package credential

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

const (
	// CredentialTypes
	OIDC     CredentialType = "oidc"
	Password CredentialType = "password"
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
	Values json.RawMessage `gorm:"not null;type:jsonb" validate:"required"`

	IdentityID  uuid.UUID `gorm:"index;not null" validate:"required,uuid4"`
	Identifiers []Identifier
}

// CredentialType defines a Credential Type
type CredentialType string

// CredentialPassword defines the structure for
// a type password's Values field
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
	Create(Credential) (*Credential, error)
	GetWithIdentifier(CredentialType, string) (*Credential, error)
	GetWithIdentityID(CredentialType, uuid.UUID) (*Credential, error)
	Update(Credential) (*Credential, error)
	Delete(uuid.UUID) error
}

