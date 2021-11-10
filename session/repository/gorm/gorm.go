package gorm

import (
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormSessionRepository struct {
	DB *gorm.DB
}

func NewGormSessionRepository(d *gorm.DB) session.Repository {
	return &gormSessionRepository{DB: d}
}

func (g *gormSessionRepository) Create(newSession session.Session) (*session.Session, error) {
	clone := newSession
	if err := g.DB.Create(&clone).Error; err != nil {
		return nil, err
	}
	return &clone, nil
}

func (g *gormSessionRepository) Get(id uuid.UUID) (*session.Session, error) {
	var found session.Session
	if err := g.DB.First(&found, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if found.IdentityID != nil {
		var user identity.Identity
		if err := g.DB.Preload("Contacts").First(&user, "id = ?", found.IdentityID).Error; err != nil {
			return nil, err
		}
		found.Identity = &user
	}
	return &found, nil
}

func (g *gormSessionRepository) GetByToken(token string) (*session.Session, error) {
	var found session.Session
	if err := g.DB.First(&found, "token = ?", token).Error; err != nil {
		return nil, err
	}
	if found.IdentityID != nil {
		var user identity.Identity
		if err := g.DB.Preload("Contacts").First(&user, "id = ?", found.IdentityID).Error; err != nil {
			return nil, err
		}
		found.Identity = &user
	}

	return &found, nil
}

func (g *gormSessionRepository) Update(updateSession session.Session) (*session.Session, error) {
	updated := updateSession
	// Make sure we're not accidentally updating the Identity
	updated.Identity = nil
	if err := g.DB.Save(&updated).Error; err != nil {
		return nil, err
	}
	updated.Identity = updateSession.Identity
	return &updated, nil
}

func (g *gormSessionRepository) Delete(id uuid.UUID) error {
	if err := g.DB.Where("id = ?", id).Delete(session.Session{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (g *gormSessionRepository) DeleteAllIdentity(identityID uuid.UUID) error {
	if err := g.DB.Where("identity_Id = ?", identityID).Delete(session.Session{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
