package login

import (
	"context"
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
	ErrInvalidPaylod = errors.New("Invalid identifier or password provided")
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
	// Identifier can either be email or username of user
	Identifier string `json:"identifier" form:"identifier" binding:"required" validate:"required,min=1,max=128"`
	// Password is what it is
	Password string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

// Repository defines the interface for repository implementations
type Repository interface {
	// Create creates a new flow
	Create(ctx context.Context, newFlow Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(ctx context.Context, id string) (*Flow, error)
	// Get retrieves a flow via FlowID
	GetByFlowID(ctx context.Context, flowID string) (*Flow, error)
	// Update updates a flow
	Update(ctx context.Context, updateFlow Flow) (*Flow, error)
	// Deletes deletes a flow via ID
	Delete(ctx context.Context, id uuid.UUID) error
}

// Services defines the interface for service implementations
type Service interface {
	// New creates a new login flow
	New(ctx context.Context, requestURL string) (*Flow, error)
	// Find does exactly that
	Find(ctx context.Context, flowID string) (*Flow, error)
	// Submit completes the flow
	Submit(ctx context.Context, flow Flow, payload Payload) (*identity.Identity, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "logins"
}

// Form creates a form for login
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
					Name:     "identifier",
					Label:    "Email or username",
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
		},
	}
}

// New creates a new flow
func New(requestURL string) (*Flow, error) {
	flowID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", internal.ErrFailedNanoID)
	}

	cfg := config.Get()
	expire := time.Now().Add(cfg.Login.Lifetime)
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Login.URL, flowID)
	form := Form(action)
	return &Flow{
		FlowID:     flowID,
		Status:     Pending,
		Form:       &form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	}, nil
}

// Valid checks the validity of flow, if the flow is expired or completed we also return error
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
