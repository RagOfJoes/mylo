package login

import (
	"time"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

type Flow struct {
	internal.Base
	RequestURL string    `json:"-" gorm:"not null" validate:"required"`
	FlowID     string    `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	Form form.Form `json:"form" gorm:"not null;type:json" validate:"required"`
}

type Payload struct {
	Identifier string `json:"identifier" form:"identifier" binding:"required" validate:"required,min=1,max=128"`
	Password   string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

type Repository interface {
	Create(Flow) (*Flow, error)
	Get(string) (*Flow, error)
	GetByFlowID(string) (*Flow, error)
	Update(Flow) (*Flow, error)
	Delete(uuid.UUID) error
}

type Service interface {
	New(requestURL string) (*Flow, error)
	Find(flowID string) (*Flow, error)
	Submit(flowID string, payload Payload) (*identity.Identity, error)
}

// TableName overrides GORM's table name
func (Flow) TableName() string {
	return "logins"
}
