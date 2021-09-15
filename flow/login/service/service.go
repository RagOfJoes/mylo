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
	errInvalidFlowID = internal.NewServiceClientError(nil, "login_flowid_invalid", "Invalid or expired flow id provided", nil)
	errNanoIDGen     = func() error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "login_nanoid_gen", "Failed to generate new nanoid token")
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
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}

	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Login.URL, fid)
	expire := time.Now().Add(cfg.Login.Lifetime)
	form := generateForm(action)
	n, err := s.r.Create(login.Flow{
		FlowID:     fid,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		return nil, internal.NewServiceClientError(err, "login_init", "Failed to create new Login", nil)
	}
	return n, nil
}

func (s *service) Find(flowID string) (*login.Flow, error) {
	if flowID == "" {
		return nil, errInvalidFlowID
	}
	f, err := s.r.GetByFlowID(flowID)
	if err != nil || f == nil || f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlowID
	}
	return f, nil
}

func (s *service) Submit(flowID string, payload login.Payload) (*identity.Identity, error) {
	// 1. Ensure that the flow is still valid
	flow, err := s.Find(flowID)
	if err != nil {
		return nil, err
	}
	// 2. Validate payload provided
	if err := validate.Check(payload); err != nil {
		return nil, err
	}
	// 3.Retrieve identity based on identifier provided
	id, err := s.is.Find(payload.Identifier)
	if err != nil {
		return nil, err
	}
	// 4. Use retrieved identity ID to then retrieve
	// the hashed password credential then decode it
	// and compare provided password attempt
	if err := s.cs.ComparePassword(id.ID, payload.Password); err != nil {
		return nil, err
	}
	// 5. If everything passes then delete flow
	if err := s.r.Delete(flow.ID); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "login_delete_fail", "Failed to delete registration flow")
	}
	return id, nil
}
