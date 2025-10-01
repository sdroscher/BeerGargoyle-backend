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

func (u *UntappedWebIntegration) FindBrewery(name string) ([]model.Brewery, error) {
	collector := colly.NewCollector(
		colly.AllowedDomains("untappd.com"),
	)

	var (
		errs    error
		results []model.Brewery
	)

	collector.OnHTML(".beer-item", func(element *colly.HTMLElement) {
		ratingString := element.ChildAttr(".rating > div.caps", "data-rating")
		rating, _ := strconv.ParseFloat(ratingString, 64)

		if rating > 0.0 {
			breweryURI := element.ChildAttr(".name > a", "href")

			brewery, err := u.getBreweryFromURI(breweryURI, collector)
			if multierr.AppendInto(&errs, err) {
				return
			}

			results = append(results, brewery)
		}
	})

	multierr.AppendInto(&errs, collector.Visit("https://untappd.com/search?q=/"+name+"&type=brewery"))

	return results, errs
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
		idLink := element.Attr("href")
		idString := idLink[strings.LastIndex(idLink, "/")+1:]

		id, err := strconv.ParseUint(idString, 10, 64)
		if err != nil {
			u.logger.Error("failed to parse brewery id", zap.String("url", idLink), zap.Error(err))
		} else {
			breweryID = id
		}
	})

	collector.OnHTML("head meta[property='og:url']", func(element *colly.HTMLElement) {
		breweryURI := element.Attr("content")
		idString := breweryURI[strings.LastIndex(breweryURI, "/")+1:]

		id, err := strconv.ParseUint(idString, 10, 64)
		if err != nil {
			u.logger.Error("failed to parse brewery id", zap.String("url", breweryURI), zap.Error(err))
		} else {
			breweryID = id
		}
	})

	multierr.AppendInto(&errs, collector.Visit("https://untappd.com/"+uri))

	if breweryID != 0 {
		brewery.ExternalID = pointy.Uint64(breweryID)
	}

	return brewery, errs
}

func stringPointer(value string) *string {
	if len(value) > 0 {
		return &value
	}

	return nil
}
