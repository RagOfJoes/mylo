package gorm

import (
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormCredentialRepository struct {
	DB *gorm.DB
}

func NewGormCredentialRepository(d *gorm.DB) credential.Repository {
	return &gormCredentialRepository{DB: d}
}

func (g *gormCredentialRepository) Create(c credential.Credential) (*credential.Credential, error) {
	n := c
	if err := g.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormCredentialRepository) GetIdentifier(id string) (*credential.Identifier, error) {
	var i credential.Identifier
	if err := g.DB.First(&i, "LOWER(value) = LOWER(?)", id).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

func (g *gormCredentialRepository) GetWithIdentifier(t credential.CredentialType, i string) (*credential.Credential, error) {
	var password credential.Credential
	var identifier credential.Identifier
	if err := g.DB.First(&identifier, "LOWER(value) = LOWER(?)", i).Error; err != nil {
		return nil, err
	}
	if err := g.DB.First(&password, "id = ?", identifier.CredentialID).Error; err != nil {
		return nil, err
	}
	return &password, nil
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
	// Update Credential
	if err := g.DB.Save(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (g *gormCredentialRepository) Delete(i uuid.UUID) error {
	if err := g.DB.Where("credential_id = ?", i).Delete(credential.Identifier{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err := g.DB.Select(clause.Associations).Delete(identity.Identity{}, "id = ?", i).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
