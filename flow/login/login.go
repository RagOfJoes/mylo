package login

import (
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
)

type Login struct {
	idp.Base
	RequestURL string    `json:"-" gorm:"not null" validate:"required"`
	FlowID     string    `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	Form form.Form `json:"form" gorm:"not null;type:json" validate:"required"`
}

type LoginPayload struct {
	Identifier string `json:"identifier" form:"identifier" binding:"required" validate:"required,min=1,max=20"`
	Password   string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

type Repository interface {
	Create(Login) (*Login, error)
	Get(string) (*Login, error)
	GetByFlowID(string) (*Login, error)
	Update(Login) (*Login, error)
	Delete(uuid.UUID) error
}

type Service interface {
	New(requestURL string) (*Login, error)
	Find(flowID string) (*Login, error)
	Submit(flowID string, payload LoginPayload) (*identity.Identity, error)
}
