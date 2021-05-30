package gorm

import (
	"github.com/RagOfJoes/idp/user/address"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type gormAddressRepository struct {
	DB *gorm.DB
}

func NewGormAddressRepository(d *gorm.DB) address.Repository {
	return &gormAddressRepository{DB: d}
}

func (g *gormAddressRepository) Create(v address.VerifiableAddress) (*address.VerifiableAddress, error) {
	n := v
	if err := g.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (g *gormAddressRepository) Update(v address.VerifiableAddress) (*address.VerifiableAddress, error) {
	u := v
	if err := g.DB.Save(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (g *gormAddressRepository) Get(i uuid.UUID) (*address.VerifiableAddress, error) {
	var v address.VerifiableAddress
	if err := g.DB.First(&v, "id = ?", i.String()).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormAddressRepository) GetByUser(i uuid.UUID) ([]*address.VerifiableAddress, error) {
	var v []*address.VerifiableAddress
	if err := g.DB.Find(&v, "identity_id = ?", i.String()).Error; err != nil {
		return nil, err
	}
	return v, nil
}

func (g *gormAddressRepository) GetByAddress(s string) (*address.VerifiableAddress, error) {
	var v address.VerifiableAddress
	if err := g.DB.First(&v, "address = ? ", s).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (g *gormAddressRepository) GetWithConditions(conds ...interface{}) ([]*address.VerifiableAddress, error) {
	var v []*address.VerifiableAddress
	if err := g.DB.Find(&v, conds).Error; err != nil {
		return nil, err
	}
	return v, nil
}

func (g *gormAddressRepository) Delete(i uuid.UUID) error {
	if err := g.DB.Where("id = ?", i.String()).Delete(address.VerifiableAddress{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (g *gormAddressRepository) DeleteAllUser(i uuid.UUID) error {
	if err := g.DB.Where("identity_id = ?", i.String()).Delete(address.VerifiableAddress{}).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}
