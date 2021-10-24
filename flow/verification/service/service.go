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
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"golang.org/x/sync/errgroup"
)

var (
	errInvalidFlowID = func(src error, i identity.Identity, f string) error {
		return internal.NewServiceClientError(src, "Verification_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Identity": i,
			"FlowID":   f,
		})
	}
	errInvalidVerifyID = func(src error, i identity.Identity, v string) error {
		return internal.NewServiceClientError(src, "Verification_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Identity": i,
			"VerifyID": v,
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

func (s *service) NewDefault(identity identity.Identity, contact contact.Contact, requestURL string) (*verification.Flow, error) {
	if !isValidContact(contact, identity) {
		return nil, errInvalidContact(identity, contact)
	}
	if existing := s.getExistingFlow(contact); existing != nil {
		return existing, nil
	}
	// Create new FlowID
	flowID, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err, identity)
	}
	// Create new VerifyID
	verifyID, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err, identity)
	}
	cfg := config.Get()

	// Create flow and store in repository
	newFlow, err := s.r.Create(verification.Flow{
		FlowID:     flowID,
		VerifyID:   verifyID,
		RequestURL: requestURL,
		Status:     verification.LinkPending,
		ExpiresAt:  time.Now().Add(cfg.Verification.Lifetime),

		Form:       nil,
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

func (s *service) NewSessionWarn(identity identity.Identity, contact contact.Contact, requestURL string) (*verification.Flow, error) {
	if !isValidContact(contact, identity) {
		return nil, errInvalidContact(identity, contact)
	}
	if existing := s.getExistingFlow(contact); existing != nil {
		return existing, nil
	}
	// Create new FlowID
	flowID, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err, identity)
	}
	// Create new VerifyID
	verifyID, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err, identity)
	}

	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, flowID)
	form := generatePasswordForm(action)
	// Create flow and store in repository
	newFlow, err := s.r.Create(verification.Flow{
		FlowID:     flowID,
		VerifyID:   verifyID,
		RequestURL: requestURL,
		Status:     verification.LinkPending,
		ExpiresAt:  time.Now().Add(cfg.Verification.Lifetime),

		Form:       &form,
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

func (s *service) Find(id string, identity identity.Identity) (*verification.Flow, error) {
	if id == "" {
		return nil, errInvalidFlowID(nil, identity, id)
	}
	// Try to get Flow
	flow, err := s.r.GetByFlowIDOrVerifyID(id)
	// Check for error or empty flow
	if err != nil || flow == nil {
		return nil, errInvalidFlowID(err, identity, id)
	}
	// Check for expired flow or if flow belongs to user
	if flow.ExpiresAt.Before(time.Now()) || flow.IdentityID != identity.ID {
		return nil, errInvalidFlow(nil, identity, *flow)
	}

	// Make sure Status and ID provided are correct
	switch flow.Status {
	case verification.SessionWarn:
		if flow.FlowID != id {
			return nil, errInvalidFlowID(nil, identity, id)
		}
	case verification.LinkPending:
		if flow.VerifyID != id {
			return nil, errInvalidVerifyID(nil, identity, id)
		}
	}
	return flow, nil
}

func (s *service) SubmitSessionWarn(flow verification.Flow, identity identity.Identity, payload verification.SessionWarnPayload) (*verification.Flow, error) {
	// Validate flow then check if flow belongs to user
	if err := validate.Check(flow); err != nil || flow.IdentityID != identity.ID {
		return nil, errInvalidFlow(err, identity, flow)
	}
	// Validate payload
	if err := validate.Check(payload); err != nil {
		return nil, errInvalidSessionWarn(err, identity, flow, payload)
	}
	// Check if password is correct
	if err := s.cs.ComparePassword(flow.IdentityID, payload.Password); err != nil {
		return nil, internal.NewServiceClientError(err, "Verification_InvalidSessionWarnPayload", "Invalid password provided", map[string]interface{}{
			"Identity": identity,
			"Flow":     flow,
		})
	}
	// Update flow appropriately
	updateFlow := flow
	updateFlow.Form = nil
	updateFlow.Status = verification.LinkPending
	updatedFlow, err := s.r.Update(updateFlow)
	if err != nil {
		return nil, err
	}
	return updatedFlow, nil
}

func (s *service) Verify(flow verification.Flow, identity identity.Identity) (*verification.Flow, error) {
	// Validate flow then check if flow belongs to user
	if err := validate.Check(flow); err != nil || flow.IdentityID != identity.ID {
		return nil, errInvalidFlow(err, identity, flow)
	}

	// Run updates concurrently
	// - Update flow
	// - Update contact
	var eg errgroup.Group
	var updatedFlow *verification.Flow

	updateFlow := flow
	updateFlow.Status = verification.Success
	eg.Go(func() error {
		update, err := s.r.Update(updateFlow)
		if err != nil {
			return errUpdateFlow(err, identity, flow, updateFlow)
		}
		updatedFlow = update
		return nil
	})
	eg.Go(func() error {
		contacts := identity.Contacts
		for idx, cont := range contacts {
			if cont.ID == flow.ContactID {
				now := time.Now()
				contacts[idx].Verified = true
				contacts[idx].UpdatedAt = &now
				contacts[idx].VerifiedAt = &now
				contacts[idx].State = contact.Completed
			}
		}
		_, err := s.cos.Add(contacts...)
		if err != nil {
			_, file, line, _ := runtime.Caller(1)
			return internal.NewServiceInternalError(err, file, line, "Verification_FailedUpdateContact", "Failed to update contact", map[string]interface{}{
				"Identity": identity,
				"Flow":     flow,
			})
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return updatedFlow, nil
}
