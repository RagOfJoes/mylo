package session

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RagOfJoes/idp/user/credential"
	"github.com/pkg/errors"
)

// CredentialMethod defines credential method used to authenticate User
type CredentialMethod struct {
	// Method is just that
	Method credential.CredentialType `json:"method"`
	// IssuedAt defines the time when credential method was used successfully
	IssuedAt time.Time `json:"issued_at"`
}

// Scan implements the Scanner interface.
func (c *CredentialMethod) Scan(value interface{}) error {
	v := fmt.Sprintf("%s", value)
	if len(v) == 0 {
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(v), c))
}

// Value implements the driver Valuer interface.
func (c CredentialMethod) Value() (driver.Value, error) {
	value, err := json.Marshal(c)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(value), nil
}

type CredentialMethods []CredentialMethod

// Scan implements the Scanner interface.
func (c *CredentialMethods) Scan(value interface{}) error {
	v := fmt.Sprintf("%s", value)
	if len(v) == 0 {
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(v), c))
}

// Value implements the driver Valuer interface.
func (c CredentialMethods) Value() (driver.Value, error) {
	value, err := json.Marshal(c)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(value), nil
}
