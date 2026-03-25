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
