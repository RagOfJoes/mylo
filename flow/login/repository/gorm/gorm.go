package gorm

import (
	"context"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormLoginRepository struct {
	DB *gorm.DB
}

func NewGormLoginRepository(d *gorm.DB) login.Repository {
	return &gormLoginRepository{DB: d}
}

func (g *gormLoginRepository) Create(ctx context.Context, newFlow login.Flow) (*login.Flow, error) {
	created := newFlow
	if err := g.DB.Create(&created).Error; err != nil {
		return nil, err
	}
	return &created, nil
}

func (g *gormLoginRepository) Get(ctx context.Context, id string) (*login.Flow, error) {
	var found login.Flow
	if err := g.DB.First(&found, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &found, nil
}

func (g *gormLoginRepository) GetByFlowID(ctx context.Context, flowID string) (*login.Flow, error) {
	var found login.Flow
	if err := g.DB.First(&found, "flow_id = ?", flowID).Error; err != nil {
		return nil, err
	}
	return &found, nil
}

func (g *gormLoginRepository) Update(ctx context.Context, updateFlow login.Flow) (*login.Flow, error) {
	updated := updateFlow
	if err := g.DB.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func (g *gormLoginRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id.String()).Delete(login.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
