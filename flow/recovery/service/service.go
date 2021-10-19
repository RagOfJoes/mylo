package service

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/RagOfJoes/idp/flow/recovery"
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/internal/validate"
	"github.com/RagOfJoes/idp/pkg/nanoid"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
)

var (
	errInvalidFlowID = func(src error, f string) error {
		return internal.NewServiceClientError(src, "Recovery_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"FlowID": f,
		})
	}
	errInvalidFlow = func(src error, f recovery.Flow) error {
		return internal.NewServiceClientError(src, "Recovery_InvalidFlow", "Invalid or expired flow", map[string]interface{}{
			"Flow": f,
		})
	}
	errInvalidIdentifierPayload = func(src error, f recovery.Flow, p recovery.IdentifierPayload) error {
		return internal.NewServiceClientError(src, "Recovery_InvalidIdentifierPayload", "Invalid payload provided", map[string]interface{}{
			"Flow":    f,
			"Payload": p,
		})
	}
	errInvalidSubmitPayload = func(src error, f recovery.Flow, p recovery.SubmitPayload) error {
		desc := "Invalid payload provided"
		if src != nil {
			desc = src.Error()
		}
		return internal.NewServiceClientError(src, "Recovery_InvalidSubmitPayload", desc, map[string]interface{}{
			"Flow":    f,
			"Payload": p,
		})
	}
	// Internal Errors
	errNanoIDGen = func(src error) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Recovery_FailedNanoID", "Failed to generate nano id", nil)
	}
	errFailedUpdate = func(src error, o recovery.Flow, n recovery.Flow) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Recovery_FailedUpdate", "Failed to update flow", map[string]interface{}{
			"OldFlow": o,
			"NewFlow": n,
		})
	}
)

type service struct {
	r   recovery.Repository
	cs  credential.Service
	cos contact.Service
}

func NewRecoveryService(r recovery.Repository, cs credential.Service, cos contact.Service) recovery.Service {
	return &service{
		r:   r,
		cs:  cs,
		cos: cos,
	}
}

func (s *service) New(requestURL string) (*recovery.Flow, error) {
	flowID, err := nanoid.New()
	if err != nil {
		return nil, errNanoIDGen(err)
	}
	cfg := config.Get()
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Recovery.URL, flowID)
	expire := time.Now().Add(cfg.Recovery.Lifetime)
	form := generateInitialForm(action)
	flow := recovery.Flow{
		FlowID:     flowID,
		ExpiresAt:  expire,
		RequestURL: requestURL,
		Status:     recovery.IdentifierPending,

		Form: &form,
	}
	newFlow, err := s.r.Create(flow)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Recovery_FailedCreate", "Failed to create new recovery flow", map[string]interface{}{
			"Flow": flow,
		})
	}
	return newFlow, nil
}

func (s *service) Find(flowID string) (*recovery.Flow, error) {
	if flowID == "" {
		return nil, errInvalidFlowID(nil, flowID)
	}
	// Try to get Flow
	flow, err := s.r.GetByFlowID(flowID)
	// Check for error or empty flow
	if err != nil || flow == nil {
		return nil, errInvalidFlowID(err, flowID)
	}
	// Check for expired flow or if flow belongs to user
	if flow.Status == recovery.Fail || flow.ExpiresAt.Before(time.Now()) {
		return nil, errInvalidFlow(nil, *flow)
	}
	return flow, nil
}

func (s *service) SubmitIdentifier(flow recovery.Flow, payload recovery.IdentifierPayload) (*recovery.Flow, error) {
	// Validate flow
	if err := validate.Check(flow); err != nil {
		return nil, errInvalidFlow(err, flow)
	}
	// Validate payload
	if err := validate.Check(payload); err != nil {
		return nil, errInvalidIdentifierPayload(err, flow, payload)
	}
	// Make sure flow has the proper state
	if flow.Status != recovery.IdentifierPending {
		return nil, errInvalidFlow(nil, flow)
	}
	credential, err := s.cs.FindPasswordWithIdentifier(payload.Identifier)
	if err != nil {
		updateFlow := flow
		updateFlow.Status = recovery.Fail
		updateFlow.Form = nil
		if _, err := s.r.Update(updateFlow); err != nil {
			// TODO: Capture Error Here
			log.Print(err)
		}
		return nil, internal.NewServiceClientError(err, "Recovery_InvalidIdentifier", "Account with identifier does not exist", map[string]interface{}{
			"Flow":    flow,
			"Payload": payload,
		})
	}
	cfg := config.Get()
	// Update flow
	updateFlow := flow
	updateFlow.Status = recovery.LinkPending
	updateFlow.IdentityID = &credential.IdentityID
	// Generate new form
	action := fmt.Sprintf("%s/%s/%s", cfg.Server.URL, cfg.Recovery.URL, flow.FlowID)
	form := generateRecoveryForm(action)
	updateFlow.Form = &form
	// Update FlowID to ensure that the intended user received the recovery request from their preferred out-of-band communication service. IE: email or phone
	newFlowID, err := nanoid.New()
	if err != nil {
		return nil, errFailedUpdate(err, flow, updateFlow)
	}
	updateFlow.FlowID = newFlowID
	updatedFlow, err := s.r.Update(updateFlow)
	if err != nil {
		return nil, errFailedUpdate(err, flow, updateFlow)
	}
	return updatedFlow, nil
}

func (s *service) SubmitUpdatePassword(flow recovery.Flow, payload recovery.SubmitPayload) (*recovery.Flow, error) {
	// Validate flow
	if err := validate.Check(flow); err != nil {
		return nil, errInvalidFlow(err, flow)
	}
	// Validate payload
	if err := validate.Check(payload); err != nil {
		return nil, errInvalidSubmitPayload(err, flow, payload)
	}
	// Make sure flow has the proper state
	if flow.IdentityID == nil || flow.Status != recovery.LinkPending {
		return nil, errInvalidFlow(nil, flow)
	}
	// Update password
	_, err := s.cs.UpdatePassword(*flow.IdentityID, payload.Password)
	if err != nil {
		_, ok := err.(internal.ClientError)
		if ok {
			return nil, internal.NewServiceClientError(err, "Recovery_FailedSubmit", err.Error(), map[string]interface{}{
				"Flow":    flow,
				"Payload": payload,
			})
		}
		return nil, err
	}
	// Update flow
	updateFlow := flow
	updateFlow.Form = nil
	updateFlow.Status = recovery.Success
	updatedFlow, err := s.r.Update(updateFlow)
	if err != nil {
		return nil, errFailedUpdate(err, flow, updateFlow)
	}
	return updatedFlow, nil
}
