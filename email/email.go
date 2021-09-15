package email

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/sendgrid/sendgrid-go"
)

var (
	errInvalidTemplate = func(t string) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "email_invalid_template", fmt.Sprintf("Invalid %s template data provided", t))
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

func (c *client) Send(to string, template Template, data interface{}) error {
	if err := validate.Var(to, "email"); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "email_invalid_email", fmt.Sprintf("Value, %s, provided for the argument `to` must be a valid email.", to))
	}

	payload := Payload{}
	personalization := &Personalization{}
	switch template {
	case Welcome:
		welcomeData, ok := data.(WelcomeTemplateData)
		if !ok {
			return errInvalidTemplate("Welcome")
		}

		mapData, err := structToMap(welcomeData)
		if err != nil {
			return err
		}
		payload.TemplateID = c.welcomeID
		personalization.DynamicTemplateData = mapData
	case Verification:
		verificationData, ok := data.(VerificationTemplateData)
		if !ok {
			return errInvalidTemplate("Verification")
		}
		mapData, err := structToMap(verificationData)
		if err != nil {
			return err
		}
		payload.TemplateID = c.welcomeID
		personalization.DynamicTemplateData = mapData
	default:
		return errInvalidTemplate(string(template))
	}
	payload.From = c.sender
	personalization.To = append(personalization.To, &Email{Email: to})
	payload.Personalizations = append(payload.Personalizations, personalization)

	body, err := json.Marshal(payload)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "email_failed_marshal", "Failed to marshal payload")
	}

	request := sendgrid.GetRequest(c.apiKey, "/v3/mail/send", c.host)
	request.Method = "POST"
	request.Body = body

	if _, err := sendgrid.API(request); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "email_failed_send", err.Error())
	}
	return nil
}

// Converts struct to map
func structToMap(d interface{}) (map[string]interface{}, error) {
	var m map[string]interface{}
	bytes, err := json.Marshal(d)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "email_failed_marshal", "Failed to marshal template data")
	}
	if err := json.Unmarshal(bytes, &m); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "email_failed_unmarshal", "Failed to unmarshal template data")
	}
	return m, nil
}
