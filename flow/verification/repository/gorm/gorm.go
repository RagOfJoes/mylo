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

func (g *gormVerificationRepository) Create(v verification.Flow) (*verification.Flow, error) {
	n := v
	if err := g.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormVerificationRepository) Get(i uuid.UUID) (*verification.Flow, error) {
	var v verification.Flow
	if err := g.DB.First(&v, "id = ?", v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormVerificationRepository) GetByFlowID(i string) (*verification.Flow, error) {
	var v verification.Flow
	if err := g.DB.First(&v, "flow_id = ?", i).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormVerificationRepository) GetByContact(c uuid.UUID) (*verification.Flow, error) {
	var v verification.Flow
	if err := g.DB.First(&v, "contact_id = ?", c).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormVerificationRepository) Update(v verification.Flow) (*verification.Flow, error) {
	n := v
	if err := g.DB.Save(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormVerificationRepository) Delete(i uuid.UUID) error {
	if err := g.DB.Where("id = ?", i.String()).Delete(verification.Flow{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
