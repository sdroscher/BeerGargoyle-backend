package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"droscher.com/BeerGargoyle/pkg/model"
)

var ErrBreweryNotFound = errors.New("brewery not found")

func (r *Repository) AddBeer(ctx context.Context, beer model.Beer) (*model.Beer, error) {
	result := r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}, {Name: "brewery_id"}},
		UpdateAll: true,
	}).Create(&beer)

	if result.Error != nil {
		return nil, result.Error
	}

	return &beer, nil
}

func (r *Repository) FindBreweryByExternalSource(ctx context.Context, externalID uint64, externalSource string) (*model.Brewery, error) {
	brewery := &model.Brewery{}
	result := r.DB.WithContext(ctx).Model(&brewery).
		Where(`external_id = ? AND external_source = ?`, externalID, externalSource).
		First(&brewery)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrBreweryNotFound
		}

		return nil, result.Error
	}

	return brewery, nil
}

func (r *Repository) AddBeerStyle(ctx context.Context, style string) (*model.BeerStyle, error) {
	beerStyle := model.BeerStyle{Name: style}
	if result := r.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&beerStyle); result.Error != nil {
		return nil, result.Error
	}

	if beerStyle.ID == 0 {
		if result := r.DB.WithContext(ctx).Where("name = ?", style).First(&beerStyle); result.Error != nil {
			return nil, result.Error
		}
	}

	return &beerStyle, nil
}

func (r *Repository) GetBeerFormats(ctx context.Context) ([]*model.BeerFormat, error) {
	var beerFormats []*model.BeerFormat

	if result := r.DB.WithContext(ctx).Order("package, size_metric").Find(&beerFormats); result.Error != nil {
		return nil, result.Error
	}

	return beerFormats, nil
}

func (r *Repository) GetTagsByNames(ctx context.Context, names []string) (map[string]model.Tag, error) {
	var tags []*model.Tag

	if result := r.DB.WithContext(ctx).Where("tag in (?)", names).Find(&tags); result.Error != nil {
		return nil, result.Error
	}

	tagsByName := make(map[string]model.Tag, len(tags))

	for index := range tags {
		tag := tags[index]
		tagsByName[tag.Tag] = *tag
	}

	return tagsByName, nil
}
