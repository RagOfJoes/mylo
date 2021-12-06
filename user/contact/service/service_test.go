package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/RagOfJoes/mylo/internal"
	mocks "github.com/RagOfJoes/mylo/mocks/user/contact"
	"github.com/RagOfJoes/mylo/user/contact"
	"github.com/gofrs/uuid"
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
				actual:      []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(id, "bar@mylo.com")},
				expect:      []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(id, "bar@mylo.com")},
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
		idTwo, err := uuid.NewV4()
		require.NoError(t, err)

		for _, test := range []struct {
			input          []contact.Contact
			expectedResult []contact.Contact
			expectedError  error
		}{
			{
				input:          []contact.Contact{},
				expectedResult: nil,
				expectedError:  contact.ErrContactInvalidLength,
			},
			{
				input:          []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(idTwo, "bar@mylo.com")},
				expectedResult: nil,
				expectedError:  contact.ErrContactNotMatchingIdentityID,
			},
			{
				input:          []contact.Contact{contact.New(id, "foo@mylo.com"), contact.New(id, "foo@mylo.com")},
				expectedResult: nil,
				expectedError:  contact.ErrContactValuesNotUnique,
			},
		} {
			mockRepo.On("DeleteAllUser", ctx, id).Return(nil)
			mockRepo.On("Create", ctx, test.input).Return(test.expectedResult, test.expectedError)

			created, err := testService.Add(ctx, test.input...)
			assert.Equal(t, test.expectedResult, created)
			assert.EqualError(t, err, test.expectedError.Error())
		}
	})
}

func TestContactServiceFind(t *testing.T) {
	t.Run("Find by ID", func(t *testing.T) {
		ctx := context.Background()
		mockRepo := mocks.Repository{}
		testService := NewContactService(&mockRepo)

		now := time.Now()
		notFoundID, err := uuid.NewV4()
		require.NoError(t, err)
		id, err := uuid.NewV4()
		require.NoError(t, err)

		for _, test := range []struct {
			input          uuid.UUID
			expectedResult *contact.Contact
			expectedError  error
		}{
			{
				input:         id,
				expectedError: nil,
				expectedResult: &contact.Contact{
					Base: internal.Base{
						ID:        id,
						CreatedAt: now,
						UpdatedAt: nil,
					},
					Verified:   false,
					VerifiedAt: nil,
					Type:       contact.Default,
					State:      contact.Sent,
					Value:      "foo@mylo.com",
					IdentityID: uuid.UUID{},
				},
			},
			{
				input:          notFoundID,
				expectedResult: nil,
				expectedError:  contact.ErrContactDoesNotExist,
			},
		} {
			mockRepo.On("Get", ctx, test.input).Return(test.expectedResult, test.expectedError)

			found, err := testService.Find(ctx, test.input.String())
			if test.expectedError != nil {
				assert.ErrorIs(t, err, contact.ErrContactDoesNotExist)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedResult, found)
		}
	})

	t.Run("Find by Value", func(t *testing.T) {
		ctx := context.Background()
		mockRepo := mocks.Repository{}
		testService := NewContactService(&mockRepo)

		now := time.Now()

		for _, test := range []struct {
			input          string
			expectedResult *contact.Contact
			expectedError  error
		}{
			{
				input:         "foo@mylo.com",
				expectedError: nil,
				expectedResult: &contact.Contact{
					Base: internal.Base{
						ID:        uuid.UUID{},
						CreatedAt: now,
						UpdatedAt: nil,
					},
					Verified:   false,
					VerifiedAt: nil,
					Type:       contact.Default,
					State:      contact.Sent,
					Value:      "foo@mylo.com",
					IdentityID: uuid.UUID{},
				},
			},
			{
				input:          "bar@mylo.com",
				expectedResult: nil,
				expectedError:  contact.ErrContactDoesNotExist,
			},
		} {
			mockRepo.On("GetByValue", ctx, test.input).Return(test.expectedResult, test.expectedError)

			found, err := testService.Find(ctx, test.input)
			if test.expectedError != nil {
				assert.Error(t, err, contact.ErrContactDoesNotExist)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedResult, found)
		}
	})
}
