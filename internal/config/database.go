package config

import "gorm.io/gorm/logger"

type Database struct {
	// Required configurations
	//

	// Driver defines the type of database.
	Driver string `validate:"required,oneof='mysql' 'postgres'"`
	// Name of the database.
	Name string `validate:"required"`
	// Username required to access database.
	Username string `validate:"required"`
	// Password required to access database.
	Password string `validate:"required"`
	// Host for the database.
	Host string `validate:"required"`
	// Port for the database.
	Port int `validate:"required"`

	// Optional configuration
	//

	// AutoMigrate will auto migrate models in startup.
	//
	// Default: true
	AutoMigrate bool
	// LogLevel determines what will be logged.
	//
	// Default: 1 || logger.Silent
	LogLevel logger.LogLevel
}
