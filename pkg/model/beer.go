package model

import "gorm.io/gorm"

type BeerStyle struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex"`
	BJCPStyleID string `gorm:"index:idx_beer_style_bjcp"`

	BJCPStyle BeerStyleBJCP `gorm:"foreignKey:BJCPStyleID;references:BJCPID"`
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

type BeerStyleBJCP struct {
	BJCPID     string `gorm:"primaryKey"`
	Name       string `gorm:"uniqueIndex"`
	CategoryID uint   `gorm:"index"`
	FamilyID   uint   `gorm:"index"`

	Category BeerCategoryBJCP `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Family   BeerStyleFamily  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type BeerCategoryBJCP struct {
	gorm.Model

	Name string `gorm:"uniqueIndex"`
}
type BeerStyleFamily struct {
	gorm.Model

	Name string `gorm:"uniqueIndex"`
}
