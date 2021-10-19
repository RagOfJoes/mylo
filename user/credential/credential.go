package credential

import (
	"time"

	"github.com/gofrs/uuid"
)

// Credential can be a Password, OTP, Device Code,
// Magic Link, etc.
//
// A User can only have only have one Credential type.
// Ie: 1 Password Credential, 1 OTP Credential, etc.
type Credential struct {
	ID        uuid.UUID  `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required,uuid4"`
	CreatedAt time.Time  `gorm:"index;not null;default:current_timestamp" validate:"required"`
	UpdatedAt *time.Time `gorm:"index;default:null"`

	Type CredentialType `gorm:"index;not null" validate:"required,oneof='oidc' 'password'"`
	// Depending on the type values stored in here
	// will differ. For example:
	// type: oidc
	// values:
	// 		- provider: google
	//		- sub: 9s988s...
	Values string `gorm:"not null;type:json" validate:"required"`

	IdentityID  uuid.UUID    `gorm:"index;not null" validate:"required,uuid4"`
	Identifiers []Identifier `gorm:"constraint:OnDelete:CASCADE"`
}

// CredentialType defines a Credential Type
type CredentialType string

const (
	// CredentialTypes
	OIDC     CredentialType = "oidc"
	Password CredentialType = "password"
)

// CredentialPassword defines the structure for
// a type password's Values field
// TODO: Look into adding more fields here like
// password score, encoding format??, etc.
type CredentialPassword struct {
	HashedPassword string `json:"hashed_password"`
}

// CredentialOIDC defines the structure for
// a type oidc's Values field
type CredentialOIDC struct {
	Provider string `json:"provider"`
	Sub      string `json:"sub"`
}

type Repository interface {
	// Create creates a new credential
	Create(newCredential Credential) (*Credential, error)
	// GetIdentifier retrieves an identifier
	GetIdentifier(identifier string) (*Identifier, error)
	// GetWithIdentifier retrieves a credential with an identifier
	GetWithIdentifier(credentialType CredentialType, identifier string) (*Credential, error)
	// GetWithIdentityID retrieves a credential with an identity id
	GetWithIdentityID(credentialType CredentialType, identityID uuid.UUID) (*Credential, error)
	// Update updates a credential
	Update(updateCredential Credential) (*Credential, error)
	// Delete deletes a credential via id
	Delete(id uuid.UUID) error
}

type Service interface {
	// CreatePassword creates a password credential
	CreatePassword(identityID uuid.UUID, password string, identifiers []Identifier) (*Credential, error)
	// ComparePassword compares a password credential
	ComparePassword(identityID uuid.UUID, password string) error
	// FindPasswordWithIdentifier finds a password with an identifier
	FindPasswordWithIdentifier(Identifier string) (*Credential, error)
	// UpdatePassword updates a password credential
	UpdatePassword(identityID uuid.UUID, newPassword string) (*Credential, error)
}
