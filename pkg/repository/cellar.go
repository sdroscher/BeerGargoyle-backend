package repository

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
)

type CellarRepository interface { //nolint:interfacebloat // this is an acceptable interface
	AddBeerToCellar(ctx context.Context, beer model.CellarEntry) (*model.CellarEntry, error)
	AddCellar(ctx context.Context, name string, description string, locations []string, owner model.User) (*model.Cellar, error)
	DeleteCellarEntry(ctx context.Context, cellarEntryID uint) error
	FindBeerRecommendations(ctx context.Context, cellarID uint64, filter *api.CellarFilter) ([]*model.CellarEntry, error)
	GetAdventCalendarByID(ctx context.Context, cellarID uint64, calendarID uint64) (*model.AdventCalendar, error)
	GetAdventCalendarByName(ctx context.Context, cellarID uint64, name string) (*model.AdventCalendar, error)
	GetAdventCalendarForDate(ctx context.Context, cellarID uint64, date time.Time) (*model.AdventCalendar, error)
	GetCellarBreweryNames(ctx context.Context, cellarID uint64) ([]*model.Brewery, error)
	GetCellarByID(ctx context.Context, cellarID uint) (*model.Cellar, error)
	GetCellarEntryByID(ctx context.Context, cellarEntryID uint) (*model.CellarEntry, error)
	GetCellarBeers(ctx context.Context, cellarID uint) ([]*model.CellarEntry, error)
	GetCellarRecommendationRanges(ctx context.Context, cellarID uint64) (*model.CellarRecommendationRanges, error)
	GetCellarStats(ctx context.Context, cellarID uint) (*model.CellarStats, error)
	GetCellarStyles(ctx context.Context, cellarID uint64) ([]*model.BeerStyle, error)
	GetCellarsForUser(ctx context.Context, user model.User) ([]*model.Cellar, error)
	SaveAdventCalendar(ctx context.Context, calendar model.AdventCalendar) (*model.AdventCalendar, error)
	UpdateAdventCalendar(ctx context.Context, cellarID uint64, calendarID uint64, day time.Time) error
	UpdateCellarEntry(ctx context.Context, entry *model.CellarEntry) (*model.CellarEntry, error)
}

func (r *Repository) AddCellar(ctx context.Context, name string, description string, locations []string, owner model.User) (*model.Cellar, error) {
	cellar := model.Cellar{
		Name:        name,
		Description: description,
		OwnerID:     owner.ID,
		Locations:   make([]model.LocationInCellar, 0, len(locations)),
	}

	for _, location := range locations {
		cellar.Locations = append(cellar.Locations, model.LocationInCellar{Name: location})
	}

	if result := r.DB.WithContext(ctx).Create(&cellar); result.Error != nil {
		return nil, result.Error
	}

	return &cellar, nil
}

func (r *Repository) GetCellarsForUser(ctx context.Context, user model.User) ([]*model.Cellar, error) {
	var cellars []*model.Cellar

	result := r.DB.WithContext(ctx).Where("owner_id = ?", user.ID).
		Joins("Owner").
		Preload("Locations").
		Find(&cellars)
	if result.Error != nil {
		r.Logger.Error("error getting cellars for user", zap.Uint("user_id", user.ID), zap.Error(result.Error))

		return nil, result.Error
	}

	return cellars, nil
}

func (r *Repository) GetCellarByID(ctx context.Context, cellarID uint) (*model.Cellar, error) {
	var cellar model.Cellar

	result := r.DB.WithContext(ctx).
		Joins("Owner").
		Preload("Locations").
		First(&cellar, cellarID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &cellar, nil
}

func (r *Repository) AddBeerToCellar(ctx context.Context, beer model.CellarEntry) (*model.CellarEntry, error) {
	if result := r.DB.WithContext(ctx).Create(&beer); result.Error != nil {
		return nil, result.Error
	}

	return &beer, nil
}

func (r *Repository) GetCellarEntryByID(ctx context.Context, cellarEntryID uint) (*model.CellarEntry, error) {
	var cellarEntry model.CellarEntry

	result := r.DB.WithContext(ctx).
		Joins("Beer").
		Joins("Location").
		Joins("Format").
		First(&cellarEntry, cellarEntryID)
	if result.Error != nil {
		return nil, result.Error
	}

	cellar, err := r.GetCellarByID(ctx, cellarEntry.CellarID)
	if err != nil {
		return nil, err
	}

	cellarEntry.Cellar = *cellar

	return &cellarEntry, nil
}

func (r *Repository) GetCellarStats(ctx context.Context, cellarID uint) (*model.CellarStats, error) {
	var stats model.CellarStats

	result := r.DB.WithContext(ctx).Table("cellar_entries as ce").
		Select("sum(quantity) as beer_count, "+
			"count(distinct ce.beer_id) as unique_count, "+
			"sum(bf.size_metric*quantity) as total_volume, "+
			"count(distinct b.brewery_id) as brewery_count, "+
			"sum(case when had_before = true then 0 else 1 end) as untried_count, "+
			"sum(case when special = true then 1 else 0 end) as special_count, "+
			"avg(b.abv) as average_abv, "+
			"avg(b.external_rating) as average_rating").
		Joins("INNER JOIN beer_formats bf on bf.id = ce.format_id").
		Joins("INNER JOIN beers b on b.id = ce.beer_id").
		Where("cellar_id = ?", cellarID).
		Where("ce.deleted_at is null").
		Scan(&stats)

	if result.Error != nil {
		return nil, result.Error
	}

	stats.CellarID = cellarID

	return &stats, nil
}

func (r *Repository) GetCellarBeers(ctx context.Context, cellarID uint) ([]*model.CellarEntry, error) {
	var beers []*model.CellarEntry

	result := r.DB.WithContext(ctx).
		Joins("Beer").
		Joins("Location").
		Joins("Format").
		Joins("Cellar").
		Preload("Tags").
		Preload("Beer.Brewery").
		Preload("Beer.Brewery.Address").
		Preload("Beer.Style").
		Where("cellar_entries.cellar_id = ?", cellarID).
		Find(&beers)
	if result.Error != nil {
		return nil, result.Error
	}

	return beers, nil
}

func (r *Repository) DeleteCellarEntry(ctx context.Context, cellarEntryID uint) error {
	result := r.DB.WithContext(ctx).Delete(&model.CellarEntry{}, cellarEntryID)

	return result.Error
}

func (r *Repository) UpdateCellarEntry(ctx context.Context, entry *model.CellarEntry) (*model.CellarEntry, error) {
	if result := r.DB.WithContext(ctx).Save(&entry); result.Error != nil {
		return nil, result.Error
	}

	return entry, nil
}

func (r *Repository) FindBeerRecommendations(ctx context.Context, cellarID uint64, filter *api.CellarFilter) ([]*model.CellarEntry, error) {
	var beers []*model.CellarEntry

	query := r.DB.WithContext(ctx).
		Joins("Beer").
		Joins("Location").
		Joins("Format").
		Joins("Cellar").
		Preload("Tags").
		Preload("Beer.Brewery").
		Preload("Beer.Brewery.Address").
		Preload("Beer.Style").
		Where("cellar_entries.cellar_id = ?", cellarID)

	updateQueryWithCriteria(filter, query)

	if result := query.Find(&beers); result.Error != nil {
		return nil, result.Error
	}

	return beers, nil
}

//nolint:cyclop // this is as simple as it can be given the number of parameters
func updateQueryWithCriteria(filter *api.CellarFilter, query *gorm.DB) {
	if filter.BreweryId != nil {
		query = query.Where(`"Beer".brewery_id = ?`, filter.GetBreweryId())
	}

	if filter.MinimumAbv != nil {
		query = query.Where(`"Beer".abv >= ?`, filter.GetMinimumAbv())
	}

	if filter.MaximumAbv != nil {
		query = query.Where(`"Beer".ABV <= ?`, filter.GetMaximumAbv())
	}

	if filter.MinimumRating != nil {
		query = query.Where(`"Beer".external_rating >= ?`, filter.GetMinimumRating())
	}

	if filter.MaximumRating != nil {
		query = query.Where(`"Beer".external_rating <= ?`, filter.GetMaximumRating())
	}

	if filter.MinimumSize != nil {
		query = query.Where(`"Format".size_metric >= ?`, filter.GetMinimumSize())
	}

	if filter.MaximumSize != nil {
		query = query.Where(`"Format".size_metric <= ?`, filter.GetMaximumSize())
	}

	if filter.Special != nil {
		query = query.Where("special = ?", filter.GetSpecial())
	}

	if filter.HadBefore != nil {
		query = query.Where("had_before = ?", filter.GetHadBefore())
	}

	if filter.StyleId != nil {
		query.Where(`"Beer".style_id = ?`, filter.GetStyleId())
	}

	if filter.OverdueToDrink != nil {
		query.Where("drink_before < ?", time.Now())
	}

	if filter.MinimumQuantity != nil {
		query.Where("quantity >= ?", filter.GetMinimumQuantity())
	}

	if filter.MinimumVintage != nil {
		query.Where("vintage >= ?", filter.GetMinimumVintage())
	}

	if filter.MaximumVintage != nil {
		query.Where("vintage <= ?", filter.GetMaximumVintage())
	}

	if len(filter.GetTags()) > 0 {
		query.Where("cellar_entries.id IN (SELECT cellar_entry_id FROM cellar_entry_tags INNER JOIN tags ON tag_id = tags.id WHERE tag IN ? GROUP BY cellar_entry_id HAVING COUNT(*) = ?)", filter.GetTags(), len(filter.GetTags()))
	}

	if filter.GetAddedBefore() != nil {
		query.Where("date_added < ?", filter.GetAddedBefore().AsTime())
	}
}

func (r *Repository) GetCellarBreweryNames(ctx context.Context, cellarID uint64) ([]*model.Brewery, error) {
	var breweries []*model.Brewery

	result := r.DB.WithContext(ctx).Table("breweries").
		Joins("INNER JOIN beers b on breweries.id = b.brewery_id").
		Joins("INNER JOIN cellar_entries ce on b.id = ce.beer_id").
		Where("ce.cellar_id = ?", cellarID).
		Distinct("breweries.id", "breweries.name").
		Order("breweries.name asc").
		Find(&breweries)

	if result.Error != nil {
		return nil, result.Error
	}

	return breweries, nil
}

func (r *Repository) GetCellarStyles(ctx context.Context, cellarID uint64) ([]*model.BeerStyle, error) {
	var styles []*model.BeerStyle

	result := r.DB.WithContext(ctx).Table("beer_styles").
		Joins("INNER JOIN beers b on beer_styles.id = b.style_id").
		Joins("INNER JOIN cellar_entries ce on b.id = ce.beer_id").
		Where("ce.cellar_id = ?", cellarID).
		Distinct("beer_styles.id", "beer_styles.name").
		Order("beer_styles.name asc").
		Find(&styles)

	if result.Error != nil {
		return nil, result.Error
	}

	return styles, nil
}

func (r *Repository) GetCellarRecommendationRanges(ctx context.Context, cellarID uint64) (*model.CellarRecommendationRanges, error) {
	var ranges *model.CellarRecommendationRanges

	result := r.DB.WithContext(ctx).Table("cellar_entries ce").
		Joins("INNER JOIN beers b on b.id = ce.beer_id").
		Joins("INNER JOIN beer_formats bf on ce.format_id = bf.id").
		Where("ce.cellar_id = ?", cellarID).
		Select("min(b.abv) as minimum_abv",
			"max(b.abv) as maximum_abv",
			"min(bf.size_metric) as minimum_size",
			"max(bf.size_metric) as maximum_size",
			"min(ce.vintage) as minimum_vintage",
			"max(ce.vintage) as maximum_vintage",
			"round(min(b.external_rating), 2) as minimum_rating",
			"round(max(b.external_rating), 2) as maximum_rating",
			"min(ce.date_added) as oldest_added_date").
		Take(&ranges)

	if result.Error != nil {
		return nil, result.Error
	}

	return ranges, nil
}

func (r *Repository) SaveAdventCalendar(ctx context.Context, calendar model.AdventCalendar) (*model.AdventCalendar, error) {
	result := r.DB.WithContext(ctx).Create(&calendar)
	if result.Error != nil {
		return nil, result.Error
	}

	return &calendar, nil
}

func (r *Repository) GetAdventCalendarByID(ctx context.Context, cellarID uint64, calendarID uint64) (*model.AdventCalendar, error) {
	var calendar *model.AdventCalendar

	result := r.DB.WithContext(ctx).
		Preload("Beers", func(db *gorm.DB) *gorm.DB { return db.Order("advent_calendar_beers.day ASC") }).
		Preload("Beers.CellarEntry", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Beers.CellarEntry.Beer").
		Preload("Beers.CellarEntry.Beer.Brewery").
		Preload("Beers.CellarEntry.Beer.Style").
		Preload("Beers.CellarEntry.Location").
		Preload("Beers.CellarEntry.Tags").
		Where("cellar_id = ?", cellarID).
		First(&calendar, calendarID)
	if result.Error != nil {
		return nil, result.Error
	}

	return calendar, nil
}

func (r *Repository) GetAdventCalendarForDate(ctx context.Context, cellarID uint64, date time.Time) (*model.AdventCalendar, error) {
	var calendar *model.AdventCalendar

	result := r.DB.WithContext(ctx).
		Preload("Beers", func(db *gorm.DB) *gorm.DB { return db.Order("advent_calendar_beers.day ASC") }).
		Preload("Beers.CellarEntry", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Beers.CellarEntry.Beer").
		Preload("Beers.CellarEntry.Beer.Brewery").
		Preload("Beers.CellarEntry.Beer.Style").
		Preload("Beers.CellarEntry.Location").
		Preload("Beers.CellarEntry.Tags").
		Where("cellar_id = ?", cellarID).
		Where("? between start_date and end_date", date).
		First(&calendar)
	if result.Error != nil {
		return nil, result.Error
	}

	return calendar, nil
}

func (r *Repository) GetAdventCalendarByName(ctx context.Context, cellarID uint64, name string) (*model.AdventCalendar, error) {
	var calendar *model.AdventCalendar

	result := r.DB.WithContext(ctx).
		Preload("Beers", func(db *gorm.DB) *gorm.DB { return db.Order("advent_calendar_beers.day ASC") }).
		Preload("Beers.CellarEntry", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Beers.CellarEntry.Beer").
		Preload("Beers.CellarEntry.Beer.Brewery").
		Preload("Beers.CellarEntry.Beer.Style").
		Preload("Beers.CellarEntry.Location").
		Preload("Beers.CellarEntry.Tags").
		Where("cellar_id = ?", cellarID).
		Where("name = ?", name).
		First(&calendar)
	if result.Error != nil {
		return nil, result.Error
	}

	return calendar, nil
}

func (r *Repository) UpdateAdventCalendar(ctx context.Context, cellarID uint64, calendarID uint64, day time.Time) error {
	result := r.DB.WithContext(ctx).Exec(
		"UPDATE advent_calendar_beers SET revealed = true, updated_at = CURRENT_TIMESTAMP"+
			" FROM advent_calendars"+
			" WHERE advent_calendar_beers.advent_calendar_id = advent_calendars.id"+
			" AND advent_calendar_id = ?"+
			" AND advent_calendars.cellar_id = ?"+
			" AND day = ?", calendarID, cellarID, day)

	return result.Error
}
