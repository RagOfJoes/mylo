package service

import (
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/identity"
	goaway "github.com/TwinProduction/go-away"
	"github.com/gofrs/uuid"
)

var (
	errInvalidID = func(src error) error {
		return internal.NewServiceClientError(src, "identity_id_invalid", "Invalid id provided", nil)
	}
	errInvalidUsername = func(src error) error {
		return internal.NewServiceClientError(src, "identity_username_invalid", "Invalid username provided. Username is either already taken or contains invalid characters", nil)
	}
)

type service struct {
	ir identity.Repository
}

func NewIdentityService(ir identity.Repository) identity.Service {
	return &service{
		ir: ir,
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
		return nil, internal.NewServiceClientError(err, "identity_create_fail", "Invalid email/username provided", nil)
	}
	// 3. Return new user
	return newUser, nil
}

func (s *service) Find(i string) (*identity.Identity, error) {
	uid, err := uuid.FromString(i)
	if err == nil {
		f, err := s.ir.Get(uid, false)
		if err != nil {
			return nil, errInvalidID(err)
		}
		return f, nil
	}
	f, err := s.ir.GetIdentifier(i, false)
	if err != nil {
		return nil, errInvalidUsername(err)
	}
	return f, nil
}

// Delete defines a delete function for User identity
func (s *service) Delete(i string, perm bool) error {
	id, err := uuid.FromString(i)
	if err != nil {
		return errInvalidID(err)
	}
	if err := s.ir.Delete(id, perm); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "identity_delete_fail", "Failed to delete Identity")
	}
	return nil
}
