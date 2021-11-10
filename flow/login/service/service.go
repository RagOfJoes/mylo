package service

import (
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

func (s *service) New(requestURL string) (*login.Flow, error) {
	newFlow, err := login.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new login flow")
	}
	return created, nil
}

func (s *service) Find(flowID string) (*login.Flow, error) {
	if flowID == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", login.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowID(flowID)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", login.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	return flow, nil
}

// TODO: Add delay to mitigate time attacks
func (s *service) Submit(flow login.Flow, payload login.Payload) (*identity.Identity, error) {
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// Retrieve identity based on identifier provided
	id, err := s.is.Find(payload.Identifier)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// Use retrieved identity ID to then retrieve
	// the hashed password credential then decode it
	// and compare provided password attempt
	if err := s.cs.ComparePassword(id.ID, payload.Password); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", login.ErrInvalidPaylod)
	}
	// If everything passes then delete flow
	// TODO: Capture error, if any, here
	go func() {
		s.r.Delete(flow.ID)
	}()
	return id, nil
}
