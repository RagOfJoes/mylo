package service

import (
	"context"
	"reflect"
	"testing"

	mocks "github.com/RagOfJoes/mylo/mocks/user/contact"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactServiceAdd(t *testing.T) {
	t.Run("Valid Contacts", func(t *testing.T) {
		ctx := context.Background()
		mockRepo := &mocks.Repository{}
		testService := NewContactService(mockRepo)

		id, err := uuid.NewV4()
		require.NoError(t, err)
		idTwo, err := uuid.NewV4()
		require.NoError(t, err)

		for _, test := range []struct {
			expectedErr error
			actual      []contact.Contact
			expect      []contact.Contact
		}{
			{
				expectedErr: nil,
				actual:      []contact.Contact{contact.New(id, "foo@mylo.com")},
				expect:      []contact.Contact{contact.New(id, "foo@mylo.com")},
			},
			{
				expectedErr: nil,
				actual:      []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(idTwo, "bar@mylo.com")},
				expect:      []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(idTwo, "bar@mylo.com")},
			},
		} {
			mockRepo.On("DeleteAllUser", ctx, id).Return(nil)
			mockRepo.On("Create", ctx, test.actual).Return(test.expect, nil)

			created, err := testService.Add(ctx, test.actual...)
			require.NoError(t, err)
			require.Len(t, created, len(test.expect))

			for _, actual := range created {
				var found bool
				for _, expect := range test.expect {
					if reflect.DeepEqual(actual, expect) {
						found = true
						break
					}
				}
				assert.True(t, found, "%+v not in %+v", actual, test.expect)
			}
		}
	})

	t.Run("Invalid Contacts", func(t *testing.T) {
		ctx := context.Background()
		mockRepo := &mocks.Repository{}
		testService := NewContactService(mockRepo)

		id, err := uuid.NewV4()
		require.NoError(t, err)
		// idTwo, err := uuid.NewV4()
		// require.NoError(t, err)

		for _, test := range []struct {
			expectedErr error
			actual      []contact.Contact
			expect      []contact.Contact
		}{
			{
				expectedErr: contact.ErrContactInvalidLength,
				actual:      []contact.Contact{},
				expect:      nil,
			},
			{
				expectedErr: errors.Errorf("Failed to create contacts for %s", id),
				actual:      []contact.Contact{contact.New(id, "")},
				expect:      nil,
			},
		} {
			mockRepo.On("DeleteAllUser", ctx, id).Return(nil)
			mockRepo.On("Create", ctx, test.actual).Return(test.expect, nil)

			created, err := testService.Add(ctx, test.actual...)
			assert.Equal(t, test.expect, created, "created not expected values")
			assert.EqualError(t, err, test.expectedErr.Error())
		}
	})
}
