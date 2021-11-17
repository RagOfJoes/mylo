package verification

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
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/identity"
	"github.com/gofrs/uuid"
)

var (
	ErrInvalidPassword  = errors.New("Invalid password provided")
	ErrNotAuthenticated = errors.New("You must be logged in to access this resource")
	ErrInvalidContact   = errors.New("Contact is either already verified or does not exist")
)

type Status string

const (
	// SessionWarn occurs when the user's session has passed its half-life. This requires the
	// user to perform a soft login by requiring them to input their password
	SessionWarn Status = "SessionWarn"
	// LinkPending occurs when the link has been sent via email/sms and is waiting to be
	// activated
	LinkPending Status = "LinkPending"
	// Complete occurs when verification has completed successfully
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

// Repository defines the interface for repository implementations
type Repository interface {
	// Create creates a new flow
	Create(ctx context.Context, newFlow Flow) (*Flow, error)
	// Get retrieves a flow via ID
	Get(ctx context.Context, id uuid.UUID) (*Flow, error)
	// GetByFlowIDOrVerifyID retrieves a flow via FlowID
	GetByFlowIDOrVerifyID(ctx context.Context, id string) (*Flow, error)
	// GetByContactID retrieves a flow via ContactID
	GetByContactID(ctx context.Context, contactID uuid.UUID) (*Flow, error)
	// Update updates a flow
	Update(ctx context.Context, updateFlow Flow) (*Flow, error)
	// Delete deletes a flow via ID
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service defines the interface for service implementations
type Service interface {
	// NewDefault creates a new flow with a Status of LinkPending
	NewDefault(ctx context.Context, identity identity.Identity, contact contact.Contact, requestURL string) (*Flow, error)
	// NewSessionWarn creates a new flow with a Status of SessionWarn. This should be called when User's session
	// has passed its half-life
	NewSessionWarn(ctx context.Context, identity identity.Identity, contact contact.Contact, requestURL string) (*Flow, error)
	// Find does exactly that
	Find(ctx context.Context, flowID string, identity identity.Identity) (*Flow, error)
	// SubmitSessionWarn requires the `SessionWarn` status and the `SessionWarnPayload` to move the flow to the next step. On success, the transport should send an email to selected contact
	SubmitSessionWarn(ctx context.Context, flow Flow, identity identity.Identity, payload SessionWarnPayload) (*Flow, error)
	// Verify either completes the flow or moves to next status
	Verify(ctx context.Context, flow Flow, identity identity.Identity) (*Flow, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "verifications"
}

// PasswordForm creates a form for flow with SessionWarn status
func PasswordForm(action string) form.Form {
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
					Label:    "Password",
				},
			},
		},
	}
}

// NewLinkPending creates a new flow with LinkPending status
func NewLinkPending(requestURL string, contactID uuid.UUID, identityID uuid.UUID) (*Flow, error) {
	// Create new FlowID
	flowID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}
	// Create new VerifyID
	verifyID, err := nanoid.New()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to generate nano id")
	}

	cfg := config.Get()
	return &Flow{
		FlowID:     flowID,
		VerifyID:   verifyID,
		RequestURL: requestURL,
		Status:     LinkPending,
		ExpiresAt:  time.Now().Add(cfg.Verification.Lifetime),

		Form:       nil,
		ContactID:  contactID,
		IdentityID: identityID,
	}, nil
}

// NewSessionWarn creates a new flow with SessionWarn status
func NewSessionWarn(requestURL string, contactID uuid.UUID, identityID uuid.UUID) (*Flow, error) {
	newFlow, err := NewLinkPending(requestURL, contactID, identityID)
	if err != nil {
		return nil, err
	}

	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, newFlow.FlowID)
	form := PasswordForm(action)
	newFlow.Form = &form
	newFlow.Status = SessionWarn
	return newFlow, nil
}

// Valid checks the validity of the flow
func (f *Flow) Valid() error {
	if err := validate.Check(f); err != nil {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", err)
	}
	if f.Status == Complete || f.ExpiresAt.Before(time.Now()) {
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", internal.ErrInvalidExpiredFlow)
	}
	return nil
}

// BelongsTo checks if flow belongs to user
func (f *Flow) BelongsTo(identityID uuid.UUID) bool {
	return f.IdentityID == identityID
}

// Next moves flow to next Status based on current Status
func (f *Flow) Next() error {
	switch f.Status {
	case SessionWarn:
		f.Form = nil
		f.Status = LinkPending
		return nil
	case LinkPending:
		f.Status = Complete
		return nil
	default:
		return internal.NewErrorf(internal.ErrorCodeInternal, "%v", internal.ErrInvalidExpiredFlow)
	}
}
