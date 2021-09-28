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
		return nil, internal.NewServiceClientError(nil, "Credential_WeakPassword", "Password provided is too weak", map[string]interface{}{
			"IdentityID":  uid,
			"Identifiers": identifiers,
		})
	}
	// 3. Hash password
	newPass, err := generateFromPassword(password)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Credential_FailedPassword", "Failed to generate a hashed password", map[string]interface{}{
			"IdentityID":  uid,
			"Identifiers": identifiers,
		})
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return nil, internal.NewServiceInternalError(err, file, line, "Credential_FailedPassword", "Failed to JSON encode hashed password", map[string]interface{}{
			"IdentityID":  uid,
			"Identifiers": identifiers,
		})
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
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Credential_FailedPassword", "Failed to decode password credentials", map[string]interface{}{
			"IdentityID": uid,
		})
	}
	match, err := comparePasswordAndHash(password, hashed.HashedPassword)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		return internal.NewServiceInternalError(err, file, line, "Credential_FailedPassword", "Failed to compare password and hash", map[string]interface{}{
			"IdentityID": uid,
		})
	}
	if !match {
		return errFailedFind(err, uid)
	}
	return nil
}
