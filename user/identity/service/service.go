package service

import (
	"errors"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	goaway "github.com/TwinProduction/go-away"
	"github.com/gofrs/uuid"
)

var (
	ErrDeleteUser        = errors.New("failed to delete user")
	ErrInvalidIdentityID = errors.New("invalid user id provided")
	ErrCreateUser        = errors.New("failed to create a new user")
	ErrInvalidUsername   = errors.New("invalid username provided. username is either already taken or contains invalid characters")
	ErrInvalidIdentifier = errors.New("invalid username or email provided. username or email is either already taken or contains invalid characters")

	errInvalidUsername = func(src error) error {
		return idp.NewServiceClientError(src, "identity_username_invalid", "Invalid username provided. Username is either already taken or contains invalid characters", nil)
	}
)

type service struct {
	ir  identity.Repository
	cs  credential.Service
	cos contact.Service
}

func NewIdentityService(ir identity.Repository, cs credential.Service, cos contact.Service) identity.Service {
	return &service{
		ir:  ir,
		cs:  cs,
		cos: cos,
	}
}

func (s *service) Create(i identity.Identity, username string, password string) (*identity.Identity, error) {
	// 1. Check for profanity in username
	if goaway.IsProfane(username) {
		return nil, errInvalidUsername(nil)
	}
	// 2. Create Identity
	builtUser := identity.Identity{
		FirstName: i.FirstName,
		LastName:  i.LastName,
		Email:     i.Email,
	}
	newUser, err := s.ir.Create(builtUser)
	if err != nil {
		return nil, idp.NewServiceClientError(err, "identity_create_fail", "Invalid email/username provided", nil)
	}
	// 3. Create Credential
	_, err = s.cs.CreatePassword(newUser.ID, password, []credential.Identifier{
		{
			Type:  "email",
			Value: i.Email,
		},
		{
			Type:  "username",
			Value: username,
		},
	})
	if err != nil {
		s.ir.Delete(newUser.ID, true)
		return nil, err
	}
	// 4. Add VerifiableAddress
	addrs := []address.VerifiableAddress{
		{
			IdentityID: newUser.ID,
			State:      address.Sent,
			Address:    newUser.Email,
		},
	}
	_, err = s.as.Add(addrs...)
	if err != nil {
		s.ir.Delete(newUser.ID, true)
		return nil, err
	}
	// 5. Return new user
	return newUser, nil
}

func (s *service) Find(i string) (*identity.Identity, error) {
	uid, err := uuid.FromString(i)
	if err == nil {
		f, err := s.ir.Get(uid, false)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	f, err := s.ir.GetIdentifier(i, false)
	if err != nil {
		return nil, err
	}
	return f, nil
}

