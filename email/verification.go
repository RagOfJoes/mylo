package email

import (
	"encoding/json"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/internal/validate"
	"github.com/RagOfJoes/mylo/user/identity"
	"github.com/sendgrid/sendgrid-go"
)

func (c *client) SendVerification(to string, user identity.Identity, verificationURL string) error {
	// Check `to` is a valid email
	if err := validate.Var(to, "email"); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Value, %s, provided for the argument `to` must be a valid email.", to)
	}
	// Build payload
	cfg := config.Get()
	pay := Payload{
		From: Email{
			Name:  cfg.SendGrid.SenderName,
			Email: cfg.SendGrid.SenderEmail,
		},
		TemplateID: cfg.SendGrid.VerificationTemplateID,
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
