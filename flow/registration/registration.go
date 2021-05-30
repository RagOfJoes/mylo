package registration

import (
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/ui/form"
)

type Registration struct {
	idp.Base
	ExpiresAt  time.Time `json:"expires_at" gorm:"index;not null" validate:"required"`
	RequestURL string    `json:"-" gorm:"not null" validate:"required"`

	CSRFToken string    `json:"csrf_token" gorm:"not null" validate:"required"`
	Form      form.Form `json:"form" gorm:"not null" validate:"required"`
}
