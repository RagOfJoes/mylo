package main

import (
	"log"
	"os"
	"time"

	registrationGorm "github.com/RagOfJoes/idp/flow/registration/repository/gorm"
	registrationService "github.com/RagOfJoes/idp/flow/registration/service"
	registrationTransport "github.com/RagOfJoes/idp/flow/registration/transport"
	"github.com/RagOfJoes/idp/persistence"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	contactGorm "github.com/RagOfJoes/idp/user/contact/repository/gorm"
	contactService "github.com/RagOfJoes/idp/user/contact/service"
	credentialGorm "github.com/RagOfJoes/idp/user/credential/repository/gorm"
	credentialService "github.com/RagOfJoes/idp/user/credential/service"
	identityGorm "github.com/RagOfJoes/idp/user/identity/repository/gorm"
	identityService "github.com/RagOfJoes/idp/user/identity/service"
	_ "github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

func init() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := persistence.NewGorm(persistence.GormConfig{
		DSN:     os.Getenv("ORM_DSN"),
		Migrate: os.Getenv("ORM_AUTOMIGRATE") == "true",
	})
	if err != nil {
		log.Panic(err.Error())
		return
	}

	// Create session
	sessionManager, err := session.NewManager(false, "sid", time.Hour*24*14)
	if err != nil {
		log.Panic(err)
	}

	httpSrvCfg := transport.HttpConfig{
		Host:   os.Getenv("HOST"),
		Port:   os.Getenv("PORT"),
		Scheme: os.Getenv("SCHEME"),

		RemoveExtraSlashes: true,
	}
	ginEng := transport.NewHttp(httpSrvCfg)
	ginEng.Use(transport.RateLimiterMiddleware(100), transport.ErrorMiddleware(), session.AuthMiddleware(sessionManager))

	// Setup repositories
	cor := contactGorm.NewGormContactRepository(db)
	cr := credentialGorm.NewGormCredentialRepository(db)
	ir := identityGorm.NewGormUserRepository(db)
	rr := registrationGorm.NewGormRegistrationRepository(db)
	// Setup services
	cos := contactService.NewContactService(cor)
	ap := credentialService.NewArgonParams(64*1024, 2, 2, 16, 32)
	cs := credentialService.NewCredentialService(ap, cr)
	is := identityService.NewIdentityService(ir, cs, cos)
	rs := registrationService.NewRegistrationService(rr, cos, cs, is)

	// Attach routes
	registrationTransport.NewRegistrationHttp(rs, sessionManager, ginEng)

	// Start server
	if err := transport.RunHttp(httpSrvCfg, sessionManager.LoadAndSave(ginEng)); err != nil {
		log.Panic(err)
	}
}
