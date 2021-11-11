package service

import (
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/identity"
	goaway "github.com/TwiN/go-away"
	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"
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
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", identity.ErrUsernameProfane)
	}
	// Check if email and username already exist
	var eg errgroup.Group
	var f *identity.Identity
	eg.Go(func() error {
		fi, err := s.ir.GetWithIdentifier(username, false)
		if err != nil {
			return internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", identity.ErrInvalidIdentifierPassword)
		}
		f = fi
		return nil
	})
	eg.Go(func() error {
		fi, err := s.ir.GetWithIdentifier(i.Email, false)
		if err != nil {
			return internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", identity.ErrInvalidIdentifierPassword)
		}
		f = fi
		return err
	})
	if eg.Wait(); f != nil {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", identity.ErrInvalidIdentifierPassword)
	}
	// Instantiate new identity
	builtUser := identity.Identity{
		FirstName: i.FirstName,
		LastName:  i.LastName,
		Email:     i.Email,
	}
	newUser, err := s.ir.Create(builtUser)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create new identity")
	}
	// 3. Return new user
	return newUser, nil
}

func (s *service) Find(id string) (*identity.Identity, error) {
	uid, err := uuid.FromString(id)
	if err == nil && uid != uuid.Nil {
		f, err := s.ir.Get(uid, false)
		if err != nil {
			return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", identity.ErrInvalidIdentifierPassword)
		}
		return f, nil
	}
	f, err := s.ir.GetWithIdentifier(id, false)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", identity.ErrInvalidIdentifierPassword)
	}
	return f, nil
}

// Delete defines a delete function for User identity
func (s *service) Delete(id string, perm bool) error {
	uid, err := uuid.FromString(id)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Account with id %s does not exist", id)
	}
	if err := s.ir.Delete(uid, perm); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to delete identity: %s", id)
	}
	return nil
}
