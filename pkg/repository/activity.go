package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
)

type ActivityRepository interface {
	CreateActivity(ctx context.Context, activity *model.Activity) (*model.Activity, error)
	FindByUntappdCheckinID(ctx context.Context, checkinID uint64) (*model.Activity, error)
}

func (r *Repository) CreateActivity(ctx context.Context, activity *model.Activity) (*model.Activity, error) {
	if result := r.DB.WithContext(ctx).Create(activity); result.Error != nil {
		return nil, result.Error
	}

	return activity, nil
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
