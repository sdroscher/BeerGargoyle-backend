package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/auth"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

// Shared test constants (goconst: repeated literals across the server_test package).
const (
	dbDownMessage = "db down"
	basementName  = "Basement"
	bobName       = "bob"
	bobEmail      = "bob@example.com"
	bobUntappd    = "bob_untappd"
	nobodyEmail   = "nobody@example.com"
	fridgeName    = "Fridge"
	shelfName     = "Shelf"
	tagSour       = "sour"
	tagGift       = "gift"
)

// ctxWithUser returns a context containing a minimal authenticated user with the given ID.
func ctxWithUser(userID uint) context.Context {
	return context.WithValue(context.Background(), auth.UserKey{}, &model.User{
		Model: gorm.Model{ID: userID},
	})
}

// ownedCellar returns a minimal Cellar with ID=1 owned by user 1.
func ownedCellar() *model.Cellar {
	return &model.Cellar{Model: gorm.Model{ID: 1}, OwnerID: 1}
}

type CellarTestSuite struct {
	suite.Suite
	cellarRepo   *mocks.CellarRepository
	beerRepo     *mocks.BeerRepository
	activityRepo *mocks.ActivityRepository
	service      *server.CellarServer
	observedLogs *observer.ObservedLogs
}

func TestCellarTestSuite(t *testing.T) {
	suite.Run(t, new(CellarTestSuite))
}

func (suite *CellarTestSuite) SetupTest() {
	suite.cellarRepo = mocks.NewCellarRepository(suite.T())
	suite.beerRepo = mocks.NewBeerRepository(suite.T())
	suite.activityRepo = mocks.NewActivityRepository(suite.T())
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	suite.observedLogs = observedLogs
	observedLogger := zap.New(observedZapCore)
	suite.service = server.NewCellarServer(suite.cellarRepo, suite.beerRepo, suite.activityRepo, observedLogger)
}

func (suite *CellarTestSuite) TestCreateAdventCalendar_ErrorMissingFilters() {
	ctx := ctxWithUser(1)
	request := &apiv1.CreateAdventCalendarRequest{
		CellarId:    1,
		Name:        "Test",
		Description: "Test",
		StartDate:   timestamppb.Now(),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 0, 1)),
		Filters:     nil,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	adventCalendar, err := suite.service.CreateAdventCalendar(ctx, &connect.Request[apiv1.CreateAdventCalendarRequest]{Msg: request})
	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Require().ErrorContains(err, "there must be a filter for each day in the calendar")
	suite.Nil(adventCalendar)
}

func (suite *CellarTestSuite) TestCreateAdventCalendar_NoCandidatesError() {
	filter1 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(8)}
	filter2 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(9)}
	request := &apiv1.CreateAdventCalendarRequest{
		CellarId:    1,
		Name:        "Test",
		Description: "Test",
		StartDate:   timestamppb.Now(),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 0, 1)),
		Filters:     []*apiv1.CellarFilter{filter1, filter2},
	}
	ctx := ctxWithUser(1)

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter1).Return(nil, nil)

	adventCalendar, err := suite.service.CreateAdventCalendar(ctx, &connect.Request[apiv1.CreateAdventCalendarRequest]{Msg: request})
	suite.Require().ErrorIs(err, server.ErrCannotCreate)
	suite.Require().ErrorContains(err, "no candidates found")
	suite.Nil(adventCalendar)
}

func (suite *CellarTestSuite) TestCreateAdventCalendar_NoUniqueEntriesError() {
	filter1 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(8)}
	filter2 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(9)}
	request := &apiv1.CreateAdventCalendarRequest{
		CellarId:    1,
		Name:        "Test",
		Description: "Test",
		StartDate:   timestamppb.Now(),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 0, 1)),
		Filters:     []*apiv1.CellarFilter{filter1, filter2},
	}
	ctx := ctxWithUser(1)
	cellarEntry := &model.CellarEntry{Model: gorm.Model{ID: 1}, CellarID: 1}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter1).Return([]*model.CellarEntry{cellarEntry}, nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter2).Return([]*model.CellarEntry{cellarEntry}, nil)

	adventCalendar, err := suite.service.CreateAdventCalendar(ctx, &connect.Request[apiv1.CreateAdventCalendarRequest]{Msg: request})
	suite.Require().ErrorIs(err, server.ErrCannotCreate)
	suite.Require().ErrorContains(err, "no unique candidate found")
	suite.Nil(adventCalendar)
}

func (suite *CellarTestSuite) TestCreateAdventCalendar_Success() {
	filter1 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(8)}
	filter2 := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(9)}
	request := &apiv1.CreateAdventCalendarRequest{
		CellarId:    1,
		Name:        "Test",
		Description: "Test",
		StartDate:   timestamppb.Now(),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 0, 1)),
		Filters:     []*apiv1.CellarFilter{filter1, filter2},
	}
	ctx := ctxWithUser(1)
	cellarEntry1 := &model.CellarEntry{Model: gorm.Model{ID: 1}, CellarID: 1}
	cellarEntry2 := &model.CellarEntry{Model: gorm.Model{ID: 2}, CellarID: 1}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter1).Return([]*model.CellarEntry{cellarEntry1}, nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter2).Return([]*model.CellarEntry{cellarEntry1, cellarEntry2}, nil)
	suite.cellarRepo.EXPECT().SaveAdventCalendar(ctx, mock.Anything).Return(&model.AdventCalendar{Model: gorm.Model{ID: 10}}, nil)
	result, err := suite.service.CreateAdventCalendar(ctx, &connect.Request[apiv1.CreateAdventCalendarRequest]{Msg: request})
	suite.Require().NoError(err)
	suite.NotNil(result)
	adventCalendar := result.Msg.GetAdventCalendar()
	suite.Len(adventCalendar.GetBeers(), 2)
	suite.Equal(uint64(1), adventCalendar.GetBeers()[0].GetBeer().GetCellarEntryId())
	suite.Equal(uint64(2), adventCalendar.GetBeers()[1].GetBeer().GetCellarEntryId())
}

func (suite *CellarTestSuite) TestGetCellarList_Success() {
	ctx := context.WithValue(context.Background(), auth.UserKey{}, &model.User{
		Model:    gorm.Model{ID: 1},
		Username: "testuser",
	})

	expectedCellars := []*model.Cellar{
		{
			Model:       gorm.Model{ID: 1},
			Name:        "Cellar 1",
			Description: "First cellar",
			OwnerID:     1,
		},
		{
			Model:       gorm.Model{ID: 2},
			Name:        "Cellar 2",
			Description: "Second cellar",
			OwnerID:     1,
		},
	}

	suite.cellarRepo.EXPECT().GetCellarsForUser(ctx, model.User{Model: gorm.Model{ID: 1}, Username: "testuser"}).Return(expectedCellars, nil)

	result, err := suite.service.GetCellarList(ctx, &connect.Request[apiv1.GetCellarListRequest]{})

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Len(result.Msg.GetCellars(), 2)
}

func (suite *CellarTestSuite) TestGetCellarList_NoUserInContext() {
	ctx := context.Background() // No user in context

	result, err := suite.service.GetCellarList(ctx, &connect.Request[apiv1.GetCellarListRequest]{})

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Nil(result)
	suite.ErrorContains(err, "no user in context")
}

func (suite *CellarTestSuite) TestGetCellar_Success() {
	ctx := ctxWithUser(1)
	expectedCellar := &model.Cellar{
		Model:       gorm.Model{ID: 1},
		Name:        "Test Cellar",
		Description: "A test cellar",
		OwnerID:     1,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(expectedCellar, nil)

	request := &apiv1.GetCellarRequest{CellarId: 1}
	result, err := suite.service.GetCellar(ctx, &connect.Request[apiv1.GetCellarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	cellar := result.Msg.GetCellar()
	suite.Equal("Test Cellar", cellar.GetName())
}

func (suite *CellarTestSuite) TestGetCellar_NotFound() {
	ctx := ctxWithUser(1)

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	request := &apiv1.GetCellarRequest{CellarId: 999}
	result, err := suite.service.GetCellar(ctx, &connect.Request[apiv1.GetCellarRequest]{Msg: request})

	suite.Require().ErrorIs(err, server.ErrCellarNotFound)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellar_Unauthorized() {
	// User 2 tries to access a cellar owned by user 1.
	ctx := ctxWithUser(2)
	cellar := &model.Cellar{Model: gorm.Model{ID: 1}, OwnerID: 1}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(cellar, nil)

	request := &apiv1.GetCellarRequest{CellarId: 1}
	result, err := suite.service.GetCellar(ctx, &connect.Request[apiv1.GetCellarRequest]{Msg: request})

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarEntry_Success() {
	ctx := ctxWithUser(1)
	expectedEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		BeerID:   100,
		Quantity: 2,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(expectedEntry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	request := &apiv1.GetCellarEntryRequest{CellarEntryId: 10}
	result, err := suite.service.GetCellarEntry(ctx, &connect.Request[apiv1.GetCellarEntryRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	entry := result.Msg.GetEntry()
	suite.Equal(uint64(10), entry.GetCellarEntryId())
}

func (suite *CellarTestSuite) TestGetCellarStats_Success() {
	ctx := ctxWithUser(1)
	expectedStats := &model.CellarStats{
		CellarID:      1,
		BeerCount:     10,
		UniqueCount:   8,
		TotalVolume:   3300.0,
		BreweryCount:  5,
		UntriedCount:  3,
		SpecialCount:  2,
		AverageABV:    7.5,
		AverageRating: 4.2,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarStats(ctx, uint(1)).Return(expectedStats, nil)

	request := &apiv1.GetCellarStatsRequest{CellarId: 1}
	result, err := suite.service.GetCellarStats(ctx, &connect.Request[apiv1.GetCellarStatsRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	stats := result.Msg.GetCellarStats()
	suite.Equal(uint64(10), stats.GetBeerCount())
	suite.Equal(uint64(8), stats.GetUniqueCount())
	suite.InDelta(3300.0, stats.GetTotalVolume(), 0.1)
}

func (suite *CellarTestSuite) TestListCellarBeers_Success() {
	ctx := ctxWithUser(1)
	expectedBeers := []*model.CellarEntry{
		{Model: gorm.Model{ID: 1}, CellarID: 1, BeerID: 100, Quantity: 2},
		{Model: gorm.Model{ID: 2}, CellarID: 1, BeerID: 200, Quantity: 1},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBeers(ctx, uint(1)).Return(expectedBeers, nil)

	request := &apiv1.ListCellarBeersRequest{CellarId: 1}
	result, err := suite.service.ListCellarBeers(ctx, &connect.Request[apiv1.ListCellarBeersRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	beers := result.Msg.GetBeers()
	suite.Len(beers, 2)
}

func (suite *CellarTestSuite) TestUpdateBeer_DeleteWhenQuantityZero() {
	ctx := ctxWithUser(1)
	request := &apiv1.UpdateBeerRequest{
		CellarEntryId: 10,
		Quantity:      pointy.Int64(0),
	}
	existingEntry := &model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(existingEntry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().DeleteCellarEntry(ctx, uint(10)).Return(nil)

	result, err := suite.service.UpdateBeer(ctx, &connect.Request[apiv1.UpdateBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Nil(result.Msg.GetBeer())
}

func (suite *CellarTestSuite) TestUpdateBeer_Success() {
	ctx := ctxWithUser(1)
	existingEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		BeerID:   100,
		Quantity: 1,
	}

	updatedEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		BeerID:   100,
		Quantity: 3,
	}

	request := &apiv1.UpdateBeerRequest{
		CellarEntryId: 10,
		Quantity:      pointy.Int64(3),
		Vintage:       pointy.Uint64(2021),
		HadBefore:     pointy.Bool(true),
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(existingEntry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().UpdateCellarEntry(ctx, mock.MatchedBy(func(entry *model.CellarEntry) bool {
		return entry.ID == 10 && entry.Quantity == 3
	})).Return(updatedEntry, nil)

	result, err := suite.service.UpdateBeer(ctx, &connect.Request[apiv1.UpdateBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	beer := result.Msg.GetBeer()
	suite.Equal(uint64(10), beer.GetCellarEntryId())
}

func (suite *CellarTestSuite) TestRecommendBeer_Success() {
	ctx := ctxWithUser(1)
	filter := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(5.0)}
	candidates := []*model.CellarEntry{
		{Model: gorm.Model{ID: 1}, CellarID: 1, BeerID: 100},
		{Model: gorm.Model{ID: 2}, CellarID: 1, BeerID: 200},
	}

	request := &apiv1.RecommendBeerRequest{
		CellarId: 1,
		Filter:   filter,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter).Return(candidates, nil)

	result, err := suite.service.RecommendBeer(ctx, &connect.Request[apiv1.RecommendBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	recommendation := result.Msg.GetRecommendation()
	suite.NotNil(recommendation)
}

func (suite *CellarTestSuite) TestRecommendBeer_NoCandidates() {
	ctx := ctxWithUser(1)
	filter := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(20.0)}

	request := &apiv1.RecommendBeerRequest{
		CellarId: 1,
		Filter:   filter,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter).Return([]*model.CellarEntry{}, nil)

	result, err := suite.service.RecommendBeer(ctx, &connect.Request[apiv1.RecommendBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Nil(result.Msg.GetRecommendation())
}

func (suite *CellarTestSuite) TestGetCellarRecommendationParams_Success() {
	ctx := ctxWithUser(1)
	expectedBreweries := []*model.Brewery{
		{Model: gorm.Model{ID: 1}, Name: "Brewery A"},
		{Model: gorm.Model{ID: 2}, Name: "Brewery B"},
	}
	expectedStyles := []*model.BeerStyle{
		{Model: gorm.Model{ID: 1}, Name: "IPA"},
		{Model: gorm.Model{ID: 2}, Name: "Stout"},
	}
	expectedRanges := &model.CellarRecommendationRanges{
		MinimumAbv:      3.5,
		MaximumAbv:      12.0,
		MinimumSize:     330,
		MaximumSize:     750,
		MinimumVintage:  2018,
		MaximumVintage:  2023,
		MinimumRating:   3.0,
		MaximumRating:   5.0,
		OldestAddedDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	request := &apiv1.GetCellarRecommendationParamsRequest{CellarId: 1}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBreweryNames(ctx, uint64(1)).Return(expectedBreweries, nil)
	suite.cellarRepo.EXPECT().GetCellarStyles(ctx, uint64(1)).Return(expectedStyles, nil)
	suite.cellarRepo.EXPECT().GetCellarRecommendationRanges(ctx, uint64(1)).Return(expectedRanges, nil)

	result, err := suite.service.GetCellarRecommendationParams(ctx, &connect.Request[apiv1.GetCellarRecommendationParamsRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	params := result.Msg
	suite.Len(params.GetBreweries(), 2)
	suite.Len(params.GetStyles(), 2)
	suite.InDelta(3.5, params.GetMinimumAbv(), 0.1)
	suite.InDelta(12.0, params.GetMaximumAbv(), 0.1)
}

func (suite *CellarTestSuite) TestGetAdventCalendar_ByID() {
	ctx := ctxWithUser(1)
	expectedCalendar := &model.AdventCalendar{
		Model:       gorm.Model{ID: 1},
		CellarID:    1,
		Name:        "Test Calendar",
		Description: "A test advent calendar",
		StartDate:   time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
	}

	request := &apiv1.GetAdventCalendarRequest{
		CellarId: 1,
		Criteria: &apiv1.GetAdventCalendarRequest_Id{Id: 1},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarByID(ctx, uint64(1), uint64(1)).Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal(uint64(1), calendar.GetId())
	suite.Equal("Test Calendar", calendar.GetName())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_ByDate() {
	ctx := ctxWithUser(1)
	testDate := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)
	expectedCalendar := &model.AdventCalendar{
		Model:       gorm.Model{ID: 1},
		CellarID:    1,
		Name:        "Test Calendar",
		Description: "A test advent calendar",
		StartDate:   time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
	}

	request := &apiv1.GetAdventCalendarRequest{
		CellarId: 1,
		Criteria: &apiv1.GetAdventCalendarRequest_ForDate{ForDate: timestamppb.New(testDate)},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarForDate(ctx, uint64(1), testDate).Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal(uint64(1), calendar.GetId())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_ByName() {
	ctx := ctxWithUser(1)
	expectedCalendar := &model.AdventCalendar{
		Model:       gorm.Model{ID: 1},
		CellarID:    1,
		Name:        "Christmas Calendar",
		Description: "A test advent calendar",
		StartDate:   time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC),
	}

	request := &apiv1.GetAdventCalendarRequest{
		CellarId: 1,
		Criteria: &apiv1.GetAdventCalendarRequest_Name{Name: "Christmas Calendar"},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarByName(ctx, uint64(1), "Christmas Calendar").Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal("Christmas Calendar", calendar.GetName())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_InvalidCriteria() {
	ctx := ctxWithUser(1)

	request := &apiv1.GetAdventCalendarRequest{
		CellarId: 1,
		Criteria: nil,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendar_Success() {
	ctx := ctxWithUser(1)
	revealDay := time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC)
	expectedDay := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC) // Truncated to day

	request := &apiv1.UpdateAdventCalendarRequest{
		CellarId:  1,
		Id:        1,
		RevealDay: timestamppb.New(revealDay),
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().UpdateAdventCalendar(ctx, uint64(1), uint64(1), expectedDay).Return(nil)

	result, err := suite.service.UpdateAdventCalendar(ctx, &connect.Request[apiv1.UpdateAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
}

func (suite *CellarTestSuite) TestDeleteAdventCalendar_Success() {
	ctx := ctxWithUser(1)

	request := &apiv1.DeleteAdventCalendarRequest{
		CellarId: 1,
		Id:       1,
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().DeleteAdventCalendar(ctx, uint64(1), uint64(1)).Return(nil)

	result, err := suite.service.DeleteAdventCalendar(ctx, &connect.Request[apiv1.DeleteAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
}

func (suite *CellarTestSuite) TestRegenerateAdventCalendarDay_Success() {
	ctx := ctxWithUser(1)
	day := time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC)
	expectedDay := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC) // Truncated to day

	adventCalendar := &model.AdventCalendar{
		Model:    gorm.Model{ID: 1},
		CellarID: 1,
		Beers: []model.AdventCalendarBeer{
			{CellarEntryID: 10},
			{CellarEntryID: 20},
		},
	}

	filter := &model.AdventCalendarFilter{
		Model:      gorm.Model{ID: 1},
		MinimumAbv: pointy.Float64(5.0),
	}

	candidates := []*model.CellarEntry{
		{Model: gorm.Model{ID: 30}, CellarID: 1, BeerID: 300},
	}

	request := &apiv1.RegenerateAdventCalendarDayRequest{
		CellarId:         1,
		AdventCalendarId: 1,
		Day:              timestamppb.New(day),
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarByID(ctx, uint64(1), uint64(1)).Return(adventCalendar, nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarFilter(ctx, uint64(1), uint64(1), expectedDay).Return(filter, nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), mock.Anything).Return(candidates, nil)
	suite.cellarRepo.EXPECT().UpdateAdventCalendarEntry(ctx, uint64(1), uint64(1), expectedDay, uint64(30)).Return(nil)

	result, err := suite.service.RegenerateAdventCalendarDay(ctx, &connect.Request[apiv1.RegenerateAdventCalendarDayRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	beer := result.Msg.GetBeer()
	suite.NotNil(beer)
	suite.Equal(uint64(30), beer.GetBeer().GetCellarEntryId())
}

func (suite *CellarTestSuite) TestAddCellarBeer_CreatesActivity() {
	ctx := ctxWithUser(1)
	cellarID := uint64(1)
	beerID := uint64(5)

	cellar := &model.Cellar{Model: gorm.Model{ID: 1}, OwnerID: 1}
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		BeerID:   5,
		Quantity: 3,
	}
	fullEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 3,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(cellarID)).Return(cellar, nil)
	suite.cellarRepo.EXPECT().AddBeerToCellarWithActivity(ctx, mock.AnythingOfType("model.CellarEntry"), mock.AnythingOfType("time.Time")).Return(entry, nil)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(fullEntry, nil)

	resp, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{
		CellarId: cellarID,
		BeerId:   beerID,
		Quantity: 3,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Beer)
}

func ctxWithFullUser(userUUID uuid.UUID) (context.Context, *model.User) {
	user := &model.User{
		Model:           gorm.Model{ID: 1},
		UUID:            userUUID,
		Username:        bobName,
		Email:           bobEmail,
		UntappdUserName: pointy.String(bobUntappd),
	}

	return context.WithValue(context.Background(), auth.UserKey{}, user), user
}

func (suite *CellarTestSuite) TestAddCellar_Success() {
	ownerUUID := uuid.New()
	ctx, user := ctxWithFullUser(ownerUUID)

	suite.cellarRepo.EXPECT().AddCellar(ctx, basementName, "Cool and dark", []string{fridgeName, shelfName}, *user).Return(&model.Cellar{
		Model: gorm.Model{ID: 7},
		Locations: []model.LocationInCellar{
			{Model: gorm.Model{ID: 1}, Name: fridgeName},
			{Model: gorm.Model{ID: 2}, Name: shelfName},
		},
	}, nil)

	result, err := suite.service.AddCellar(ctx, connect.NewRequest(&apiv1.AddCellarRequest{
		OwnerUuid:   ownerUUID.String(),
		Name:        basementName,
		Description: "Cool and dark",
		Locations:   []string{fridgeName, shelfName},
	}))

	suite.Require().NoError(err)
	cellar := result.Msg.GetCellar()
	suite.Equal(uint64(7), cellar.GetCellarId())
	suite.Equal(basementName, cellar.GetName())
	suite.Equal(bobName, cellar.GetOwner().GetUserName())
	suite.Equal(bobUntappd, cellar.GetOwner().GetUntappedUsername())
	suite.Require().Len(cellar.GetLocations(), 2)
	suite.Equal(shelfName, cellar.GetLocations()[1].GetName())
}

func (suite *CellarTestSuite) TestAddCellar_NoUserInContext() {
	result, err := suite.service.AddCellar(context.Background(), connect.NewRequest(&apiv1.AddCellarRequest{OwnerUuid: uuid.NewString()}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestAddCellar_InvalidOwnerUUID() {
	ctx, _ := ctxWithFullUser(uuid.New())

	result, err := suite.service.AddCellar(ctx, connect.NewRequest(&apiv1.AddCellarRequest{OwnerUuid: "not-a-uuid"}))

	suite.Require().Error(err)
	suite.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestAddCellar_OwnerUUIDMismatch() {
	ctx, _ := ctxWithFullUser(uuid.New())

	result, err := suite.service.AddCellar(ctx, connect.NewRequest(&apiv1.AddCellarRequest{OwnerUuid: uuid.NewString()}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestAddCellar_RepositoryError() {
	ownerUUID := uuid.New()
	ctx, user := ctxWithFullUser(ownerUUID)
	suite.cellarRepo.EXPECT().AddCellar(ctx, basementName, "", []string(nil), *user).Return(nil, errors.New("duplicate cellar"))

	result, err := suite.service.AddCellar(ctx, connect.NewRequest(&apiv1.AddCellarRequest{OwnerUuid: ownerUUID.String(), Name: basementName}))

	suite.Require().ErrorContains(err, "duplicate cellar")
	suite.Nil(result)
}

// captureAddedEntry stubs the cellar ownership check and the save, returning a pointer that is
// populated with the entry (and timestamp) the handler passed to AddBeerToCellarWithActivity.
func (suite *CellarTestSuite) captureAddedEntry(ctx context.Context) (*model.CellarEntry, *time.Time) {
	var (
		captured   model.CellarEntry
		capturedAt time.Time
	)

	saved := &model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().AddBeerToCellarWithActivity(ctx,
		mock.MatchedBy(func(entry model.CellarEntry) bool {
			captured = entry

			return true
		}),
		mock.MatchedBy(func(occurredAt time.Time) bool {
			capturedAt = occurredAt

			return true
		}),
	).Return(saved, nil)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(saved, nil)

	return &captured, &capturedAt
}

func (suite *CellarTestSuite) TestAddCellarBeer_MapsAllOptionalFields() {
	ctx := ctxWithUser(1)
	captured, capturedAt := suite.captureAddedEntry(ctx)
	drinkBefore := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	cellarUntil := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	dateAdded := time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)

	_, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{
		CellarId:    1,
		BeerId:      5,
		LocationId:  3,
		Quantity:    6,
		FormatId:    pointy.Uint64(2),
		Vintage:     pointy.Uint64(2020),
		HadBefore:   true,
		Special:     true,
		DrinkBefore: timestamppb.New(drinkBefore),
		CellarUntil: timestamppb.New(cellarUntil),
		DateAdded:   timestamppb.New(dateAdded),
	}))

	suite.Require().NoError(err)
	suite.Equal(uint(5), captured.BeerID)
	suite.Equal(int64(6), captured.Quantity)
	suite.Require().NotNil(captured.LocationID)
	suite.Equal(uint(3), *captured.LocationID)
	suite.Require().NotNil(captured.FormatID)
	suite.Equal(uint(2), *captured.FormatID)
	suite.Require().NotNil(captured.Vintage)
	suite.Equal(uint64(2020), *captured.Vintage)
	suite.True(captured.HadBefore)
	suite.True(captured.Special)
	suite.Equal(drinkBefore, *captured.DrinkBefore)
	suite.Equal(cellarUntil, *captured.CellarUntil)
	suite.Equal(dateAdded, *captured.DateAdded)
	suite.Equal(dateAdded, *capturedAt, "activity timestamp should be the supplied date_added")
}

func (suite *CellarTestSuite) TestAddCellarBeer_UnsetOptionalsStayNil() {
	ctx := ctxWithUser(1)
	captured, capturedAt := suite.captureAddedEntry(ctx)

	_, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{CellarId: 1, BeerId: 5, Quantity: 1}))

	suite.Require().NoError(err)
	suite.Nil(captured.LocationID, "location_id 0 means 'none', not a FK to row 0")
	suite.Nil(captured.FormatID)
	suite.Nil(captured.Vintage)
	suite.Nil(captured.DateAdded)
	suite.Nil(captured.DrinkBefore)
	suite.Nil(captured.CellarUntil)
	suite.WithinDuration(time.Now().UTC(), *capturedAt, 5*time.Second)
}

func (suite *CellarTestSuite) TestAddCellarBeer_ReusesExistingTagsAndCreatesNewOnes() {
	ctx := ctxWithUser(1)
	captured, _ := suite.captureAddedEntry(ctx)
	suite.beerRepo.EXPECT().GetTagsByNames(ctx, []string{tagSour, tagGift}).
		Return(map[string]model.Tag{tagSour: {Model: gorm.Model{ID: 4}, Tag: tagSour}}, nil)

	_, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{
		CellarId: 1, BeerId: 5, Quantity: 1, Tags: []string{tagSour, tagGift},
	}))

	suite.Require().NoError(err)
	suite.Require().Len(captured.Tags, 2)
	suite.Equal(uint(4), captured.Tags[0].ID)
	suite.Equal(tagGift, captured.Tags[1].Tag)
	suite.Zero(captured.Tags[1].ID)
}

func (suite *CellarTestSuite) TestAddCellarBeer_TagLookupErrorFallsBackToNewTags() {
	ctx := ctxWithUser(1)
	captured, _ := suite.captureAddedEntry(ctx)
	suite.beerRepo.EXPECT().GetTagsByNames(ctx, []string{tagSour}).Return(nil, errors.New(dbDownMessage))

	_, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{
		CellarId: 1, BeerId: 5, Quantity: 1, Tags: []string{tagSour},
	}))

	suite.Require().NoError(err)
	suite.Require().Len(captured.Tags, 1)
	suite.Equal(tagSour, captured.Tags[0].Tag)
	suite.Equal(1, suite.observedLogs.FilterMessage("error getting tags by name").Len())
}

func (suite *CellarTestSuite) TestAddCellarBeer_ReloadFailureFallsBackToSavedEntry() {
	ctx := ctxWithUser(1)
	saved := &model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1, Quantity: 2}
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().AddBeerToCellarWithActivity(ctx, mock.AnythingOfType("model.CellarEntry"), mock.AnythingOfType("time.Time")).Return(saved, nil)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(nil, errors.New("reload failed"))

	result, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{CellarId: 1, BeerId: 5, Quantity: 2}))

	suite.Require().NoError(err)
	suite.Equal(uint64(10), result.Msg.GetBeer().GetCellarEntryId())
	suite.Equal(1, suite.observedLogs.FilterMessage("error loading cellar entry after saving").Len())
}

func (suite *CellarTestSuite) TestAddCellarBeer_SaveError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().AddBeerToCellarWithActivity(ctx, mock.AnythingOfType("model.CellarEntry"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("fk violation"))

	result, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{CellarId: 1, BeerId: 5, Quantity: 1}))

	suite.Require().ErrorContains(err, "fk violation")
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestAddCellarBeer_NotOwner() {
	ctx := ctxWithUser(2)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	result, err := suite.service.AddCellarBeer(ctx, connect.NewRequest(&apiv1.AddCellarBeerRequest{CellarId: 1, BeerId: 5, Quantity: 1}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestRequireOwner_NoUserInContext() {
	result, err := suite.service.GetCellar(context.Background(), connect.NewRequest(&apiv1.GetCellarRequest{CellarId: 1}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestRequireOwner_RepositoryErrorPropagates() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.GetCellar(ctx, connect.NewRequest(&apiv1.GetCellarRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Require().NotErrorIs(err, server.ErrCellarNotFound)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarList_RepositoryError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarsForUser(ctx, model.User{Model: gorm.Model{ID: 1}}).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.GetCellarList(ctx, connect.NewRequest(&apiv1.GetCellarListRequest{}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarEntry_RepositoryError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(nil, errors.New("entry not found"))

	result, err := suite.service.GetCellarEntry(ctx, connect.NewRequest(&apiv1.GetCellarEntryRequest{CellarEntryId: 10}))

	suite.Require().Error(err)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarEntry_NotOwner() {
	ctx := ctxWithUser(2)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(&model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	result, err := suite.service.GetCellarEntry(ctx, connect.NewRequest(&apiv1.GetCellarEntryRequest{CellarEntryId: 10}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarStats_RepositoryError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarStats(ctx, uint(1)).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.GetCellarStats(ctx, connect.NewRequest(&apiv1.GetCellarStatsRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestListCellarBeers_RepositoryError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBeers(ctx, uint(1)).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.ListCellarBeers(ctx, connect.NewRequest(&apiv1.ListCellarBeersRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateBeer_EntryLookupError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(nil, errors.New("entry not found"))

	result, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{CellarEntryId: 10}))

	suite.Require().Error(err)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateBeer_NotOwner() {
	ctx := ctxWithUser(2)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(&model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	result, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{CellarEntryId: 10, Quantity: pointy.Int64(0)}))

	suite.Require().ErrorIs(err, server.ErrNotAuthorized)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateBeer_DeleteError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(&model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().DeleteCellarEntry(ctx, uint(10)).Return(errors.New(dbDownMessage))

	result, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{CellarEntryId: 10, Quantity: pointy.Int64(0)}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateBeer_UpdateError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(&model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1}, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().UpdateCellarEntry(ctx, mock.Anything).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{CellarEntryId: 10, Quantity: pointy.Int64(2)}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateBeer_AppliesEveryProvidedField() {
	ctx := ctxWithUser(1)
	existing := &model.CellarEntry{
		Model: gorm.Model{ID: 10}, CellarID: 1, Quantity: 1,
		Location: &model.LocationInCellar{Name: "old"}, Format: &model.BeerFormat{Package: "old"},
	}
	drinkBefore := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	cellarUntil := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	dateAdded := time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(existing, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.beerRepo.EXPECT().GetTagsByNames(ctx, []string{tagGift}).Return(map[string]model.Tag{}, nil)
	suite.cellarRepo.EXPECT().UpdateCellarEntry(ctx, existing).Return(existing, nil) // the handler mutates the entry in place

	_, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{
		CellarEntryId: 10,
		LocationId:    pointy.Uint64(4),
		FormatId:      pointy.Uint64(2),
		Vintage:       pointy.Uint64(2019),
		Quantity:      pointy.Int64(5),
		HadBefore:     pointy.Bool(true),
		Special:       pointy.Bool(true),
		DrinkBefore:   timestamppb.New(drinkBefore),
		CellarUntil:   timestamppb.New(cellarUntil),
		DateAdded:     timestamppb.New(dateAdded),
		Tags:          &apiv1.Tags{Tags: []string{tagGift}},
	}))

	suite.Require().NoError(err)
	suite.Require().NotNil(existing.LocationID)
	suite.Equal(uint(4), *existing.LocationID)
	suite.Nil(existing.Location, "stale preloaded location must be cleared")
	suite.Require().NotNil(existing.FormatID)
	suite.Equal(uint(2), *existing.FormatID)
	suite.Nil(existing.Format, "stale preloaded format must be cleared")
	suite.Require().NotNil(existing.Vintage)
	suite.Equal(uint64(2019), *existing.Vintage)
	suite.Equal(int64(5), existing.Quantity)
	suite.True(existing.HadBefore)
	suite.True(existing.Special)
	suite.True(existing.DrinkBefore.Equal(drinkBefore))
	suite.True(existing.CellarUntil.Equal(cellarUntil))
	suite.True(existing.DateAdded.Equal(dateAdded))
	suite.Require().Len(existing.Tags, 1)
	suite.Equal(tagGift, existing.Tags[0].Tag)
}

func (suite *CellarTestSuite) TestUpdateBeer_UnsetFieldsAreLeftAlone() {
	ctx := ctxWithUser(1)
	vintage := uint64(2018)
	existing := &model.CellarEntry{Model: gorm.Model{ID: 10}, CellarID: 1, Quantity: 3, Vintage: &vintage, HadBefore: true, Special: true}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(existing, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().UpdateCellarEntry(ctx, existing).Return(existing, nil)

	_, err := suite.service.UpdateBeer(ctx, connect.NewRequest(&apiv1.UpdateBeerRequest{CellarEntryId: 10}))

	suite.Require().NoError(err)
	suite.Equal(int64(3), existing.Quantity)
	suite.Equal(uint64(2018), *existing.Vintage)
	suite.True(existing.HadBefore)
	suite.True(existing.Special)
	suite.Nil(existing.Tags)
}

func (suite *CellarTestSuite) TestRecommendBeer_RepositoryError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), mock.Anything).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.RecommendBeer(ctx, connect.NewRequest(&apiv1.RecommendBeerRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarRecommendationParams_BreweriesError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBreweryNames(ctx, uint64(1)).Return(nil, errors.New("breweries failed"))

	result, err := suite.service.GetCellarRecommendationParams(ctx, connect.NewRequest(&apiv1.GetCellarRecommendationParamsRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, "breweries failed")
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarRecommendationParams_StylesError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBreweryNames(ctx, uint64(1)).Return(nil, nil)
	suite.cellarRepo.EXPECT().GetCellarStyles(ctx, uint64(1)).Return(nil, errors.New("styles failed"))

	result, err := suite.service.GetCellarRecommendationParams(ctx, connect.NewRequest(&apiv1.GetCellarRecommendationParamsRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, "styles failed")
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarRecommendationParams_RangesError() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetCellarBreweryNames(ctx, uint64(1)).Return(nil, nil)
	suite.cellarRepo.EXPECT().GetCellarStyles(ctx, uint64(1)).Return(nil, nil)
	suite.cellarRepo.EXPECT().GetCellarRecommendationRanges(ctx, uint64(1)).Return(nil, errors.New("ranges failed"))

	result, err := suite.service.GetCellarRecommendationParams(ctx, connect.NewRequest(&apiv1.GetCellarRecommendationParamsRequest{CellarId: 1}))

	suite.Require().ErrorContains(err, "ranges failed")
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestAdventCalendar_RepositoryErrors() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil).Times(3)
	suite.cellarRepo.EXPECT().UpdateAdventCalendar(ctx, uint64(1), uint64(2), mock.Anything).Return(errors.New("update failed"))
	suite.cellarRepo.EXPECT().DeleteAdventCalendar(ctx, uint64(1), uint64(2)).Return(errors.New("delete failed"))
	suite.cellarRepo.EXPECT().GetAdventCalendarByID(ctx, uint64(1), uint64(2)).Return(nil, errors.New("get failed"))

	_, err := suite.service.UpdateAdventCalendar(ctx, connect.NewRequest(&apiv1.UpdateAdventCalendarRequest{CellarId: 1, Id: 2, RevealDay: timestamppb.Now()}))
	suite.Require().ErrorContains(err, "update failed")

	_, err = suite.service.DeleteAdventCalendar(ctx, connect.NewRequest(&apiv1.DeleteAdventCalendarRequest{CellarId: 1, Id: 2}))
	suite.Require().ErrorContains(err, "delete failed")

	_, err = suite.service.GetAdventCalendar(ctx, connect.NewRequest(&apiv1.GetAdventCalendarRequest{CellarId: 1, Criteria: &apiv1.GetAdventCalendarRequest_Id{Id: 2}}))
	suite.Require().ErrorContains(err, "get failed")
}

func (suite *CellarTestSuite) TestRegenerateAdventCalendarDay_MissingDay() {
	ctx := ctxWithUser(1)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarByID(ctx, uint64(1), uint64(1)).Return(&model.AdventCalendar{Model: gorm.Model{ID: 1}}, nil)

	result, err := suite.service.RegenerateAdventCalendarDay(ctx, connect.NewRequest(&apiv1.RegenerateAdventCalendarDayRequest{CellarId: 1, AdventCalendarId: 1}))

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Nil(result)
}
