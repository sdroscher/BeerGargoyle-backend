package model

import (
	"gorm.io/gorm"
)

type Brewery struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex:idx_brewery_unique"`
	Description    string
	AddressID      int
	Address        Address
	ImageURL       string
	ExternalID     *uint64 `gorm:"uniqueIndex:idx_brewery_unique"`
	ExternalSource *string
	ExternalRating *float64
}

type Address struct {
	gorm.Model
	Country       string
	Locality      string
	Region        *string
	PostalCode    *string
	StreetAddress *string
}
