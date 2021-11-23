package gorm

import (
	"context"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/RagOfJoes/mylo/user/identity"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormUserRepository struct {
	DB *gorm.DB
}

func NewGormUserRepository(d *gorm.DB) identity.Repository {
	return &gormUserRepository{DB: d}
}

func (g *gormUserRepository) Create(ctx context.Context, newIdentity identity.Identity) (*identity.Identity, error) {
	clone := newIdentity
	if err := g.DB.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormUserRepository) Get(ctx context.Context, id uuid.UUID, c bool) (*identity.Identity, error) {
	var found identity.Identity
	if err := g.DB.Preload("Credentials").Preload("Contacts").First(&found, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if !c {
		found.Credentials = nil
	}
	return &found, nil
}

func (g *gormUserRepository) GetWithIdentifier(ctx context.Context, identifier string, critical bool) (*identity.Identity, error) {
	// First check the credentials to make sure that the identifier provided is valid
	var cred credential.Credential
	var idenf credential.Identifier
	db := g.DB.WithContext(ctx)
	if err := db.First(&idenf, "LOWER(value) = LOWER(?)", identifier).Error; err != nil {
		return nil, err
	}
	if err := db.First(&cred, "id = ?", idenf.CredentialID).Error; err != nil {
		return nil, err
	}
	// Use the credential found to search for the actual identity
	var user identity.Identity
	if err := db.Preload("Credentials").Preload("Contacts").First(&user, "id = ?", cred.IdentityID).Error; err != nil {
		return nil, err
	}
	if !critical {
		user.Credentials = nil
	}
	return &user, nil
}

func (g *gormUserRepository) Update(ctx context.Context, updateIdentity identity.Identity) (*identity.Identity, error) {
	var found identity.Identity
	if err := g.DB.Model(&found).Omit("Credentials").Updates(updateIdentity).Error; err != nil {
		return nil, err
	}
	return &found, nil
}

func (g *gormUserRepository) Delete(ctx context.Context, id uuid.UUID, permanent bool) error {
	i := identity.Identity{
		BaseSoftDelete: internal.BaseSoftDelete{
			ID: id,
		},
	}
	db := g.DB
	if permanent {
		db = db.Unscoped()
	}
	if err := db.Select(clause.Associations).Delete(&i).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
