package main

import (
	"log"

	"github.com/RagOfJoes/idp/email"
	loginGorm "github.com/RagOfJoes/idp/flow/login/repository/gorm"
	loginService "github.com/RagOfJoes/idp/flow/login/service"
	loginTransport "github.com/RagOfJoes/idp/flow/login/transport"
	registrationGorm "github.com/RagOfJoes/idp/flow/registration/repository/gorm"
	registrationService "github.com/RagOfJoes/idp/flow/registration/service"
	registrationTransport "github.com/RagOfJoes/idp/flow/registration/transport"
	verificationGorm "github.com/RagOfJoes/idp/flow/verification/repository/gorm"
	verificationService "github.com/RagOfJoes/idp/flow/verification/service"
	verificationTransport "github.com/RagOfJoes/idp/flow/verification/transport"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/persistence"
	"github.com/RagOfJoes/idp/session"
	"github.com/RagOfJoes/idp/transport"
	contactGorm "github.com/RagOfJoes/idp/user/contact/repository/gorm"
	contactService "github.com/RagOfJoes/idp/user/contact/service"
	credentialGorm "github.com/RagOfJoes/idp/user/credential/repository/gorm"
	credentialService "github.com/RagOfJoes/idp/user/credential/service"
	identityGorm "github.com/RagOfJoes/idp/user/identity/repository/gorm"
	identityService "github.com/RagOfJoes/idp/user/identity/service"
	identityTransport "github.com/RagOfJoes/idp/user/identity/transport"
)

func init() {
	// Load configuration
	if err := config.Setup("config", "yaml", "."); err != nil {
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
	cor := contactGorm.NewGormContactRepository(db)
	cr := credentialGorm.NewGormCredentialRepository(db)
	ir := identityGorm.NewGormUserRepository(db)
	vr := verificationGorm.NewGormVerificationRepository(db)
	rr := registrationGorm.NewGormRegistrationRepository(db)
	lr := loginGorm.NewGormLoginRepository(db)
	// Setup services
	cos := contactService.NewContactService(cor)
	cs := credentialService.NewCredentialService(cr)
	is := identityService.NewIdentityService(ir)
	// Flow Services
	// These will essentially stitch all other services together
	vs := verificationService.NewVerificationService(vr, cos, cs, is)
	rs := registrationService.NewRegistrationService(rr, cos, cs, is)
	ls := loginService.NewLoginService(lr, cos, cs, is)

	// Create session manager
	sessionManager, err := session.NewManager()
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
	// 3. Auth Middleware (Checks session for identity if found then passes to context)
	// 4. Execute route
	// 5. Error Middleware handles any errors that were generated from route execution
	if cfg.Server.RPS > 0 {
		router.Use(transport.RateLimiterMiddleware(cfg.Server.RPS))
	}
	router.Use(transport.SecurityMiddleware(), session.AuthMiddleware(sessionManager, is), transport.ErrorMiddleware())

	// Attach routes
	identityTransport.NewIdentityHttp(sessionManager, router)
	verificationTransport.NewVerificationHttp(email, sessionManager, vs, router)
	registrationTransport.NewRegistrationHttp(email, sessionManager, rs, vs, router)
	loginTransport.NewLoginHttp(ls, sessionManager, router)

	// Start HTTP server
	if err := transport.RunHttp(sessionManager.LoadAndSave(router)); err != nil {
		log.Panic(err)
	}
}
