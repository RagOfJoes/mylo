package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/nanoid"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/RagOfJoes/idp/validate"
)

var (
	errInvalidFlowID = idp.NewServiceClientError(nil, "registration_flowid_invalid", "Invalid or expired flow id provided", nil)
	errNanoIDGen     = func() error {
		_, file, line, _ := runtime.Caller(1)
		return idp.NewServiceInternalError(file, line, "registration_nanoid_gen", "Failed to generate new nanoid token")
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
	action := fmt.Sprintf("/registration/%s", fid)
	expire := time.Now().Add(time.Minute * 10)
	form := generateForm(action)
	n, err := s.r.Create(registration.Registration{
		FlowID:     fid,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		return nil, idp.NewServiceClientError(err, "registration_init", "Failed to create new Registration", nil)
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
	flow, err := s.Find(flowID)
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
	chanErr := make(chan error)
	go func() {
		defer close(chanErr)
		// 4. Create a new verifiable contact with email provided
		vc, err := s.cos.Add([]contact.VerifiableContact{
			{
				IdentityID: newUser.ID,
				State:      contact.Sent,
				Value:      payload.Email,
			},
		}...)
		if err != nil {
			return
		}
		// 5. Create a new password credential
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
			chanErr <- err
			return
		}
		// 6. Append VerifiableContacts and Credentials to Identity
		// This is to mimic the behavior for the all subsequent flows
		var vcf []contact.VerifiableContact
		for _, c := range vc {
			vcf = append(vcf, *c)
		}
		newUser.VerifiableContacts = vcf
		newUser.Credentials = append(newUser.Credentials, *cr)
	}()
	// 7. If an error has occurred while adding new verifiable
	// contact or new password credential then delete identity
	if err := <-chanErr; err != nil {
		s.is.Delete(newUser.ID.String(), true)
		return nil, err
	}
	// 8. If everything passes then delete flow
	if err := s.r.Delete(flow.ID); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, idp.NewServiceInternalError(file, line, "registration_delete_fail", "Failed to delete registration flow")
	}
	return newUser, nil
}
