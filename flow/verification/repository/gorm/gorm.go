package gorm

import (
	"context"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormVerificationRepository struct {
	DB *gorm.DB
}

func NewGormVerificationRepository(d *gorm.DB) verification.Repository {
	return &gormVerificationRepository{DB: d}
}

func (g *gormVerificationRepository) Create(ctx context.Context, newFlow verification.Flow) (*verification.Flow, error) {
	created := newFlow
	if err := g.DB.Create(&created).Error; err != nil {
		return nil, err
	}
	return &created, nil
}

func (g *gormVerificationRepository) Get(ctx context.Context, id uuid.UUID) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.First(ctx, &flow, "id = ?", flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) GetByFlowIDOrVerifyID(ctx context.Context, id string) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.Where("flow_id = ?", id).Or("verify_id = ?", id).Find(&flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) GetByContactID(ctx context.Context, contactID uuid.UUID) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.First(&flow, "contact_id = ?", contactID).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) Update(ctx context.Context, updateFlow verification.Flow) (*verification.Flow, error) {
	updated := updateFlow
	if err := g.DB.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func (g *gormVerificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id.String()).Delete(verification.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
