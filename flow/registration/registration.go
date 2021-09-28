package registration

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

type Flow struct {
	internal.Base
	// RequestURL defines the url that initiated flow. This can be used to pass any
	// relevant data from urls path or query. This can also be used to find locate
	// or security issues.
	RequestURL string `json:"-" gorm:"not null" validate:"required"`
	// FlowID defines the unique identifier that user's will use to access the flow
	FlowID string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`
	// Form defines additional information required to continue with flow
	Form form.Form `json:"form" gorm:"not null;type:json" validate:"required"`
}

// Payload deinfes the data required to complete the flow
type Payload struct {
	// Email is what it is
	Email string `json:"email" form:"email" binding:"required" validate:"required,min=1,email"`
	// Username is what it is
	Username string `json:"username" form:"username" binding:"required" validate:"required,min=4,max=20,alphanum"`
	// FirstName is what it is
	FirstName string `json:"first_name" form:"first_name" validate:"max=64,alphanumunicode"`
	// LastName is what it is
	LastName string `json:"last_name" form:"last_name" validate:"max=64,alphanumunicode"`
	// Password is what it is
	Password string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

// Repository defines the interface for repository implementations
type Repository interface {
	// Creates a new flow
	Create(Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(string) (*Flow, error)
	// GetByFlowID retrieves a flow via FlowID
	GetByFlowID(string) (*Flow, error)
	// Update updates a flow
	Update(Flow) (*Flow, error)
	// Delete deletes a flow via ID
	Delete(uuid.UUID) error
}

// Service defines the interface for service implementations
type Service interface {
	// New creates a new registration flow
	New(requestURL string) (*Flow, error)
	// Find does exactly that
	Find(flowID string) (*Flow, error)
	// Submit completes the flow
	Submit(flow Flow, payload Payload) (*identity.Identity, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "registrations"
}
