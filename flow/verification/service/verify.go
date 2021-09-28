package service

import (
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
	"golang.org/x/sync/errgroup"
)

// Verifies LinkPending flow
func (s *service) verifyLinkPending(flow verification.Flow, identity identity.Identity) (*verification.Flow, error) {
	// Run updates concurrently
	// - Update flow
	// - Update contact
	var eg errgroup.Group
	var updated *verification.Flow

	newFlow := flow
	newFlow.Status = verification.Success
	eg.Go(func() error {
		up, err := s.r.Update(newFlow)
		if err != nil {
			return errUpdateFlow(err, identity, flow, newFlow)
		}
		updated = up
		return nil
	})
	eg.Go(func() error {
		vcs := identity.Contacts
		for i, vc := range vcs {
			if vc.ID == flow.ContactID {
				now := time.Now()
				vcs[i].Verified = true
				vcs[i].UpdatedAt = &now
				vcs[i].VerifiedAt = &now
				vcs[i].State = contact.Completed
			}
		}
		_, err := s.cos.Add(vcs...)
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
	return updated, nil
}

// Verifies SessionWarning flow
func (s *service) verifySessionWarning(flow verification.Flow, identity identity.Identity, payload verification.SessionWarnPayload) (*verification.Flow, error) {
	// Check that payload has all the required information
	if err := validate.Check(payload); err != nil {
		return nil, errInvalidSessionWarn(err, identity, flow, payload)
	}
	// Compare passwords and find contact concurrently
	var eg errgroup.Group
	var foundContact contact.Contact
	eg.Go(func() error {
		// Compare passwords
		if err := s.cs.ComparePassword(flow.IdentityID, payload.Password); err != nil {
			return err
		}
		return nil
	})
	eg.Go(func() error {
		f, err := s.cos.Find(flow.ContactID.String())
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
	if err := s.sendEmail(flow, identity, foundContact.Value, false); err != nil {
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
