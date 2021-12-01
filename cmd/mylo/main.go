package main

import (
	"log"

	"github.com/RagOfJoes/mylo/email"
	loginGorm "github.com/RagOfJoes/mylo/flow/login/repository/gorm"
	loginService "github.com/RagOfJoes/mylo/flow/login/service"
	loginTransport "github.com/RagOfJoes/mylo/flow/login/transport"
	recoveryGorm "github.com/RagOfJoes/mylo/flow/recovery/repository/gorm"
	recoveryService "github.com/RagOfJoes/mylo/flow/recovery/service"
	recoveryTransport "github.com/RagOfJoes/mylo/flow/recovery/transport"
	registrationGorm "github.com/RagOfJoes/mylo/flow/registration/repository/gorm"
	registrationService "github.com/RagOfJoes/mylo/flow/registration/service"
	registrationTransport "github.com/RagOfJoes/mylo/flow/registration/transport"
	verificationGorm "github.com/RagOfJoes/mylo/flow/verification/repository/gorm"
	verificationService "github.com/RagOfJoes/mylo/flow/verification/service"
	verificationTransport "github.com/RagOfJoes/mylo/flow/verification/transport"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/persistence"
	sessionGorm "github.com/RagOfJoes/mylo/session/repository/gorm"
	sessionService "github.com/RagOfJoes/mylo/session/service"
	sessionTransport "github.com/RagOfJoes/mylo/session/transport"
	"github.com/RagOfJoes/mylo/transport"
	contactGorm "github.com/RagOfJoes/mylo/user/contact/repository/gorm"
	contactService "github.com/RagOfJoes/mylo/user/contact/service"
	credentialGorm "github.com/RagOfJoes/mylo/user/credential/repository/gorm"
	credentialService "github.com/RagOfJoes/mylo/user/credential/service"
	identityGorm "github.com/RagOfJoes/mylo/user/identity/repository/gorm"
	identityService "github.com/RagOfJoes/mylo/user/identity/service"
	identityTransport "github.com/RagOfJoes/mylo/user/identity/transport"
	"github.com/gorilla/sessions"
)

func main() {
	cfgPtr, err := config.New("mylo", "yaml", "/home/mylo/")
	if err != nil {
		log.Panic(err)
		return
	}
	cfg := *cfgPtr

	db, err := persistence.NewGorm(cfg)
	if err != nil {
		log.Panic(err)
		return
	}

	// Setup Email client
	email := email.New(cfg)

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
	identityService := identityService.NewIdentityService(identityRepository)
	credentialService := credentialService.NewCredentialService(cfg, credentialRepository)
	// Flow Services
	// These will essentially stitch all other services together
	recoveryService := recoveryService.NewRecoveryService(cfg, recoveryRepository, credentialService, contactService)
	loginService := loginService.NewLoginService(cfg, loginRepository, contactService, credentialService, identityService)
	registrationService := registrationService.NewRegistrationService(cfg, registrationRepository, contactService, credentialService, identityService)
	verificationService := verificationService.NewVerificationService(cfg, verificationRepository, contactService, credentialService, identityService)

	// Create session manager
	store := sessions.NewCookieStore([]byte(cfg.Session.Cookie.Name))
	sessionHttp := sessionTransport.NewSessionHttp(cfg, store, sessionService)

	// Setup HTTP Server
	router := transport.NewHttp(cfg)

	// Attach Middlewares
	//
	// Order of execution:
	// 1. Rate Limiter
	// 2. Security Middleware (Adds essential security headers to request)
	// 3. Error Middleware handles any errors that were generated from route execution
	if cfg.Server.RPS > 0 {
		router.Use(transport.RateLimiterMiddleware(cfg.Server.RPS))
	}
	router.Use(transport.SecurityMiddleware(cfg), transport.ErrorMiddleware())

	// Attach routes
	identityTransport.NewIdentityHttp(*sessionHttp, router)
	loginTransport.NewLoginHttp(cfg, *sessionHttp, loginService, router)
	verificationTransport.NewVerificationHttp(cfg, email, *sessionHttp, verificationService, router)
	recoveryTransport.NewRecoveryHttp(cfg, email, *sessionHttp, recoveryService, identityService, router)
	registrationTransport.NewRegistrationHttp(cfg, email, *sessionHttp, registrationService, verificationService, router)

	// Start HTTP server
	if err := transport.RunHttp(cfg, router); err != nil {
		log.Panic(err)
	}
}
