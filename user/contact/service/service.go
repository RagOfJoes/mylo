package service

import (
	"fmt"
	"runtime"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/gofrs/uuid"
)

var (
	errAddInvalidContacts = func(err error) error {
		return idp.NewServiceClientError(err, "verifiable_contact_add", "Invalid contact values provided", nil)
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

func (s *service) Add(args ...contact.VerifiableContact) ([]*contact.VerifiableContact, error) {
	if len(args) == 0 {
		return nil, errAddInvalidContacts(nil)
	}

	identityID := ""
	for _, arg := range args {
		if _, err := uuid.FromString(identityID); identityID != "" && err != nil {
			return nil, idp.NewServiceClientError(err, "verifiable_contact_add", "Invalid user provided", nil)
		}
		identityID = arg.IdentityID.String()
	}

	if err := s.cr.DeleteAllUser(uuid.FromStringOrNil(identityID)); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, idp.NewServiceInternalError(file, line, "verifiable_contact_add", fmt.Sprintf("Failed to delete %s contacts", identityID))
	}

	n, err := s.cr.Create(args...)
	if err != nil {
		return nil, errAddInvalidContacts(err)
	}
	return n, nil
}
