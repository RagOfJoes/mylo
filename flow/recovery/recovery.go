package recovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/pkg/nanoid"
	"github.com/RagOfJoes/mylo/ui/form"
	"github.com/RagOfJoes/mylo/ui/node"
	"github.com/gofrs/uuid"
)

var (
	ErrInvalidIdentifierPaylod = errors.New("Invalid identifier provided")
	ErrAccountDoesNotExist     = errors.New("Account with identifier does not exist")
	ErrAlreadyAuthenticated    = errors.New("Cannot access this resource while logged in")
)

type Status string

const (
	// IdentifierPending occurs when the flow has just been initialized and must now submit an identifier
	IdentifierPending Status = "IdentifierPending"
	// Fail occurs when an invalid identifier has been provided
	Fail Status = "Fail"
	// LinkPending occurs when the link has been sent via email/sms and is waiting to be activated
	LinkPending Status = "LinkPending"
	// Complete occurs when recovery has completed successfully
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
	// RecoverID defines the unique identifier that user's will use to complete the flow
	RecoverID string `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	// ExpiresAt defines the time when this flow will no longer be valid
	ExpiresAt time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	// Form defines additional information required to continue with flow
	Form *form.Form `json:"form,omitempty" gorm:"type:json;default:null"`

	// IdentityID defines the user that this flow belongs to
	IdentityID *uuid.UUID `json:"-" gorm:"type:uuid;index" validate:"required_if=Status LinkPending"`
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
	Create(ctx context.Context, newFlow Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(ctx context.Context, id uuid.UUID) (*Flow, error)
	// GetByFlowIDOrRecoverID retrieves a flow via FlowID or RecoverID
	GetByFlowIDOrRecoverID(ctx context.Context, id string) (*Flow, error)
	// GetByIdentityID retrieves a flow via identity ID
	GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*Flow, error)
	// Update updates a flow
	Update(ctx context.Context, updateFlow Flow) (*Flow, error)
	// Delete deletes a flow via ID
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service defines
type Service interface {
	// New creates a new flow
	New(ctx context.Context, requestURL string) (*Flow, error)
	// Find retrieves flow via FlowID or RecoverID
	Find(ctx context.Context, id string) (*Flow, error)
	// SubmitIdentifier requires the `IdentifierPending` status and the `IdentifierPayload` to move the flow to the next step. An email should also be sent to all backup contacts in transport implementation
	SubmitIdentifier(ctx context.Context, flow Flow, payload IdentifierPayload) (*Flow, error)
	// SubmitUpdatePassword completes the flow
	SubmitUpdatePassword(ctx context.Context, flow Flow, payload SubmitPayload) (*Flow, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "recoveries"
}

// IdentifierForm creates a form for IdentifierPending
func IdentifierForm(action string) form.Form {
	return form.Form{
		Action: action,
		Method: "POST",
		Nodes: node.Nodes{
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "text",
					Name:     "identifier",
					Label:    "Identifier",
				},
			},
		},
	}
}

// RecoverForm creates a form for LinkPending
func RecoverForm(action string) form.Form {
	return form.Form{
		Action: action,
		Method: "POST",
		Nodes: node.Nodes{
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Name:     "password",
					Type:     "password",
					Label:    "New Password",
				},
			},
			&node.Node{
				Type:  node.Input,
				Group: node.Default,
				Attributes: &node.InputAttribute{
					Required: true,
					Type:     "password",
					Name:     "confirm_password",
					Label:    "Confirm New Password",
				},
			},
		},
	}
}

// New creates a new flow with IdentifierPending status
func New(requestURL string) (*Flow, error) {
	flowID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}
	recoverID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate uuid")
	}
	cfg := config.Get()
	expire := time.Now().Add(cfg.Recovery.Lifetime)
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Recovery.URL, flowID)
	form := IdentifierForm(action)
	return &Flow{
		FlowID:     flowID,
		ExpiresAt:  expire,
		RecoverID:  recoverID,
		RequestURL: requestURL,
		Status:     IdentifierPending,

		Form: &form,
	}, nil
}

// Valid checks the validity of flow, if the flow is expired or completed we also return error
func (f *Flow) Valid() error {
	if err := validate.Check(f); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", internal.ErrInvalidExpiredFlow)
	}
	if f.Status == Fail || f.Status == Complete || f.ExpiresAt.Before(time.Now()) {
		return internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", internal.ErrInvalidExpiredFlow)
	}
	if f.Status == LinkPending && f.IdentityID == nil {
		return internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", internal.ErrInvalidExpiredFlow)
	}
	return nil
}

// Fail updates flow to Fail status
func (f *Flow) Fail() {
	f.Form = nil
	f.Status = Fail
}

// Complete updates flow to Complete status
func (f *Flow) Complete() {
	f.Form = nil
	f.Status = Complete
}

// LinkPending updates flow to LinkPending status
func (f *Flow) LinkPending(identityID uuid.UUID) error {
	if f.Status != IdentifierPending {
		return internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", internal.ErrInvalidExpiredFlow)
	}

	cfg := config.Get()
	f.Status = LinkPending
	f.IdentityID = &identityID
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Recovery.URL, f.RecoverID)
	form := RecoverForm(action)
	f.Form = &form
	return nil
}
