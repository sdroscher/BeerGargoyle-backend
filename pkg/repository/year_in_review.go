package repository

import (
	"context"
	"time"

	"droscher.com/BeerGargoyle/pkg/model"
)

// scanBeerTallies returns one row per beer consumed in the year, with its pour count and last pour.
func (r *Repository) scanBeerTallies(ctx context.Context, cellarID uint, start, end time.Time) ([]model.BeerTally, error) {
	var tallies []model.BeerTally

	err := r.DB.WithContext(ctx).Raw(`
		SELECT b.id              AS beer_id,
		       b.name            AS name,
		       COALESCE(br.name, '') AS brewery_name,
		       COALESCE(bs.name, '') AS style_name,
		       b.abv             AS abv,
		       b.external_rating AS external_rating,
		       SUM(a.quantity)   AS count,
		       MAX(a.occurred_at) AS last_consumed_at
		FROM activities a
		JOIN cellar_entries ce ON ce.id = a.cellar_entry_id
		JOIN beers b           ON b.id  = ce.beer_id
		LEFT JOIN breweries br ON br.id = b.brewery_id
		LEFT JOIN beer_styles bs ON bs.id = b.style_id
		WHERE a.cellar_id     = ?
		  AND a.activity_type = ?
		  AND a.occurred_at  >= ?
		  AND a.occurred_at   < ?
		  AND a.deleted_at   IS NULL
		GROUP BY b.id, b.name, br.name, bs.name, b.abv, b.external_rating
	`, cellarID, model.ActivityTypeBeerConsumed, start, end).Scan(&tallies).Error
	if err != nil {
		return nil, err
	}

	return tallies, nil
}

// scanBiggestDay finds the local calendar day with the most beers consumed; nil when there are none.
func (r *Repository) scanBiggestDay(ctx context.Context, result *model.YearInReview, cellarID uint, start, end time.Time, loc *time.Location) error {
	var days []model.DayCount

	err := r.DB.WithContext(ctx).Raw(`
		SELECT to_char(occurred_at AT TIME ZONE ?, 'YYYY-MM-DD') AS date, SUM(quantity) AS count
		FROM activities
		WHERE cellar_id     = ?
		  AND activity_type = ?
		  AND occurred_at  >= ?
		  AND occurred_at   < ?
		  AND deleted_at   IS NULL
		GROUP BY date
		ORDER BY count DESC, date ASC
		LIMIT 1
	`, loc.String(), cellarID, model.ActivityTypeBeerConsumed, start, end).Scan(&days).Error
	if err != nil {
		return err
	}

	if len(days) > 0 {
		result.BiggestDay = &days[0]
	}

	return nil
}
