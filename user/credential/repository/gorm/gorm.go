package gorm

import (
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormCredentialRepository struct {
	DB *gorm.DB
}

func NewGormUserRepository(d *gorm.DB) credential.Repository {
	return &gormCredentialRepository{DB: d}
}

func (g *gormCredentialRepository) Create(c credential.Credential) (*credential.Credential, error) {
	n := c
	if err := g.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormCredentialRepository) GetWithIdentifier(t credential.CredentialType, i string) (*credential.Credential, error) {
	var c credential.Credential
	if err := g.DB.Preload("Identifiers", "value = ?", i).First(&c, "type = ?", t).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (g *gormCredentialRepository) GetWithIdentityID(t credential.CredentialType, i uuid.UUID) (*credential.Credential, error) {
	str := i.String()
	var c credential.Credential
	if err := g.DB.First(&c, "type = ? AND identity_id = ?", t, str).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (g *gormCredentialRepository) Update(n credential.Credential) (*credential.Credential, error) {
	r := n
	if err := g.DB.Unscoped().Where("credential_id = ?", r.ID).Delete(credential.Identifier{}).Error; err != nil {
		return nil, err
	}
	// Update Credential
	if err := g.DB.Save(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (g *gormCredentialRepository) Delete(i uuid.UUID) error {
	str := i.String()
	if err := g.DB.Unscoped().Where("credential_id = ?", str).Delete(credential.Identifier{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err := g.DB.Where("id = ?", str).Delete(credential.Credential{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
