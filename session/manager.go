package session

import (
	"context"
	"encoding/gob"
	"net/http"
	"sync"
	"time"

	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/alexedwards/scs/v2"
)

var (
	registerSessionOnce sync.Once
)

type Manager struct {
	*scs.SessionManager
}

func NewManager() (*Manager, error) {
	cfg := config.Get()
	secure := cfg.Environment == config.Production
	cookieName := cfg.Session.CookieName
	lifetime := cfg.Session.Lifetime
	registerSessionOnce.Do(func() {
		gob.Register(Session{})
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

func (m *Manager) Insert(ctx context.Context, user *identity.Identity, credentialTypes []credential.CredentialType) (*Session, error) {
	// Separate contacts from the identity
	// to ease readability
	vc := user.Contacts
	user.Contacts = nil

	newSession, err := New(time.Now().Add(m.Lifetime), *user, credentialTypes, vc)
	if err != nil || newSession == nil {
		return nil, err
	}
	// Trim session that will be inserted into session store
	ins := *newSession
	ins.Identity = nil
	ins.Contacts = []contact.Contact{}
	m.Put(ctx, "sess", ins)
	return newSession, nil
}

func (m *Manager) Retrieve(ctx context.Context, strict bool) *Session {
	// Look if context has an auth session
	session, ok := m.Get(ctx, "sess").(Session)
	if !ok {
		return nil
	}
	// Disregard validity of session if strict is false
	if !strict {
		return &session
	}
	// Run checks to make sure session is valid
	if session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	return &session
}
