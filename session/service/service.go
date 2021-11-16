package service

import (
	"context"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/session"
	"github.com/gofrs/uuid"
)

type service struct {
	r session.Repository
}

func NewSessionService(r session.Repository) session.Service {
	return &service{
		r: r,
	}
}

func (s *service) New(ctx context.Context, newSession session.Session) (*session.Session, error) {
	if err := validate.Check(newSession); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSession)
	}
	created, err := s.r.Create(ctx, newSession)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new session")
	}
	return created, nil
}

func (s *service) FindByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	found, err := s.r.Get(ctx, id)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnauthorized, "%v", session.ErrInvalidSessionID)
	}
	if err := found.Valid(); err != nil {
		return nil, err
	}

	stripSession(found)
	return found, nil
}

func (s *service) FindByToken(ctx context.Context, token string) (*session.Session, error) {
	found, err := s.r.GetByToken(ctx, token)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSessionToken)
	}
	if err := found.Valid(); err != nil {
		return nil, err
	}

	stripSession(found)
	return found, nil
}

func (s *service) Update(ctx context.Context, currentSession session.Session) (*session.Session, error) {
	if err := validate.Check(currentSession); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSession)
	}
	updated, err := s.r.Update(ctx, currentSession)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update session: %s", currentSession.ID)
	}
	return updated, nil
}

func (s *service) Destroy(ctx context.Context, id uuid.UUID) error {
	err := s.r.Delete(ctx, id)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to delete session: %s", id)
	}
	return nil
}

func (s *service) DestroyAllIdentity(ctx context.Context, identityID uuid.UUID) error {
	err := s.r.DeleteAllIdentity(ctx, identityID)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to delete all the session for: %s", identityID)
	}
	return nil
}

func stripSession(s *session.Session) {
	// Just incase Identity was left over, make sure to remove it before sending back to client
	if s.State == session.Unauthenticated {
		s.IdentityID = nil
		s.Identity = nil
	}
}
