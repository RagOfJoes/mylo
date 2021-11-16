package gorm

import (
	"context"

	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormCredentialRepository struct {
	DB *gorm.DB
}

func NewGormCredentialRepository(d *gorm.DB) credential.Repository {
	return &gormCredentialRepository{DB: d}
}

func (g *gormCredentialRepository) Create(ctx context.Context, newCredential credential.Credential) (*credential.Credential, error) {
	clone := newCredential
	if err := g.DB.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormCredentialRepository) GetIdentifier(ctx context.Context, id string) (*credential.Identifier, error) {
	var identifier credential.Identifier
	if err := g.DB.Preload("Identifiers").First(&identifier, "LOWER(value) = LOWER(?)", id).Error; err != nil {
		return nil, err
	}
	return &identifier, nil
}

func (g *gormCredentialRepository) GetWithIdentifier(ctx context.Context, credentialType credential.CredentialType, id string) (*credential.Credential, error) {
	var password credential.Credential
	var identifier credential.Identifier
	if err := g.DB.First(&identifier, "LOWER(value) = LOWER(?)", id).Error; err != nil {
		return nil, err
	}
	if err := g.DB.Preload("Identifiers").First(&password, "id = ?", identifier.CredentialID).Error; err != nil {
		return nil, err
	}
	return &password, nil
}

func (g *gormCredentialRepository) GetWithIdentityID(ctx context.Context, credentialType credential.CredentialType, identityID uuid.UUID) (*credential.Credential, error) {
	var found credential.Credential
	if err := g.DB.Preload("Identifiers").First(&found, "type = ? AND identity_id = ?", credentialType, identityID).Error; err != nil {
		return nil, err
	}
	return &found, nil
}

func (g *gormCredentialRepository) Update(ctx context.Context, update credential.Credential) (*credential.Credential, error) {
	updated := update
	// Update Credential
	if err := g.DB.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func (g *gormCredentialRepository) Delete(ctx context.Context, credentialID uuid.UUID) error {
	if err := g.DB.Where("credential_id = ?", credentialID).Delete(credential.Identifier{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		g.DB.Rollback()
		return err
	}
	if err := g.DB.Where("id = ?", credentialID).Delete(credential.Credential{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		g.DB.Rollback()
		return err
	}
	return nil
}
