package credential

import (
	"time"

	"github.com/gofrs/uuid"
)

// IdentifierType defines an Identifier Type
type IdentifierType string

const (
	// IdentifierTypes
	Email    IdentifierType = "email"
	Username IdentifierType = "username"
)

// Identifier is a unique value that a User will use for authentication
// Each Identifier can belong to multiple Credentials
type Identifier struct {
	// ID just a unique identifier
	ID uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required,uuid4"`
	// CreatedAt meta data about Identifier
	CreatedAt time.Time `gorm:"index;not null;default:current_timestamp" validate:"required"`
	// UpdatedAt meta data about Identifier
	UpdatedAt *time.Time `gorm:"index;default:null"`

	// ForeignKey ID to User that an Identifier
	// is linked to
	CredentialID uuid.UUID `gorm:"not null;index;type:uuid" validate:"required,uuid4"`

	// Type of Identifier. Supported are: email and username
	Type IdentifierType `gorm:"not null;" validate:"required,oneof:'email' 'username'"`
	// Value of Identifier. Has to be unique for all types
	Value string `gorm:"not null;uniqueIndex" validate:"required"`
}
