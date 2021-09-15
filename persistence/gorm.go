package persistence

import (
	"errors"
	"fmt"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewGorm() (db *gorm.DB, err error) {
	cfg := config.Get()

	driver := cfg.Database.Driver
	username := cfg.Database.Username
	password := cfg.Database.Password
	host := cfg.Database.Host
	port := cfg.Database.Port
	name := cfg.Database.Name
	autoMigrate := cfg.Database.AutoMigrate

	isDev := cfg.Environment == config.Development

	gc := gorm.Config{
		Logger: logger.Default.LogMode(cfg.Database.LogLevel),
	}

	switch driver {
	case "mysql":
		dsn := fmt.Sprintf("mysql%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, name)
		db, err = gorm.Open(mysql.Open(dsn), &gc)
		if err != nil {
			return nil, err
		}
		break
	case "postgres":
		sslMode := "disable"
		if !isDev {
			sslMode = "enable"
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s", host, username, password, name, port, sslMode)
		db, err = gorm.Open(postgres.Open(dsn), &gc)
		if err != nil {
			return nil, err
		}
		break
	default:
		return nil, errors.New("invalid database driver provided")
	}

	if autoMigrate {
		if err := migrate(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&identity.Identity{},
		&contact.Contact{},
		&credential.Identifier{},
		&credential.Credential{},

		&login.Flow{},
		&verification.Flow{},
		&registration.Flow{},
	)
}
