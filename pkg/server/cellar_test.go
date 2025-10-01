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
