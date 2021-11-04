package service

import (
	"runtime"

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
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_InvalidSession", "Invalid session provided", map[string]interface{}{
			"Session": newSession,
		})
	}

	created, err := s.r.Create(newSession)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_FailedCreate", "Failed to create new session", nil)
	}
	return created, nil
}

func (s *service) FindByID(id uuid.UUID) (*session.Session, error) {
	found, err := s.r.Get(id)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_InvalidID", "Invalid session id provided", map[string]interface{}{
			"SessionID": id,
		})
	}

	stripSession(found)
	return found, nil
}

func (s *service) FindByToken(token string) (*session.Session, error) {
	found, err := s.r.GetByToken(token)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_InvalidToken", "Invalid session token provided", map[string]interface{}{
			"SessionToken": token,
		})
	}

	stripSession(found)
	return found, nil
}

func (s *service) Update(currentSession session.Session) (*session.Session, error) {
	if err := validate.Check(currentSession); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_InvalidSession", "Invalid session provided", map[string]interface{}{
			"Session": currentSession,
		})
	}
	updated, err := s.r.Update(currentSession)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_FailedUpdate", "Failed to update session", map[string]interface{}{
			"Session": currentSession,
		})
	}
	return updated, nil
}

func (s *service) Destroy(id uuid.UUID) error {
	return s.r.Delete(id)
}

func (s *service) DestroyAllIdentity(identityID uuid.UUID) error {
	return s.r.DeleteAllIdentity(identityID)
}

func stripSession(s *session.Session) {
	// Just incase Identity was left over, make sure to remove it before sending back to client
	if s.State == session.Unauthenticated {
		s.IdentityID = nil
		s.Identity = nil
	}
}
