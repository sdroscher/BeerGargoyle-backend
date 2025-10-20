package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/mocks"
	"droscher.com/BeerGargoyle/pkg/auth"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
	apiv1 "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
)

type CellarTestSuite struct {
	suite.Suite
	cellarRepo   *mocks.CellarRepository
	service      *server.CellarServer
	observedLogs *observer.ObservedLogs
}

func TestCellarTestSuite(t *testing.T) {
	suite.Run(t, new(CellarTestSuite))
}

func (suite *CellarTestSuite) SetupTest() {
	suite.cellarRepo = mocks.NewCellarRepository(suite.T())
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	suite.observedLogs = observedLogs
	observedLogger := zap.New(observedZapCore)
	suite.service = server.NewCellarServer(suite.cellarRepo, nil, nil, observedLogger)
}

func (suite *CellarTestSuite) TestCreateAdventCalendar_ErrorMissingFilters() {
	request := &apiv1.CreateAdventCalendarRequest{
		CellarId:    1,
		Name:        "Test",
		Description: "Test",
		StartDate:   timestamppb.Now(),
		EndDate:     timestamppb.New(time.Now().AddDate(0, 0, 1)),
		Filters:     nil,
	}
	adventCalendar, err := suite.service.CreateAdventCalendar(context.Background(), &connect.Request[apiv1.CreateAdventCalendarRequest]{Msg: request})
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
	ctx := context.Background()

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
	ctx := context.Background()
	cellarEntry := &model.CellarEntry{Model: gorm.Model{ID: 1}, CellarID: 1}

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
	ctx := context.Background()
	cellarEntry1 := &model.CellarEntry{Model: gorm.Model{ID: 1}, CellarID: 1}
	cellarEntry2 := &model.CellarEntry{Model: gorm.Model{ID: 2}, CellarID: 1}

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
	ctx := context.Background()
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
	ctx := context.Background()

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	request := &apiv1.GetCellarRequest{CellarId: 999}
	result, err := suite.service.GetCellar(ctx, &connect.Request[apiv1.GetCellarRequest]{Msg: request})

	suite.Require().ErrorIs(err, gorm.ErrRecordNotFound)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestGetCellarEntry_Success() {
	ctx := context.Background()
	expectedEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		BeerID:   100,
		Quantity: 2,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(expectedEntry, nil)

	request := &apiv1.GetCellarEntryRequest{CellarEntryId: 10}
	result, err := suite.service.GetCellarEntry(ctx, &connect.Request[apiv1.GetCellarEntryRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	entry := result.Msg.GetEntry()
	suite.Equal(uint64(10), entry.GetCellarEntryId())
}

func (suite *CellarTestSuite) TestGetCellarStats_Success() {
	ctx := context.Background()
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
	ctx := context.Background()
	expectedBeers := []*model.CellarEntry{
		{Model: gorm.Model{ID: 1}, CellarID: 1, BeerID: 100, Quantity: 2},
		{Model: gorm.Model{ID: 2}, CellarID: 1, BeerID: 200, Quantity: 1},
	}

	suite.cellarRepo.EXPECT().GetCellarBeers(ctx, uint(1)).Return(expectedBeers, nil)

	request := &apiv1.ListCellarBeersRequest{CellarId: 1}
	result, err := suite.service.ListCellarBeers(ctx, &connect.Request[apiv1.ListCellarBeersRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	beers := result.Msg.GetBeers()
	suite.Len(beers, 2)
}

func (suite *CellarTestSuite) TestUpdateBeer_DeleteWhenQuantityZero() {
	ctx := context.Background()
	request := &apiv1.UpdateBeerRequest{
		CellarEntryId: 10,
		Quantity:      pointy.Int64(0),
	}

	suite.cellarRepo.EXPECT().DeleteCellarEntry(ctx, uint(10)).Return(nil)

	result, err := suite.service.UpdateBeer(ctx, &connect.Request[apiv1.UpdateBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Nil(result.Msg.GetBeer())
}

func (suite *CellarTestSuite) TestUpdateBeer_Success() {
	ctx := context.Background()
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
	ctx := context.Background()
	filter := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(5.0)}
	candidates := []*model.CellarEntry{
		{Model: gorm.Model{ID: 1}, CellarID: 1, BeerID: 100},
		{Model: gorm.Model{ID: 2}, CellarID: 1, BeerID: 200},
	}

	request := &apiv1.RecommendBeerRequest{
		CellarId: 1,
		Filter:   filter,
	}

	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter).Return(candidates, nil)

	result, err := suite.service.RecommendBeer(ctx, &connect.Request[apiv1.RecommendBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	recommendation := result.Msg.GetRecommendation()
	suite.NotNil(recommendation)
}

func (suite *CellarTestSuite) TestRecommendBeer_NoCandidates() {
	ctx := context.Background()
	filter := &apiv1.CellarFilter{MinimumAbv: pointy.Float64(20.0)}

	request := &apiv1.RecommendBeerRequest{
		CellarId: 1,
		Filter:   filter,
	}

	suite.cellarRepo.EXPECT().FindBeerRecommendations(ctx, uint64(1), filter).Return([]*model.CellarEntry{}, nil)

	result, err := suite.service.RecommendBeer(ctx, &connect.Request[apiv1.RecommendBeerRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Nil(result.Msg.GetRecommendation())
}

func (suite *CellarTestSuite) TestGetCellarRecommendationParams_Success() {
	ctx := context.Background()
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
	ctx := context.Background()
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

	suite.cellarRepo.EXPECT().GetAdventCalendarByID(ctx, uint64(1), uint64(1)).Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal(uint64(1), calendar.GetId())
	suite.Equal("Test Calendar", calendar.GetName())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_ByDate() {
	ctx := context.Background()
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

	suite.cellarRepo.EXPECT().GetAdventCalendarForDate(ctx, uint64(1), testDate).Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal(uint64(1), calendar.GetId())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_ByName() {
	ctx := context.Background()
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

	suite.cellarRepo.EXPECT().GetAdventCalendarByName(ctx, uint64(1), "Christmas Calendar").Return(expectedCalendar, nil)

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
	calendar := result.Msg.GetAdventCalendar()
	suite.Equal("Christmas Calendar", calendar.GetName())
}

func (suite *CellarTestSuite) TestGetAdventCalendar_InvalidCriteria() {
	ctx := context.Background()

	request := &apiv1.GetAdventCalendarRequest{
		CellarId: 1,
		Criteria: nil,
	}

	result, err := suite.service.GetAdventCalendar(ctx, &connect.Request[apiv1.GetAdventCalendarRequest]{Msg: request})

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Nil(result)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendar_Success() {
	ctx := context.Background()
	revealDay := time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC)
	expectedDay := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC) // Truncated to day

	request := &apiv1.UpdateAdventCalendarRequest{
		CellarId:  1,
		Id:        1,
		RevealDay: timestamppb.New(revealDay),
	}

	suite.cellarRepo.EXPECT().UpdateAdventCalendar(ctx, uint64(1), uint64(1), expectedDay).Return(nil)

	result, err := suite.service.UpdateAdventCalendar(ctx, &connect.Request[apiv1.UpdateAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
}

func (suite *CellarTestSuite) TestDeleteAdventCalendar_Success() {
	ctx := context.Background()

	request := &apiv1.DeleteAdventCalendarRequest{
		CellarId: 1,
		Id:       1,
	}

	suite.cellarRepo.EXPECT().DeleteAdventCalendar(ctx, uint64(1), uint64(1)).Return(nil)

	result, err := suite.service.DeleteAdventCalendar(ctx, &connect.Request[apiv1.DeleteAdventCalendarRequest]{Msg: request})

	suite.Require().NoError(err)
	suite.NotNil(result)
}

func (suite *CellarTestSuite) TestRegenerateAdventCalendarDay_Success() {
	ctx := context.Background()
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
