package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"droscher.com/BeerGargoyle/pkg/model"
)

type ActivityRepository interface {
	CreateActivity(ctx context.Context, activity *model.Activity) (*model.Activity, error)

	// GetFeed returns activities for a cellar sorted by occurred_at DESC.
	// Returns the page of activities and the total count (before pagination).
	GetFeed(ctx context.Context, cellarID uint, from, until *time.Time, limit, offset int) ([]*model.Activity, int64, error)

	// GetYearInReview returns aggregate stats for a cellar for a given calendar year.
	GetYearInReview(ctx context.Context, cellarID uint, year int) (*model.YearInReview, error)

	// GetUntappdCheckinIDsForCellar returns the set of Untappd checkin IDs already stored for a cellar.
	GetUntappdCheckinIDsForCellar(ctx context.Context, cellarID uint) (map[uint64]struct{}, error)

	// GetOrInitImportState returns the import watermark for a cellar, creating it (seeded from the
	// oldest cellar entry) if it does not exist yet.
	GetOrInitImportState(ctx context.Context, cellarID uint) (*model.UntappdImportState, error)

	// GetConsumedActivitiesForCellar returns beer_consumed activities that have no Untappd checkin
	// linked yet, indexed by cellar_entry_id. Used to correlate CSV rows with backfill records.
	GetConsumedActivitiesForCellar(ctx context.Context, cellarID uint) (map[uint][]model.ConsumedActivitySummary, error)

	// ImportActivitiesWithState bulk-inserts new activities, applies Untappd metadata updates,
	// and advances the watermark atomically.
	// Returns the number of rows inserted and the number of rows updated.
	ImportActivitiesWithState(ctx context.Context, activities []*model.Activity, updates []model.ActivityUntappdUpdate, state *model.UntappdImportState, newWatermark time.Time) (int64, int64, error)
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

func (r *Repository) GetUntappdCheckinIDsForCellar(ctx context.Context, cellarID uint) (map[uint64]struct{}, error) {
	var ids []uint64

	err := r.DB.WithContext(ctx).
		Model(&model.Activity{}).
		Where("cellar_id = ? AND untappd_checkin_id IS NOT NULL", cellarID).
		Pluck("untappd_checkin_id", &ids).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}

	return result, nil
}

func (r *Repository) GetOrInitImportState(ctx context.Context, cellarID uint) (*model.UntappdImportState, error) {
	var state model.UntappdImportState

	result := r.DB.WithContext(ctx).Where("cellar_id = ?", cellarID).First(&state)
	if result.Error == nil {
		return &state, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	// Seed watermark from the oldest cellar entry so we skip pre-app history.
	var minCreatedAt time.Time

	err := r.DB.WithContext(ctx).Raw(
		`SELECT COALESCE(MIN(created_at), '1970-01-01 00:00:00') FROM cellar_entries WHERE cellar_id = ?`,
		cellarID,
	).Scan(&minCreatedAt).Error
	if err != nil {
		return nil, err
	}

	state = model.UntappdImportState{CellarID: cellarID, LastImportAt: minCreatedAt}

	// Use FirstOrCreate to handle the unlikely concurrent-insert race on the unique index.
	err = r.DB.WithContext(ctx).Where("cellar_id = ?", cellarID).FirstOrCreate(&state).Error
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func (r *Repository) GetConsumedActivitiesForCellar(ctx context.Context, cellarID uint) (map[uint][]model.ConsumedActivitySummary, error) {
	var summaries []model.ConsumedActivitySummary

	err := r.DB.WithContext(ctx).Raw(`
		SELECT id, cellar_entry_id AS entry_id, occurred_at
		FROM activities
		WHERE cellar_id = ?
		  AND activity_type = ?
		  AND untappd_checkin_id IS NULL
		  AND cellar_entry_id IS NOT NULL
		  AND deleted_at IS NULL
	`, cellarID, model.ActivityTypeBeerConsumed).Scan(&summaries).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]model.ConsumedActivitySummary, len(summaries))
	for _, s := range summaries {
		result[s.EntryID] = append(result[s.EntryID], s)
	}

	return result, nil
}

func (r *Repository) ImportActivitiesWithState(
	ctx context.Context,
	activities []*model.Activity,
	updates []model.ActivityUntappdUpdate,
	state *model.UntappdImportState,
	newWatermark time.Time,
) (int64, int64, error) {
	var imported, updated int64

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(activities) > 0 {
			result := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "untappd_checkin_id"}},
				DoNothing: true,
			}).Create(&activities)
			if result.Error != nil {
				return result.Error
			}

			imported = result.RowsAffected
		}

		for _, upd := range updates {
			cols := map[string]any{"untappd_checkin_id": upd.CheckinID}
			if upd.Rating != nil {
				cols["rating"] = upd.Rating
			}

			if upd.Note != nil {
				cols["note"] = upd.Note
			}

			result := tx.Model(&model.Activity{}).Where("id = ?", upd.ActivityID).Updates(cols)
			if result.Error != nil {
				return result.Error
			}

			updated += result.RowsAffected
		}

		hadBeforeErr := markEntriesHadBefore(tx, activities, updates)
		if hadBeforeErr != nil {
			return hadBeforeErr
		}

		return tx.Model(state).Update("last_import_at", newWatermark).Error
	})
	if err != nil {
		return 0, 0, err
	}

	return imported, updated, nil
}

// markEntriesHadBefore sets had_before = TRUE on cellar entries touched by the import.
func markEntriesHadBefore(tx *gorm.DB, activities []*model.Activity, updates []model.ActivityUntappdUpdate) error {
	entryIDs := make([]uint, 0, len(activities)+len(updates))

	for _, act := range activities {
		if act.CellarEntryID != nil {
			entryIDs = append(entryIDs, *act.CellarEntryID)
		}
	}

	for _, upd := range updates {
		entryIDs = append(entryIDs, upd.EntryID)
	}

	if len(entryIDs) == 0 {
		return nil
	}

	err := tx.Unscoped().Model(&model.CellarEntry{}).
		Where("id IN ?", entryIDs).
		Update("had_before", true).Error

	return err
}
