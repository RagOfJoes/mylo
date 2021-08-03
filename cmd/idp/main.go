package main

import (
	"log"
	"os"
	"time"

	loginGorm "github.com/RagOfJoes/idp/flow/login/repository/gorm"
	loginService "github.com/RagOfJoes/idp/flow/login/service"
	loginTransport "github.com/RagOfJoes/idp/flow/login/transport"
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
	"github.com/alexedwards/scs/redisstore"
	_ "github.com/go-playground/validator/v10"
	"github.com/gomodule/redigo/redis"
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

	// Setup repositories
	cor := contactGorm.NewGormContactRepository(db)
	cr := credentialGorm.NewGormCredentialRepository(db)
	ir := identityGorm.NewGormUserRepository(db)
	rr := registrationGorm.NewGormRegistrationRepository(db)
	lr := loginGorm.NewGormLoginRepository(db)
	// Setup services
	cos := contactService.NewContactService(cor)
	ap := credentialService.NewArgonParams(64*1024, 2, 2, 16, 32)
	cs := credentialService.NewCredentialService(ap, cr)
	is := identityService.NewIdentityService(ir)
	// Flow Services
	// These will essentially stitch all other services together
	rs := registrationService.NewRegistrationService(rr, cos, cs, is)
	ls := loginService.NewLoginService(lr, cos, cs, is)

	// Setup HTTP transport
	// Create session manager
	pool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", os.Getenv("REDIS_URL"))
			if err != nil {
				return nil, err
			}
			if _, err := conn.Do("AUTH", os.Getenv("REDIS_PASSWORD")); err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	sessionManager, err := session.NewManager(false, "sid", time.Hour*24*14)
	sessionManager.Store = redisstore.New(pool)
	if err != nil {
		log.Panic(err)
	}
	// Setup HTTP config
	httpSrvCfg := transport.HttpConfig{
		Host:   os.Getenv("HOST"),
		Port:   os.Getenv("PORT"),
		Scheme: os.Getenv("SCHEME"),

		RemoveExtraSlashes: true,
	}
	// Setup HTTP Server
	ginEng := transport.NewHttp(httpSrvCfg)
	// Attach Middlewares
	//
	// Order of execution:
	// 1. Rate Limiter
	// 2. Auth Middleware (Checks session for identity if found then passes to context)
	// 3. Execute route
	// 4. Error Middleware handles any errors that were generated from route execution
	ginEng.Use(transport.RateLimiterMiddleware(100), session.AuthMiddleware(sessionManager), transport.ErrorMiddleware())

	// Attach routes
	registrationTransport.NewRegistrationHttp(rs, sessionManager, ginEng)
	loginTransport.NewLoginHttp(ls, sessionManager, ginEng)

	// Start HTTP server
	if err := transport.RunHttp(httpSrvCfg, sessionManager.LoadAndSave(ginEng)); err != nil {
		log.Panic(err)
	}
}
