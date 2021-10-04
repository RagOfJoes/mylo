package email

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/sendgrid/sendgrid-go"
)

// SendWelcome sends a welcome email to new user
func (c *client) SendWelcome(to string, user identity.Identity, verificationURL string) error {
	// Check `to` is a valid email
	if err := validate.Var(to, "email"); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "", fmt.Sprintf("Value, %s, provided for the argument `to` must be a valid email.", to), map[string]interface{}{
			"Email": to,
		})
	}
	// Build payload
	cfg := config.Get()
	pay := Payload{
		From: Email{
			Name:  cfg.SendGrid.SenderName,
			Email: cfg.SendGrid.SenderEmail,
		},
		TemplateID: cfg.SendGrid.WelcomeTemplateID,
		Personalizations: []*Personalization{
			{
				To: []*Email{
					{
						Email: to,
						Name:  user.FirstName,
					},
				},
				DynamicTemplateData: map[string]interface{}{
					"ApplicationName": cfg.Name,
					"FirstName":       user.FirstName,
					"VerificationURL": verificationURL,
				},
			},
		},
	}
	body, err := json.Marshal(pay)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Email_FailedMarshal", "Failed to marshal payload", map[string]interface{}{
			"Email":   to,
			"Payload": pay,
		})
	}
	// Make request to SendGrid
	request := sendgrid.GetRequest(c.apiKey, "/v3/mail/send", c.host)
	request.Method = "POST"
	request.Body = body
	if _, err := sendgrid.API(request); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Email_FailedSend", err.Error(), map[string]interface{}{
			"Email":   to,
			"Payload": pay,
		})
	}
	return nil
}
