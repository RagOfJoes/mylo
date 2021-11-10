package service

import (
	"time"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
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
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", verification.ErrInvalidContact)
	}
	if existing := s.getExistingFlow(contact); existing != nil {
		return existing, nil
	}
	newFlow, err := verification.NewLinkPending(requestURL, contact.ID, identity.ID)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new verification flow")
	}
	return created, nil
}

func (s *service) NewSessionWarn(identity identity.Identity, contact contact.Contact, requestURL string) (*verification.Flow, error) {
	if !isValidContact(contact, identity) {
		return nil, internal.NewErrorf(internal.ErrorCodeInternal, "%v", verification.ErrInvalidContact)
	}
	if existing := s.getExistingFlow(contact); existing != nil {
		return existing, nil
	}
	newFlow, err := verification.NewSessionWarn(requestURL, contact.ID, identity.ID)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new verification flow with status of `SessionWarn`")
	}
	return created, nil
}

func (s *service) Find(id string, identity identity.Identity) (*verification.Flow, error) {
	if id == "" {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
	}

	flow, err := s.r.GetByFlowIDOrVerifyID(id)
	if err != nil || flow == nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if !flow.BelongsTo(identity.ID) {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
	}
	switch flow.Status {
	case verification.SessionWarn:
		if flow.FlowID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
		}
	case verification.LinkPending:
		if flow.VerifyID != id {
			return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
		}
	}
	return flow, nil
}

func (s *service) SubmitSessionWarn(flow verification.Flow, identity identity.Identity, payload verification.SessionWarnPayload) (*verification.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if !flow.BelongsTo(identity.ID) {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
	}
	if err := validate.Check(payload); err != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", err)
	}

	// Check if password is correct
	if err := s.cs.ComparePassword(flow.IdentityID, payload.Password); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", verification.ErrInvalidPassword)
	}
	// Update flow to next Status
	if err := flow.Next(); err != nil {
		return nil, err
	}
	updated, err := s.r.Update(flow)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update verification flow: %s", flow.ID)
	}
	return updated, nil
}

func (s *service) Verify(flow verification.Flow, identity identity.Identity) (*verification.Flow, error) {
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if !flow.BelongsTo(identity.ID) {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", verification.ErrInvalidExpiredFlow)
	}

	for idx, cont := range identity.Contacts {
		if cont.ID == flow.ContactID {
			now := time.Now()
			identity.Contacts[idx].Verified = true
			identity.Contacts[idx].UpdatedAt = &now
			identity.Contacts[idx].VerifiedAt = &now
			identity.Contacts[idx].State = contact.Completed
			break
		}
	}
	_, err := s.cos.Add(identity.Contacts...)
	if err != nil {
		return nil, err
	}
	// Update flow to next Status
	if err := flow.Next(); err != nil {
		return nil, err
	}
	verified, err := s.r.Update(flow)
	// TODO: Revert contacts on error
	if err != nil {
		return nil, err
	}
	return verified, nil
}
