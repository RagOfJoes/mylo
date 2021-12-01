package email

import (
	"encoding/json"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/validate"
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
	pay := Payload{
		From:       c.sender,
		TemplateID: c.recoveryID,
		Personalizations: []*Personalization{
			{
				To: emails,
				DynamicTemplateData: map[string]interface{}{
					"ApplicationName": c.appName,
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
	request := sendgrid.GetRequest(c.apiKey, "/v3/mail/send", "")
	request.Method = "POST"
	request.Body = body
	if _, err := sendgrid.API(request); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to send email")
	}
	return nil
}
