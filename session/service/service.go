package service

import (
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

func (s *service) New(newSession session.Session) (*session.Session, error) {
	if err := validate.Check(newSession); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSession)
	}
	created, err := s.r.Create(newSession)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new session")
	}
	return created, nil
}

func (s *service) FindByID(id uuid.UUID) (*session.Session, error) {
	found, err := s.r.Get(id)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnauthorized, "%v", session.ErrInvalidSessionID)
	}
	if err := found.Valid(); err != nil {
		return nil, err
	}

	stripSession(found)
	return found, nil
}

func (s *service) FindByToken(token string) (*session.Session, error) {
	found, err := s.r.GetByToken(token)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSessionToken)
	}
	if err := found.Valid(); err != nil {
		return nil, err
	}

	stripSession(found)
	return found, nil
}

func (s *service) Update(currentSession session.Session) (*session.Session, error) {
	if err := validate.Check(currentSession); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", session.ErrInvalidSession)
	}
	updated, err := s.r.Update(currentSession)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update session: %s", currentSession.ID)
	}
	return updated, nil
}

func (s *service) Destroy(id uuid.UUID) error {
	err := s.r.Delete(id)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to delete session: %s", id)
	}
	return nil
}

func (s *service) DestroyAllIdentity(identityID uuid.UUID) error {
	err := s.r.DeleteAllIdentity(identityID)
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
