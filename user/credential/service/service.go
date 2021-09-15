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
		return nil, internal.NewServiceClientError(nil, "credential_password_weak", "Password provided is too weak", nil)
	}
	// 3. Hash password
	newPass, err := generateFromPassword(password)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "credential_password_fail", "Failed to generate a hashed password")
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(file, line, "credential_password_fail", "Failed to JSON encode hashed password")
	}
	// 4. Build Credential
	newCredential := credential.Credential{
		Type:        credential.Password,
		IdentityID:  uid,
		Identifiers: identifiers,
		Values:      string(jsonPass[:]),
	}
	// 5. Create Credential in repository
	ncp, err := s.cr.Create(newCredential)
	if err != nil {
		return nil, internal.NewServiceClientError(err, "credential_password_create", "Invalid email/username provided", nil)
	}
	return ncp, nil
}

func (s *service) ComparePassword(uid uuid.UUID, password string) error {
	cred, err := s.cr.GetWithIdentityID(credential.Password, uid)
	if err != nil {
		return internal.NewServiceClientError(err, "invalid_identity", "Invalid email/username provided", nil)
	}
	var hashed credential.CredentialPassword
	if err := json.Unmarshal([]byte(cred.Values), &hashed); err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "credential_password_fail", "Failed to decode password credential")
	}
	match, err := comparePasswordAndHash(password, hashed.HashedPassword)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(file, line, "credential_password_fail", err.Error())
	}
	if !match {
		return internal.NewServiceClientError(err, "invalid_password", "Wrong password. Click on Forgot Password to reset it.", nil)
	}
	return nil
}
