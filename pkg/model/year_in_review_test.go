package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"droscher.com/BeerGargoyle/pkg/model"
)

const (
	nameBourbonStout = "Goose Island Bourbon Stout"
	nameTableBeer    = "Fonta Flora Table Beer"
	nameHeadyTopper  = "Heady Topper"
	nameMysteryLager = "Mystery Lager"
)

func baseTime() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }

func tally(id uint, name string, abv, rating *float64, count int64, daysAfter int) model.BeerTally {
	return model.BeerTally{
		YearInReviewBeer: model.YearInReviewBeer{
			BeerID: id, Name: name, ABV: abv, ExternalRating: rating, Count: count,
		},
		LastConsumedAt: baseTime().AddDate(0, 0, daysAfter),
	}
}

func TestPickBeerHighlights_EmptyInputIsZeroValue(t *testing.T) {
	assert.Equal(t, model.BeerHighlights{}, model.PickBeerHighlights(nil))
}

func TestPickBeerHighlights_MostHadTieBreaksOnMostRecent(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameBourbonStout, nil, nil, 3, 5),
		tally(2, nameTableBeer, nil, nil, 3, 9),
		tally(3, nameHeadyTopper, nil, nil, 1, 20),
	})

	require.NotNil(t, got.MostHad)
	assert.Equal(t, nameTableBeer, got.MostHad.Name)
}

func TestPickBeerHighlights_IgnoresNilAndZeroAbv(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameMysteryLager, nil, nil, 9, 1),
		tally(2, nameHeadyTopper, new(0.0), nil, 9, 2),
		tally(3, nameBourbonStout, new(14.2), nil, 1, 3),
		tally(4, nameTableBeer, new(2.8), nil, 1, 4),
	})

	require.NotNil(t, got.Strongest)
	require.NotNil(t, got.Gentlest)
	assert.Equal(t, nameBourbonStout, got.Strongest.Name)
	assert.Equal(t, nameTableBeer, got.Gentlest.Name)
}

func TestPickBeerHighlights_SingleAbvBeerOmitsGentlest(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameBourbonStout, new(14.2), nil, 1, 1),
		tally(2, nameMysteryLager, nil, nil, 1, 2),
	})

	require.NotNil(t, got.Strongest)
	assert.Nil(t, got.Gentlest)
}

func TestPickBeerHighlights_AbvTieBreaksOnCountThenRecency(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameBourbonStout, new(8.0), nil, 2, 9),
		tally(2, nameTableBeer, new(8.0), nil, 5, 1),
		tally(3, nameHeadyTopper, new(8.0), nil, 5, 4),
		tally(4, nameMysteryLager, new(2.4), nil, 1, 1),
	})

	require.NotNil(t, got.Strongest)
	assert.Equal(t, nameHeadyTopper, got.Strongest.Name)
	require.NotNil(t, got.Gentlest)
	assert.Equal(t, nameMysteryLager, got.Gentlest.Name)
}

func TestPickBeerHighlights_IgnoresUnratedBeers(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameMysteryLager, nil, nil, 1, 1),
		tally(2, nameHeadyTopper, nil, new(0.0), 1, 2),
		tally(3, nameBourbonStout, nil, new(4.5), 1, 3),
		tally(4, nameTableBeer, nil, new(3.1), 1, 4),
	})

	require.NotNil(t, got.HighestRated)
	require.NotNil(t, got.LowestRated)
	assert.Equal(t, nameBourbonStout, got.HighestRated.Name)
	assert.Equal(t, nameTableBeer, got.LowestRated.Name)
}

func TestPickBeerHighlights_SingleRatedBeerOmitsLowest(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameBourbonStout, nil, new(4.5), 1, 1),
	})

	require.NotNil(t, got.HighestRated)
	assert.Nil(t, got.LowestRated)
}

func TestPickBeerHighlights_SameBeerNeverHighestAndLowest(t *testing.T) {
	got := model.PickBeerHighlights([]model.BeerTally{
		tally(1, nameBourbonStout, new(8.0), new(4.0), 1, 1),
		tally(2, nameTableBeer, new(8.0), new(4.0), 1, 2),
	})

	require.NotNil(t, got.HighestRated)
	assert.Nil(t, got.LowestRated)
	require.NotNil(t, got.Strongest)
	assert.Nil(t, got.Gentlest)
}
