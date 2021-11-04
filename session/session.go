package session

import (
	"errors"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

var (
	ErrLockedSession = errors.New("Identity has been locked")
)

// State defines the current state of the session
type State string

const (
	// Unauthenticated is the default State
	Unauthenticated State = "Unauthenticated"
	// Locked occurs when the User has 5 consecutive failed login attempts. The User must now go through the Recovery flow
	Locked State = "Locked"
	// Authenticated occurs when the User has successfully authenticated
	Authenticated State = "Authenticated"
)

// Session defines the session model
//
// This will persist throughout
type Session struct {
	// ID defines the unique id for the session
	ID uuid.UUID `json:"id" gorm:"not null" validate:"required"`
	// Token can be used by API clients to fetch current session by passing token in `X-Session-Token` Header
	//
	// Will only be provided once to the client and that's on successful login. This can occur in two flows: Login and Registration
	Token string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// State defines the current state of the session
	State State `json:"state" gorm:"not null;default:Unauthenticated" validate:"required"`
	// CreatedAt defines when the session was created
	CreatedAt time.Time `json:"created_at" gorm:"index;not null;default:current_timestamp" validate:"required"`
	// ExpiresAt defines the expiration of the session. This'll only be applicable when `State` is `Authenticated`
	ExpiresAt time.Time `json:"expires_at" validate:"required_if=State Authenticated"`
	// AuthenticatedAt defines the time when user was successfully logged in
	AuthenticatedAt time.Time `json:"authenticated_at" validate:"required_if=State Authenticated"`
	// CredentialMethods defines the list of credentials used to authenticate the user
	CredentialMethods CredentialMethods `json:"credential_methods" gorm:"type:json" validate:"required_if=State Authenticated"`

	// IdentityID defines the ID of the User that the session belongs to
	IdentityID *uuid.UUID `json:"-" validate:"required_if=State Authenticated"`
	// Identity is the identity, if any, that the session belongs to
	Identity *identity.Identity `json:"identity,omitempty" validate:"required_if=State Authenticated"`
}

type Repository interface {
	// Create creates a new Session
	Create(newSession Session) (*Session, error)
	// Get retrieves a session via ID
	Get(id uuid.UUID) (*Session, error)
	// GetByToken retrieves a session via Token
	GetByToken(token string) (*Session, error)
	// Update updates a session
	Update(updateSession Session) (*Session, error)
	// Delete deletes a session via ID
	Delete(id uuid.UUID) error
	// DeleteAllIdentity deletes all the session that belongs to an identity
	DeleteAllIdentity(identityID uuid.UUID) error
}
	//
	// TODO: Determine whether this is necessary or not
	Identity *identity.Identity `json:"identity,omitempty"`
	// Contacts are contact methods be it
	// email, sms, etc.
	Contacts []contact.Contact `json:"contacts,omitempty"`
}

func NewUnauthenticated() (*Session, error) {
	id, err := uuid.NewV4()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_FailedUUID", "Failed to generate uuid", nil)
	}
	token, err := nanoid.New()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Session_FailedNanoID", "Failed to generate nano id", nil)
	}

	now := time.Now()
	return &Session{
		ID:        id,
		Token:     token,
		CreatedAt: now,
		State:     Unauthenticated,
	}, nil
}

func NewAuthenticated(identity identity.Identity, methods ...credential.CredentialType) (*Session, error) {
	newSession, err := NewUnauthenticated()
	if err != nil {
		return nil, err
	}

	for _, method := range methods {
		newSession.AddCredential(method)
	}
	if err := newSession.Authenticate(identity); err != nil {
		return nil, err
	}
	return newSession, nil
}

func (s *Session) AddCredential(method credential.CredentialType) error {
	if s.State == Locked {
		return internal.NewServiceClientError(nil, "Session_FailedUpdate", "Account has been locked. Reset password to unlock account", map[string]interface{}{
			"Session": s,
		})
	}
	s.CredentialMethods = append(s.CredentialMethods, CredentialMethod{
		Method:   method,
		IssuedAt: time.Now(),
	})
	return nil
}

func (s *Session) Authenticate(identity identity.Identity) error {
	if s.State == Locked {
		return internal.NewServiceClientError(nil, "Session_FailedUpdate", "Account has been locked. Reset password to unlock account", map[string]interface{}{
			"Identity": identity,
			"Session":  s,
		})
	}
	cfg := config.Get()
	now := time.Now()
	s.State = Authenticated
	s.AuthenticatedAt = now
	s.ExpiresAt = s.AuthenticatedAt.Add(cfg.Session.Lifetime)
	s.IdentityID = &identity.ID
	s.Identity = &identity
	return nil
}

func (s *Session) Authenticated() bool {
	if s.State == Authenticated && s.ExpiresAt.After(time.Now()) && s.IdentityID != nil && s.Identity != nil {
		return true
	}
	return false
}
