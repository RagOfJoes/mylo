package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RagOfJoes/mylo/internal"
	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/RagOfJoes/mylo/user/credential"
	"github.com/gofrs/uuid"
	"github.com/nbutton23/zxcvbn-go"
)

type service struct {
	cfg config.Configuration

	cr credential.Repository
}

func NewCredentialService(cfg config.Configuration, cr credential.Repository) credential.Service {
	return &service{
		cfg: cfg,

		cr: cr,
	}
}

func (s *service) CreatePassword(ctx context.Context, uid uuid.UUID, password string, identifiers []credential.Identifier) (*credential.Credential, error) {
	// Get inputs to test password strength
	var ids []string
	for _, i := range identifiers {
		ids = append(ids, i.Value)
	}
	// Test password strength
	passStrength := zxcvbn.PasswordStrength(password, ids)
	if passStrength.Score <= s.cfg.Credential.MinimumScore {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", credential.ErrWeakPassword)
	}
	// Hash password
	newPass, err := generateFromPassword(s.cfg.Credential.Argon, password)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedGeneratePassword)
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedJSONEncodePassword)
	}
	// Build Credential
	newCredential := credential.Credential{
		Type:        credential.Password,
		IdentityID:  uid,
		Identifiers: identifiers,
		Values:      string(jsonPass[:]),
	}
	// Create Credential in repository
	created, err := s.cr.Create(ctx, newCredential)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to create password credential")
	}
	return created, nil
}

func (s *service) ComparePassword(ctx context.Context, uid uuid.UUID, password string) error {
	found, err := s.cr.GetWithIdentityID(ctx, credential.Password, uid)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "%v", credential.ErrInvalidIdentifierPassword)
	}
	var hashed credential.CredentialPassword
	if err := json.Unmarshal([]byte(found.Values), &hashed); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedJSONDecodePassword)
	}
	match, err := comparePasswordAndHash(password, hashed.HashedPassword)
	if err != nil {
		internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedPasswordCompare)
	}
	if !match {
		return internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", credential.ErrInvalidIdentifierPassword)
	}
	return nil
}

func (s *service) FindPasswordWithIdentifier(ctx context.Context, identifier string) (*credential.Credential, error) {
	credential, err := s.cr.GetWithIdentifier(ctx, credential.Password, identifier)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "Invalid identifier provided")
	}
	return credential, nil
}

func (s *service) UpdatePassword(ctx context.Context, uid uuid.UUID, newPassword string) (*credential.Credential, error) {
	// Find existing credential
	cred, err := s.cr.GetWithIdentityID(ctx, credential.Password, uid)
	// TODO: In this scenario should we just create a new password credential behind the scene?
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeNotFound, "The account doesn't exist or the account doesn't have a password credential setup")
	}
	// Get identifiers to test password strength
	ids := []string{}
	for _, id := range cred.Identifiers {
		ids = append(ids, id.Value)
	}
	// Test password strength
	passStrength := zxcvbn.PasswordStrength(newPassword, ids)
	if passStrength.Score <= s.cfg.Credential.MinimumScore {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", credential.ErrWeakPassword)
	}
	// Compare new and old password
	var hashed credential.CredentialPassword
	if err := json.Unmarshal([]byte(cred.Values), &hashed); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedJSONDecodePassword)
	}
	match, err := comparePasswordAndHash(newPassword, hashed.HashedPassword)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedPasswordCompare)
	}
	if match {
		return nil, internal.NewErrorf(internal.ErrorCodeInvalidArgument, "%v", credential.ErrInvalidIdentifierPassword)
	}
	// Create new password
	newPass, err := generateFromPassword(s.cfg.Credential.Argon, newPassword)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedGeneratePassword)
	}
	credPass := credential.CredentialPassword{
		HashedPassword: newPass,
	}
	jsonPass, err := json.Marshal(credPass)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "%v", credential.ErrFailedJSONEncodePassword)
	}
	// Rebuild Credential
	uc := *cred
	updatedAt := time.Now()
	uc.UpdatedAt = &updatedAt
	uc.Values = string(jsonPass[:])
	uc.Identifiers = cred.Identifiers
	// Delete previous password credential
	if err := s.cr.Delete(ctx, cred.ID); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update password credential: %s", cred.ID)
	}
	// Create new
	updated, err := s.cr.Create(ctx, uc)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeInternal, "Failed to update password credential: %s", cred.ID)
	}
	return updated, nil
}
