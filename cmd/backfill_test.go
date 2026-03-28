package cmd_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/cmd"
	"droscher.com/BeerGargoyle/pkg/model"
)

func TestBuildCandidateActivities_AddedOnly(t *testing.T) {
	entries := []model.CellarEntry{
		{
			Model:    gorm.Model{ID: 1, CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			CellarID: 10,
			Quantity: 4,
		},
	}

	got := cmd.BuildCandidateActivities(entries)

	assert.Len(t, got, 1)
	assert.Equal(t, model.ActivityTypeBeerAdded, got[0].ActivityType)
	assert.Equal(t, int64(4), got[0].Quantity)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), got[0].OccurredAt)
}

func TestBuildCandidateActivities_UsesDateAddedWhenSet(t *testing.T) {
	dateAdded := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	entries := []model.CellarEntry{
		{
			Model:     gorm.Model{ID: 2, CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			CellarID:  10,
			Quantity:  1,
			DateAdded: &dateAdded,
		},
	}

	got := cmd.BuildCandidateActivities(entries)

	assert.Len(t, got, 1)
	assert.Equal(t, dateAdded, got[0].OccurredAt)
}

func TestBuildCandidateActivities_SoftDeletedEntryGetsBothActivities(t *testing.T) {
	deletedAt := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	entries := []model.CellarEntry{
		{
			Model: gorm.Model{
				ID:        3,
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true},
			},
			CellarID: 10,
			Quantity: 0,
		},
	}

	got := cmd.BuildCandidateActivities(entries)

	assert.Len(t, got, 2)
	assert.Equal(t, model.ActivityTypeBeerAdded, got[0].ActivityType)
	assert.Equal(t, model.ActivityTypeBeerConsumed, got[1].ActivityType)
	assert.Equal(t, int64(1), got[1].Quantity) // best-guess fallback
	assert.Equal(t, deletedAt, got[1].OccurredAt)
}

func TestBuildCandidateActivities_ZeroQuantityDefaultsToOne(t *testing.T) {
	entries := []model.CellarEntry{
		{
			Model:    gorm.Model{ID: 4},
			CellarID: 10,
			Quantity: 0,
		},
	}

	got := cmd.BuildCandidateActivities(entries)

	assert.Equal(t, int64(1), got[0].Quantity)
}
