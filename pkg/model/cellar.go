package model

import (
	"time"

	"gorm.io/gorm"
)

type Cellar struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex:idx_name_owner"`
	Description string
	OwnerID     uint `gorm:"uniqueIndex:idx_name_owner"`
	Locations   []LocationInCellar

	Owner User `gorm:"foreignKey:OwnerID"`
}

type LocationInCellar struct {
	gorm.Model
	Name     string
	CellarID uint
}

type CellarEntry struct {
	gorm.Model
	CellarID    uint
	BeerID      uint
	Vintage     *uint64
	Quantity    int64
	LocationID  *uint
	FormatID    *uint
	HadBefore   bool
	DateAdded   *time.Time
	DrinkBefore *time.Time
	CellarUntil *time.Time
	Special     bool
	Tags        []Tag `gorm:"many2many:cellar_entry_tags;"`

	Cellar   Cellar            `gorm:"foreignKey:CellarID"`
	Beer     Beer              `gorm:"foreignKey:BeerID"`
	Location *LocationInCellar `gorm:"foreignKey:LocationID"`
	Format   *BeerFormat       `gorm:"foreignKey:FormatID"`
}

type CellarStats struct {
	CellarID      uint
	BeerCount     uint64
	UniqueCount   uint64
	TotalVolume   float64
	BreweryCount  uint64
	UntriedCount  uint64
	SpecialCount  uint64
	AverageABV    float64
	AverageRating float64
}

type CellarRecommendationRanges struct {
	MinimumAbv, MaximumAbv         float64
	MinimumSize, MaximumSize       int64
	MinimumVintage, MaximumVintage uint64
	MinimumRating, MaximumRating   float64
	OldestAddedDate                time.Time
}

type AdventCalendar struct {
	gorm.Model
	CellarID    uint   `gorm:"uniqueIndex:idx_advent_cellar_name"`
	Name        string `gorm:"uniqueIndex:idx_advent_cellar_name"`
	Description string
	StartDate   time.Time
	EndDate     time.Time
	Beers       []AdventCalendarBeer
}

type AdventCalendarBeer struct {
	gorm.Model
	AdventCalendarID uint
	CellarEntryID    uint
	Day              time.Time
	Revealed         bool

	CellarEntry CellarEntry `gorm:"foreignKey:CellarEntryID"`
}
