package model

import "time"

// BeerTally is one beer's consumption within a year. Count is the number of pours.
type BeerTally struct {
	YearInReviewBeer

	LastConsumedAt time.Time
}

// BeerHighlights are the notable beers picked from a year's tallies. Any may be nil.
type BeerHighlights struct {
	MostHad, Strongest, Gentlest, HighestRated, LowestRated *YearInReviewBeer
}

// PickBeerHighlights picks the most-had, strongest, gentlest, highest and lowest rated beers.
// ABV considers only beers with abv > 0 and ratings only beers with external_rating > 0. Gentlest
// and lowest rated are nil unless a different beer is strictly lower than the strongest/highest.
func PickBeerHighlights(tallies []BeerTally) BeerHighlights {
	var highlights BeerHighlights

	if best := pickTally(tallies, allQualify, tieWins); best != nil {
		highlights.MostHad = &best.YearInReviewBeer
	}

	highlights.Strongest, highlights.Gentlest = pickExtremes(tallies,
		func(t BeerTally) bool { return t.ABV != nil && *t.ABV > 0 },
		func(t BeerTally) float64 { return *t.ABV })
	highlights.HighestRated, highlights.LowestRated = pickExtremes(tallies,
		func(t BeerTally) bool { return t.ExternalRating != nil && *t.ExternalRating > 0 },
		func(t BeerTally) float64 { return *t.ExternalRating })

	return highlights
}

func allQualify(BeerTally) bool { return true }

// tieWins reports whether a beats b on count, then on most recent consumption.
func tieWins(a, b BeerTally) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	}

	return a.LastConsumedAt.After(b.LastConsumedAt)
}

// pickExtremes returns the highest and lowest beers by value among qualifying tallies, ties going
// to tieWins. The lowest is nil unless its value is strictly below the highest's.
func pickExtremes(tallies []BeerTally, qualifies func(BeerTally) bool, value func(BeerTally) float64) (*YearInReviewBeer, *YearInReviewBeer) {
	highest := pickTally(tallies, qualifies, func(a, b BeerTally) bool {
		return value(a) > value(b) || value(a) == value(b) && tieWins(a, b)
	})
	lowest := pickTally(tallies, qualifies, func(a, b BeerTally) bool {
		return value(a) < value(b) || value(a) == value(b) && tieWins(a, b)
	})

	if highest == nil {
		return nil, nil
	}

	if value(*lowest) >= value(*highest) {
		return &highest.YearInReviewBeer, nil
	}

	return &highest.YearInReviewBeer, &lowest.YearInReviewBeer
}

// pickTally returns a copy of the best qualifying tally according to better, or nil if none qualify.
func pickTally(tallies []BeerTally, qualifies func(BeerTally) bool, better func(a, b BeerTally) bool) *BeerTally {
	var best *BeerTally

	for _, candidate := range tallies {
		if qualifies(candidate) && (best == nil || better(candidate, *best)) {
			best = &candidate
		}
	}

	return best
}
