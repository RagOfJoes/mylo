package main

import (
	"log"

	"github.com/RagOfJoes/idp/email"
	loginGorm "github.com/RagOfJoes/idp/flow/login/repository/gorm"
	loginService "github.com/RagOfJoes/idp/flow/login/service"
	loginTransport "github.com/RagOfJoes/idp/flow/login/transport"
	recoveryGorm "github.com/RagOfJoes/idp/flow/recovery/repository/gorm"
	recoveryService "github.com/RagOfJoes/idp/flow/recovery/service"
	recoveryTransport "github.com/RagOfJoes/idp/flow/recovery/transport"
	registrationGorm "github.com/RagOfJoes/idp/flow/registration/repository/gorm"
	registrationService "github.com/RagOfJoes/idp/flow/registration/service"
	registrationTransport "github.com/RagOfJoes/idp/flow/registration/transport"
	verificationGorm "github.com/RagOfJoes/idp/flow/verification/repository/gorm"
	verificationService "github.com/RagOfJoes/idp/flow/verification/service"
	verificationTransport "github.com/RagOfJoes/idp/flow/verification/transport"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/persistence"
	sessionGorm "github.com/RagOfJoes/idp/session/repository/gorm"
	sessionService "github.com/RagOfJoes/idp/session/service"
	sessionTransport "github.com/RagOfJoes/idp/session/transport"
	"github.com/RagOfJoes/idp/transport"
	contactGorm "github.com/RagOfJoes/idp/user/contact/repository/gorm"
	contactService "github.com/RagOfJoes/idp/user/contact/service"
	credentialGorm "github.com/RagOfJoes/idp/user/credential/repository/gorm"
	credentialService "github.com/RagOfJoes/idp/user/credential/service"
	identityGorm "github.com/RagOfJoes/idp/user/identity/repository/gorm"
	identityService "github.com/RagOfJoes/idp/user/identity/service"
	identityTransport "github.com/RagOfJoes/idp/user/identity/transport"
	"github.com/gorilla/sessions"
)

func init() {
	// Load configuration
	if err := config.Setup("config", "yaml", "/home/raggy/"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	cfg := config.Get()

	db, err := persistence.NewGorm()
	if err != nil {
		log.Panic(err.Error())
		return
	}

	// Setup Email client
	email := email.New()

	// Setup repositories
	sessionRepository := sessionGorm.NewGormSessionRepository(db)
	contactRepository := contactGorm.NewGormContactRepository(db)
	credentialRepository := credentialGorm.NewGormCredentialRepository(db)
	identityRepository := identityGorm.NewGormUserRepository(db)
	recoveryRepository := recoveryGorm.NewGormRecoveryRepository(db)
	verificationRepository := verificationGorm.NewGormVerificationRepository(db)
	registrationRepository := registrationGorm.NewGormRegistrationRepository(db)
	loginRepository := loginGorm.NewGormLoginRepository(db)
	// Setup services
	sessionService := sessionService.NewSessionService(sessionRepository)
	contactService := contactService.NewContactService(contactRepository)
	credentialService := credentialService.NewCredentialService(credentialRepository)
	identityService := identityService.NewIdentityService(identityRepository)
	// Flow Services
	// These will essentially stitch all other services together
	verificationService := verificationService.NewVerificationService(verificationRepository, contactService, credentialService, identityService)
	registrationService := registrationService.NewRegistrationService(registrationRepository, contactService, credentialService, identityService)
	loginService := loginService.NewLoginService(loginRepository, contactService, credentialService, identityService)
	recoveryService := recoveryService.NewRecoveryService(recoveryRepository, credentialService, contactService)

	// Create session manager
	store := sessions.NewCookieStore([]byte(cfg.Session.Cookie.Name))
	sessionHttp := sessionTransport.NewSessionHttp(store, sessionService)
	if err != nil {
		log.Panic(err)
	}

	// Setup HTTP Server
	router := transport.NewHttp()

	// Attach Middlewares
	//
	// Order of execution:
	// 1. Rate Limiter
	// 2. Security Middleware (Adds essential security headers to request)
	// 3. Error Middleware handles any errors that were generated from route execution
	if cfg.Server.RPS > 0 {
		router.Use(transport.RateLimiterMiddleware(cfg.Server.RPS))
	}
	router.Use(transport.SecurityMiddleware(), transport.ErrorMiddleware())

	// Attach routes
	identityTransport.NewIdentityHttp(*sessionHttp, router)
	verificationTransport.NewVerificationHttp(email, *sessionHttp, verificationService, router)
	registrationTransport.NewRegistrationHttp(email, *sessionHttp, registrationService, verificationService, router)
	loginTransport.NewLoginHttp(*sessionHttp, loginService, router)
	recoveryTransport.NewRecoveryHttp(email, *sessionHttp, recoveryService, identityService, router)

	// Start HTTP server
	if err := transport.RunHttp(router); err != nil {
		log.Panic(err)
	}
}
