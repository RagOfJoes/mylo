package service

import (
	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/user/address"
)

type service struct {
	ar address.Repository
}

func NewAddressService(ar address.Repository) address.Service {
	return &service{
		ar: ar,
	}
}

func (s *service) Add(args ...address.VerifiableAddress) ([]*address.VerifiableAddress, error) {
	n, err := s.ar.Create(args...)
	if err != nil {
		return nil, idp.NewServiceClientError(err, "address_verifiable_create", "Invalid address provided", nil)
	}
	return n, nil
}
