package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"golang.org/x/sync/errgroup"
)

var (
	errInvalidFlowID = internal.NewServiceClientError(nil, "registration_flowid_invalid", "Invalid or expired flow id provided", nil)
	errNanoIDGen     = func() error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "registration_nanoid_gen", "Failed to generate new nanoid token")
	}
)

type service struct {
	r   registration.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewRegistrationService(r registration.Repository, cos contact.Service, cs credential.Service, is identity.Service) registration.Service {
	return &service{
		r:   r,
		cs:  cs,
		is:  is,
		cos: cos,
	}
}

func (s *service) New(requestURL string) (*registration.Registration, error) {
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}

	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Registration.URL, fid)
	expire := time.Now().Add(cfg.Registration.Lifetime)
	form := generateForm(action)
	n, err := s.r.Create(registration.Registration{
		FlowID:     fid,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		return nil, internal.NewServiceClientError(err, "registration_init", "Failed to create new Registration", nil)
	}
	return n, nil
}

func (s *service) Find(flowID string) (*registration.Registration, error) {
	if flowID == "" {
		return nil, errInvalidFlowID
	}
	f, err := s.r.GetByFlowID(flowID)
	if err != nil || f == nil || f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlowID
	}
	return f, nil
}

func (s *service) Submit(flowID string, payload registration.RegistrationPayload) (*identity.Identity, error) {
	// 1. Make sure the flow is still valid
	_, err := s.Find(flowID)
	if err != nil {
		return nil, err
	}
	// 2. Validate payload provided
	if err := validate.Check(payload); err != nil {
		return nil, err
	}
	// 3. Create new Identity
	tempIdentity := identity.Identity{
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
	}
	newUser, err := s.is.Create(tempIdentity, payload.Username, payload.Password)
	if err != nil {
		return nil, err
	}
	// 4. Create error group to execute concurrent service calls
	var eg errgroup.Group
	// 5. Create verifiable contacts and append to newUser for response
	eg.Go(func() error {
		vc, err := s.cos.Add([]contact.VerifiableContact{
			{
				IdentityID: newUser.ID,
				State:      contact.Sent,
				Value:      payload.Email,
			},
		}...)
		if err != nil {
			return err
		}

		var vcf []contact.VerifiableContact
		for _, c := range vc {
			vcf = append(vcf, c)
		}
		newUser.VerifiableContacts = vcf
		return nil
	})
	// 6. Create password credential and append to newUser
	eg.Go(func() error {
		cr, err := s.cs.CreatePassword(newUser.ID, payload.Password, []credential.Identifier{
			{
				Type:  "email",
				Value: payload.Email,
			},
			{
				Type:  "username",
				Value: payload.Username,
			},
		})
		if err != nil {
			return err
		}
		newUser.Credentials = append(newUser.Credentials, *cr)
		return nil
	})
	// 7. Check if error group executed with any errors. If so then delete
	// new identity
	if err := eg.Wait(); err != nil {
		s.is.Delete(newUser.ID.String(), true)
		return nil, err
	}
	return newUser, nil
}
