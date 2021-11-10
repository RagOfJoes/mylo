package login

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
	ErrInvalidExpiredFlow = errors.New("Invalid or expired login flow")
	ErrInvalidPaylod      = errors.New("Invalid identifier or password provided")
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
	// Form defines additional information required to continue with the flow
	Form form.Form `json:"form" gorm:"not null;type:json" validate:"required"`
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
	Create(Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(string) (*Flow, error)
	// Get retrieves a flow via FlowID
	GetByFlowID(string) (*Flow, error)
	// Update updates a flow
	Update(Flow) (*Flow, error)
	// Deletes deletes a flow via ID
	Delete(uuid.UUID) error
}

// Services defines the interface for service implementations
type Service interface {
	// New creates a new login flow
	New(requestURL string) (*Flow, error)
	// Find does exactly that
	Find(flowID string) (*Flow, error)
	// Submit completes the flow
	Submit(flow Flow, payload Payload) (*identity.Identity, error)
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
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}

	cfg := config.Get()
	expire := time.Now().Add(cfg.Login.Lifetime)
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Login.URL, flowID)
	form := Form(action)
	return &Flow{
		FlowID:     flowID,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	}, nil
}

// Valid checks the validity of flow
func (f *Flow) Valid() error {
	if err := validate.Check(f); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", ErrInvalidExpiredFlow)
	}
	if f.ExpiresAt.Before(time.Now()) {
		return internal.NewErrorf(internal.ErrorCodeNotFound, "%v", ErrInvalidExpiredFlow)
	}
	return nil
}
