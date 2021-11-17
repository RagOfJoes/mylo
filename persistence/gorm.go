package persistence

import (
	"errors"
	"fmt"

	"github.com/RagOfJoes/mylo/flow/login"
	"github.com/RagOfJoes/mylo/flow/recovery"
	"github.com/RagOfJoes/mylo/flow/registration"
	"github.com/RagOfJoes/mylo/flow/verification"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/session"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/RagOfJoes/mylo/user/identity"
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
		&session.Session{},
		&identity.Identity{},
		&contact.Contact{},
		&credential.Identifier{},
		&credential.Credential{},

		&login.Flow{},
		&recovery.Flow{},
		&verification.Flow{},
		&registration.Flow{},
	)
}
