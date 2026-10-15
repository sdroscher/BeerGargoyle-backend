package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/configs"
	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/integrations"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

const fakeIntegration = "fake"

type BeerServerTestSuite struct {
	suite.Suite
	beerRepo     *mocks.BeerRepository
	integration  *mocks.Integration
	config       *configs.Config
	observedLogs *observer.ObservedLogs
	service      *server.BeerServer
}

func TestBeerServerTestSuite(t *testing.T) {
	suite.Run(t, new(BeerServerTestSuite))
}

func (suite *BeerServerTestSuite) SetupTest() {
	suite.beerRepo = mocks.NewBeerRepository(suite.T())
	suite.integration = mocks.NewIntegration(suite.T())
	suite.config = &configs.Config{Integrations: configs.Integrations{Beer: []string{fakeIntegration}}}

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	suite.observedLogs = observedLogs

	resolver := func(name string, _ *configs.Config, _ *zap.Logger) integrations.Integration {
		if name == fakeIntegration {
			return suite.integration
		}

		return nil
	}

	suite.service = server.NewBeerServer(suite.beerRepo, zap.New(observedZapCore), suite.config, server.WithIntegrationResolver(resolver))
}

func (suite *BeerServerTestSuite) TestGetBeerFormats_Success() {
	ctx := context.Background()
	formats := []*model.BeerFormat{
		{Model: gorm.Model{ID: 1}, Package: "Bottle", SizeMetric: 330},
		{Model: gorm.Model{ID: 2}, Package: "Can", SizeMetric: 473},
	}
	suite.beerRepo.EXPECT().GetBeerFormats(ctx).Return(formats, nil)

	result, err := suite.service.GetBeerFormats(ctx, connect.NewRequest(&apiv1.GetBeerFormatsRequest{}))

	suite.Require().NoError(err)
	suite.Len(result.Msg.GetFormats(), 2)
	suite.Equal("Bottle", result.Msg.GetFormats()[0].GetPackageType())
}

func (suite *BeerServerTestSuite) TestGetBeerFormats_RepositoryError() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().GetBeerFormats(ctx).Return(nil, errors.New("db down"))

	result, err := suite.service.GetBeerFormats(ctx, connect.NewRequest(&apiv1.GetBeerFormatsRequest{}))

	suite.Require().ErrorContains(err, "db down")
	suite.Nil(result)
}
