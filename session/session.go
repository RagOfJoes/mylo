package session

import (
	"context"
	"errors"
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

var (
	ErrLockedSession       = errors.New("Identity has been locked")
	ErrInvalidSession      = errors.New("Invalid session provided")
	ErrSessionNotFound     = errors.New("No active session found")
	ErrInvalidSessionID    = errors.New("Invalid session id provided")
	ErrInvalidSessionToken = errors.New("Invalid session token provided")
)

// TODO: Check if Locked state is at all useful here
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
// A Session will only be assigned when one of the following occur:
// - A User attempts to access a protected resource without being Authenticated (ie. /verificaition)
// - A User successfully passes first factor (ie. Login Flow via Password)
// - (If MFA is active) A User successfully passes second factor (ie. TOTP via authenticator app)
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
	ExpiresAt *time.Time `json:"expires_at" validate:"required_if=State Authenticated"`
	// AuthenticatedAt defines the time when user was successfully logged in
	AuthenticatedAt *time.Time `json:"authenticated_at" validate:"required_if=State Authenticated"`
	// CredentialMethods defines the list of credentials used to authenticate the user
	CredentialMethods CredentialMethods `json:"credential_methods,omitempty" gorm:"type:json;default:null" validate:"required_if=State Authenticated"`

	// IdentityID defines the ID of the User that the session belongs to
	IdentityID *uuid.UUID `json:"-" validate:"required_if=State Authenticated"`
	// Identity is the identity, if any, that the session belongs to
	Identity *identity.Identity `json:"identity,omitempty" validate:"required_if=State Authenticated"`
}

type Repository interface {
	// Create creates a new Session
	Create(ctx context.Context, newSession Session) (*Session, error)
	// Get retrieves a session via ID
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	// GetByToken retrieves a session via Token
	GetByToken(ctx context.Context, token string) (*Session, error)
	// Update updates a session
	Update(ctx context.Context, updateSession Session) (*Session, error)
	// Delete deletes a session via ID
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteAllIdentity deletes all the session that belongs to an identity
	DeleteAllIdentity(ctx context.Context, identityID uuid.UUID) error
}

type Service interface {
	// New creates a session
	New(ctx context.Context, newSession Session) (*Session, error)
	// FindByID finds a session via ID
	FindByID(ctx context.Context, id uuid.UUID) (*Session, error)
	// FindByToken finds a session via Token
	FindByToken(ctx context.Context, token string) (*Session, error)
	// Update updates a session
	Update(ctx context.Context, updateSession Session) (*Session, error)
	// Destroy deletes session
	Destroy(ctx context.Context, id uuid.UUID) error
	// DestroyAllIdentity deletes all the session that belongs to an identity
	DestroyAllIdentity(ctx context.Context, identityID uuid.UUID) error
}

func NewUnauthenticated() (*Session, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate uuid")
	}
	token, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}
	now := time.Now()
	return &Session{
		ID:        id,
		CreatedAt: now,
		Token:     token,
		State:     Unauthenticated,
	}, nil
}

func NewAuthenticated(identity identity.Identity, methods ...credential.CredentialType) (*Session, error) {
	newSession, err := NewUnauthenticated()
	if err != nil {
		return nil, err
	}
	if err := newSession.Authenticate(identity, methods...); err != nil {
		return nil, err
	}
	return newSession, nil
}

func (s *Session) Valid() error {
	if err := validate.Check(s); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", ErrInvalidSession)
	}
	if s.State == Authenticated && s.ExpiresAt == nil {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", ErrInvalidSession)
	}
	if s.ExpiresAt != nil && s.ExpiresAt.Before(time.Now()) {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", ErrInvalidSession)
	}
	return nil
}

func (s *Session) AddCredential(method credential.CredentialType) error {
	if s.State == Locked {
		return internal.NewErrorf(internal.ErrorCodeUnauthorized, "Account has been locked. Reset password to unlock account")
	}
	s.CredentialMethods = append(s.CredentialMethods, CredentialMethod{
		Method:   method,
		IssuedAt: time.Now(),
	})
	return nil
}

func (s *Session) Authenticate(identity identity.Identity, methods ...credential.CredentialType) error {
	if s.State == Locked {
		return internal.NewErrorf(internal.ErrorCodeUnauthorized, "Account has been locked. Reset password to unlock account")
	}
	for _, method := range methods {
		if err := s.AddCredential(method); err != nil {
			return err
		}
	}

	cfg := config.Get()
	now := time.Now()
	expire := now.Add(cfg.Session.Lifetime)
	s.State = Authenticated
	s.ExpiresAt = &expire
	s.AuthenticatedAt = &now
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
