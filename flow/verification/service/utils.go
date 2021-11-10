package service

import (
	"github.com/RagOfJoes/idp/flow/verification"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/identity"
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
func (s *service) getExistingFlow(contact contact.Contact) *verification.Flow {
	existing, _ := s.r.GetByContactID(contact.ID)
	if existing != nil && existing.Valid() == nil {
		return existing
	}
	return nil
}
