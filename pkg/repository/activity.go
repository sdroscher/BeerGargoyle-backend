package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
)

type ActivityRepository interface {
	CreateActivity(ctx context.Context, activity *model.Activity) (*model.Activity, error)
	FindByUntappdCheckinID(ctx context.Context, checkinID uint64) (*model.Activity, error)

	// GetFeed returns activities for a cellar sorted by occurred_at DESC.
	// Returns the page of activities and the total count (before pagination).
	GetFeed(ctx context.Context, cellarID uint, from, until *time.Time, limit, offset int) ([]*model.Activity, int64, error)

	// GetYearInReview returns aggregate stats for a cellar for a given calendar year.
	GetYearInReview(ctx context.Context, cellarID uint, year int) (*model.YearInReview, error)
}

func (r *Repository) CreateActivity(ctx context.Context, activity *model.Activity) (*model.Activity, error) {
	if result := r.DB.WithContext(ctx).Create(activity); result.Error != nil {
		return nil, result.Error
	}

	return activity, nil
}

func (r *Repository) GetFeed(ctx context.Context, cellarID uint, from, until *time.Time, limit, offset int) ([]*model.Activity, int64, error) {
	var total int64

	err := r.feedQuery(ctx, cellarID, from, until).Model(&model.Activity{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var activities []*model.Activity

	err = r.feedQuery(ctx, cellarID, from, until).
		Preload("CellarEntry", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("CellarEntry.Beer").
		Preload("CellarEntry.Beer.Brewery").
		Preload("CellarEntry.Beer.Brewery.Address").
		Preload("CellarEntry.Beer.Style").
		Preload("CellarEntry.Beer.Style.BJCPStyle").
		Preload("CellarEntry.Beer.Style.BJCPStyle.Family").
		Preload("CellarEntry.Beer.Style.BJCPStyle.Category").
		Preload("CellarEntry.Format").
		Preload("CellarEntry.Location").
		Preload("CellarEntry.Tags").
		Order("occurred_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&activities).Error
	if err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}

func (r *Repository) feedQuery(ctx context.Context, cellarID uint, from, until *time.Time) *gorm.DB {
	query := r.DB.WithContext(ctx).Where("cellar_id = ?", cellarID)

	if from != nil {
		query = query.Where("occurred_at >= ?", from)
	}

	if until != nil {
		query = query.Where("occurred_at <= ?", until)
	}

	return query
}

const topNLimit = 5

func (r *Repository) GetYearInReview(ctx context.Context, cellarID uint, year int) (*model.YearInReview, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := yearStart.AddDate(1, 0, 0)

	result := &model.YearInReview{Year: year, CellarID: cellarID}

	err := r.scanConsumptionSummary(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	err = r.scanAddedSummary(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	err = r.scanTopStyles(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	err = r.scanTopCategories(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	err = r.scanTopBreweries(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	err = r.scanByMonth(ctx, result, cellarID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) scanConsumptionSummary(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	type summary struct {
		BeersConsumed int64
		UniqueBeers   int64
		TotalVolumeMl float64
		AverageRating float64
	}

	var row summary

	err := r.DB.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(a.quantity), 0)                    AS beers_consumed,
			COUNT(DISTINCT a.cellar_entry_id)               AS unique_beers,
			COALESCE(SUM(a.quantity * bf.size_metric), 0)   AS total_volume_ml,
			COALESCE(AVG(a.rating), 0)                      AS average_rating
		FROM activities a
		LEFT JOIN cellar_entries ce ON ce.id = a.cellar_entry_id
		LEFT JOIN beer_formats bf   ON bf.id = ce.format_id
		WHERE a.cellar_id      = ?
		  AND a.activity_type  = ?
		  AND a.occurred_at   >= ?
		  AND a.occurred_at    < ?
		  AND a.deleted_at    IS NULL
	`, cellarID, model.ActivityTypeBeerConsumed, yearStart, yearEnd).Scan(&row).Error
	if err != nil {
		return err
	}

	result.BeersConsumed = row.BeersConsumed
	result.UniqueBeers = row.UniqueBeers
	result.TotalVolumeMl = row.TotalVolumeMl
	result.AverageRating = row.AverageRating

	return nil
}

func (r *Repository) scanAddedSummary(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	var beersAdded int64

	err := r.DB.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(quantity), 0)
		FROM activities
		WHERE cellar_id     = ?
		  AND activity_type = ?
		  AND occurred_at  >= ?
		  AND occurred_at   < ?
		  AND deleted_at   IS NULL
	`, cellarID, model.ActivityTypeBeerAdded, yearStart, yearEnd).Scan(&beersAdded).Error
	if err != nil {
		return err
	}

	result.BeersAdded = beersAdded

	return nil
}

func (r *Repository) scanTopStyles(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	err := r.DB.WithContext(ctx).Raw(`
		SELECT bs.name AS name, SUM(a.quantity) AS count
		FROM activities a
		JOIN cellar_entries ce ON ce.id = a.cellar_entry_id
		JOIN beers b           ON b.id  = ce.beer_id
		JOIN beer_styles bs    ON bs.id = b.style_id
		WHERE a.cellar_id     = ?
		  AND a.activity_type = ?
		  AND a.occurred_at  >= ?
		  AND a.occurred_at   < ?
		  AND a.deleted_at   IS NULL
		GROUP BY bs.name
		ORDER BY count DESC
		LIMIT ?
	`, cellarID, model.ActivityTypeBeerConsumed, yearStart, yearEnd, topNLimit).
		Scan(&result.TopStyles).Error

	return err
}

func (r *Repository) scanTopCategories(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	err := r.DB.WithContext(ctx).Raw(`
		SELECT bc.name AS name, SUM(a.quantity) AS count
		FROM activities a
		JOIN cellar_entries ce  ON ce.id    = a.cellar_entry_id
		JOIN beers b            ON b.id     = ce.beer_id
		JOIN beer_styles bs     ON bs.id    = b.style_id
		JOIN beer_style_bjcps bsb ON bsb.bjcp_id = bs.bjcp_style_id
		JOIN beer_category_bjcps bc ON bc.id = bsb.category_id
		WHERE a.cellar_id     = ?
		  AND a.activity_type = ?
		  AND a.occurred_at  >= ?
		  AND a.occurred_at   < ?
		  AND a.deleted_at   IS NULL
		GROUP BY bc.name
		ORDER BY count DESC
		LIMIT ?
	`, cellarID, model.ActivityTypeBeerConsumed, yearStart, yearEnd, topNLimit).
		Scan(&result.TopCategories).Error

	return err
}

func (r *Repository) scanTopBreweries(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	err := r.DB.WithContext(ctx).Raw(`
		SELECT br.name AS name, SUM(a.quantity) AS count
		FROM activities a
		JOIN cellar_entries ce ON ce.id  = a.cellar_entry_id
		JOIN beers b           ON b.id   = ce.beer_id
		JOIN breweries br      ON br.id  = b.brewery_id
		WHERE a.cellar_id     = ?
		  AND a.activity_type = ?
		  AND a.occurred_at  >= ?
		  AND a.occurred_at   < ?
		  AND a.deleted_at   IS NULL
		GROUP BY br.name
		ORDER BY count DESC
		LIMIT ?
	`, cellarID, model.ActivityTypeBeerConsumed, yearStart, yearEnd, topNLimit).
		Scan(&result.TopBreweries).Error

	return err
}

func (r *Repository) scanByMonth(ctx context.Context, result *model.YearInReview, cellarID uint, yearStart, yearEnd time.Time) error {
	err := r.DB.WithContext(ctx).Raw(`
		SELECT EXTRACT(MONTH FROM occurred_at)::int AS month, SUM(quantity) AS count
		FROM activities
		WHERE cellar_id     = ?
		  AND activity_type = ?
		  AND occurred_at  >= ?
		  AND occurred_at   < ?
		  AND deleted_at   IS NULL
		GROUP BY month
		ORDER BY month
	`, cellarID, model.ActivityTypeBeerConsumed, yearStart, yearEnd).
		Scan(&result.ByMonth).Error

	return err
}

func (r *Repository) FindByUntappdCheckinID(ctx context.Context, checkinID uint64) (*model.Activity, error) {
	var activity model.Activity

	result := r.DB.WithContext(ctx).Where("untappd_checkin_id = ?", checkinID).First(&activity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil //nolint:nilnil // nil means "not found", not an error
		}

		return nil, result.Error
	}

	return &activity, nil
}
