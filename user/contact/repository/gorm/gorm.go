package gorm

import (
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormContactRepository struct {
	DB *gorm.DB
}

func NewGormContactRepository(d *gorm.DB) contact.Repository {
	return &gormContactRepository{DB: d}
}

func (g *gormContactRepository) Create(v ...contact.Contact) ([]contact.Contact, error) {
	n := v
	if err := g.DB.CreateInBatches(n, len(n)).Error; err != nil {
		return nil, err
	}
	return n, nil
}

func (g *gormContactRepository) Update(v contact.Contact) (*contact.Contact, error) {
	u := v
	if err := g.DB.Save(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (g *gormContactRepository) Get(i uuid.UUID) (*contact.Contact, error) {
	var v contact.Contact
	if err := g.DB.Where("id = ?", i).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormContactRepository) GetByValue(s string) (*contact.Contact, error) {
	var v contact.Contact
	if err := g.DB.First(&v, "value = ?", s).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormContactRepository) Delete(i uuid.UUID) error {
	if err := g.DB.Where("id = ?", i).Delete(contact.Contact{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (g *gormContactRepository) DeleteAllUser(i uuid.UUID) error {
	if err := g.DB.Where("identity_id = ?", i).Delete(contact.Contact{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
