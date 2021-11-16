package service

import (
	"context"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
)

type service struct {
	r   login.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewLoginService(r login.Repository, cos contact.Service, cs credential.Service, is identity.Service) login.Service {
	return &service{
		r:   r,
		cs:  cs,
		is:  is,
		cos: cos,
	}
}

func (s *service) New(ctx context.Context, requestURL string) (*login.Flow, error) {
	newFlow, err := login.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(ctx, *newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new login flow")
	}
	return created, nil
}

func (s *service) Find(ctx context.Context, flowID string) (*login.Flow, error) {
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

// TODO: Add delay to mitigate time attacks
func (s *service) Submit(ctx context.Context, flow login.Flow, payload login.Payload) (*identity.Identity, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// Retrieve identity based on identifier provided
	id, err := s.is.Find(ctx, payload.Identifier)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// Use retrieved identity ID to then retrieve
	// the hashed password credential then decode it
	// and compare provided password attempt
	if err := s.cs.ComparePassword(ctx, id.ID, payload.Password); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// Complete the flow
	flow.Complete()
	if _, err := s.r.Update(ctx, flow); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update login flow: %s", flow.ID)
	}
	return id, nil
}
