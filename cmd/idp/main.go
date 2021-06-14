package main

import (
	"log"
	"os"

	registrationGorm "github.com/RagOfJoes/idp/flow/registration/repository/gorm"
	registrationService "github.com/RagOfJoes/idp/flow/registration/service"
	registrationTransport "github.com/RagOfJoes/idp/flow/registration/transport"
	"github.com/RagOfJoes/idp/persistence"
	"github.com/RagOfJoes/idp/transport"
	addressGorm "github.com/RagOfJoes/idp/user/address/repository/gorm"
	addressService "github.com/RagOfJoes/idp/user/address/service"
	credentialGorm "github.com/RagOfJoes/idp/user/credential/repository/gorm"
	credentialService "github.com/RagOfJoes/idp/user/credential/service"
	identityGorm "github.com/RagOfJoes/idp/user/identity/repository/gorm"
	identityService "github.com/RagOfJoes/idp/user/identity/service"
	_ "github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

func init() {
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
	httpSrvCfg := transport.HttpConfig{
		Host:   os.Getenv("HOST"),
		Port:   os.Getenv("PORT"),
		Scheme: os.Getenv("SCHEME"),

		RemoveExtraSlashes: true,
	}
	ginEng := transport.NewHttp(httpSrvCfg)
	ginEng.Use(transport.RateLimiterMiddleware(100), transport.ErrorMiddleware())

	// Setup repositories
	ar := addressGorm.NewGormAddressRepository(db)
	cr := credentialGorm.NewGormCredentialRepository(db)
	ir := identityGorm.NewGormUserRepository(db)
	rr := registrationGorm.NewGormRegistrationRepository(db)
	// Setup services
	as := addressService.NewAddressService(ar)
	ap := credentialService.NewArgonParams(64*1024, 3, 2, 16, 32)
	cs := credentialService.NewCredentialService(ap, cr)
	is := identityService.NewIdentityService(ir, cs, as)
	rs := registrationService.NewRegistrationService(rr, is)

	// Attach routes
	registrationTransport.NewRegistrationHttp(rs, ginEng)

	if err := transport.RunHttp(httpSrvCfg, ginEng); err != nil {
		log.Panic(err)
	}
}
