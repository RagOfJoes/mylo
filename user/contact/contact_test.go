package contact

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContact(t *testing.T) {
	id, err := uuid.NewV4()
	require.NoError(t, err)

	value := "mylo@test.com"
	var nilTime *time.Time = nil
	newContact := New(id, value)

	assert.Equal(t, false, newContact.Verified)
	assert.Equal(t, nilTime, newContact.VerifiedAt)
	assert.Equal(t, Default, newContact.Type)
	assert.Equal(t, Sent, newContact.State)
	assert.Equal(t, value, newContact.Value)
	assert.Equal(t, id, newContact.IdentityID)
}
