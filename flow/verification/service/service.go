package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/email"
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/ui/form"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"
)

var (
	errInvalidFlowID = func(src error, f string, i uuid.UUID) error {
		return internal.NewServiceClientError(src, "verification_flowid_invalid", "Invalid or expired flow id provided", &map[string]interface{}{
			"FlowID":     f,
			"IdentityID": i,
		})
	}
	errInvalidContact = func(i identity.Identity, c contact.VerifiableContact) error {
		return internal.NewServiceClientError(nil, "verification_invalid_contact", "Contact is either already verified or does not exist", &map[string]interface{}{
			"Identity": i,
			"Contact":  c,
		})
	}
	errInvalidContactMatch = func(i uuid.UUID, c contact.VerifiableContact) error {
		return internal.NewServiceClientError(nil, "verification_invalid_contact", "Contact is either already verified or does not exist", &map[string]interface{}{
			"IdentityID": i,
			"Contact":    c,
		})
	}
	errInvalidUser = func(f string, i uuid.UUID) error {
		return internal.NewServiceClientError(nil, "verification_invalid_user", "Account does not exist or invalid verification code provided", &map[string]interface{}{
			"FlowID":     f,
			"IdentityID": i,
		})
	}
	errInvalidPayload = func(f verification.Verification) error {
		return internal.NewServiceClientError(nil, "verification_invalid_payload", "Invalid payload provided", &map[string]interface{}{
			"Flow": f,
		})
	}
	errNanoIDGen = func() error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "verification_nanoid_gen", "Failed to generate new nanoid token")
	}
	errEmailSend = func() error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "verification_email_send", "Failed to send email")
	}
	errUpdateFlow = func(src error, f verification.Verification, i identity.Identity) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "verification_update_failed", "Failed to update flow")
	}
	errUpdateContact = func(src error, f verification.Verification, i identity.Identity) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "verification_update_failed", "Failed to update contact")
	}
)

type service struct {
	e   email.Client
	r   verification.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewVerificationService(e email.Client, r verification.Repository, cos contact.Service, cs credential.Service, is identity.Service) verification.Service {
	return &service{
		e:   e,
		r:   r,
		cos: cos,
		cs:  cs,
		is:  is,
	}
}

// TODO: Check if contact has an existing flow and reuse if it does
func (s *service) New(identity identity.Identity, contact contact.VerifiableContact, requestURL string, status verification.VerificationStatus) (*verification.Verification, error) {
	// Make sure contact provided belongs to identity
	flag := false
	for _, c := range identity.VerifiableContacts {
		if c.ID.String() == contact.ID.String() {
			flag = true
			break
		}
	}
	if !flag {
		return nil, errInvalidContactMatch(identity.ID, contact)
	}
	// Check if already verified
	if contact.Verified {
		return nil, errInvalidContact(identity, contact)
	}
	// Check if contact has an existing flow and reuse if it does
	ext, _ := s.r.GetByContact(contact.ID)
	if ext != nil && ext.Status != verification.Success && ext.ExpiresAt.After(time.Now()) {
		return ext, nil
	}
	// Create new FlowID
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	cfg := config.Get()
	// Generate form depending on status
	var newForm *form.Form
	switch status {
	case verification.SessionWarn:
		action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, fid)
		gen := generatePasswordForm(action)
		newForm = &gen
		break
	default:
		newForm = nil
	}
	// Make new flow with given parameters
	expire := time.Now().Add(time.Minute * 10)
	// If status is LinkPending then send email
	if status == verification.LinkPending {
		if err := s.sendEmail(fid, identity, contact.Value, false); err != nil {
			return nil, err
		}
	}
	// Create flow and store in repository
	created, err := s.r.Create(verification.Verification{
		FlowID:     fid,
		Status:     status,
		ExpiresAt:  expire,
		RequestURL: requestURL,

		Form:                newForm,
		VerifiableContactID: contact.ID,
		IdentityID:          identity.ID,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *service) NewWelcome(identity identity.Identity, contact contact.VerifiableContact, requestURL string) (*verification.Verification, error) {
	// Make sure contact provided belongs to identity
	flag := false
	for _, c := range identity.VerifiableContacts {
		if c.ID == contact.ID {
			flag = true
		}
	}
	if !flag {
		return nil, errInvalidContactMatch(identity.ID, contact)
	}
	// New FlowID
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	cfg := config.Get()
	// New Flow
	expire := time.Now().Add(cfg.Verification.Lifetime)
	newFlow := verification.Verification{
		FlowID:     fid,
		ExpiresAt:  expire,
		RequestURL: requestURL,
		Status:     verification.LinkPending,

		VerifiableContactID: contact.ID,
		IdentityID:          identity.ID,
	}
	// Send email
	if err := s.sendEmail(fid, identity, contact.Value, true); err != nil {
		return nil, err
	}
	// Create and store flow in repository
	created, err := s.r.Create(newFlow)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *service) Find(flowID string, identityID uuid.UUID) (*verification.Verification, error) {
	if flowID == "" {
		return nil, errInvalidFlowID(nil, flowID, identityID)
	}
	f, err := s.r.GetByFlowID(flowID)
	if err != nil || f == nil || f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlowID(err, flowID, identityID)
	}
	if f.IdentityID != identityID {
		return nil, errInvalidUser(flowID, identityID)
	}
	return f, nil
}

func (s *service) Verify(flowID string, identity identity.Identity, payload interface{}) (*verification.Verification, error) {
	f, err := s.Find(flowID, identity.ID)
	if err != nil {
		return nil, err
	}
	switch f.Status {
	case verification.LinkPending:
		return s.verifyLinkPending(identity, *f)
	case verification.SessionWarn:
		pay, ok := payload.(verification.SessionWarnPayload)
		// Check that valid payload was provided
		if !ok {
			return nil, errInvalidPayload(*f)
		}
		return s.verifySessionWarning(identity, *f, pay)
	}
	return nil, internal.NewServiceClientError(nil, "verification_invalid_flow", "Invalid verification flow provided", &map[string]interface{}{
		"Flow": f,
	})
}

// Internal methods
//
func (s *service) verifyLinkPending(identity identity.Identity, flow verification.Verification) (*verification.Verification, error) {
	// Compare IdentityID to make sure user attempting to verify is the proper user
	if flow.IdentityID != identity.ID {
		return nil, errInvalidUser(flow.FlowID, identity.ID)
	}
	// Run updates concurrently
	// - Update flow
	// - Update contact
	var eg errgroup.Group
	var updated *verification.Verification
	eg.Go(func() error {
		flow.Status = verification.Success
		up, err := s.r.Update(flow)
		if err != nil {
			return errUpdateFlow(err, flow, identity)
		}
		updated = up
		return nil
	})
	eg.Go(func() error {
		vcs := identity.VerifiableContacts
		for i, vc := range vcs {
			if vc.ID == flow.VerifiableContactID {
				now := time.Now()
				vcs[i].Verified = true
				vcs[i].UpdatedAt = &now
				vcs[i].VerifiedAt = &now
				vcs[i].State = contact.Completed
			}
		}
		_, err := s.cos.Add(vcs...)
		if err != nil {
			return errUpdateContact(err, flow, identity)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *service) verifySessionWarning(identity identity.Identity, flow verification.Verification, payload verification.SessionWarnPayload) (*verification.Verification, error) {
	// Check that payload has all the required information
	if err := validate.Check(payload); err != nil {
		return nil, err
	}
	// Compare IdentityID to make sure user attempting to verify is the proper user
	if flow.IdentityID != identity.ID {
		return nil, errInvalidUser(flow.FlowID, identity.ID)
	}
	// Compare passwords and find contact concurrently
	var eg errgroup.Group
	var foundContact contact.VerifiableContact
	eg.Go(func() error {
		// Compare passwords
		if err := s.cs.ComparePassword(flow.IdentityID, payload.Password); err != nil {
			return err
		}
		return nil
	})
	eg.Go(func() error {
		f, err := s.cos.Find(flow.VerifiableContactID.String())
		if err != nil {
			return err
		}
		foundContact = *f
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	// Send email
	if err := s.sendEmail(flow.FlowID, identity, foundContact.Value, false); err != nil {
		return nil, err
	}
	// Update flow appropriately
	flow.Form = nil
	flow.Status = verification.LinkPending
	updated, err := s.r.Update(flow)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *service) sendEmail(flowID string, identity identity.Identity, to string, isNew bool) error {
	cfg := config.Get()
	if isNew {
		wd := email.WelcomeTemplateData{
			ApplicationName: cfg.Name,
			FirstName:       identity.FirstName,
			VerificationURL: fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, flowID),
		}
		if err := s.e.Send(to, email.Welcome, wd); err != nil {
			return err
		}
		return nil
	}
	td := email.VerificationTemplateData{
		ApplicationName: cfg.Name,
		FirstName:       identity.FirstName,
		VerificationURL: fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Verification.URL, flowID),
	}
	// Send Email
	if err := s.e.Send(to, email.Verification, td); err != nil {
		return errEmailSend()
	}
	return nil
}
