package registration

import (
	"errors"
	"fmt"
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/ui/node"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

var (
	ErrInvalidPaylod        = errors.New("Invalid identifier(s) or password provided")
	ErrAlreadyAuthenticated = errors.New("Cannot access this resource while logged in")
)

type Status string

const (
	// Pending occurs when login flow is awaiting first factor ie. Password, Passwordless code
	Pending Status = "Pending"
	// Complete occurs when login has completed successfully
	Complete Status = "Complete"
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
	FlowID string `json:"flow_id" gorm:"not null;uniqueIndex" validate:"required"`
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`
	// Form defines additional information required to continue with the flow
	Form *form.Form `json:"form" gorm:"type:json" validate:"required_unless=Status Complete"`
}

// Payload defines the data required to complete the flow
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

// Form creates a form for registration
func Form(action string) form.Form {
	return form.Form{
		Action: action,
		Method: form.POST,
		Nodes: node.Nodes{
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "text",
					Label:    "Username",
					Name:     "username",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "email",
					Name:     "email",
					Label:    "Email Address",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "password",
					Name:     "password",
					Label:    "Password",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					Type:  "text",
					Name:  "first_name",
					Label: "First Name",
				},
			},
			{
				Type:  node.Input,
				Group: node.Password,
				Attributes: &node.InputAttribute{
					Type:  "text",
					Name:  "last_name",
					Label: "Last Name",
				},
			},
		},
	}
}

// New creates a new Flow
func New(requestURL string) (*Flow, error) {
	flowID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}

	cfg := config.Get()
	expire := time.Now().Add(cfg.Registration.Lifetime)
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Registration.URL, flowID)
	form := Form(action)
	return &Flow{
		FlowID:     flowID,
		Status:     Pending,
		Form:       &form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	}, nil
}

// Valid checks the validity of flow
func (f *Flow) Valid() error {
	if err := validate.Check(f); err != nil {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", err)
	}
	if f.Status == Complete || f.ExpiresAt.Before(time.Now()) {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", internal.ErrInvalidExpiredFlow)
	}
	return nil
}

// Complete updates flow to Complete status
func (f *Flow) Complete() {
	f.Form = nil
	f.Status = Complete
}
