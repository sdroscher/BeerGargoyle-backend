package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"go.openly.dev/pointy"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"droscher.com/BeerGargoyle/pkg/auth"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
	"droscher.com/BeerGargoyle/pkg/server/grpc"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
	"droscher.com/BeerGargoyle/pkg/server/grpc/api/v1/apiv1connect"
)

type CellarServer struct {
	apiv1connect.UnimplementedCellarServiceHandler
	logger           *zap.Logger
	cellarRepository repository.CellarRepository
	beerRepository   beerRepository
	userRepository   userRepository
}

const (
	hoursPerDay = 24
)

var (
	ErrCellarNotFound = errors.New("cellar not found")
	ErrInvalidInput   = errors.New("bad request")
	ErrCannotCreate   = errors.New("cannot create advent calendar")
)

type userRepository interface {
	GetUserByUUID(ctx context.Context, uuid uuid.UUID) (*model.User, error)
}

type beerRepository interface {
	GetTagsByNames(ctx context.Context, names []string) (map[string]model.Tag, error)
}

func NewCellarServer(cellarRepo repository.CellarRepository, beerRepo beerRepository, userRepo userRepository, logger *zap.Logger) *CellarServer {
	return &CellarServer{cellarRepository: cellarRepo, beerRepository: beerRepo, userRepository: userRepo, logger: logger}
}

func (c *CellarServer) AddCellar(ctx context.Context, request *connect.Request[api.AddCellarRequest]) (*connect.Response[api.AddCellarResponse], error) {
	user, err := c.userRepository.GetUserByUUID(ctx, uuid.MustParse(request.Msg.GetOwnerUuid()))
	if err != nil {
		return nil, err
	}

	owner := api.User{
		Id:       user.UUID.String(),
		UserName: user.Username,
		Email:    user.Email,
	}

	if user.UntappdUserName != nil {
		owner.UntappedUsername = pointy.String(*user.UntappdUserName)
	}

	cellar, err := c.cellarRepository.AddCellar(ctx, request.Msg.GetName(), request.Msg.GetDescription(), request.Msg.GetLocations(), *user)
	if err != nil {
		return nil, err
	}

	pbCellar := api.Cellar{
		CellarId:    uint64(cellar.ID),
		Owner:       &owner,
		Name:        request.Msg.GetName(),
		Description: request.Msg.GetDescription(),
	}

	for _, location := range cellar.Locations {
		pbCellar.Locations = append(pbCellar.Locations, &api.LocationInCellar{
			LocationId: uint64(location.ID),
			Name:       location.Name,
		})
	}

	response := api.AddCellarResponse{Cellar: &pbCellar}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) GetCellarList(ctx context.Context, _ *connect.Request[api.GetCellarListRequest]) (*connect.Response[api.GetCellarListResponse], error) {
	user, ok := ctx.Value(auth.UserKey{}).(*model.User)
	if !ok {
		return nil, fmt.Errorf("%w: no user in context", ErrInvalidInput)
	}

	cellars, err := c.cellarRepository.GetCellarsForUser(ctx, *user)
	if err != nil {
		return nil, err
	}

	response := api.GetCellarListResponse{Cellars: grpc.CellarsFromModel(cellars)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) GetCellar(ctx context.Context, request *connect.Request[api.GetCellarRequest]) (*connect.Response[api.GetCellarResponse], error) {
	cellar, err := c.cellarRepository.GetCellarByID(ctx, uint(request.Msg.GetCellarId()))
	if err != nil {
		return nil, err
	}

	pbCellar := grpc.CellarFromModel(cellar)

	response := api.GetCellarResponse{Cellar: pbCellar}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) AddCellarBeer(ctx context.Context, request *connect.Request[api.AddCellarBeerRequest]) (*connect.Response[api.AddCellarBeerResponse], error) {
	cellar, err := c.cellarRepository.GetCellarByID(ctx, uint(request.Msg.GetCellarId()))
	if err != nil {
		return nil, err
	}

	if cellar == nil {
		return nil, fmt.Errorf("%w: id %d", ErrCellarNotFound, request.Msg.GetCellarId())
	}

	beer := model.CellarEntry{
		CellarID:   uint(request.Msg.GetCellarId()),
		BeerID:     uint(request.Msg.GetBeerId()),
		Quantity:   request.Msg.GetQuantity(),
		LocationID: pointy.Uint(uint(request.Msg.GetLocationId())),
		HadBefore:  request.Msg.GetHadBefore(),
		Special:    request.Msg.GetSpecial(),
	}

	if request.Msg.GetDrinkBefore() != nil {
		drinkBefore := request.Msg.GetDrinkBefore().AsTime()
		beer.DrinkBefore = &drinkBefore
	}

	if request.Msg.GetCellarUntil() != nil {
		cellarUntil := request.Msg.GetCellarUntil().AsTime()
		beer.CellarUntil = &cellarUntil
	}

	if request.Msg.GetFormatId() != 0 {
		beer.FormatID = pointy.Uint(uint(request.Msg.GetFormatId()))
	}

	if request.Msg.GetVintage() != 0 {
		beer.Vintage = pointy.Uint64(request.Msg.GetVintage())
	}

	if request.Msg.GetDateAdded() != nil {
		dateAdded := request.Msg.GetDateAdded().AsTime()
		beer.DateAdded = &dateAdded
	}

	if len(request.Msg.GetTags()) > 0 {
		tags := c.fetchTags(ctx, request.Msg.GetTags())
		beer.Tags = tags
	}

	cellarEntry, err := c.cellarRepository.AddBeerToCellar(ctx, beer)
	if err != nil {
		return nil, err
	}

	fullCellarEntry, err := c.cellarRepository.GetCellarEntryByID(ctx, cellarEntry.ID)
	if err != nil {
		c.logger.Error("error loading cellar entry after saving", zap.Uint("id", cellarEntry.ID), zap.String("beer", cellarEntry.Beer.Name), zap.Error(err))
		fullCellarEntry = cellarEntry
	}

	reply := api.AddCellarBeerResponse{Beer: grpc.CellarBeerFromModel(fullCellarEntry)}

	return connect.NewResponse(&reply), nil
}

func (c *CellarServer) fetchTags(ctx context.Context, requestTags []string) []model.Tag {
	tags := make([]model.Tag, 0, len(requestTags))

	tagsByName, err := c.beerRepository.GetTagsByNames(ctx, requestTags)
	if err != nil {
		c.logger.Error("error getting tags by name", zap.Error(err))

		tagsByName = map[string]model.Tag{}
	}

	for _, tagName := range requestTags {
		if tag, ok := tagsByName[tagName]; ok {
			tags = append(tags, tag)
		} else {
			tags = append(tags, model.Tag{Tag: tagName})
		}
	}

	return tags
}

func (c *CellarServer) GetCellarEntry(ctx context.Context, request *connect.Request[api.GetCellarEntryRequest]) (*connect.Response[api.GetCellarEntryResponse], error) {
	cellarEntry, err := c.cellarRepository.GetCellarEntryByID(ctx, uint(request.Msg.GetCellarEntryId()))
	if err != nil {
		return nil, err
	}

	response := api.GetCellarEntryResponse{Entry: grpc.CellarBeerFromModel(cellarEntry)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) GetCellarStats(ctx context.Context, request *connect.Request[api.GetCellarStatsRequest]) (*connect.Response[api.GetCellarStatsResponse], error) {
	stats, err := c.cellarRepository.GetCellarStats(ctx, uint(request.Msg.GetCellarId()))
	if err != nil {
		return nil, err
	}

	response := api.GetCellarStatsResponse{CellarStats: grpc.CellarStatsFromModel(stats)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) ListCellarBeers(ctx context.Context, request *connect.Request[api.ListCellarBeersRequest]) (*connect.Response[api.ListCellarBeersResponse], error) {
	beers, err := c.cellarRepository.GetCellarBeers(ctx, uint(request.Msg.GetCellarId()))
	if err != nil {
		return nil, err
	}

	response := api.ListCellarBeersResponse{Beers: grpc.CellarBeersFromModel(beers)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) UpdateBeer(ctx context.Context, request *connect.Request[api.UpdateBeerRequest]) (*connect.Response[api.UpdateBeerResponse], error) {
	if request.Msg.Quantity != nil && request.Msg.GetQuantity() == 0 {
		err := c.cellarRepository.DeleteCellarEntry(ctx, uint(request.Msg.GetCellarEntryId()))
		if err != nil {
			return nil, err
		}

		return connect.NewResponse(&api.UpdateBeerResponse{Beer: nil}), nil
	}

	cellarEntry, err := c.cellarRepository.GetCellarEntryByID(ctx, uint(request.Msg.GetCellarEntryId()))
	if err != nil {
		return nil, err
	}

	c.updateCellarEntry(ctx, request, cellarEntry)

	updatedEntry, err := c.cellarRepository.UpdateCellarEntry(ctx, cellarEntry)
	if err != nil {
		return nil, err
	}

	response := api.UpdateBeerResponse{Beer: grpc.CellarBeerFromModel(updatedEntry)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) updateCellarEntry(ctx context.Context, request *connect.Request[api.UpdateBeerRequest], cellarEntry *model.CellarEntry) {
	if request.Msg.GetLocationId() != 0 {
		cellarEntry.LocationID = pointy.Uint(uint(request.Msg.GetLocationId()))
		cellarEntry.Location = nil
	}

	if request.Msg.GetVintage() != 0 {
		cellarEntry.Vintage = pointy.Uint64(request.Msg.GetVintage())
	}

	if request.Msg.Quantity != nil {
		cellarEntry.Quantity = request.Msg.GetQuantity()
	}

	if request.Msg.GetFormatId() != 0 {
		cellarEntry.FormatID = pointy.Uint(uint(request.Msg.GetFormatId()))
		cellarEntry.Format = nil
	}

	if request.Msg.HadBefore != nil {
		cellarEntry.HadBefore = request.Msg.GetHadBefore()
	}

	if request.Msg.GetDrinkBefore() != nil {
		drinkBefore := request.Msg.GetDrinkBefore().AsTime()
		cellarEntry.DrinkBefore = &drinkBefore
	}

	if request.Msg.GetCellarUntil() != nil {
		cellarUntil := request.Msg.GetCellarUntil().AsTime()
		cellarEntry.CellarUntil = &cellarUntil
	}

	if request.Msg.Special != nil {
		cellarEntry.Special = request.Msg.GetSpecial()
	}

	if request.Msg.GetDateAdded() != nil {
		dateAdded := request.Msg.GetDateAdded().AsTime()
		cellarEntry.DateAdded = &dateAdded
	}

	if request.Msg.GetTags() != nil {
		cellarEntry.Tags = c.fetchTags(ctx, request.Msg.GetTags().GetTags())
	}
}

func (c *CellarServer) RecommendBeer(ctx context.Context, request *connect.Request[api.RecommendBeerRequest]) (*connect.Response[api.RecommendBeerResponse], error) {
	candidates, err := c.cellarRepository.FindBeerRecommendations(ctx, request.Msg.GetCellarId(), request.Msg.GetFilter())
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return connect.NewResponse(&api.RecommendBeerResponse{}), nil
	}

	recommendation := candidates[rand.Intn(len(candidates))] //nolint: gosec // we don't need crypto security here

	response := api.RecommendBeerResponse{Recommendation: grpc.CellarBeerFromModel(recommendation)}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) GetCellarRecommendationParams(ctx context.Context, request *connect.Request[api.GetCellarRecommendationParamsRequest]) (*connect.Response[api.GetCellarRecommendationParamsResponse], error) {
	breweries, err := c.cellarRepository.GetCellarBreweryNames(ctx, request.Msg.GetCellarId())
	if err != nil {
		return nil, err
	}

	styles, err := c.cellarRepository.GetCellarStyles(ctx, request.Msg.GetCellarId())
	if err != nil {
		return nil, err
	}

	params, err := c.cellarRepository.GetCellarRecommendationRanges(ctx, request.Msg.GetCellarId())
	if err != nil {
		return nil, err
	}

	response := api.GetCellarRecommendationParamsResponse{
		Breweries:       grpc.BreweriesFromModel(breweries),
		Styles:          grpc.StylesFromModel(styles),
		MinimumAbv:      params.MinimumAbv,
		MaximumAbv:      params.MaximumAbv,
		MinimumSize:     params.MinimumSize,
		MaximumSize:     params.MaximumSize,
		OldestVintage:   params.MinimumVintage,
		NewestVintage:   params.MaximumVintage,
		MinimumRating:   params.MinimumRating,
		MaximumRating:   params.MaximumRating,
		OldestAddedDate: timestamppb.New(params.OldestAddedDate),
	}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) CreateAdventCalendar(ctx context.Context, request *connect.Request[api.CreateAdventCalendarRequest]) (*connect.Response[api.CreateAdventCalendarResponse], error) {
	startDate := truncateToDay(request.Msg.GetStartDate().AsTime())
	endDate := truncateToDay(request.Msg.GetEndDate().AsTime())

	beers, pbBeers, err := c.createAdventCalendar(ctx, request.Msg.GetCellarId(), startDate, endDate, request.Msg.GetFilters())
	if err != nil {
		return nil, err
	}

	adventCalendar := model.AdventCalendar{
		CellarID:    uint(request.Msg.GetCellarId()),
		Name:        request.Msg.GetName(),
		Description: request.Msg.GetDescription(),
		StartDate:   startDate,
		EndDate:     endDate,
		Beers:       beers,
	}

	pbAdventCalendar := api.AdventCalendar{
		CellarId:    request.Msg.GetCellarId(),
		Name:        request.Msg.GetName(),
		Description: request.Msg.GetDescription(),
		StartDate:   timestamppb.New(startDate),
		EndDate:     timestamppb.New(endDate),
		Beers:       pbBeers,
	}

	savedCalendar, err := c.cellarRepository.SaveAdventCalendar(ctx, adventCalendar)
	if err != nil {
		return nil, err
	}

	pbAdventCalendar.Id = uint64(savedCalendar.ID)
	response := api.CreateAdventCalendarResponse{
		AdventCalendar: &pbAdventCalendar,
	}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) createAdventCalendar(ctx context.Context, cellarID uint64, startDate time.Time, endDate time.Time, filters []*api.CellarFilter) ([]model.AdventCalendarBeer, []*api.AdventCalendarBeer, error) {
	days := int(math.Round(endDate.Sub(startDate).Hours()/hoursPerDay)) + 1
	if len(filters) != days {
		return nil, nil, fmt.Errorf("%w: there must be a filter for each day in the calendar", ErrInvalidInput)
	}

	index := 0
	beers := make([]model.AdventCalendarBeer, 0, days)
	pbBeers := make([]*api.AdventCalendarBeer, 0, days)
	beerMap := make(map[uint64]struct{}, days)

	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		recommendation, err := c.uniqueRecommendation(ctx, cellarID, filters[index], beerMap)
		if err != nil {
			return nil, nil, err
		}

		beerMap[recommendation.GetCellarEntryId()] = struct{}{}

		beer := model.AdventCalendarBeer{
			CellarEntryID: uint(recommendation.GetCellarEntryId()),
			Day:           day,
			Revealed:      false,
		}

		pbBeer := api.AdventCalendarBeer{
			Beer:     recommendation,
			Day:      timestamppb.New(day),
			Revealed: false,
		}

		beers = append(beers, beer)
		pbBeers = append(pbBeers, &pbBeer)

		index++
	}

	return beers, pbBeers, nil
}

func (c *CellarServer) uniqueRecommendation(ctx context.Context, cellarID uint64, filter *api.CellarFilter, beerMap map[uint64]struct{}) (*api.CellarBeer, error) {
	var result *api.CellarBeer

	candidates, err := c.cellarRepository.FindBeerRecommendations(ctx, cellarID, filter)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: no candidates found for filter %v", ErrCannotCreate, filter)
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	for index := 0; index < len(candidates) && result == nil; index++ {
		beer := grpc.CellarBeerFromModel(candidates[index])
		if _, found := beerMap[beer.GetCellarEntryId()]; !found {
			result = beer
		}
	}

	if result == nil {
		return nil, fmt.Errorf("%w: no unique candidate found for filter %v", ErrCannotCreate, filter)
	}

	return result, nil
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (c *CellarServer) GetAdventCalendar(ctx context.Context, request *connect.Request[api.GetAdventCalendarRequest]) (*connect.Response[api.GetAdventCalendarResponse], error) {
	var (
		calendar *model.AdventCalendar
		err      error
	)

	switch requestType := request.Msg.GetCriteria().(type) {
	case *api.GetAdventCalendarRequest_Id:
		calendar, err = c.cellarRepository.GetAdventCalendarByID(ctx, request.Msg.GetCellarId(), requestType.Id)
	case *api.GetAdventCalendarRequest_ForDate:
		calendar, err = c.cellarRepository.GetAdventCalendarForDate(ctx, request.Msg.GetCellarId(), requestType.ForDate.AsTime())
	case *api.GetAdventCalendarRequest_Name:
		calendar, err = c.cellarRepository.GetAdventCalendarByName(ctx, request.Msg.GetCellarId(), requestType.Name)
	default:
		err = fmt.Errorf("%w: no handler for %v", ErrInvalidInput, requestType)
	}

	if err != nil {
		return nil, err
	}

	pbAdventCalendar := api.AdventCalendar{
		Id:          uint64(calendar.ID),
		CellarId:    request.Msg.GetCellarId(),
		Name:        calendar.Name,
		Description: calendar.Description,
		StartDate:   timestamppb.New(calendar.StartDate),
		EndDate:     timestamppb.New(calendar.EndDate),
		Beers:       grpc.AdventCalendarBeersFromModel(calendar.Beers),
	}

	response := api.GetAdventCalendarResponse{AdventCalendar: &pbAdventCalendar}

	return connect.NewResponse(&response), nil
}

func (c *CellarServer) UpdateAdventCalendar(ctx context.Context, request *connect.Request[api.UpdateAdventCalendarRequest]) (*connect.Response[api.UpdateAdventCalendarResponse], error) {
	day := truncateToDay(request.Msg.GetRevealDay().AsTime())

	err := c.cellarRepository.UpdateAdventCalendar(ctx, request.Msg.GetCellarId(), request.Msg.GetId(), day)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.UpdateAdventCalendarResponse{}), nil
}

func (c *CellarServer) DeleteAdventCalendar(ctx context.Context, request *connect.Request[api.DeleteAdventCalendarRequest]) (*connect.Response[api.DeleteAdventCalendarResponse], error) {
	err := c.cellarRepository.DeleteAdventCalendar(ctx, request.Msg.GetCellarId(), request.Msg.GetId())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.DeleteAdventCalendarResponse{}), nil
}

func (c *CellarServer) RegenerateAdventCalendarDay(ctx context.Context, request *connect.Request[api.RegenerateAdventCalendarDayRequest]) (*connect.Response[api.RegenerateAdventCalendarDayResponse], error) {
	adventCalendar, err := c.cellarRepository.GetAdventCalendarByID(ctx, request.Msg.GetCellarId(), request.Msg.GetAdventCalendarId())
	if err != nil {
		return nil, err
	}

	if request.Msg.Day == nil {
		return nil, fmt.Errorf("%w: day must be set", ErrInvalidInput)
	}

	day := truncateToDay(request.Msg.GetDay().AsTime())

	filter, err := c.cellarRepository.GetAdventCalendarFilter(ctx, request.Msg.GetCellarId(), request.Msg.GetAdventCalendarId(), day)
	if err != nil {
		return nil, err
	}

	beerMap := make(map[uint64]struct{}, len(adventCalendar.Beers))

	for _, beer := range adventCalendar.Beers {
		beerMap[uint64(beer.CellarEntryID)] = struct{}{}
	}

	recommendation, err := c.uniqueRecommendation(ctx, request.Msg.GetCellarId(), grpc.CellarFilterFromModel(filter), beerMap)
	if err != nil {
		return nil, err
	}

	err = c.cellarRepository.UpdateAdventCalendarEntry(ctx, request.Msg.GetCellarId(), request.Msg.GetAdventCalendarId(), day, recommendation.GetCellarEntryId())
	if err != nil {
		return nil, err
	}

	beer := api.AdventCalendarBeer{
		Beer:     recommendation,
		Day:      timestamppb.New(day),
		Revealed: false,
	}

	return connect.NewResponse(&api.RegenerateAdventCalendarDayResponse{Beer: &beer}), nil
}
