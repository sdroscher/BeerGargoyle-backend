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

const (
	chromeVersion = "124"
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + chromeVersion + ".0.0.0 Safari/537.36"
	secChUa       = `"Chromium";v="` + chromeVersion + `", "Google Chrome";v="` + chromeVersion + `", "Not-A.Brand";v="99"`
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
		colly.UserAgent(userAgent),
	)

	setBrowserHeaders(collector)

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

	u.registerDebugHandlers(collector)

	u.logger.Info("scraping query results", zap.String("query", name))
	multierr.AppendInto(&errs, collector.Visit("https://untappd.com/search?q=/"+name))

	results = make([]model.Beer, 0, len(scrapedPages))

	var beerWG sync.WaitGroup

	beerChan := make(chan scrapeResults, len(scrapedPages))

	for _, scraped := range scrapedPages {
		beerWG.Add(1)

		go func() {
			defer beerWG.Done()
			u.getBeerData(collector.Clone(), scraped, breweries, beerChan)
		}()
	}

	beerWG.Wait()

	for range scrapedPages {
		result := <-beerChan
		results = append(results, result.beers...)
		multierr.AppendInto(&errs, result.err)
	}

	u.logger.Info("finished scraping query results", zap.Any("results", results), zap.Error(errs))

	return results, errs
}

func (u *UntappedWebIntegration) getBeerData(detailCollector *colly.Collector, scraped BeerScraped, breweries map[string]model.Brewery, beerChan chan scrapeResults) {
	setBrowserHeaders(detailCollector)
	u.registerDebugHandlers(detailCollector)

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
		if !strings.HasPrefix(beerJSON.Image.ContentURL, "https://next.untappd.com/og/") {
			beer.ImageURL = beerJSON.Image.ContentURL
		}
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
	clone := collector.Clone()
	setBrowserHeaders(clone)
	u.registerDebugHandlers(clone)

	brewery, err := u.getBreweryFromURI(breweryURI, clone)
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

func (u *UntappedWebIntegration) registerDebugHandlers(collector *colly.Collector) {
	collector.OnResponse(func(response *colly.Response) {
		u.logger.Info("scrape response", zap.String("url", responseURL(response)), zap.Int("status_code", response.StatusCode))
	})

	collector.OnError(func(response *colly.Response, err error) {
		fields := []zap.Field{
			zap.String("url", responseURL(response)),
			zap.Error(err),
			zap.Int("status_code", response.StatusCode),
		}
		if response.Headers != nil {
			for key, vals := range *response.Headers {
				fields = append(fields, zap.String("resp_header_"+key, strings.Join(vals, ", ")))
			}
		}
		if len(response.Body) > 0 {
			runes := []rune(string(response.Body))
			if len(runes) > 500 {
				runes = runes[:500]
			}
			fields = append(fields, zap.String("resp_body_snippet", string(runes)))
		}
		u.logger.Error("error while scraping beer search results", fields...)
	})
}

func responseURL(response *colly.Response) string {
	if response.Request == nil || response.Request.URL == nil {
		return ""
	}

	return response.Request.URL.String()
}

func setBrowserHeaders(collector *colly.Collector) {
	collector.OnRequest(func(req *colly.Request) {
		req.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		req.Headers.Set("Cache-Control", "no-cache")
		req.Headers.Set("Pragma", "no-cache")
		req.Headers.Set("Sec-Ch-Ua", secChUa)
		req.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Headers.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Headers.Set("Sec-Fetch-Dest", "document")
		req.Headers.Set("Sec-Fetch-Mode", "navigate")
		req.Headers.Set("Sec-Fetch-Site", "none")
		req.Headers.Set("Sec-Fetch-User", "?1")
		req.Headers.Set("Upgrade-Insecure-Requests", "1")
	})
}

func extractIBU(details BeerScraped) *uint64 {
	if !strings.HasPrefix(details.IBU, "N/A") {
		ibu, _ := strconv.ParseUint(strings.Split(details.IBU, " ")[0], 0, 64)

		return pointy.Uint64(ibu)
	}

	return nil
}
