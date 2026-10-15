package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/configs"
	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/integrations"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
	"droscher.com/BeerGargoyle/pkg/server"
)

const (
	fakeIntegration = "fake"
	ipaQuery        = "ipa"
	hazyIPA         = "Hazy IPA"
	breweryA        = "Brew A"
	goseStyle       = "Gose"
	saltyBeer       = "Salty"
)

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
	suite.beerRepo.EXPECT().GetBeerFormats(ctx).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.GetBeerFormats(ctx, connect.NewRequest(&apiv1.GetBeerFormatsRequest{}))

	suite.Require().ErrorContains(err, "db down")
	suite.Nil(result)
}

func (suite *BeerServerTestSuite) TestFindBeer_Success() {
	suite.integration.EXPECT().FindBeer(ipaQuery).Return([]model.Beer{
		{Name: hazyIPA, Brewery: model.Brewery{Name: "Brew A"}},
		{Name: "West Coast IPA", Brewery: model.Brewery{Name: "Brew B"}},
	}, nil)

	result, err := suite.service.FindBeer(context.Background(), connect.NewRequest(&apiv1.FindBeerRequest{Query: ipaQuery}))

	suite.Require().NoError(err)
	suite.Require().Len(result.Msg.GetBeers(), 2)
	suite.Equal("Hazy IPA", result.Msg.GetBeers()[0].GetName())
}

func (suite *BeerServerTestSuite) TestFindBeer_IntegrationErrorIsLoggedAndSkipped() {
	suite.integration.EXPECT().FindBeer(ipaQuery).Return(nil, errors.New("algolia down"))

	result, err := suite.service.FindBeer(context.Background(), connect.NewRequest(&apiv1.FindBeerRequest{Query: ipaQuery}))

	suite.Require().NoError(err)
	suite.Empty(result.Msg.GetBeers())
	suite.Equal(1, suite.observedLogs.FilterMessage("failed beer search").Len())
}

func (suite *BeerServerTestSuite) TestFindBeer_UnknownIntegrationIsSkipped() {
	suite.config.Integrations.Beer = []string{"typo", fakeIntegration}
	suite.integration.EXPECT().FindBeer(ipaQuery).Return([]model.Beer{{Name: hazyIPA}}, nil)

	result, err := suite.service.FindBeer(context.Background(), connect.NewRequest(&apiv1.FindBeerRequest{Query: ipaQuery}))

	suite.Require().NoError(err)
	suite.Len(result.Msg.GetBeers(), 1)
	suite.Equal(1, suite.observedLogs.FilterMessage("unknown beer integration").Len())
}

func (suite *BeerServerTestSuite) TestFindBeer_NoIntegrationsConfigured() {
	suite.config.Integrations.Beer = nil

	result, err := suite.service.FindBeer(context.Background(), connect.NewRequest(&apiv1.FindBeerRequest{Query: ipaQuery}))

	suite.Require().NoError(err)
	suite.Empty(result.Msg.GetBeers())
}

func addBeerRequest(beer *apiv1.Beer) *connect.Request[apiv1.AddBeerRequest] {
	return connect.NewRequest(&apiv1.AddBeerRequest{Beer: beer})
}

func (suite *BeerServerTestSuite) TestAddBeer_ExistingBreweryAndStyleIDs() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Name == hazyIPA && beer.BreweryID == 7 && beer.StyleID == 3
	})).Return(&model.Beer{Model: gorm.Model{ID: 99}, Name: hazyIPA}, nil)

	result, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:    hazyIPA,
		Brewery: &apiv1.Brewery{Id: 7},
		Style:   &apiv1.BeerStyle{Id: 3},
	}))

	suite.Require().NoError(err)
	suite.Equal(uint64(99), result.Msg.GetBeer().GetId())
}

func (suite *BeerServerTestSuite) TestAddBeer_NilBeerIsInvalidArgument() {
	result, err := suite.service.AddBeer(context.Background(), connect.NewRequest(&apiv1.AddBeerRequest{}))

	suite.Require().Error(err)
	suite.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *BeerServerTestSuite) TestAddBeer_LooksUpExistingBreweryByExternalSource() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().FindBreweryByExternalSource(ctx, uint64(55), "untappd_web").
		Return(&model.Brewery{Model: gorm.Model{ID: 12}}, nil)
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.BreweryID == 12
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:    hazyIPA,
		Brewery: &apiv1.Brewery{ExternalId: pointy.Uint64(55), ExternalSource: pointy.String("untappd_web")},
	}))

	suite.Require().NoError(err)
}

func (suite *BeerServerTestSuite) TestAddBeer_NewBreweryIsEnriched() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().FindBreweryByExternalSource(ctx, uint64(55), fakeIntegration).Return(nil, repository.ErrBreweryNotFound)
	suite.integration.EXPECT().GetBreweryDetails(uint64(55)).Return(model.Brewery{
		Description:    "A fine brewery",
		ExternalRating: pointy.Float64(4.1),
		Address:        model.Address{Locality: "Vancouver"},
		ImageURL:       "http://img/brewery.png",
	}, nil)
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.BreweryID == 0 &&
			beer.Brewery.Name == breweryA &&
			beer.Brewery.Description == "A fine brewery" &&
			beer.Brewery.Address.Locality == "Vancouver" &&
			beer.Brewery.ImageURL == "http://img/brewery.png" &&
			beer.Brewery.ExternalRating != nil
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:    hazyIPA,
		Brewery: &apiv1.Brewery{Name: breweryA, ExternalId: pointy.Uint64(55), ExternalSource: pointy.String(fakeIntegration)},
	}))

	suite.Require().NoError(err)
	suite.Equal(1, suite.observedLogs.FilterMessage("brewery not found").Len())
}

func (suite *BeerServerTestSuite) TestAddBeer_BreweryLookupErrorStillSaves() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().FindBreweryByExternalSource(ctx, uint64(55), "other").Return(nil, errors.New(dbDownMessage))
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Brewery.Name == breweryA
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:    hazyIPA,
		Brewery: &apiv1.Brewery{Name: breweryA, ExternalId: pointy.Uint64(55), ExternalSource: pointy.String("other")},
	}))

	suite.Require().NoError(err)
	suite.Equal(1, suite.observedLogs.FilterMessage("error looking for brewery").Len())
}

func (suite *BeerServerTestSuite) TestAddBeer_BreweryEnrichmentFailureStillSaves() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().FindBreweryByExternalSource(ctx, uint64(55), fakeIntegration).Return(nil, repository.ErrBreweryNotFound)
	suite.integration.EXPECT().GetBreweryDetails(uint64(55)).Return(model.Brewery{}, errors.New("cloudflare challenge"))
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Brewery.Name == breweryA && beer.Brewery.Description == ""
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:    hazyIPA,
		Brewery: &apiv1.Brewery{Name: breweryA, ExternalId: pointy.Uint64(55), ExternalSource: pointy.String(fakeIntegration)},
	}))

	suite.Require().NoError(err)
	suite.Equal(1, suite.observedLogs.FilterMessage("failed to fetch brewery details for save").Len())
}

func (suite *BeerServerTestSuite) TestAddBeer_NewStyleDefaultsBJCPToOther() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeerStyle(ctx, model.BeerStyle{Name: goseStyle, BJCPStyleID: "OTHER"}).
		Return(&model.BeerStyle{Model: gorm.Model{ID: 21}}, nil)
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.StyleID == 21
	})).Return(&model.Beer{Name: saltyBeer}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{Name: saltyBeer, Style: &apiv1.BeerStyle{Name: goseStyle}}))

	suite.Require().NoError(err)
}

func (suite *BeerServerTestSuite) TestAddBeer_NewStyleKeepsBJCPID() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeerStyle(ctx, model.BeerStyle{Name: goseStyle, BJCPStyleID: "23G"}).
		Return(&model.BeerStyle{Model: gorm.Model{ID: 21}}, nil)
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.Anything).Return(&model.Beer{Name: saltyBeer}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name:  saltyBeer,
		Style: &apiv1.BeerStyle{Name: goseStyle, BjcpStyle: &apiv1.BeerStyleBJCP{Id: "23G"}},
	}))

	suite.Require().NoError(err)
}

func (suite *BeerServerTestSuite) TestAddBeer_StyleErrorAborts() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeerStyle(ctx, mock.Anything).Return(nil, errors.New("style insert failed"))

	result, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{Name: saltyBeer, Style: &apiv1.BeerStyle{Name: goseStyle}}))

	suite.Require().ErrorContains(err, "style insert failed")
	suite.Nil(result)
}

func (suite *BeerServerTestSuite) TestAddBeer_FetchesMissingDescription() {
	ctx := context.Background()
	suite.integration.EXPECT().GetBeerDescription(uint64(8)).Return("Juicy and hazy", nil)
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Description == "Juicy and hazy"
	})).Return(&model.Beer{Name: hazyIPA, Description: "Juicy and hazy"}, nil)

	result, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name: hazyIPA, ExternalId: pointy.Uint64(8), ExternalSource: pointy.String(fakeIntegration),
	}))

	suite.Require().NoError(err)
	suite.Equal("Juicy and hazy", result.Msg.GetBeer().GetDescription())
}

func (suite *BeerServerTestSuite) TestAddBeer_ExistingDescriptionNotOverwritten() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Description == "Mine"
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name: hazyIPA, Description: "Mine", ExternalId: pointy.Uint64(8), ExternalSource: pointy.String(fakeIntegration),
	}))

	suite.Require().NoError(err) // no GetBeerDescription expectation: the mock fails the test if it is called
}

func (suite *BeerServerTestSuite) TestAddBeer_DescriptionFailureStillSaves() {
	ctx := context.Background()
	suite.integration.EXPECT().GetBeerDescription(uint64(8)).Return("", errors.New("cloudflare challenge"))
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.MatchedBy(func(beer model.Beer) bool {
		return beer.Description == ""
	})).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name: hazyIPA, ExternalId: pointy.Uint64(8), ExternalSource: pointy.String(fakeIntegration),
	}))

	suite.Require().NoError(err)
	suite.Equal(1, suite.observedLogs.FilterMessage("failed to fetch beer description for save").Len())
}

func (suite *BeerServerTestSuite) TestAddBeer_UnknownSourceSkipsEnrichment() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.Anything).Return(&model.Beer{Name: hazyIPA}, nil)

	_, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{
		Name: hazyIPA, ExternalId: pointy.Uint64(8), ExternalSource: pointy.String("typo"),
	}))

	suite.Require().NoError(err)
}

func (suite *BeerServerTestSuite) TestAddBeer_RepositoryError() {
	ctx := context.Background()
	suite.beerRepo.EXPECT().AddBeer(ctx, mock.Anything).Return(nil, errors.New("insert failed"))

	result, err := suite.service.AddBeer(ctx, addBeerRequest(&apiv1.Beer{Name: hazyIPA}))

	suite.Require().ErrorContains(err, "insert failed")
	suite.Nil(result)
}
