package service

import (
	"fmt"
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/gofrs/uuid"
)

var (
	errAddInvalidContacts = func(err error) error {
		return internal.NewServiceClientError(err, "contact_add", "Invalid contact values provided", nil)
	}
)

type service struct {
	cr contact.Repository
}

func NewContactService(cr contact.Repository) contact.Service {
	return &service{
		cr: cr,
	}
}

func (s *service) Add(args ...contact.Contact) ([]contact.Contact, error) {
	if len(args) == 0 {
		return nil, errAddInvalidContacts(nil)
	}

	identityID := args[0].IdentityID
	if err := s.cr.DeleteAllUser(identityID); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "contact_add", fmt.Sprintf("Failed to delete %s contacts", identityID))
	}

	n, err := s.cr.Create(args...)
	if err != nil {
		return nil, errAddInvalidContacts(err)
	}
	return n, nil
}

func (s *service) Find(i string) (*contact.Contact, error) {
	uid, err := uuid.FromString(i)
	if err == nil {
		f, err := s.cr.Get(uid)
		if err != nil {
			return nil, internal.NewServiceClientError(err, "contact_find", "Invalid ID provided", &map[string]interface{}{
				"ContactID": uid,
			})
		}
		return f, nil
	}
	f, err := s.cr.GetByValue(i)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "contact_find", "Invalid contact value provided", &map[string]interface{}{
			"Contact": contact.Contact{Value: i},
		})
	}
	return f, nil
}
