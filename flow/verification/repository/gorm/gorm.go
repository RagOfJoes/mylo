package gorm

import (
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

func (g *gormVerificationRepository) Create(newFlow verification.Flow) (*verification.Flow, error) {
	clone := newFlow
	if err := g.DB.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormVerificationRepository) Get(id uuid.UUID) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.First(&flow, "id = ?", flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) GetByFlowIDOrVerifyID(id string) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.Where("flow_id = ?", id).Or("verify_id = ?", id).Find(&flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) GetByContactID(contactID uuid.UUID) (*verification.Flow, error) {
	var flow verification.Flow
	if err := g.DB.First(&flow, "contact_id = ?", contactID).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (g *gormVerificationRepository) Update(updateFlow verification.Flow) (*verification.Flow, error) {
	clone := updateFlow
	if err := g.DB.Save(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormVerificationRepository) Delete(id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id.String()).Delete(verification.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
