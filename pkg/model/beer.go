package model

import "gorm.io/gorm"

type BeerStyle struct {
	gorm.Model
	Name string `gorm:"uniqueIndex"`
}

type Beer struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex:idx_beer_unique"`
	Description    string
	ImageURL       string
	BreweryID      uint `gorm:"uniqueIndex:idx_beer_unique"`
	StyleID        uint
	ABV            *float64
	IBU            *uint64
	ExternalID     *uint64
	ExternalSource *string
	ExternalRating *float64
	Tags           []Tag `gorm:"many2many:beer_tags;"`

	Brewery Brewery   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Style   BeerStyle `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type BeerFormat struct {
	gorm.Model
	Package      string
	SizeMetric   float64
	SizeImperial float64
}

type Tag struct {
	gorm.Model
	Tag string
}
