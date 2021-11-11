package email

import (
	"encoding/json"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/sendgrid/sendgrid-go"
)

func (c *client) SendRecovery(to []string, recoveryURL string) error {
	// Check `to` is a valid email and build Email
	var emails []*Email
	var validationErr error
	for _, e := range to {
		if err := validate.Var(e, "email"); err != nil {
			validationErr = internal.WrapErrorf(err, internal.ErrorCodeInternal, "Value, %s, provided for the argument `to` must be a valid email.", to)
			break
		}
		emails = append(emails, &Email{
			Email: e,
		})
	}
	if validationErr != nil {
		return validationErr
	}
	// Build payload
	cfg := config.Get()
	pay := Payload{
		From: Email{
			Name:  cfg.SendGrid.SenderName,
			Email: cfg.SendGrid.SenderEmail,
		},
		TemplateID: cfg.SendGrid.RecoveryTemplateID,
		Personalizations: []*Personalization{
			{
				To: emails,
				DynamicTemplateData: map[string]interface{}{
					"ApplicationName": cfg.Name,
					"RecoveryURL":     recoveryURL,
				},
			},
		},
	}
	body, err := json.Marshal(pay)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to marshal payload")
	}
	// Make request to SendGrid
	request := sendgrid.GetRequest(c.apiKey, "/v3/mail/send", c.host)
	request.Method = "POST"
	request.Body = body
	if _, err := sendgrid.API(request); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to send email")
	}
	return nil
}
