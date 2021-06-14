package address

import (
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/gofrs/uuid"
)

type VerifiableAddressType string
type VerifiableAddressMethod string
type VerifiableAddressState string

const (
	Default VerifiableAddressType = "default"
	Backup  VerifiableAddressType = "backup"

	Email VerifiableAddressMethod = "email"

	Sent      VerifiableAddressState = "sent"
	Completed VerifiableAddressState = "completed"
)

type VerifiableAddress struct {
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
	Type VerifiableAddressType `json:"type" gorm:"index;not null;default:default"`
	// Method is the delivery method for verification
	Method VerifiableAddressMethod `json:"method"`
	// State is the current state of the verification process.
	//
	// enum{ "sent", "completed" }
	//
	// "sent" means the verification link, email, sms, etc.
	// was sent.
	// "completed" means the verification process been fulfilled
	// by the user.
	State VerifiableAddressState `json:"state" gorm:"not null" validate:"oneof='sent' 'completed'"`
	// Address is the actual address to be verified. This can
	// be an email, phone number, etc.
	Address    string    `json:"address" gorm:"uniqueIndex;not null;" validate:"required,min=1"`
	IdentityID uuid.UUID `gorm:"index;not null" validate:"required,uuid4"`
}

type Repository interface {
	// Create creates a new VerifiableAddress
	Create(...VerifiableAddress) ([]*VerifiableAddress, error)
	// Update updates a new VerifiableAddress
	Update(VerifiableAddress) (*VerifiableAddress, error)
	// Get retrieves a single VerifiableAddress given its
	// id or identity id
	Get(uuid.UUID) (*VerifiableAddress, error)
	// GetByAddress retrieves a VerifiableAddress given an address
	// value
	GetByAddress(string) (*VerifiableAddress, error)
	// GetWithConditions retrieves all VerifiableAddress given
	// a custom conditional values
	GetWithConditions(...interface{}) ([]*VerifiableAddress, error)
	// Delete deletes a single VerifiableAddress
	Delete(uuid.UUID) error
	// DeleteAllUser deletes all VerifiableAddress of a User
	// given an IdentityID
	DeleteAllUser(uuid.UUID) error
}

type Service interface {
	Add(...VerifiableAddress) ([]*VerifiableAddress, error)
}
