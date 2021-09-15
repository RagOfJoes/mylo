package contact

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/gofrs/uuid"
)

type Type string
type Method string
type State string

const (
	Default Type = "default"
	Backup  Type = "backup"

	Email Method = "email"

	Sent      State = "sent"
	Completed State = "completed"
)

type Contact struct {
	internal.Base
	// Verified flag
	Verified bool `json:"verified" gorm:"default:false"`
	// VerifiedAt is the verification date
	VerifiedAt *time.Time `json:"verified_at" gorm:"default:null"`

	// Type is the type of address.
	//
	// Any type besides default will be ignored
	// if state != "completed" or if verified is false.
	//
	// enum{ "default", "backup" }
	//
	// "default" means this is just a contact.
	// "backup" means this address is also a backup so user's
	// can use it for recovering accounts.
	Type Type `json:"type" gorm:"index;not null;default:default"`
	// Method is the delivery method for verification
	Method Method `json:"method" gorm:"default:email" validate:"oneof='email'"`
	// State is the current state of the verification process.
	//
	// "sent" means the verification link, email, sms, etc.
	// was sent.
	// "completed" means the verification process been fulfilled
	// by the user.
	State State `json:"state" gorm:"not null" validate:"oneof='sent' 'completed'"`
	// Value is the actual value to be verified. This can
	// be an email, phone number, etc.
	Value      string    `json:"value" gorm:"uniqueIndex;not null;" validate:"required,min=1"`
	IdentityID uuid.UUID `json:"-" gorm:"index;not null" validate:"required,uuid4"`
}

type Repository interface {
	// Create creates a new Contact
	Create(...Contact) ([]Contact, error)
	// Update updates a new Contact
	Update(Contact) (*Contact, error)
	// Get retrieves a single Contact given its
	// id
	Get(uuid.UUID) (*Contact, error)
	// GetByValue retrieves a Contact given an address
	// value
	GetByValue(string) (*Contact, error)
	// Delete deletes a single Contact
	Delete(uuid.UUID) error
	// DeleteAllUser deletes all Contact of a User
	// given an IdentityID
	DeleteAllUser(uuid.UUID) error
}

type Service interface {
	// Find finds a single contact based on
	Find(string) (*Contact, error)
	// Add adds a single or a collection of contacts. This should ideally
	// merge new and old Contact that a user owns
	Add(...Contact) ([]Contact, error)
}
