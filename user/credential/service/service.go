package service

import (
	"encoding/json"
	"runtime"

	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/internal/config"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gofrs/uuid"
	"github.com/nbutton23/zxcvbn-go"
)

var (
	errFailedFind = func(src error, i uuid.UUID) error {
		return internal.NewServiceClientError(src, "Credential_FailedFind", "Invalid identifier/password provided", map[string]interface{}{
			"IdentityID": i,
		})
	}
	errWeakPassword = func(src error, i uuid.UUID, identifiers []credential.Identifier) error {
		return internal.NewServiceClientError(nil, "Credential_WeakPassword", "Password provided is too weak", map[string]interface{}{
			"IdentityID":  i,
			"Identifiers": identifiers,
		})
	}
	errFailedGeneratePassword = func(src error, i uuid.UUID, identifiers []credential.Identifier) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Credential_FailedPassword", "Failed to generate a hashed password", map[string]interface{}{
			"IdentityID":  i,
			"Identifiers": identifiers,
		})
	}
	errFailedCompare = func(src error, i uuid.UUID) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Credential_FailedPassword", "Failed to compare password and hash", map[string]interface{}{
			"IdentityID": i,
		})
	}
	errFailedDecodePassword = func(src error, i uuid.UUID) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Credential_FailedPassword", "Failed to decode password credentials", map[string]interface{}{
			"IdentityID": i,
		})
	}
	errFailedEncodePassword = func(src error, i uuid.UUID, ids []credential.Identifier) error {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(src, file, line, "Credential_FailedPassword", "Failed to JSON encode hashed password", map[string]interface{}{
			"IdentityID":  i,
			"Identifiers": ids,
		})

	}
)

type service struct {
	cr credential.Repository
}

func NewCredentialService(cr credential.Repository) credential.Service {
	return &service{
		cr: cr,
	}
}

func (s *service) CreatePassword(uid uuid.UUID, password string, identifiers []credential.Identifier) (*credential.Credential, error) {
	cfg := config.Get()
	// Get inputs to test password strength
	var ids []string
	for _, i := range identifiers {
		ids = append(ids, i.Value)
	}
	// Test password strength
	passStrength := zxcvbn.PasswordStrength(password, ids)
	if passStrength.Score <= cfg.Credential.MinimumScore {
		return nil, errWeakPassword(nil, uid, identifiers)
	}
	// Hash password
	newPass, err := generateFromPassword(password)
	if err != nil {
		return nil, errFailedGeneratePassword(err, uid, identifiers)
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		return nil, errFailedEncodePassword(err, uid, identifiers)
	}
	// Build Credential
	newCredential := credential.Credential{
		Type:        credential.Password,
		IdentityID:  uid,
		Identifiers: identifiers,
		Values:      string(jsonPass[:]),
	}
	// Create Credential in repository
	ncp, err := s.cr.Create(newCredential)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "Credential_FailedCreate", "Invalid identifier(s)/password provided", map[string]interface{}{
			"IdentityID":  uid,
			"Identifiers": identifiers,
		})
	}
	return ncp, nil
}

func (s *service) ComparePassword(uid uuid.UUID, password string) error {
	cred, err := s.cr.GetWithIdentityID(credential.Password, uid)
	if err != nil {
		return errFailedFind(err, uid)
	}
	var hashed credential.CredentialPassword
	if err := json.Unmarshal([]byte(cred.Values), &hashed); err != nil {
		return errFailedDecodePassword(err, uid)
	}
	match, err := comparePasswordAndHash(password, hashed.HashedPassword)
	if err != nil {
		return errFailedCompare(err, uid)
	}
	if !match {
		return errFailedFind(err, uid)
	}
	return nil
}

func (s *service) FindPasswordWithIdentifier(identifier string) (*credential.Credential, error) {
	credential, err := s.cr.GetWithIdentifier(credential.Password, identifier)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "Credential_FailedFind", "Invalid identifier provided", map[string]interface{}{
			"Identifier": identifier,
		})
	}
	return credential, nil
}

func (s *service) UpdatePassword(uid uuid.UUID, newPassword string) (*credential.Credential, error) {
	// Find existing credential
	cred, err := s.cr.GetWithIdentityID(credential.Password, uid)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "Credential_FailedFind", "The account doesn't exist or the account doesn't have a password credential setup", map[string]interface{}{
			"IdentityID": uid,
		})
	}
	cfg := config.Get()
	// Get identifiers to test password strength
	ids := []string{}
	for _, id := range cred.Identifiers {
		ids = append(ids, id.Value)
	}
	// Test password strength
	passStrength := zxcvbn.PasswordStrength(newPassword, ids)
	if passStrength.Score <= cfg.Credential.MinimumScore {
		return nil, errWeakPassword(nil, uid, cred.Identifiers)
	}
	// Compare new and old password
	var hashed credential.CredentialPassword
	if err := json.Unmarshal([]byte(cred.Values), &hashed); err != nil {
		return nil, errFailedDecodePassword(err, uid)
	}
	match, err := comparePasswordAndHash(newPassword, hashed.HashedPassword)
	if err != nil {
		return nil, errFailedCompare(err, uid)
	}
	if match {
		return nil, internal.NewServiceClientError(nil, "Credential_FailedUpdate", "Invalid password provided", map[string]interface{}{
			"IdentityID":  uid,
			"NewPassword": newPassword,
		})
	}
	// Create new password
	newPass, err := generateFromPassword(newPassword)
	if err != nil {
		return nil, errFailedGeneratePassword(err, uid, cred.Identifiers)
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		return nil, errFailedEncodePassword(err, uid, cred.Identifiers)
	}
	// Build Credential
	uc := *cred
	uc.Values = string(jsonPass[:])
	// Update password
	updated, err := s.cr.Update(uc)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Credential_FailedUpdate", "Failed to update credential", map[string]interface{}{
			"IdentityID":        uid,
			"UpdatedCredential": uc,
		})
	}
	return updated, nil
}
