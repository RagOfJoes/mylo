package config

import (
	"net/http"
	"time"

	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

// Environment of the API. Note that certain features will be disabled in Production.
type Environment string

var (
	Development Environment = "Development"
	Production  Environment = "Production"
)

// Configuration is just that.
type Configuration struct {
	// Name of the API.
	Name string `validate:"required"`
	// Environment of the API.
	//
	// Default: Development
	Environment Environment `validate:"required,oneof='Development' 'Production'"`

	// Flows
	//
	//

	Login        Login
	Recovery     Recovery
	Registration Registration
	Verification Verification

	// Essentials
	//

	Server     Server
	Session    Session
	Database   Database
	Credential Credential

	// 3rd party
	//

	SendGrid SendGrid
}

var c Configuration

// Setup retrieves configuration provided
// Override any default values
// Initialize singleton object
func Setup(filename string, filetype string, filepath string) error {
	conf := Configuration{
		Environment: Development,

		// Flows
		//
		//

		Login: Login{
			URL:      "login",
			Lifetime: time.Minute * 10,
		},
		Recovery: Recovery{
			URL:      "recovery",
			Lifetime: time.Minute * 10,
		},
		Registration: Registration{
			URL:      "registration",
			Lifetime: time.Minute * 10,
		},
		Verification: Verification{
			URL:      "verification",
			Lifetime: time.Minute * 10,
		},

		// Essentials
		//
		//

		Session: Session{
			// 2 hours
			Lifetime: time.Hour * 336,
			Cookie: Cookie{
				Path:     "",
				Domain:   "",
				Persist:  true,
				HttpOnly: true,
				Name:     "raggy_sid",
				SameSite: http.SameSiteLaxMode,
			},
		},
		Credential: Credential{
			MinimumScore: 0,
			Argon: Argon{
				Memory:      64 * 1024,
				Iterations:  2,
				Parallelism: 2,
				SaltLength:  16,
				KeyLength:   32,
			},
		},
		Server: Server{
			Port:   80,
			Host:   ":",
			RPS:    100,
			Scheme: "http",
			AccessControl: AccessControl{
				AllowOrigin:      "*",
				MaxAge:           86400,
				AllowCredentials: true,
				AllowMethods:     []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Content-Type", "Content-Length", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With", "X-Session-Token"},
			},
			Security: secure.Options{
				IsDevelopment:     false,
				ReferrerPolicy:    "same-origin",
				HostsProxyHeaders: []string{"X-Forwarded-Hosts"},
			},
		},
	}

	viper.SetConfigName(filename)
	viper.SetConfigType(filetype)
	viper.AddConfigPath(filepath)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(&conf); err != nil {
		return err
	}
	if err := validate.Check(conf); err != nil {
		return err
	}

	c = conf
	if err := setupServer(&c); err != nil {
		return err
	}
	return nil
}

func Get() Configuration {
	return c
}
