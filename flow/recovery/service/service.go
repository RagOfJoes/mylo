package service

import (
	"context"
	"fmt"
	"log"

	"github.com/RagOfJoes/mylo/flow/recovery"
	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/credential"
)

type service struct {
	cfg config.Configuration

	r   recovery.Repository
	cs  credential.Service
	cos contact.Service
}

func NewRecoveryService(cfg config.Configuration, r recovery.Repository, cs credential.Service, cos contact.Service) recovery.Service {
	return &service{
		cfg: cfg,

		r:   r,
		cs:  cs,
		cos: cos,
	}
}

func (s *service) New(ctx context.Context, requestURL string) (*recovery.Flow, error) {
	serverURL := fmt.Sprintf("%s/%s", s.cfg.Server.URL, s.cfg.Recovery.URL)
	newFlow, err := recovery.New(s.cfg.Recovery.Lifetime, serverURL, requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(ctx, *newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new recovery flow")
	}
	return created, nil
}

func (s *service) Find(ctx context.Context, id string) (*recovery.Flow, error) {
	if id == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowIDOrRecoverID(ctx, id)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	switch flow.Status {
	case recovery.IdentifierPending:
		if flow.FlowID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
		}
	case recovery.LinkPending:
		if flow.RecoverID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
		}
	}
	return flow, nil
}

func (s *service) SubmitIdentifier(ctx context.Context, flow recovery.Flow, payload recovery.IdentifierPayload) (*recovery.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if flow.Status != recovery.IdentifierPending {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", recovery.ErrInvalidIdentifierPaylod)
	}

	credential, err := s.cs.FindPasswordWithIdentifier(ctx, payload.Identifier)
	if err != nil {
		flow.Fail()
		if _, err := s.r.Update(ctx, flow); err != nil {
			// TODO: Capture Error Here
			log.Print(err)
		}
		// It's backwards so we can use Is method
		return nil, internal.WrapErrorf(recovery.ErrAccountDoesNotExist, internal.ErrorCodeInternal, "%v", err)
	}
	// Update flow to LinkPending
	serverURL := fmt.Sprintf("%s/%s", s.cfg.Server.URL, s.cfg.Recovery.URL)
	if err := flow.LinkPending(serverURL, credential.IdentityID); err != nil {
		return nil, err
	}
	updated, err := s.r.Update(ctx, flow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update recovery flow: %s", flow.ID)
	}
	return updated, nil
}

func (s *service) SubmitUpdatePassword(ctx context.Context, flow recovery.Flow, payload recovery.SubmitPayload) (*recovery.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if flow.Status != recovery.LinkPending {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", internal.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, err.Error())
	}

	// Update password
	_, err := s.cs.UpdatePassword(ctx, *flow.IdentityID, payload.Password)
	if err != nil {
		return nil, err
	}
	// Complete flow
	flow.Complete()
	updated, err := s.r.Update(ctx, flow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update recovery flow: %s", flow.ID)
	}
	return updated, nil
}
