package gorm

import (
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
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

func (g *gormUserRepository) Create(u identity.Identity) (*identity.Identity, error) {
	c := u
	if err := g.DB.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (g *gormUserRepository) Get(u uuid.UUID, c bool) (*identity.Identity, error) {
	str := u.String()
	var f identity.Identity
	if err := g.DB.Preload("Credentials").Preload("VerifiableContacts").Find(&f, "id = ?", str).Error; err != nil {
		return nil, err
	}
	if !c {
		f.Credentials = nil
	}
	return &f, nil
}

func (g *gormUserRepository) GetIdentifier(s string, c bool) (*identity.Identity, error) {
	// First check the credentials to make sure that the identifier provided is valid
	var cred credential.Credential
	var idenf credential.Identifier
	if err := g.DB.First(&idenf, "value = ?", s).Error; err != nil {
		return nil, err
	}
	if err := g.DB.First(&cred, "id = ?", idenf.CredentialID).Error; err != nil {
		return nil, err
	}

	// Use the credential found to search for the actual identity
	var ident identity.Identity
	if err := g.DB.Preload("Credentials").Preload("VerifiableContacts").First(&ident, "id = ?", cred.IdentityID).Error; err != nil {
		return nil, err
	}
	if !c {
		ident.Credentials = nil
	}
	return &ident, nil
}

func (g *gormUserRepository) Update(u identity.Identity) (*identity.Identity, error) {
	var i identity.Identity
	if err := g.DB.Model(&i).Omit("Credentials").Updates(u).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

func (g *gormUserRepository) Delete(id uuid.UUID, permanent bool) error {
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
