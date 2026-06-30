package integrations

import (
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
	"droscher.com/BeerGargoyle/pkg/model"
)

type Integration interface {
	FindBeer(name string) ([]model.Beer, error)
	FindBrewery(name string) ([]model.Brewery, error)
	// GetBeerDescription fetches a single beer's long description from its
	// detail page (used to enrich a beer at save time).
	GetBeerDescription(externalID uint64) (string, error)
	// GetBreweryDetails fetches a brewery's description, address and rating from
	// its detail page (used to enrich a new brewery at save time).
	GetBreweryDetails(externalID uint64) (model.Brewery, error)
}

func GetIntegration(name string, config *configs.Config, logger *zap.Logger) Integration {
	if name == untappdweb.IntegrationName {
		untappd := config.Integrations.Untappd
		proxyURL, insecureSkipVerify := untappd.ResolveProxy()

		return untappdweb.NewUntappedWebIntegration(logger, untappdweb.Config{
			Proxy: untappdweb.ProxyConfig{
				URL:                proxyURL,
				InsecureSkipVerify: insecureSkipVerify,
			},
			Algolia: untappdweb.AlgoliaConfig{
				AppID:        untappd.AlgoliaAppID,
				SearchKey:    untappd.AlgoliaSearchKey,
				BeerIndex:    untappd.AlgoliaBeerIndex,
				BreweryIndex: untappd.AlgoliaBreweryIndex,
			},
		})
	}

	return nil
}
