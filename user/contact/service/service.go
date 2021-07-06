package service

import (
	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/user/contact"
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
	n, err := s.cr.Create(args...)
	if err != nil {
		return nil, idp.NewServiceClientError(err, "address_verifiable_create", "Invalid address provided", nil)
	}
	return n, nil
}
