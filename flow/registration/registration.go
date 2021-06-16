package registration

import (
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/gofrs/uuid"
)

type Registration struct {
	idp.Base
	RequestURL string    `json:"-" gorm:"not null" validate:"required"`
	FlowID     string    `json:"-" gorm:"not null;uniqueIndex" validate:"required"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`

	CSRFToken string    `json:"-" gorm:"not null" validate:"required"`
	Form      form.Form `json:"form" gorm:"not null;type:json" validate:"required"`
}

type RegistrationPayload struct {
	Email     string `json:"email" form:"email" binding:"required" validate:"required,min=1,email"`
	Username  string `json:"username" form:"username" binding:"required" validate:"required,min=4,max=20,alphanum"`
	FirstName string `json:"first_name" form:"first_name" validate:"max=64,alphanumunicode"`
	LastName  string `json:"last_name" form:"last_name" validate:"max=64,alphanumunicode"`
	Password  string `json:"password" form:"password" binding:"required" validate:"required,min=6,max=128"`
}

type Repository interface {
	Create(Registration) (*Registration, error)
	Get(string) (*Registration, error)
	GetByFlowID(string) (*Registration, error)
	Update(Registration) (*Registration, error)
	Delete(uuid.UUID) error
}

type Service interface {
	New(requestURL string) (*Registration, error)
	Find(flowID string) (*Registration, error)
	Submit(flowID string, payload RegistrationPayload) error
}
