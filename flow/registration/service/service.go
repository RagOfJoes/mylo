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
	errInvalidFlowID = func(src error, f string) error {
		return internal.NewServiceClientError(src, "Registration_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"FlowID": f,
		})
	}
	errInvalidFlow = func(src error, f registration.Flow) error {
		return internal.NewServiceClientError(src, "Registration_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Flow": f,
		})
	}
	errInvalidPayload = func(src error, f registration.Flow, p registration.Payload) error {
		return internal.NewServiceClientError(src, "Registration_InvalidPayload", "Invalid identifier(s) or password provided", map[string]interface{}{
			"Flow":    f,
			"Payload": p,
		})
	}
	// Internal Errors
	errNanoIDGen = func(src error) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Registration_FailedNanoID", "Failed to generate nano id", nil)
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

func (s *service) New(requestURL string) (*registration.Flow, error) {
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err)
	}

	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Registration.URL, fid)
	expire := time.Now().Add(cfg.Registration.Lifetime)
	form := generateForm(action)
	n, err := s.r.Create(registration.Flow{
		FlowID:     fid,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Registration_FailedCreate", "Failed to create new registration flow", map[string]interface{}{
			"Flow": n,
		})
	}
	return n, nil
}

func (s *service) Find(flowID string) (*registration.Flow, error) {
	if flowID == "" {
		return nil, errInvalidFlowID(nil, flowID)
	}
	// Try to get Flow
	f, err := s.r.GetByFlowID(flowID)
	// Check for error or empty flow
	if err != nil || f == nil {
		return nil, errInvalidFlowID(err, flowID)
	}
	// Check for expired flow
	if f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlow(nil, *f)
	}
	return f, nil
}

func (s *service) Submit(flow registration.Flow, payload registration.Payload) (*identity.Identity, error) {
	// Validate flow
	if err := validate.Check(flow); err != nil {
		return nil, errInvalidFlow(err, flow)
	}
	// Validate payload
	if err := validate.Check(payload); err != nil {
		return nil, err
	}
	// Instantiate new identity
	tempIdentity := identity.Identity{
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
	}
	// Create new identity
	// TODO: Determine whether or not we should wrap this error
	newUser, err := s.is.Create(tempIdentity, payload.Username, payload.Password)
	if err != nil {
		return nil, err
	}

	// Run multiple actions concurrently
	// - Use Contact Service to create new contact for user
	// - Use Credential Service to create new password credential for user
	var eg errgroup.Group
	eg.Go(func() error {
		vc, err := s.cos.Add([]contact.Contact{
			{
				IdentityID: newUser.ID,
				State:      contact.Sent,
				Value:      payload.Email,
			},
		}...)
		if err != nil {
			return errInvalidPayload(err, flow, payload)
		}

		// Append new contact to instantiated identity
		newUser.Contacts = append(newUser.Contacts, vc...)
		return nil
	})
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
		// TODO: Determine whether or not we should wrap this error
		if err != nil {
			return err
		}
		// Append new credential to instantited identity
		newUser.Credentials = append(newUser.Credentials, *cr)
		return nil
	})
	// Check if any of the concurrent actions error'd out and if so
	// perform a cascade delete on the user
	if err := eg.Wait(); err != nil {
		s.is.Delete(newUser.ID.String(), true)
		return nil, err
	}
	// Delete service in background
	go func() {
		s.r.Delete(flow.ID)
	}()
	return newUser, nil
}
