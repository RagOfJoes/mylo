package service

import (
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/identity"
	goaway "github.com/TwinProduction/go-away"
	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"
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
	// Check for profanity in username
	if goaway.IsProfane(username) {
		return nil, internal.NewServiceClientError(nil, "Identity_FailedCreate", "Username must not contain any profanity", map[string]interface{}{
			"Identity": i,
			"Username": username,
		})
	}
	// Check if email and username already exist
	var eg errgroup.Group
	var f *identity.Identity
	eg.Go(func() error {
		fi, err := s.ir.GetIdentifier(username, false)
		if err != nil {
			return err
		}
		f = fi
		return nil
	})
	eg.Go(func() error {
		fi, err := s.ir.GetIdentifier(i.Email, false)
		if err != nil {
			return err
		}
		f = fi
		return err
	})
	if eg.Wait(); f != nil {
		return nil, internal.NewServiceClientError(nil, "Identity_FailedCreate", "Invalid identifier(s)/password provided", map[string]interface{}{
			"Identity": i,
			"Username": username,
		})
	}
	// Instantiate new identity
	builtUser := identity.Identity{
		FirstName: i.FirstName,
		LastName:  i.LastName,
		Email:     i.Email,
	}
	newUser, err := s.ir.Create(builtUser)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "Identity_FailedCreate", "Invalid identifier(s)/password provided", map[string]interface{}{
			"Identity": i,
			"Username": username,
		})
	}
	// 3. Return new user
	return newUser, nil
}

func (s *service) Find(i string) (*identity.Identity, error) {
	uid, err := uuid.FromString(i)
	if err == nil && uid != uuid.Nil {
		f, err := s.ir.Get(uid, false)
		if err != nil {
			return nil, internal.NewServiceClientError(err, "Identity_FailedFind", "Invalid identifier(s)/password provided", map[string]interface{}{
				"IdentityID": i,
			})
		}
		return f, nil
	}
	f, err := s.ir.GetIdentifier(i, false)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "Identity_FailedFind", "Invalid identifier(s)/password provided", map[string]interface{}{
			"Identifier": i,
		})
	}
	return f, nil
}

// Delete defines a delete function for User identity
func (s *service) Delete(i string, perm bool) error {
	id, err := uuid.FromString(i)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Identity_FailedFind", "Invalid IdentityID provided", map[string]interface{}{
			"IdentityID": i,
		})
	}
	if err := s.ir.Delete(id, perm); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Identity_FailedDelete", "Failed to delete identity", map[string]interface{}{
			"IdentityID": i,
		})
	}
	return nil
}
