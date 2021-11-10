package contact

import (
	"errors"
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/gofrs/uuid"
)

var (
	ErrContactDoesNotExist = errors.New("Contact does not exist")
)

// Type defines the type of contact
type Type string

const (
	// Default means that this contact can be used as a valid identifier
	// pair with a credential to log user in
	Default Type = "Default"
	// Backup means that this contact can be used to recover account
	Backup Type = "Backup"
)

// State defines the current state of verification for this particular contact
type State string

const (
	// Sent means the verification link was sent to an out of band communication provider
	// ie: Email
	Sent State = "Sent"
	// Completed means the contact has successfully been verified
	Completed State = "Completed"
)

type Contact struct {
	internal.Base
	// Verified flag
	Verified bool `json:"verified" gorm:"default:false"`
	// VerifiedAt is the verification date
	VerifiedAt *time.Time `json:"verified_at" gorm:"default:null"`

	// Type defines the type of contact
	//
	// Any type besides default will be ignored
	// if state != "Complete" or if verified is false.
	Type Type `json:"type" gorm:"index;not null;default:default"`
	// State defines the current state of verification for this particular contact
	//
	// "Sent" means the verification link, email, sms, etc.
	// was sent.
	// "Completed" means the verification process been fulfilled
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
	// Find finds a single contact based on the value provided
	Find(value string) (*Contact, error)
	// Add adds a single or a collection of contacts. This should ideally
	// merge new and old Contact that a user owns
	Add(contacts ...Contact) ([]Contact, error)
}
