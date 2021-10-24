package verification

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

type Status string

const (
	// SessionWarn occurs when the user's session has passed its half-life. This requires the
	// user to perform a soft login by requiring them to input their password
	SessionWarn Status = "SessionWarn"
	// LinkPending occurs when the link has been sent via email/sms and is waiting to be
	// activated
	LinkPending Status = "LinkPending"
	// Success occurs when verification has completed successfully
	Success Status = "Success"
)

type Flow struct {
	internal.Base
	// RequestURL defines the url that initiated flow. This can be used to pass any
	// relevant data from urls path or query. This can also be used to find locate
	// or security issues.
	RequestURL string `json:"-" gorm:"not null" validate:"required"`
	// Status defines the current state of the flow
	Status Status `json:"status" gorm:"not null" validate:"required"`
	// FlowID defines the unique identifier that user's will use to access the flow
	FlowID string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// VerifyID defines the unique identifier that user's will use to complete the flow
	VerifyID string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	// Form defines additional information required to continue with flow
	Form *form.Form `json:"form,omitempty" gorm:"type:json;default:null"`

	// ContactID defines the contact that this flow belongs to
	ContactID uuid.UUID `json:"-" gorm:"type:uuid;index;not null" validate:"required"`
	// IdentityID defines the user that this flow belongs to
	IdentityID uuid.UUID `json:"-" gorm:"type:uuid;index;not null" validate:"required"`
}

// SessionWarnPayload defines the form that will be rendered
// when a User's session has passed half of the expiration time
type SessionWarnPayload struct {
	// Password should be provided by the user
	Password string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

// Repository defines
type Repository interface {
	// Create creates a new flow
	Create(newFlow Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(id uuid.UUID) (*Flow, error)
	// GetByFlowIDOrVerifyID retrieves a flow via FlowID
	GetByFlowIDOrVerifyID(id string) (*Flow, error)
	// GetByContactID retrieves a flow via ContactID
	GetByContactID(contactID uuid.UUID) (*Flow, error)
	// Update updates a flow
	Update(updateFlow Flow) (*Flow, error)
	// Delete deletes a flow via ID
	Delete(id uuid.UUID) error
}

// Service defines
type Service interface {
	// NewDefault creates a new flow with a Status of LinkPending
	NewDefault(identity identity.Identity, contact contact.Contact, requestURL string) (*Flow, error)
	// NewSessionWarn creates a new flow with a Status of SessionWarn. This should be called when User's session
	// has passed its half-life
	NewSessionWarn(identity identity.Identity, contact contact.Contact, requestURL string) (*Flow, error)
	// Find does exactly that
	Find(flowID string, identity identity.Identity) (*Flow, error)
	// SubmitSessionWarn requires the `SessionWarn` status and the `SessionWarnPayload` to move the flow to the next step. On success, the transport should send an email to selected contact
	SubmitSessionWarn(flow Flow, identity identity.Identity, payload SessionWarnPayload) (*Flow, error)
	// Verify either completes the flow or moves to next status
	Verify(flow Flow, identity identity.Identity) (*Flow, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "verifications"
}
