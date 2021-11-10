package service

import (
	"log"

	"github.com/RagOfJoes/idp/flow/recovery"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
)

type service struct {
	r   recovery.Repository
	cs  credential.Service
	cos contact.Service
}

func NewRecoveryService(r recovery.Repository, cs credential.Service, cos contact.Service) recovery.Service {
	return &service{
		r:   r,
		cs:  cs,
		cos: cos,
	}
}

func (s *service) New(requestURL string) (*recovery.Flow, error) {
	newFlow, err := recovery.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new recovery flow")
	}
	return created, nil
}

func (s *service) Find(id string) (*recovery.Flow, error) {
	if id == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowIDOrRecoverID(id)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	switch flow.Status {
	case recovery.IdentifierPending:
		if flow.FlowID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
		}
	case recovery.LinkPending:
		if flow.RecoverID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
		}
	}
	return flow, nil
}

func (s *service) SubmitIdentifier(flow recovery.Flow, payload recovery.IdentifierPayload) (*recovery.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if flow.Status != recovery.IdentifierPending {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", recovery.ErrInvalidIdentifierPaylod)
	}

	credential, err := s.cs.FindPasswordWithIdentifier(payload.Identifier)
	if err != nil {
		flow.Fail()
		if _, err := s.r.Update(flow); err != nil {
			// TODO: Capture Error Here
			log.Print(err)
		}
		// Wrap error with internal code
		return nil, internal.NewErrorf(internal.ErrorCodeInternal, "%v", err)
	}
	// Update flow to LinkPending
	if err := flow.LinkPending(credential.IdentityID); err != nil {
		return nil, err
	}
	updated, err := s.r.Update(flow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update recovery flow: %s", flow.ID)
	}
	return updated, nil
}

func (s *service) SubmitUpdatePassword(flow recovery.Flow, payload recovery.SubmitPayload) (*recovery.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
	}
	if flow.Status != recovery.LinkPending {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", recovery.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, err.Error())
	}

	// Update password
	_, err := s.cs.UpdatePassword(*flow.IdentityID, payload.Password)
	if err != nil {
		return nil, err
	}
	// Complete flow
	flow.Complete()
	updated, err := s.r.Update(flow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update recovery flow: %s", flow.ID)
	}
	return updated, nil
}
