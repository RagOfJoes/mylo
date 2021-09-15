package identity

import (
	"github.com/RagOfJoes/idp/internal"
	"github.com/RagOfJoes/idp/user/contact"
	"github.com/RagOfJoes/idp/user/credential"
	"github.com/gofrs/uuid"
)

// Identity defines the base Identity model
type Identity struct {
	internal.BaseSoftDelete
	Avatar    string `json:"avatar" gorm:"size:1024;" validate:"url,min=1,max=1024"`
	FirstName string `json:"first_name" gorm:"size:64" validate:"max=64,alphanumunicode"`
	LastName  string `json:"last_name" gorm:"size:64" validate:"max=64,alphanumunicode"`
	// Email is the primary email that will be used for account
	// security related notifications
	Email string `json:"email" gorm:"uniqueIndex;not null;" validate:"email,required"`

	Credentials        []credential.Credential     `json:"-"`
	VerifiableContacts []contact.VerifiableContact `json:"-"`
}

// Repository defines an interface that allows
// Identity domain data to be persisted through different
// dbs
type Repository interface {
	Create(Identity) (*Identity, error)
	Get(id uuid.UUID, critical bool) (*Identity, error)
	GetIdentifier(identifier string, critical bool) (*Identity, error)
	Update(Identity) (*Identity, error)
	Delete(id uuid.UUID, permanent bool) error
}

type Service interface {
	Find(string) (*Identity, error)
	Create(user Identity, username string, password string) (*Identity, error)
	Delete(id string, permanent bool) error
}
