package config

type SendGrid struct {
	APIKey      string `validate:"required"`
	SenderName  string `validate:"required"`
	SenderEmail string `validate:"required,email"`

	WelcomeTemplateID      string `validate:"required"`
	VerificationTemplateID string `validate:"required"`
	RecoveryTemplateID     string `validate:"required"`
}
