package gorm

import (
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormRegistrationRepository struct {
	DB *gorm.DB
}

func NewGormRegistrationRepository(d *gorm.DB) registration.Repository {
	return &gormRegistrationRepository{DB: d}
}

func (g *gormRegistrationRepository) Create(r registration.Registration) (*registration.Registration, error) {
	n := r
	if err := g.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormRegistrationRepository) Get(i string) (*registration.Registration, error) {
	var n registration.Registration
	if err := g.DB.First(&n, "id = ?", i).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormRegistrationRepository) GetByFlowID(i string) (*registration.Registration, error) {
	var n registration.Registration
	if err := g.DB.First(&n, "flow_id = ?", i).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormRegistrationRepository) Update(r registration.Registration) (*registration.Registration, error) {
	n := r
	if err := g.DB.Save(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormRegistrationRepository) Delete(i uuid.UUID) error {
	if err := g.DB.Where("id = ?", i.String()).Delete(registration.Registration{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
