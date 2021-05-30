package idp

import (
	"time"

	"github.com/gofrs/uuid"
)

type Base struct {
	ID        uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required,uuid4"`
	CreatedAt time.Time  `json:"created_at" gorm:"index;not null;default:current_timestamp" validate:"required"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index;default:null"`
}

type BaseSoftDelete struct {
	ID        uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required"`
	CreatedAt time.Time  `json:"created_at" gorm:"index;not null;default:current_timestamp" validate:"required"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index;default:null"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index;default:null"`
}
