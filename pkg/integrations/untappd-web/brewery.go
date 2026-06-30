package untappdweb

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"go.openly.dev/pointy"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/model"
)

type BreweryJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       struct {
		ContentURL string `json:"contentUrl"`
		URL        string `json:"url"`
	} `json:"image"`
	AggregateRating struct {
		RatingValue float64 `json:"ratingValue"`
		BestRating  string  `json:"bestRating"`
		ReviewCount int     `json:"reviewCount"`
	} `json:"aggregateRating"`
	Address struct {
		StreetAddress   string `json:"streetAddress"`
		AddressLocality string `json:"addressLocality"`
		AddressRegion   string `json:"addressRegion"`
	} `json:"address"`
}

// FindBrewery searches Untappd's Algolia "brewery" index, which returns
// structured JSON and is not behind Cloudflare, so it requires no proxy.
func (u *UntappedWebIntegration) FindBrewery(name string) ([]model.Brewery, error) {
	u.logger.Info("searching untappd breweries", zap.String("query", name))

	hits, err := u.algolia.searchBreweries(name)
	if err != nil {
		return nil, err
	}

	results := make([]model.Brewery, 0, len(hits))
	for _, hit := range hits {
		results = append(results, breweryFromHit(hit))
	}

	return results, nil
}

func breweryFromHit(hit algoliaBreweryHit) model.Brewery {
	return model.Brewery{
		Name:     hit.BreweryName,
		ImageURL: hit.BreweryImage,
		Address: model.Address{
			Country:       hit.BreweryCountry,
			Locality:      hit.BreweryCity,
			Region:        stringPointer(hit.BreweryState),
			StreetAddress: stringPointer(hit.BreweryAddress),
		},
		ExternalSource: pointy.String(IntegrationName),
		ExternalID:     pointy.Uint64(hit.BreweryID),
	}
}

// GetBreweryDetails fetches a brewery's description, address and rating from its
// Untappd page (behind Cloudflare, so routed through the configured proxy). Used
// to enrich a brewery when a beer is first saved.
func (u *UntappedWebIntegration) GetBreweryDetails(externalID uint64) (model.Brewery, error) {
	collector := u.newCollector()
	u.registerDebugHandlers(collector)

	return u.getBreweryFromURI("brewery/"+strconv.FormatUint(externalID, 10), collector)
}

func (u *UntappedWebIntegration) getBreweryFromURI(uri string, collector *colly.Collector) (model.Brewery, error) {
	var (
		errs      error
		brewery   model.Brewery
		breweryID uint64
	)

	collector.OnHTML("head script[type='application/ld+json']", func(element *colly.HTMLElement) {
		var breweryJSON BreweryJSON
		_ = json.Unmarshal([]byte(element.Text), &breweryJSON)

		brewery = model.Brewery{
			Name:        breweryJSON.Name,
			Description: breweryJSON.Description,
			Address: model.Address{
				Locality:      breweryJSON.Address.AddressLocality,
				Region:        stringPointer(breweryJSON.Address.AddressRegion),
				StreetAddress: stringPointer(breweryJSON.Address.StreetAddress),
			},
			ImageURL:       breweryJSON.Image.ContentURL,
			ExternalSource: pointy.String(IntegrationName),
			ExternalRating: pointy.Float64(breweryJSON.AggregateRating.RatingValue),
		}
	})

	collector.OnHTML("p.rss a", func(element *colly.HTMLElement) {
		breweryID = parseTrailingID(u.logger, element.Attr("href"))
	})

	collector.OnHTML("head meta[property='og:url']", func(element *colly.HTMLElement) {
		breweryID = parseTrailingID(u.logger, element.Attr("content"))
	})

	multierr.AppendInto(&errs, collector.Visit("https://untappd.com/"+uri))

	if breweryID != 0 {
		brewery.ExternalID = pointy.Uint64(breweryID)
	}

	return brewery, errs
}

func parseTrailingID(logger *zap.Logger, link string) uint64 {
	idString := link[strings.LastIndex(link, "/")+1:]

	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		logger.Error("failed to parse brewery id", zap.String("url", link), zap.Error(err))

		return 0
	}

	return id
}

func stringPointer(value string) *string {
	if len(value) > 0 {
		return &value
	}

	return nil
}
