package gorm

import (
	"github.com/RagOfJoes/idp"
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
	if err := g.DB.Preload("Credentials").Preload("VerifiableAddresses").Find(&f, "id = ?", str).Error; err != nil {
		return nil, err
	}
	if !c {
		f.Credentials = nil
	}
	return &f, nil
}

func (g *gormUserRepository) GetIdentifier(s string, c bool) (*identity.Identity, error) {
	var i identity.Identity
	var f credential.Credential
	if err := g.DB.Preload("Identifiers", "value = ?", s).First(&f).Error; err != nil {
		return nil, err
	}
	if err := g.DB.Preload("Credentials").Preload("VerifiableAddresses").Find(&i, "id = ?", f.IdentityID).Error; err != nil {
		return nil, err
	}
	if !c {
		i.Credentials = nil
	}
	return &i, nil
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
		BaseSoftDelete: idp.BaseSoftDelete{
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
