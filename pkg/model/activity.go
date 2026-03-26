package model

import (
	"time"

	"gorm.io/gorm"
)

type ActivityType string

const (
	ActivityTypeBeerAdded    ActivityType = "beer_added"
	ActivityTypeBeerConsumed ActivityType = "beer_consumed"
)

type NameCount struct {
	Name  string
	Count int64
}

type MonthlyCount struct {
	Month int32
	Count int64
}

type YearInReview struct {
	Year          int
	CellarID      uint
	BeersConsumed int64
	UniqueBeers   int64
	TotalVolumeMl float64
	AverageRating float64
	BeersAdded    int64
	TopStyles     []NameCount
	TopCategories []NameCount
	TopBreweries  []NameCount
	ByMonth       []MonthlyCount
}

type Activity struct {
	gorm.Model
	CellarID         uint         `gorm:"index"`
	CellarEntryID    *uint        `gorm:"index:idx_activity_entry_type"`
	ActivityType     ActivityType `gorm:"type:varchar(32);index:idx_activity_entry_type"`
	Quantity         int64
	OccurredAt       time.Time `gorm:"index"`
	Note             *string
	Rating           *float64
	UntappdCheckinID *uint64 `gorm:"uniqueIndex"`

	Cellar      Cellar       `gorm:"foreignKey:CellarID"`
	CellarEntry *CellarEntry `gorm:"foreignKey:CellarEntryID"`
}
