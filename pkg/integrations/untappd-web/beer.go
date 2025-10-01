package untappdweb

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"go.openly.dev/pointy"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/model"
)

type BeerJSON struct {
	Description string `json:"description"`
	Brand       struct {
		Name string `json:"name"`
	} `json:"brand"`
	Image struct {
		ContentURL string `json:"contentUrl"`
	} `json:"image"`
	Sku             uint64 `json:"sku"`
	AggregateRating struct {
		RatingValue float64 `json:"ratingValue"`
		BestRating  string  `json:"bestRating"`
		ReviewCount int     `json:"reviewCount"`
	} `json:"aggregateRating"`
}

type BeerScraped struct {
	IDLink        string `attr:"href"          selector:"a.label"`
	Name          string `selector:".name > a"`
	BreweryIDLink string `attr:"href"          selector:".brewery > a"`
	Style         string `selector:".style"`
	ABV           string `selector:".abv"`
	IBU           string `selector:".ibu"`
}

type BeerContent struct {
	Description string `selector:".beer-descrption-read-more"`
	ImageURL    string `attr:"src"                            selector:"a.label > img"`
	Rating      string `selector:".details .num"`
}

type scrapeResults struct {
	beers []model.Beer
	err   error
}

func (u *UntappedWebIntegration) FindBeer(name string) ([]model.Beer, error) {
	collector := colly.NewCollector(
		colly.AllowedDomains("untappd.com"),
		colly.UserAgent("Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1"),
	)

	var (
		errs         error
		results      []model.Beer
		scrapedPages []BeerScraped
	)

	breweries := make(map[string]model.Brewery, 0)

	collector.OnHTML(".beer-item", func(element *colly.HTMLElement) {
		scraped := BeerScraped{}

		err := element.Unmarshal(&scraped)
		if multierr.AppendInto(&errs, err) {
			u.logger.Error("failed to unmarshal scraped beer", zap.Error(err))

			return
		}

		idString := scraped.IDLink[strings.LastIndex(scraped.IDLink, "/")+1:]

		u.logger.Info("successfully scraped item from results", zap.String("id", idString), zap.String("name", scraped.Name))

		if _, found := breweries[scraped.BreweryIDLink]; !found {
			err = u.cacheBreweryDetails(scraped.BreweryIDLink, collector, breweries)
			if multierr.AppendInto(&errs, err) {
				return
			}
		}

		scrapedPages = append(scrapedPages, scraped)
	})

	collector.OnError(func(response *colly.Response, err error) {
		u.logger.Error("error while scraping beer search results", zap.String("url", response.Request.URL.String()), zap.Error(err))
	})

	u.logger.Info("scraping query results", zap.String("query", name))
	multierr.AppendInto(&errs, collector.Visit("https://untappd.com/search?q=/"+name))

	var beerWG sync.WaitGroup

	beerChan := make(chan scrapeResults, len(scrapedPages))

	appendResult := func() {
		scraped := <-beerChan
		results = append(results, scraped.beers...)
		multierr.AppendInto(&errs, scraped.err)
		beerWG.Done()
	}

	for _, scraped := range scrapedPages {
		beerWG.Add(1)

		go u.getBeerData(collector.Clone(), scraped, breweries, beerChan)
		go appendResult()
	}

	beerWG.Wait()

	u.logger.Info("finished scraping query results", zap.Any("results", results), zap.Error(errs))

	return results, errs
}

func (u *UntappedWebIntegration) getBeerData(detailCollector *colly.Collector, scraped BeerScraped, breweries map[string]model.Brewery, beerChan chan scrapeResults) {
	beer := model.Beer{
		Name:           scraped.Name,
		ExternalSource: pointy.String(IntegrationName),
		Brewery:        breweries[scraped.BreweryIDLink],
		Style:          model.BeerStyle{Name: scraped.Style},
		ABV:            extractABV(scraped),
		IBU:            extractIBU(scraped),
	}

	detailCollector.OnHTML("head script[type='application/ld+json']", func(element *colly.HTMLElement) {
		var beerJSON BeerJSON
		_ = json.Unmarshal([]byte(element.Text), &beerJSON)

		u.logger.Info("successfully scraped beer from JSON data", zap.Uint64("id", beerJSON.Sku), zap.String("description", beerJSON.Description))

		beer.Description = beerJSON.Description
		beer.ImageURL = beerJSON.Image.ContentURL
		beer.ExternalID = pointy.Uint64(beerJSON.Sku)
		beer.ExternalRating = pointy.Float64(beerJSON.AggregateRating.RatingValue)
	})

	detailCollector.OnHTML(".content", func(element *colly.HTMLElement) {
		beerContent := BeerContent{}

		err := element.Unmarshal(&beerContent)
		if err != nil {
			return
		}

		if len(beer.Description) == 0 {
			beer.Description = beerContent.Description
		}

		if len(beer.ImageURL) == 0 {
			beer.ImageURL = beerContent.ImageURL
		}

		if beer.ExternalRating == nil {
			rating, err := strconv.ParseFloat(beerContent.Rating, 64)
			if err == nil {
				beer.ExternalRating = pointy.Float64(rating)
			}
		}
	})

	idString := scraped.IDLink[strings.LastIndex(scraped.IDLink, "/")+1:]
	u.logger.Info("scraping beer page", zap.String("id", idString))

	err := detailCollector.Visit("https://untappd.com/beer/" + idString)
	if err == nil && beer.ExternalID == nil {
		externalID, err := strconv.ParseUint(idString, 10, 64)
		if err == nil {
			beer.ExternalID = pointy.Uint64(externalID)
		}
	}

	beerChan <- scrapeResults{beers: []model.Beer{beer}, err: err}
}

func (u *UntappedWebIntegration) cacheBreweryDetails(breweryURI string, collector *colly.Collector, breweries map[string]model.Brewery) error {
	brewery, err := u.getBreweryFromURI(breweryURI, collector)
	if err != nil {
		return err
	}

	breweries[breweryURI] = brewery

	return nil
}

func extractABV(details BeerScraped) *float64 {
	if strings.Contains(details.ABV, "%") {
		abv, _ := strconv.ParseFloat(details.ABV[:strings.Index(details.ABV, "%")], 64) //nolint: gocritic // We know we won't get -1

		return &abv
	}

	return nil
}

func extractIBU(details BeerScraped) *uint64 {
	if !strings.HasPrefix(details.IBU, "N/A") {
		ibu, _ := strconv.ParseUint(strings.Split(details.IBU, " ")[0], 0, 64)

		return pointy.Uint64(ibu)
	}

	return nil
}
