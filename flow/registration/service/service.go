package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/flow/registration"
	"github.com/RagOfJoes/idp/nanoid"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/RagOfJoes/idp/validate"
)

var (
	errInvalidFlowID = idp.NewServiceClientError(nil, "registration_flowid_invalid", "Invalid or expired flow id provided", nil)
	errNanoIDGen     = func() error {
		_, file, line, _ := runtime.Caller(1)
		return idp.NewServiceInternalError(file, line, "registration_nanoid_gen", "Failed to generate new nanoid token. Please try again later")
	}
)

type service struct {
	r  registration.Repository
	is identity.Service
}

func NewRegistrationService(r registration.Repository, is identity.Service) registration.Service {
	return &service{
		r:  r,
		is: is,
	}
}

func (s *service) New(requestURL string) (*registration.Registration, error) {
	tok, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	action := fmt.Sprintf("/registration/%s", fid)
	expire := time.Now().Add(time.Minute * 10)
	form := generateForm(action)
	n, err := s.r.Create(registration.Registration{
		FlowID:     fid,
		CSRFToken:  tok,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		return nil, idp.NewServiceClientError(err, "registration_init", "Failed to create new Registration", nil)
	}
	return n, nil
}

func (s *service) Find(flowID string) (*registration.Registration, error) {
	if flowID == "" {
		return nil, errInvalidFlowID
	}
	f, err := s.r.GetByFlowID(flowID)
	if err != nil {
		return nil, errInvalidFlowID
	}
	if f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlowID
	}
	return f, nil
}

func (s *service) Submit(flowID string, payload registration.RegistrationPayload) error {
	flow, err := s.Find(flowID)
	if err != nil {
		return err
	}

	if err := validate.Check(payload); err != nil {
		return err
	}
	newUser := identity.Identity{
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
	}
	if _, err := s.is.Create(newUser, payload.Username, payload.Password); err != nil {
		return err
	}
	if err := s.r.Delete(flow.ID); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return idp.NewServiceInternalError(file, line, "registration_delete_fail", "Failed to delete registration flow")
	}
	return nil
}
