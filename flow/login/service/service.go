package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
)

var (
	errInvalidFlowID = func(src error, f string) error {
		return internal.NewServiceClientError(src, "Login_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"FlowID": f,
		})
	}
	errInvalidFlow = func(src error, f login.Flow) error {
		return internal.NewServiceClientError(src, "Login_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Flow": f,
		})
	}
	errInvalidPayload = func(src error, f login.Flow, p login.Payload) error {
		return internal.NewServiceClientError(src, "Login_InvalidPayload", "Invalid identifier or password provided", map[string]interface{}{
			"Flow":    f,
			"Payload": p,
		})
	}
	// Internal Errors
	errNanoIDGen = func(src error) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Login_FailedNanoID", "Failed to generate nano id", nil)
	}
)

type service struct {
	r   login.Repository
	cos contact.Service
	cs  credential.Service
	is  identity.Service
}

func NewLoginService(r login.Repository, cos contact.Service, cs credential.Service, is identity.Service) login.Service {
	return &service{
		r:   r,
		cs:  cs,
		is:  is,
		cos: cos,
	}
}

func (s *service) New(requestURL string) (*login.Flow, error) {
	newFlow, err := login.New(requestURL)
	if err != nil {
		return nil, err
	}
	created, err := s.r.Create(*newFlow)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Login_FailedCreate", "Failed to create new login flow", map[string]interface{}{
			"Flow": f,
		})
	}
	return created, nil
}

func (s *service) Find(flowID string) (*login.Flow, error) {
	if flowID == "" {
		return nil, errInvalidFlowID(nil, flowID)
	}

	flow, err := s.r.GetByFlowID(flowID)
	if err != nil || flow == nil {
		return nil, internal.NewErrorf(internal.ErrorCodeNotFound, "%v", login.ErrInvalidExpiredFlow)
	}
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	return flow, nil
}

func (s *service) Submit(flow login.Flow, payload login.Payload) (*identity.Identity, error) {
	if err := flow.Valid(); err != nil {
		return nil, err
	}
	if err := validate.Check(payload); err != nil {
		return nil, errInvalidPayload(err, flow, payload)
	}
	// Retrieve identity based on identifier provided
	id, err := s.is.Find(payload.Identifier)
	if err != nil {
		return nil, errInvalidPayload(err, flow, payload)
	}
	// Use retrieved identity ID to then retrieve
	// the hashed password credential then decode it
	// and compare provided password attempt
	if err := s.cs.ComparePassword(id.ID, payload.Password); err != nil {
		return nil, errInvalidPayload(err, flow, payload)
	}
	// If everything passes then delete flow
	// TODO: Capture error, if any, here
	go func() {
		s.r.Delete(flow.ID)
	}()
	return id, nil
}
