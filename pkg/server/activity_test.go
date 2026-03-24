package server_test

import (
	"context"
	"testing"

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
	updatedEntry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 2,
		Beer:     entry.Beer,
		Format:   format,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.activityRepo.EXPECT().CreateActivity(ctx, mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 2
	})).Return(activity, nil)
	suite.cellarRepo.EXPECT().UpdateCellarEntry(ctx, entry).Return(updatedEntry, nil)

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

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.activityRepo.EXPECT().CreateActivity(ctx, mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 1
	})).Return(activity, nil)
	suite.cellarRepo.EXPECT().DeleteCellarEntry(ctx, uint(10)).Return(nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      1,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Consumption)
	suite.NotNil(resp.Msg.Consumption.CellarBeer)
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
