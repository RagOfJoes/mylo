package email

import (
	"github.com/RagOfJoes/idp/internal/config"
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
	// Template ID for Recovery template
	recoveryID string
}

func New() Client {
	cfg := config.Get()
	return &client{
		appName: cfg.Name,
		apiKey:  cfg.SendGrid.APIKey,

		welcomeID:      cfg.SendGrid.WelcomeTemplateID,
		verificationID: cfg.SendGrid.VerificationTemplateID,
		recoveryID:     cfg.SendGrid.RecoveryTemplateID,
		sender: Email{
			Name:  cfg.SendGrid.SenderName,
			Email: cfg.SendGrid.SenderEmail,
		},
	}
}
