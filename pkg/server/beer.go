package server

import (
	"context"
	"errors"

	"github.com/bufbuild/connect-go"
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/integrations"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
	"droscher.com/BeerGargoyle/pkg/server/grpc"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
	"droscher.com/BeerGargoyle/pkg/server/grpc/api/v1/apiv1connect"
)

type BeerServer struct {
	apiv1connect.UnimplementedBeerServiceHandler
	repository *repository.Repository
	logger     *zap.Logger
	config     *configs.Config
}

func NewBeerServer(repository *repository.Repository, logger *zap.Logger, config *configs.Config) *BeerServer {
	return &BeerServer{repository: repository, logger: logger, config: config}
}

func (b *BeerServer) FindBeer(_ context.Context, request *connect.Request[api.FindBeerRequest]) (*connect.Response[api.FindBeerResponse], error) {
	var beers []*api.Beer

	for _, integration := range b.config.Integrations.Beer {
		beerIntegration := integrations.GetIntegration(integration, b.logger)

		foundBeers, err := beerIntegration.FindBeer(request.Msg.GetQuery())
		if err != nil {
			b.logger.Error("failed beer search", zap.String("integration", integration), zap.Error(err))

			continue
		}

		beers = append(beers, grpc.BeersFromModel(foundBeers)...)
	}

	response := api.FindBeerResponse{Beers: beers}

	return connect.NewResponse(&response), nil
}

func (b *BeerServer) AddBeer(ctx context.Context, request *connect.Request[api.AddBeerRequest]) (*connect.Response[api.AddBeerResponse], error) {
	beer := grpc.BeerToModel(request.Msg.GetBeer())

	if request.Msg.GetBeer().GetBrewery() != nil {
		if request.Msg.GetBeer().GetBrewery().GetId() != 0 {
			beer.BreweryID = uint(request.Msg.GetBeer().GetBrewery().GetId())
		} else {
			b.loadBrewery(ctx, request.Msg.GetBeer().GetBrewery(), &beer)
		}
	}

	if request.Msg.GetBeer().GetStyle() != nil {
		err := b.assignBeerStyle(ctx, request.Msg.GetBeer().GetStyle(), &beer)
		if err != nil {
			return nil, err
		}
	}

	newBeer, err := b.repository.AddBeer(ctx, beer)
	if err != nil {
		return nil, err
	}

	response := api.AddBeerResponse{
		Beer: grpc.BeerFromModel(*newBeer),
	}

	return connect.NewResponse(&response), nil
}

func (b *BeerServer) assignBeerStyle(ctx context.Context, beerStyle *api.BeerStyle, beer *model.Beer) error {
	if beerStyle.GetId() != 0 {
		beer.StyleID = uint(beerStyle.GetId())
	} else {
		style, err := b.repository.AddBeerStyle(ctx, beerStyle.GetName())
		if err != nil {
			return err
		}

		beer.StyleID = style.ID
	}

	return nil
}

func (b *BeerServer) loadBrewery(ctx context.Context, pbBrewery *api.Brewery, beer *model.Beer) {
	brewery, err := b.repository.FindBreweryByExternalSource(ctx, pbBrewery.GetExternalId(), pbBrewery.GetExternalSource())
	if err != nil {
		if errors.Is(err, repository.ErrBreweryNotFound) {
			b.logger.Warn("brewery not found", zap.Uint64("external ID", pbBrewery.GetExternalId()), zap.String("source", pbBrewery.GetExternalSource()))
		} else {
			b.logger.Error("error looking for brewery", zap.Uint64("external ID", pbBrewery.GetExternalId()), zap.String("source", pbBrewery.GetExternalSource()), zap.Error(err))
		}

		beer.Brewery = grpc.BreweryToModel(pbBrewery)
	} else {
		beer.BreweryID = brewery.ID
	}
}

func (b *BeerServer) GetBeerFormats(ctx context.Context, _ *connect.Request[api.GetBeerFormatsRequest]) (*connect.Response[api.GetBeerFormatsResponse], error) {
	formats, err := b.repository.GetBeerFormats(ctx)
	if err != nil {
		return nil, err
	}

	response := api.GetBeerFormatsResponse{
		Formats: grpc.FormatsFromModel(formats),
	}

	return connect.NewResponse(&response), nil
}
