package internal

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Base defines the base model for domain objects
type Base struct {
	ID        uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required"`
	CreatedAt time.Time  `json:"created_at" gorm:"index;not null;default:current_timestamp" validate:"required"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index;default:null"`
}

// BaseSoftDelete defines the base model with soft delete functionality for domain objects
type BaseSoftDelete struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" validate:"required"`
	CreatedAt time.Time      `json:"created_at" gorm:"index;not null;default:current_timestamp" validate:"required"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty" gorm:"index;default:null"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index;default:null"`
}
