package service

import (
	"context"

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

func (s *service) New(ctx context.Context, requestURL string) (*registration.Flow, error) {
	newFlow, err := registration.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(ctx, *newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new registration flow")
	}
	return created, nil
}

func (s *service) Find(ctx context.Context, flowID string) (*registration.Flow, error) {
	if flowID == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowID(ctx, flowID)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	return flow, nil
}

func (s *service) Submit(ctx context.Context, flow registration.Flow, payload registration.Payload) (*identity.Identity, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
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
	newUser, err := s.is.Create(ctx, tempIdentity, payload.Username, payload.Password)
	if err != nil {
		return nil, err
	}

	// Run multiple actions concurrently
	// - Use Contact Service to create new contact for user
	// - Use Credential Service to create new password credential for user
	var eg errgroup.Group
	eg.Go(func() error {
		vc, err := s.cos.Add(ctx, []contact.Contact{
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
		cr, err := s.cs.CreatePassword(ctx, newUser.ID, payload.Password, []credential.Identifier{
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
		s.is.Delete(ctx, newUser.ID.String(), true)
		return nil, err
	}
	// Complete the flow
	flow.Complete()
	if _, err := s.r.Update(ctx, flow); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update registration flow: %s", flow.ID)
	}
	return newUser, nil
}
