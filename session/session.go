package session

import (
	"time"

	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

// Session is a session object related to auth
// state of the User
type Session struct {
	ID uuid.UUID `json:"id"`
	// IssuedAt defines the session was created
	IssuedAt time.Time `json:"issued_at"`
	// ExpiresAt defines the expiration of the session
	ExpiresAt time.Time `json:"expires_at"`
	// AuthenticatedAt defines the time when user was successfully
	// logged meaning all requirements were met
	AuthenticatedAt time.Time `json:"authenticated_at"`
	// IdentityID is just that
	IdentityID uuid.UUID `json:"-"`
	// Credentials used to authenticate user
	// This will never be passed on to the client
	Credentials []credential.CredentialType `json:"-"`
	// Identity is the identity, if any, that the session belongs to
	//
	// TODO: Determine whether this is necessary or not
	Identity *identity.Identity `json:"identity,omitempty"`
	// Contacts are contact methods be it
	// email, sms, etc.
	Contacts []contact.Contact `json:"contacts,omitempty"`
}

// New creates a new session
func New(exp time.Time, i identity.Identity, c []credential.CredentialType, v []contact.Contact) (*Session, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	newSession := Session{
		ID:              uid,
		IssuedAt:        time.Now(),
		AuthenticatedAt: time.Now(),
		// ExpiresAt 2 weeks
		ExpiresAt: exp,

		IdentityID:  i.ID,
		Credentials: c,

		Identity: &i,
		Contacts: v,
	}
	return &newSession, nil
}
