package session

import (
	"context"
	"encoding/gob"
	"net/http"
	"sync"
	"time"

	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/alexedwards/scs/v2"
	"github.com/gofrs/uuid"
)

var (
	registerSessionOnce sync.Once
)

type Manager struct {
	*scs.SessionManager
}

// AuthSession is a session object related to auth
// state of the User
type AuthSession struct {
	ID              uuid.UUID         `json:"id"`
	Active          bool              `json:"active"`
	IssuedAt        time.Time         `json:"issued_at"`
	ExpiresAt       time.Time         `json:"expires_at"`
	AuthenticatedAt time.Time         `json:"authenticated_at"`
	Identity        identity.Identity `json:"identity"`
	// Credentials used to authenticate user
	Credentials []credential.CredentialType `json:"-"`
}

func New(exp time.Time, i identity.Identity, c []credential.CredentialType) (*AuthSession, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	newSession := AuthSession{
		ID:              uid,
		Active:          true,
		IssuedAt:        time.Now(),
		AuthenticatedAt: time.Now(),
		// ExpiresAt 2 weeks
		ExpiresAt: exp,

		Identity:    i,
		Credentials: c,
	}
	return &newSession, nil
}

// Manager
//
func NewManager(secure bool, cookieName string, lifetime time.Duration) (*Manager, error) {
	registerSessionOnce.Do(func() {
		gob.Register(AuthSession{})
	})

	manager := scs.New()
	manager.Lifetime = lifetime
	manager.Cookie.Name = cookieName
	if secure {
		manager.Cookie.Secure = true
		manager.Cookie.Persist = true
		manager.Cookie.HttpOnly = true
		manager.Cookie.SameSite = http.SameSiteLaxMode
	}
	return &Manager{manager}, nil
}

func (m *Manager) PutAuth(ctx context.Context, i identity.Identity, c []credential.CredentialType) error {
	newSession, err := New(time.Now().Add(m.Lifetime), i, c)
	if err != nil {
		return err
	}
	m.Put(ctx, "auth", newSession)
	return nil
}

func (m *Manager) GetAuth(ctx context.Context, strict bool) *AuthSession {
	// Look if context has an auth session
	sess, ok := m.Get(ctx, "auth").(AuthSession)
	if !ok {
		return nil
	}

	// Disregard status or expiration of session
	// if strict is false
	if !strict {
		return &sess
	}

	// Run checks to make sure session is valid
	if !sess.Active || sess.ExpiresAt.Before(time.Now()) {
		return nil
	}

	return &sess
}
