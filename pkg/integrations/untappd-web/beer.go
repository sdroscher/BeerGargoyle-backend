package untappdweb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
	"go.openly.dev/pointy"
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/model"
)

const (
	chromeVersion     = "124"
	chromeFullVersion = chromeVersion + ".0.6367.155"
	userAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + chromeVersion + ".0.0.0 Safari/537.36"
	secChUa           = `"Chromium";v="` + chromeVersion + `", "Google Chrome";v="` + chromeVersion + `", "Not-A.Brand";v="99"`
	secChUaFullList   = `"Chromium";v="` + chromeFullVersion + `", "Google Chrome";v="` + chromeFullVersion + `", "Not-A.Brand";v="99.0.0.0"`
)

// homebrewStylePrefix preserves the original style formatting for homebrews,
// which Untappd's pages rendered as "Homebrew  |  <style>".
const homebrewStylePrefix = "Homebrew  |  "

// BeerJSON maps the ld+json embedded in an Untappd beer detail page. Only the
// description is read; everything else now comes from Algolia search.
type BeerJSON struct {
	Description string `json:"description"`
}

// FindBeer searches Untappd via its Algolia backend, which returns structured
// JSON and is not behind Cloudflare, so it requires no proxy.
func (u *UntappedWebIntegration) FindBeer(name string) ([]model.Beer, error) {
	u.logger.Info("searching untappd beers", zap.String("query", name))

	hits, err := u.algolia.searchBeers(name)
	if err != nil {
		return nil, err
	}

	results := make([]model.Beer, 0, len(hits))
	for _, hit := range hits {
		results = append(results, beerFromHit(hit))
	}

	u.logger.Info("finished untappd beer search", zap.String("query", name), zap.Int("results", len(results)))

	return results, nil
}

func beerFromHit(hit algoliaBeerHit) model.Beer {
	beer := model.Beer{
		Name:           hit.BeerName,
		ImageURL:       firstNonEmpty(hit.BeerLabelHD, hit.BeerLabel),
		Style:          model.BeerStyle{Name: styleName(hit)},
		ExternalSource: pointy.String(IntegrationName),
		ExternalID:     pointy.Uint64(hit.BID),
		Brewery: model.Brewery{
			Name:           hit.BreweryName,
			ImageURL:       hit.BreweryLabel,
			ExternalSource: pointy.String(IntegrationName),
			ExternalID:     pointy.Uint64(hit.BreweryID),
		},
	}

	if hit.BeerABV > 0 {
		beer.ABV = pointy.Float64(hit.BeerABV)
	}

	if hit.BeerIBU > 0 {
		beer.IBU = pointy.Uint64(uint64(hit.BeerIBU))
	}

	if hit.RatingScore > 0 {
		beer.ExternalRating = pointy.Float64(hit.RatingScore)
	}

	return beer
}

func styleName(hit algoliaBeerHit) string {
	if hit.Homebrew == 1 {
		return homebrewStylePrefix + hit.TypeName
	}

	return hit.TypeName
}

// GetBeerDescription fetches a single beer's long description from its Untappd
// detail page. The page is behind Cloudflare, so this goes through the
// configured proxy when running from a datacenter IP.
func (u *UntappedWebIntegration) GetBeerDescription(externalID uint64) (string, error) {
	collector := u.newCollector()
	u.registerDebugHandlers(collector)

	var description string

	collector.OnHTML("head script[type='application/ld+json']", func(element *colly.HTMLElement) {
		var beerJSON BeerJSON

		err := json.Unmarshal([]byte(element.Text), &beerJSON)
		if err != nil {
			return
		}

		if beerJSON.Description != "" {
			description = beerJSON.Description
		}
	})

	err := collector.Visit(fmt.Sprintf("https://untappd.com/beer/%d", externalID))

	return description, err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
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
		u.logger.Error("error while scraping beer detail page", fields...)
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
		req.Headers.Set("Sec-Ch-Ua-Arch", `"x86"`)
		req.Headers.Set("Sec-Ch-Ua-Bitness", `"64"`)
		req.Headers.Set("Sec-Ch-Ua-Full-Version", `"`+chromeFullVersion+`"`)
		req.Headers.Set("Sec-Ch-Ua-Full-Version-List", secChUaFullList)
		req.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Headers.Set("Sec-Ch-Ua-Model", `""`)
		req.Headers.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Headers.Set("Sec-Ch-Ua-Platform-Version", `"10.0.0"`)
		req.Headers.Set("Sec-Fetch-Dest", "document")
		req.Headers.Set("Sec-Fetch-Mode", "navigate")
		req.Headers.Set("Sec-Fetch-Site", "none")
		req.Headers.Set("Sec-Fetch-User", "?1")
		req.Headers.Set("Upgrade-Insecure-Requests", "1")
	})
}
