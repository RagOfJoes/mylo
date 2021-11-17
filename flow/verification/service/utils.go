package service

import (
	"context"

	"github.com/RagOfJoes/mylo/flow/verification"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/RagOfJoes/mylo/user/identity"
)

// Check if Contact provided actually belongs to the User
func isValidContact(contact contact.Contact, identity identity.Identity) bool {
	flag := false
	for _, c := range identity.Contacts {
		if c.ID.String() == contact.ID.String() {
			flag = true
			break
		}
	}
	if contact.Verified {
		return false
	}
	return flag
}

// Check if user has an existing valid flow
func (s *service) getExistingFlow(ctx context.Context, contact contact.Contact) *verification.Flow {
	existing, _ := s.r.GetByContactID(ctx, contact.ID)
	if existing != nil && existing.Valid() == nil {
		return existing
	}
	return nil
}
