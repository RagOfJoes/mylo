package recovery

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/gofrs/uuid"
)

type Status string

const (
	// IdentifierPending occurs when the flow has just been initialized and must now submit an identifier
	IdentifierPending Status = "Pending"
	// LinkPending occurs when the link has been sent via email/sms and is waiting to be activated
	LinkPending Status = "LinkPending"
	// Success occurs when recovery has completed successfully
	Success Status = "Success"
	// Fail occurs when an invalid identifier has been provided
	Fail Status = "Fail"
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
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	// Form defines additional information required to continue with flow
	Form *form.Form `json:"form,omitempty" gorm:"type:json;default:null"`

	// IdentityID defines the user that this flow belongs to
	IdentityID *uuid.UUID `json:"-" gorm:"index" validate:"required_if=Status LinkPending"`
}

// IdentifierPayload defines the payload required to move to `LinkPending`
type IdentifierPayload struct {
	Identifier string `json:"identifier" form:"identifier" binding:"required" validate:"required"`
}

// SubmitPayload defines the payload required to complete the flow
type SubmitPayload struct {
	Password        string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password" binding:"required" validate:"required,eqfield=Password"`
}

// Repository defines
type Repository interface {
	// Create creates a new flow
	Create(newFlow Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(id uuid.UUID) (*Flow, error)
	// GetByFlowID retrieves a flow via FlowID
	GetByFlowID(flowID string) (*Flow, error)
	// GetByIdentityID retrieves a flow via identity ID
	GetByIdentityID(identityID uuid.UUID) (*Flow, error)
	// Update updates a flow
	Update(updateFlow Flow) (*Flow, error)
	// Delete deletes a flow via ID
	Delete(id uuid.UUID) error
}

// Service defines
type Service interface {
	// New creates a new flow
	New(requestURL string) (*Flow, error)
	// Find does exactly that
	Find(flowID string) (*Flow, error)
	// SubmitIdentifier requires the `IdentifierPending` status and the `IdentifierPayload` to move the flow to the next step. An email should also be sent to all backup contacts in transport implementation
	SubmitIdentifier(flow Flow, payload IdentifierPayload) (*Flow, error)
	// SubmitUpdatePassword completes the flow
	SubmitUpdatePassword(flow Flow, payload SubmitPayload) (*Flow, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "recoveries"
}
