package email

import (
	"fmt"
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
)

var (
	errInvalidTemplate = func(src error, template, to string) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Email_InvalidTemplate", fmt.Sprintf("Invalid %s template data provided", template), map[string]interface{}{
			"Email": to,
		})
	}
)

type client struct {
	sender  Email
	host    string
	apiKey  string
	appName string
	// Template ID for Welcome template
	welcomeID string
	// Template ID for Verification template
	verificationID string
}

func New() Client {
	cfg := config.Get()
	return &client{
		appName:        cfg.Name,
		apiKey:         cfg.SendGrid.APIKey,
		welcomeID:      cfg.SendGrid.WelcomeTemplateID,
		verificationID: cfg.SendGrid.VerificationTemplateID,
		sender: Email{
			Name:  cfg.SendGrid.SenderName,
			Email: cfg.SendGrid.SenderEmail,
		},
	}
}
