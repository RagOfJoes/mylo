package email

import (
	"github.com/RagOfJoes/mylo/internal/config"
)

type client struct {
	sender  Email
	apiKey  string
	appName string
	// Template ID for Welcome template
	welcomeID string
	// Template ID for Verification template
	verificationID string
	// Template ID for Recovery template
	recoveryID string
}

func New(cfg config.Configuration) Client {
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
