package grpc

import (
	"go.openly.dev/pointy"
	"google.golang.org/protobuf/types/known/timestamppb"

	"droscher.com/BeerGargoyle/pkg/model"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
)

func BeersFromModel(beers []model.Beer) []*api.Beer {
	pbBeers := make([]*api.Beer, 0, len(beers))

	for _, beer := range beers {
		pbBeer := BeerFromModel(beer)
		pbBeers = append(pbBeers, pbBeer)
	}

	return pbBeers
}

func BeerFromModel(beer model.Beer) *api.Beer {
	address := api.Address{Locality: beer.Brewery.Address.Locality}
	brewery := api.Brewery{
		Name:     beer.Brewery.Name,
		Address:  &address,
		ImageUrl: beer.Brewery.ImageURL,
	}

	if beer.Brewery.ID != 0 {
		brewery.Id = uint64(beer.Brewery.ID)
	}

	if beer.Brewery.ExternalID != nil {
		brewery.ExternalId = pointy.Uint64(*beer.Brewery.ExternalID)
	}

	if beer.Brewery.ExternalSource != nil {
		brewery.ExternalSource = pointy.String(*beer.Brewery.ExternalSource)
	}

	if beer.Brewery.ExternalRating != nil {
		brewery.ExternalRating = pointy.Float64(*beer.Brewery.ExternalRating)
	}

	pbBeer := api.Beer{
		Id:          uint64(beer.ID),
		Name:        beer.Name,
		Description: beer.Description,
		Style:       &api.BeerStyle{Name: beer.Style.Name},
		ImageUrl:    pointy.String(beer.ImageURL),
		Brewery:     &brewery,
	}

	if beer.ABV != nil {
		pbBeer.Abv = *beer.ABV
	}

	if beer.IBU != nil {
		pbBeer.Ibu = pointy.Uint64(*beer.IBU)
	}

	if beer.ExternalID != nil {
		pbBeer.ExternalId = pointy.Uint64(*beer.ExternalID)
	}

	if beer.ExternalSource != nil {
		pbBeer.ExternalSource = pointy.String(*beer.ExternalSource)
	}

	if beer.ExternalRating != nil {
		pbBeer.ExternalRating = pointy.Float64(*beer.ExternalRating)
	}

	return &pbBeer
}

func BeerToModel(pbBeer *api.Beer) model.Beer {
	beer := model.Beer{
		Name:        pbBeer.Name,
		Description: pbBeer.Description,
		ABV:         pointy.Float64(pbBeer.Abv),
	}

	if pbBeer.ImageUrl != nil {
		beer.ImageURL = *pbBeer.ImageUrl
	}

	if pbBeer.Ibu != nil {
		beer.IBU = pointy.Uint64(*pbBeer.Ibu)
	}

	if pbBeer.ExternalId != nil {
		beer.ExternalID = pointy.Uint64(*pbBeer.ExternalId)
	}

	if pbBeer.ExternalSource != nil {
		beer.ExternalSource = pointy.String(*pbBeer.ExternalSource)
	}

	if pbBeer.ExternalRating != nil {
		beer.ExternalRating = pointy.Float64(*pbBeer.ExternalRating)
	}

	return beer
}

func BreweryToModel(pbBrewery *api.Brewery) model.Brewery {
	brewery := model.Brewery{
		Name:        pbBrewery.Name,
		Description: pbBrewery.Description,
		ImageURL:    pbBrewery.ImageUrl,
	}

	if pbBrewery.Address != nil {
		brewery.Address = AddressToModel(pbBrewery.Address)
	}

	if pbBrewery.ExternalId != nil {
		brewery.ExternalID = pointy.Uint64(*pbBrewery.ExternalId)
	}

	if pbBrewery.ExternalSource != nil {
		brewery.ExternalSource = pointy.String(*pbBrewery.ExternalSource)
	}

	if pbBrewery.ExternalRating != nil {
		brewery.ExternalRating = pointy.Float64(*pbBrewery.ExternalRating)
	}

	return brewery
}

func AddressToModel(pbAddress *api.Address) model.Address {
	return model.Address{
		Country:       pbAddress.Country,
		Locality:      pbAddress.Locality,
		Region:        &pbAddress.Region,
		PostalCode:    &pbAddress.PostalCode,
		StreetAddress: &pbAddress.StreetAddress,
	}
}

func CellarsFromModel(cellars []*model.Cellar) []*api.Cellar {
	pbCellars := make([]*api.Cellar, 0, len(cellars))

	for index := range cellars {
		pbCellar := CellarFromModel(cellars[index])
		pbCellars = append(pbCellars, pbCellar)
	}

	return pbCellars
}

func CellarFromModel(cellar *model.Cellar) *api.Cellar {
	return &api.Cellar{
		CellarId:    uint64(cellar.ID),
		Owner:       UserFromModel(cellar.Owner),
		Name:        cellar.Name,
		Description: cellar.Description,
		Locations:   LocationsFromModel(cellar.Locations),
	}
}

func CellarBeersFromModel(cellarEntries []*model.CellarEntry) []*api.CellarBeer {
	beers := make([]*api.CellarBeer, 0, len(cellarEntries))
	for index := range cellarEntries {
		beer := CellarBeerFromModel(cellarEntries[index])
		if beer != nil {
			beers = append(beers, beer)
		}
	}

	return beers
}

func CellarBeerFromModel(cellarEntry *model.CellarEntry) *api.CellarBeer {
	cellarBeer := api.CellarBeer{
		CellarEntryId: uint64(cellarEntry.ID),
		Cellar:        CellarFromModel(&cellarEntry.Cellar),
		Beer:          BeerFromModel(cellarEntry.Beer),
		Quantity:      cellarEntry.Quantity,
		HadBefore:     cellarEntry.HadBefore,
		Special:       cellarEntry.Special,
	}

	if cellarEntry.Location != nil {
		cellarBeer.Location = LocationFromModel(*cellarEntry.Location)
	}

	if cellarEntry.Format != nil {
		cellarBeer.Format = FormatFromModel(*cellarEntry.Format)
	}

	if cellarEntry.Vintage != nil {
		cellarBeer.Vintage = pointy.Uint64(*cellarEntry.Vintage)
	}

	if cellarEntry.DateAdded != nil {
		cellarBeer.DateAdded = timestamppb.New(*cellarEntry.DateAdded)
	}

	if cellarEntry.DrinkBefore != nil {
		cellarBeer.DrinkBefore = timestamppb.New(*cellarEntry.DrinkBefore)
	}

	if cellarEntry.CellarUntil != nil {
		cellarBeer.CellarUntil = timestamppb.New(*cellarEntry.CellarUntil)
	}

	if len(cellarEntry.Tags) > 0 {
		cellarBeer.Tags = TagsFromModel(cellarEntry.Tags)
	}

	return &cellarBeer
}

func TagsFromModel(tags []model.Tag) []string {
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Tag)
	}

	return tagNames
}

func UserFromModel(user model.User) *api.User {
	pbUser := api.User{
		Id:       user.UUID.String(),
		UserName: user.Username,
		Email:    user.Email,
	}

	if user.UntappdUserName != nil {
		pbUser.UntappedUsername = pointy.String(*user.UntappdUserName)
	}

	return &pbUser
}

func LocationsFromModel(locations []model.LocationInCellar) []*api.LocationInCellar {
	pbLocations := make([]*api.LocationInCellar, 0, len(locations))

	for _, location := range locations {
		pbLocation := LocationFromModel(location)
		pbLocations = append(pbLocations, pbLocation)
	}

	return pbLocations
}

func LocationFromModel(location model.LocationInCellar) *api.LocationInCellar {
	return &api.LocationInCellar{
		LocationId: uint64(location.ID),
		Name:       location.Name,
	}
}

func FormatFromModel(format model.BeerFormat) *api.BeerFormat {
	return &api.BeerFormat{
		FormatId:     uint64(format.ID),
		PackageType:  format.Package,
		MetricSize:   format.SizeMetric,
		ImperialSize: format.SizeImperial,
	}
}

func FormatsFromModel(formats []*model.BeerFormat) []*api.BeerFormat {
	pbFormats := make([]*api.BeerFormat, 0, len(formats))

	for _, format := range formats {
		pbFormat := FormatFromModel(*format)
		pbFormats = append(pbFormats, pbFormat)
	}

	return pbFormats
}

func CellarStatsFromModel(stats *model.CellarStats) *api.CellarStats {
	return &api.CellarStats{
		CellarId:      uint64(stats.CellarID),
		BeerCount:     stats.BeerCount,
		UniqueCount:   stats.UniqueCount,
		TotalVolume:   stats.TotalVolume,
		BreweryCount:  stats.BreweryCount,
		UntriedCount:  stats.UntriedCount,
		AverageAbv:    stats.AverageABV,
		AverageRating: stats.AverageRating,
		SpecialCount:  stats.SpecialCount,
	}
}

func BreweriesFromModel(breweries []*model.Brewery) []*api.Brewery {
	pbBreweries := make([]*api.Brewery, 0, len(breweries))

	for index := range breweries {
		brewery := breweries[index]
		pbBrewery := api.Brewery{Id: uint64(brewery.ID), Name: brewery.Name}
		pbBreweries = append(pbBreweries, &pbBrewery)
	}

	return pbBreweries
}

func StylesFromModel(styles []*model.BeerStyle) []*api.BeerStyle {
	pbStyles := make([]*api.BeerStyle, 0, len(styles))

	for index := range styles {
		style := styles[index]
		pbStyle := api.BeerStyle{Id: uint64(style.ID), Name: style.Name}
		pbStyles = append(pbStyles, &pbStyle)
	}

	return pbStyles
}

func AdventCalendarBeersFromModel(beers []model.AdventCalendarBeer) []*api.AdventCalendarBeer {
	pbBeers := make([]*api.AdventCalendarBeer, 0, len(beers))
	for _, beer := range beers {
		pbBeer := api.AdventCalendarBeer{
			Beer:     CellarBeerFromModel(&beer.CellarEntry),
			Day:      timestamppb.New(beer.Day),
			Revealed: beer.Revealed,
		}
		pbBeers = append(pbBeers, &pbBeer)
	}

	return pbBeers
}

//nolint:cyclop,funlen // this many ifs required for optional fields
func CellarFilterFromModel(filter *model.AdventCalendarFilter) *api.CellarFilter {
	pbFilter := api.CellarFilter{}

	if filter == nil {
		return &pbFilter
	}

	if filter.BreweryID != nil {
		pbFilter.BreweryId = pointy.Uint64(*filter.BreweryID)
	}

	if filter.MinimumAbv != nil {
		pbFilter.MinimumAbv = pointy.Float64(*filter.MinimumAbv)
	}

	if filter.MaximumAbv != nil {
		pbFilter.MaximumAbv = pointy.Float64(*filter.MaximumAbv)
	}

	if filter.StyleID != nil {
		pbFilter.StyleId = pointy.Uint64(*filter.StyleID)
	}

	if filter.MinimumVintage != nil {
		pbFilter.MinimumVintage = pointy.Uint64(*filter.MinimumVintage)
	}

	if filter.MaximumVintage != nil {
		pbFilter.MaximumVintage = pointy.Uint64(*filter.MaximumVintage)
	}

	if filter.OverdueToDrink != nil {
		pbFilter.OverdueToDrink = pointy.Bool(*filter.OverdueToDrink)
	}

	if filter.HadBefore != nil {
		pbFilter.HadBefore = pointy.Bool(*filter.HadBefore)
	}

	if filter.Special != nil {
		pbFilter.Special = pointy.Bool(*filter.Special)
	}

	if filter.MinimumQuantity != nil {
		pbFilter.MinimumQuantity = pointy.Int64(*filter.MinimumQuantity)
	}

	if filter.MinimumSize != nil {
		pbFilter.MinimumSize = pointy.Int64(*filter.MinimumSize)
	}

	if filter.MaximumSize != nil {
		pbFilter.MaximumSize = pointy.Int64(*filter.MaximumSize)
	}

	if filter.MinimumRating != nil {
		pbFilter.MinimumRating = pointy.Float64(*filter.MinimumRating)
	}

	if filter.MaximumRating != nil {
		pbFilter.MaximumRating = pointy.Float64(*filter.MaximumRating)
	}

	if filter.AddedBefore != nil {
		pbFilter.AddedBefore = timestamppb.New(*filter.AddedBefore)
	}

	if len(filter.Tags) > 0 {
		pbFilter.Tags = TagsFromModel(filter.Tags)
	}

	return &pbFilter
}
