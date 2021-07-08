package persistence

import (
	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GormConfig struct {
	DSN     string
	Migrate bool
}

func NewGorm(cfg GormConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if cfg.Migrate {
		err := migrate(db)
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&identity.Identity{},
		&contact.VerifiableContact{},
		&credential.Identifier{},
		&credential.Credential{},

		&login.Login{},
		&registration.Registration{},
	)
}
