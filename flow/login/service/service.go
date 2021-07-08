package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp"
	"github.com/RagOfJoes/idp/flow/login"
	"github.com/RagOfJoes/idp/nanoid"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/RagOfJoes/idp/user/identity"
	"github.com/RagOfJoes/idp/validate"
)

var (
	errInvalidFlowID = idp.NewServiceClientError(nil, "login_flowid_invalid", "Invalid or expired flow id provided", nil)
	errNanoIDGen     = func() error {
		_, file, line, _ := runtime.Caller(1)
		return idp.NewServiceInternalError(file, line, "registration_nanoid_gen", "Failed to generate new nanoid token")
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

func (s *service) New(requestURL string) (*login.Login, error) {
	tok, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	fid, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen()
	}
	action := fmt.Sprintf("/login/%s", fid)
	expire := time.Now().Add(time.Minute * 10)
	form := generateForm(action, tok)
	n, err := s.r.Create(login.Login{
		FlowID:     fid,
		CSRFToken:  tok,
		Form:       form,
		ExpiresAt:  expire,
		RequestURL: requestURL,
	})
	if err != nil {
		return nil, idp.NewServiceClientError(err, "login_init", "Failed to create new Login", nil)
	}
	return n, nil
}

func (s *service) Find(flowID string) (*login.Login, error) {
	if flowID == "" {
		return nil, errInvalidFlowID
	}
	f, err := s.r.GetByFlowID(flowID)
	if err != nil || f == nil || f.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlowID
	}
	return f, nil
}

func (s *service) Submit(flowID string, payload login.LoginPayload) (*identity.Identity, error) {
	_, err := s.Find(flowID)
	if err != nil {
		return nil, err
	}
	if err := validate.Check(payload); err != nil {
		return nil, err
	}
	id, err := s.is.Find(payload.Identifier)
	if err != nil {
		return nil, err
	}
	if err := s.cs.ComparePassword(id.ID, payload.Password); err != nil {
		return nil, err
	}
	return id, nil
}
