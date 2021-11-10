package service

import (
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"golang.org/x/sync/errgroup"
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
	newFlow, err := registration.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new registration flow")
	}
	return created, nil
}

func (s *service) Find(flowID string) (*registration.Flow, error) {
	if flowID == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", registration.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowID(flowID)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", registration.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	return flow, nil
}

func (s *service) Submit(flow registration.Flow, payload registration.Payload) (*identity.Identity, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", registration.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", err)
	}
	// Instantiate new identity
	// TODO: Use identity function here to create identity
	tempIdentity := identity.Identity{
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
	}
	// Create new identity
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
			return internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", registration.ErrInvalidPaylod)
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
	// If everything passes then delete flow
	// TODO: Capture error, if any, here
	go func() {
		s.r.Delete(flow.ID)
	}()
	return newUser, nil
}
