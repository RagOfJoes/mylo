package gorm

import (
	"context"

	"github.com/RagOfJoes/idp/flow/recovery"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormRecoveryRepository struct {
	DB *gorm.DB
}

func NewGormRecoveryRepository(d *gorm.DB) recovery.Repository {
	return &gormRecoveryRepository{DB: d}
}

func (g *gormRecoveryRepository) Create(ctx context.Context, newFlow recovery.Flow) (*recovery.Flow, error) {
	clone := newFlow
	if err := g.DB.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormRecoveryRepository) Get(ctx context.Context, id uuid.UUID) (*recovery.Flow, error) {
	var flow recovery.Flow
	if err := g.DB.First(&flow, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormRecoveryRepository) GetByFlowIDOrRecoverID(ctx context.Context, id string) (*recovery.Flow, error) {
	var flow recovery.Flow
	if err := g.DB.Where("flow_id = ?", id).Or("recover_id = ?", id).Find(&flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormRecoveryRepository) GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*recovery.Flow, error) {
	var flow recovery.Flow
	if err := g.DB.First(&flow, "identity_id = ?", identityID).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormRecoveryRepository) Update(ctx context.Context, updateFlow recovery.Flow) (*recovery.Flow, error) {
	clone := updateFlow
	if err := g.DB.Save(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormRecoveryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id).Delete(recovery.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
