package integrations

import (
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
	"droscher.com/BeerGargoyle/pkg/model"
)

type Integration interface {
	FindBeer(name string) ([]model.Beer, error)
	FindBrewery(name string) ([]model.Brewery, error)
}

func GetIntegration(name string, logger *zap.Logger) Integration {
	if name == untappdweb.IntegrationName {
		return untappdweb.NewUntappedWebIntegration(logger)
	}

	return nil
}
