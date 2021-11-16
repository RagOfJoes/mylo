package gorm

import (
	"context"

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

func (g *gormContactRepository) Create(ctx context.Context, contacts ...contact.Contact) ([]contact.Contact, error) {
	clone := contacts
	if err := g.DB.CreateInBatches(clone, len(clone)).Error; err != nil {
		return nil, err
	}
	return clone, nil
}

func (g *gormContactRepository) Update(ctx context.Context, updateContact contact.Contact) (*contact.Contact, error) {
	clone := updateContact
	if err := g.DB.Save(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormContactRepository) Get(ctx context.Context, contactID uuid.UUID) (*contact.Contact, error) {
	var contact contact.Contact
	if err := g.DB.Where("id = ?", contactID).First(&contact).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

func (g *gormContactRepository) GetByValue(ctx context.Context, value string) (*contact.Contact, error) {
	var contact contact.Contact
	if err := g.DB.First(&contact, "LOWER(v) = LOWER(?)", value).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

func (g *gormContactRepository) Delete(ctx context.Context, contactID uuid.UUID) error {
	if err := g.DB.Where("id = ?", contactID).Delete(contact.Contact{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (g *gormContactRepository) DeleteAllUser(ctx context.Context, identityID uuid.UUID) error {
	if err := g.DB.Where("identity_id = ?", identityID).Delete(contact.Contact{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
