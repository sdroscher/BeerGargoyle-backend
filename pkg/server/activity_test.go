package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

type ActivityTestSuite struct {
	suite.Suite
	activityRepo *mocks.ActivityRepository
	cellarRepo   *mocks.CellarRepository
	service      *server.CellarServer
}

func TestActivityTestSuite(t *testing.T) {
	suite.Run(t, new(ActivityTestSuite))
}

func (suite *ActivityTestSuite) SetupTest() {
	suite.activityRepo = mocks.NewActivityRepository(suite.T())
	suite.cellarRepo = mocks.NewCellarRepository(suite.T())
	suite.service = server.NewCellarServer(suite.cellarRepo, nil, nil, suite.activityRepo, zap.NewNop())
}

func (suite *ActivityTestSuite) TestRecordConsumption_Success() {
	ctx := context.Background()
	format := &model.BeerFormat{Package: "Can", SizeMetric: 473}
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 4,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
		Format:   format,
	}
	entryID := uint(10)
	activity := &model.Activity{
		Model:         gorm.Model{ID: 1},
		CellarID:      1,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      2,
	}
	activityMatcher := mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 2
	})
	entryAfterDecrement := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 2,
		Beer:     entry.Beer,
		Format:   format,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, entryAfterDecrement, activityMatcher).Return(activity, nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      2,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Consumption)
	suite.Equal(apiv1.ActivityEventType_ACTIVITY_EVENT_TYPE_BEER_CONSUMED, resp.Msg.Consumption.Type)
	suite.Equal(int64(2), resp.Msg.Consumption.Quantity)
	suite.NotNil(resp.Msg.Consumption.CellarBeer)
}

func (suite *ActivityTestSuite) TestRecordConsumption_LastBottle() {
	ctx := context.Background()
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 1,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}
	entryID := uint(10)
	activity := &model.Activity{
		Model:         gorm.Model{ID: 1},
		CellarID:      1,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      1,
	}

	activityMatcher := mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 1
	})
	entryAfterDecrement := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 0,
		Beer:     entry.Beer,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, entryAfterDecrement, activityMatcher).Return(activity, nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      1,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Consumption)
	suite.NotNil(resp.Msg.Consumption.CellarBeer)
}

func (suite *ActivityTestSuite) TestGetActivityFeed_Basic() {
	ctx := context.Background()
	entryID := uint(10)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 3,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}
	activities := []*model.Activity{
		{
			Model:         gorm.Model{ID: 1},
			CellarID:      1,
			CellarEntryID: &entryID,
			ActivityType:  model.ActivityTypeBeerConsumed,
			Quantity:      1,
			CellarEntry:   entry,
		},
	}

	suite.activityRepo.EXPECT().GetFeed(ctx, uint(1), (*time.Time)(nil), (*time.Time)(nil), 25, 0).
		Return(activities, int64(1), nil)

	resp, err := suite.service.GetActivityFeed(ctx, connect.NewRequest(&apiv1.GetActivityFeedRequest{
		CellarId: 1,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(1), resp.Msg.Total)
	suite.False(resp.Msg.HasMore)
	suite.Len(resp.Msg.Events, 1)
	suite.Equal(apiv1.ActivityEventType_ACTIVITY_EVENT_TYPE_BEER_CONSUMED, resp.Msg.Events[0].Type)
}

func (suite *ActivityTestSuite) TestGetActivityFeed_Pagination() {
	ctx := context.Background()
	entryID := uint(10)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}

	activities := make([]*model.Activity, 0, 5)

	for range 5 {
		activities = append(activities, &model.Activity{
			CellarID:      1,
			CellarEntryID: &entryID,
			ActivityType:  model.ActivityTypeBeerConsumed,
			Quantity:      1,
			CellarEntry:   entry,
		})
	}

	suite.activityRepo.EXPECT().GetFeed(ctx, uint(1), (*time.Time)(nil), (*time.Time)(nil), 5, 5).
		Return(activities, int64(15), nil)

	resp, err := suite.service.GetActivityFeed(ctx, connect.NewRequest(&apiv1.GetActivityFeedRequest{
		CellarId: 1,
		PageSize: 5,
		Page:     2,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(15), resp.Msg.Total)
	suite.True(resp.Msg.HasMore)
	suite.Len(resp.Msg.Events, 5)
}

func (suite *ActivityTestSuite) TestRecordConsumption_ConsumeEntryFails() {
	ctx := context.Background()
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 3,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}},
	}
	dbErr := errors.New("db error")

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, mock.Anything, mock.Anything).Return(nil, dbErr)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      1,
	}))

	suite.Require().Error(err)
	suite.Nil(resp)
}

func (suite *ActivityTestSuite) TestRecordConsumption_ExceedsQuantity() {
	ctx := context.Background()
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 2,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}},
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      5,
	}))

	suite.Require().Error(err)
	suite.Require().ErrorIs(err, server.ErrInvalidQuantity)
	suite.Nil(resp)
}

func (suite *ActivityTestSuite) TestGetYearInReview_ExplicitYear() {
	ctx := context.Background()
	yir := &model.YearInReview{
		Year:          2024,
		CellarID:      1,
		BeersConsumed: 42,
		UniqueBeers:   15,
		TotalVolumeMl: 19866,
		AverageRating: 4.1,
		BeersAdded:    60,
		TopStyles: []model.NameCount{
			{Name: "American IPA", Count: 18},
			{Name: "Imperial Stout", Count: 10},
		},
		TopCategories: []model.NameCount{
			{Name: "IPA", Count: 28},
			{Name: "Dark Beer", Count: 10},
		},
		TopBreweries: []model.NameCount{
			{Name: "Boombox", Count: 20},
		},
		ByMonth: []model.MonthlyCount{
			{Month: 1, Count: 3},
			{Month: 6, Count: 12},
		},
	}

	suite.activityRepo.EXPECT().GetYearInReview(ctx, uint(1), 2024).Return(yir, nil)

	resp, err := suite.service.GetYearInReview(ctx, connect.NewRequest(&apiv1.GetYearInReviewRequest{
		CellarId: 1,
		Year:     2024,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(2024), resp.Msg.YearInReview.Year)
	suite.Equal(int64(42), resp.Msg.YearInReview.BeersConsumed)
	suite.Equal(int64(15), resp.Msg.YearInReview.UniqueBeers)
	suite.Equal(int64(60), resp.Msg.YearInReview.BeersAdded)
	suite.InDelta(4.1, resp.Msg.YearInReview.AverageRating, 0.001)
	suite.Len(resp.Msg.YearInReview.TopStyles, 2)
	suite.Equal("American IPA", resp.Msg.YearInReview.TopStyles[0].Name)
	suite.Len(resp.Msg.YearInReview.TopCategories, 2)
	suite.Equal("IPA", resp.Msg.YearInReview.TopCategories[0].Name)
	suite.Len(resp.Msg.YearInReview.TopBreweries, 1)
	suite.Len(resp.Msg.YearInReview.ByMonth, 2)
}

func (suite *ActivityTestSuite) TestGetYearInReview_DefaultsToCurrentYear() {
	ctx := context.Background()
	currentYear := time.Now().UTC().Year()

	suite.activityRepo.EXPECT().GetYearInReview(ctx, uint(1), currentYear).
		Return(&model.YearInReview{Year: currentYear, CellarID: 1}, nil)

	resp, err := suite.service.GetYearInReview(ctx, connect.NewRequest(&apiv1.GetYearInReviewRequest{
		CellarId: 1,
	}))

	suite.Require().NoError(err)
	suite.Greater(resp.Msg.YearInReview.Year, int32(2000))
}
