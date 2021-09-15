package verification

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

type VerificationStatus string

const (
	// SessionWarn occurs when the user's session has passed its half-life. This requires the
	// user to perform a soft login by requiring them to input their password
	SessionWarn VerificationStatus = "SessionWarn"
	// LinkPending occurs when the link has been sent via email/sms and is waiting to be
	// activated
	LinkPending VerificationStatus = "LinkPending"
	// Success occurs when verification has completed successfully
	Success VerificationStatus = "Success"
)

type Verification struct {
	internal.Base
	// RequestURL defines the url that initiated flow. This can be used to pass any
	// relevant data from urls path or query. This can also be used to find locate
	// or security issues.
	RequestURL string `json:"-" gorm:"not null" validate:"required"`
	// Status defines the current state of the flow
	Status VerificationStatus `json:"status" gorm:"not null" validate:"required"`
	// FlowID defines the unique identifier that will a user will use to
	FlowID string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	// Form defines additional information required to continue with flow
	Form *form.Form `json:"form,omitempty" gorm:"type:json;default:null"`

	// VerifiableContactID defines the contact that this flow belongs to
	VerifiableContactID uuid.UUID `json:"-" gorm:"index;not null" validate:"required,uuid4"`
	// IdentityID defines the user that this flow belongs to
	IdentityID uuid.UUID `json:"-" gorm:"index;not null" validate:"required,uuid4"`
}

// NewPayload defines the data required to initiate the flow
type NewPayload struct {
	// Contact should be the id of whatever contact the user wants to verify
	Contact string `json:"contact" form:"contact" binding:"required" validate:"required,uuid4"`
}

// SessionWarnPayload defines the form that will be rendered
// when a User's session has passed half of the expiration time
type SessionWarnPayload struct {
	// Password should be provided by the user
	Password string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

// Repository defines
type Repository interface {
	// Create creates a new Verification
	Create(newFlow Verification) (*Verification, error)
	// Get retrieves a flow via ID
	Get(id uuid.UUID) (*Verification, error)
	// GetByFlowID retrieves a flow via FlowID
	GetByFlowID(flowID string) (*Verification, error)
	// GetByContact retrieves a flow via ContactID
	GetByContact(contactID uuid.UUID) (*Verification, error)
	// Update updates a flow
	Update(updateFlow Verification) (*Verification, error)
	// Delete deletes a flow via ID
	Delete(id uuid.UUID) error
}

// Service defines
type Service interface {
	// New creates a new verification flow
	New(identity identity.Identity, contact contact.VerifiableContact, requestURL string, status VerificationStatus) (*Verification, error)
	// NewWelcome creates a new verification flow for a new user
	NewWelcome(identity identity.Identity, contact contact.VerifiableContact, requestURL string) (*Verification, error)
	Find(flowID string, identityID uuid.UUID) (*Verification, error)
	// Verify either completes the flow or moves to next status
	Verify(flowID string, identity identity.Identity, payload interface{}) (*Verification, error)
}
