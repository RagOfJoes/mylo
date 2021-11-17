package gorm

import (
	"context"

	"github.com/RagOfJoes/mylo/flow/registration"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormRegistrationRepository struct {
	DB *gorm.DB
}

func NewGormRegistrationRepository(d *gorm.DB) registration.Repository {
	return &gormRegistrationRepository{DB: d}
}

func (g *gormRegistrationRepository) Create(ctx context.Context, newFlow registration.Flow) (*registration.Flow, error) {
	created := newFlow
	if err := g.DB.Create(&created).Error; err != nil {
		return nil, err
	}
	return &created, nil
}

func (g *gormRegistrationRepository) Get(ctx context.Context, id string) (*registration.Flow, error) {
	var flow registration.Flow
	if err := g.DB.First(&flow, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormRegistrationRepository) GetByFlowID(ctx context.Context, flowID string) (*registration.Flow, error) {
	var flow registration.Flow
	if err := g.DB.First(&flow, "flow_id = ?", flowID).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormRegistrationRepository) Update(ctx context.Context, updateFlow registration.Flow) (*registration.Flow, error) {
	updated := updateFlow
	if err := g.DB.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func (g *gormRegistrationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id.String()).Delete(registration.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
