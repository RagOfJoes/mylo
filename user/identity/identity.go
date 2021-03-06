package identity

import (
	"context"
	"errors"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/gofrs/uuid"
)

var (
	ErrUsernameProfane           = errors.New("Username must not contain any profanity")
	ErrInvalidIdentifierPassword = errors.New("Invalid identifier(s) or password provided")
)

// Identity defines the base Identity model
type Identity struct {
	internal.BaseSoftDelete
	Avatar    string `json:"avatar" gorm:"size:1024;" validate:"max=1024"`
	FirstName string `json:"first_name" gorm:"size:64" validate:"max=64,alphanumunicode"`
	LastName  string `json:"last_name" gorm:"size:64" validate:"max=64,alphanumunicode"`
	// Email is the primary email that will be used for account
	// security related notifications
	Email string `json:"email" gorm:"uniqueIndex;not null;" validate:"email,required"`

	Credentials []credential.Credential `json:"-"`
	Contacts    []contact.Contact       `json:"contacts"`
}

type Repository interface {
	// Create creates a new identity
	Create(ctx context.Context, newIdentity Identity) (*Identity, error)
	// Get retrieves an identity with id
	Get(ctx context.Context, id uuid.UUID, critical bool) (*Identity, error)
	// GetWithIdentifier retrieves an identity with identifier
	GetWithIdentifier(ctx context.Context, identifier string, critical bool) (*Identity, error)
	// Update updates an identity
	Update(ctx context.Context, updateIdentity Identity) (*Identity, error)
	// Delete deletes an identity
	Delete(ctx context.Context, id uuid.UUID, permanent bool) error
}

type Service interface {
	// Create creates an identity
	Create(ctx context.Context, user Identity, username string, password string) (*Identity, error)
	// Find finds an identity with either its id or an identifier
	Find(ctx context.Context, id string) (*Identity, error)
	// Delete deletes an identity
	Delete(ctx context.Context, id string, permanent bool) error
}
