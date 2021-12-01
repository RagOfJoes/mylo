package service

import (
	"context"
	"fmt"

	"github.com/RagOfJoes/mylo/flow/registration"
	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/RagOfJoes/mylo/user/identity"
	"golang.org/x/sync/errgroup"
)

type service struct {
	cfg config.Configuration

	r   registration.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewRegistrationService(cfg config.Configuration, r registration.Repository, cos contact.Service, cs credential.Service, is identity.Service) registration.Service {
	return &service{
		cfg: cfg,

		r:   r,
		cs:  cs,
		is:  is,
		cos: cos,
	}
}

func (s *service) New(ctx context.Context, requestURL string) (*registration.Flow, error) {
	serverURL := fmt.Sprintf("%s/%s", s.cfg.Server.URL, s.cfg.Registration.URL)
	newFlow, err := registration.New(s.cfg.Registration.Lifetime, serverURL, requestURL)
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
	tempIdentity := identity.New(payload.FirstName, payload.LastName, payload.Email)
	newUser, err := s.is.Create(ctx, tempIdentity, payload.Username, payload.Password)
	if err != nil {
		return nil, err
	}

	// Run multiple actions concurrently
	// - Use Contact Service to create new contact for user
	// - Use Credential Service to create new password credential for user
	var eg errgroup.Group
	eg.Go(func() error {
		newContact := contact.New(newUser.ID, payload.Email)
		vc, err := s.cos.Add(ctx, []contact.Contact{
			newContact,
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
