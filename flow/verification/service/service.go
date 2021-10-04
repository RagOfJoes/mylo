package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
)

var (
	errInvalidFlowID = func(src error, i identity.Identity, f string) error {
		return internal.NewServiceClientError(src, "Verification_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Identity": i,
			"FlowID":   f,
		})
	}
	errInvalidFlow = func(src error, i identity.Identity, f verification.Flow) error {
		return internal.NewServiceClientError(src, "Verification_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Identity": i,
			"Flow":     f,
		})
	}
	errInvalidContact = func(i identity.Identity, c contact.Contact) error {
		return internal.NewServiceClientError(nil, "Verification_InvalidContact", "Contact is either already verified or does not exist", map[string]interface{}{
			"Identity": i,
			"Contact":  c,
		})
	}
	errInvalidSessionWarn = func(src error, i identity.Identity, f verification.Flow, p verification.SessionWarnPayload) error {
		err := "Invalid payload provided"
		if src != nil {
			err = src.Error()
		}
		return internal.NewServiceClientError(src, "Verification_InvalidSessionWarnPayload", err, map[string]interface{}{
			"Identity": i,
			"Flow":     f,
			"Payload":  p,
		})
	}
	// Internal Errors
	errNanoIDGen = func(src error, i identity.Identity) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Verification_FailedNanoID", "Failed to generate nano id", map[string]interface{}{
			"Identity": i,
		})
	}
	errUpdateFlow = func(src error, i identity.Identity, o verification.Flow, n verification.Flow) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Verification_FailedUpdate", "Failed to update flow", map[string]interface{}{
			"Identity": i,
			"OldFlow":  o,
			"NewFlow":  n,
		})
	}
)

type service struct {
	r   verification.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewVerificationService(r verification.Repository, cos contact.Service, cs credential.Service, is identity.Service) verification.Service {
	return &service{
		r:   r,
		cos: cos,
		cs:  cs,
		is:  is,
	}
}

func (s *service) New(identity identity.Identity, contact contact.Contact, requestURL string, status verification.Status) (*verification.Flow, error) {
	// Make sure contact provided belongs to identity
	flag := false
	for _, c := range identity.Contacts {
		if c.ID.String() == contact.ID.String() {
			flag = true
			break
		}
	}
	// Check if already verified
	if !flag || contact.Verified {
		return nil, errInvalidContact(identity, contact)
	}
	// Check if contact has a valid existing flow and reuse if it does
	ext, _ := s.r.GetByContact(contact.ID)
	if ext != nil && ext.Status != verification.Success && ext.ExpiresAt.After(time.Now()) {
		return ext, nil
	}
	// Create new FlowID
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err, identity)
	}
	cfg := config.Get()
	// Generate form depending on status
	var newForm *form.Form
	switch status {
	case verification.SessionWarn:
		action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, fid)
		gen := generatePasswordForm(action)
		newForm = &gen
	default:
		newForm = nil
	}
	// Create flow and store in repository
	newFlow, err := s.r.Create(verification.Flow{
		FlowID:     fid,
		Status:     status,
		RequestURL: requestURL,
		ExpiresAt:  time.Now().Add(cfg.Verification.Lifetime),

		Form:       newForm,
		ContactID:  contact.ID,
		IdentityID: identity.ID,
	})
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Verification_FailedCreate", "Failed to create new verification flow", map[string]interface{}{
			"Identity": identity,
			"Flow":     newFlow,
		})
	}
	return newFlow, nil
}

func (s *service) Find(flowID string, identity identity.Identity) (*verification.Flow, error) {
	if flowID == "" {
		return nil, errInvalidFlowID(nil, identity, flowID)
	}
	// Try to get Flow
	f, err := s.r.GetByFlowID(flowID)
	// Check for error or empty flow
	if err != nil || f == nil {
		return nil, errInvalidFlowID(err, identity, flowID)
	}
	// Check for expired flow or if flow belongs to user
	if f.ExpiresAt.Before(time.Now()) || f.IdentityID != identity.ID {
		return nil, errInvalidFlow(nil, identity, *f)
	}
	return f, nil
}

func (s *service) Verify(flow verification.Flow, identity identity.Identity, payload interface{}) (*verification.Flow, error) {
	// Validate flow then check if flow belongs to user
	if err := validate.Check(flow); err != nil || flow.IdentityID != identity.ID {
		return nil, errInvalidFlow(err, identity, flow)
	}
	// Check status
	switch flow.Status {
	case verification.LinkPending:
		return s.verifyLinkPending(flow, identity)
	case verification.SessionWarn:
		pay, ok := payload.(verification.SessionWarnPayload)
		// Check that valid payload was provided
		if !ok {
			return nil, errInvalidSessionWarn(nil, identity, flow, verification.SessionWarnPayload{})
		}
		return s.verifySessionWarning(flow, identity, pay)
	case verification.Success:
		return &flow, nil
	}
	// If the status provided is invalid
	return nil, errInvalidFlow(nil, identity, flow)
}
