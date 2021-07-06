package contact

import (
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/gofrs/uuid"
)

type VerifiableContactType string
type VerifiableContactMethod string
type VerifiableContactState string

const (
	Default VerifiableContactType = "default"
	Backup  VerifiableContactType = "backup"

	Email VerifiableContactMethod = "email"

	Sent      VerifiableContactState = "sent"
	Completed VerifiableContactState = "completed"
)

type VerifiableContact struct {
	idp.Base
	Verified   bool       `json:"verified" gorm:"default:false"`
	VerifiedAt *time.Time `json:"verified_at" gorm:"default:null"`

	// Type is the type of address.
	//
	// Any type besides default will be ignored
	// if state != "completed" or if verified is false.
	//
	// enum{ "default", "backup" }
	//
	// "default" means this is just a verifiable address.
	// "backup" means this address is also a backup so user's
	// can use it for recovering accounts.
	Type VerifiableContactType `json:"type" gorm:"index;not null;default:default"`
	// Method is the delivery method for verification
	Method VerifiableContactMethod `json:"method"`
	// State is the current state of the verification process.
	//
	// enum{ "sent", "completed" }
	//
	// "sent" means the verification link, email, sms, etc.
	// was sent.
	// "completed" means the verification process been fulfilled
	// by the user.
	State VerifiableContactState `json:"state" gorm:"not null" validate:"oneof='sent' 'completed'"`
	// Address is the actual address to be verified. This can
	// be an email, phone number, etc.
	Address    string    `json:"address" gorm:"uniqueIndex;not null;" validate:"required,min=1"`
	IdentityID uuid.UUID `gorm:"index;not null" validate:"required,uuid4"`
}

type Repository interface {
	// Create creates a new VerifiableContact
	Create(...VerifiableContact) ([]*VerifiableContact, error)
	// Update updates a new VerifiableContact
	Update(VerifiableContact) (*VerifiableContact, error)
	// Get retrieves a single VerifiableContact given its
	// id or identity id
	Get(uuid.UUID) (*VerifiableContact, error)
	// GetByAddress retrieves a VerifiableContact given an address
	// value
	GetByAddress(string) (*VerifiableContact, error)
	// GetWithConditions retrieves all VerifiableContact given
	// a custom conditional values
	GetWithConditions(...interface{}) ([]*VerifiableContact, error)
	// Delete deletes a single VerifiableContact
	Delete(uuid.UUID) error
	// DeleteAllUser deletes all VerifiableContact of a User
	// given an IdentityID
	DeleteAllUser(uuid.UUID) error
}

type Service interface {
	Add(...VerifiableContact) ([]*VerifiableContact, error)
}