package untappdweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ErrAlgolia is returned when an Algolia search request fails.
var ErrAlgolia = errors.New("algolia search request failed")

const algoliaTimeout = 15 * time.Second

// AlgoliaConfig holds the credentials and index names for Untappd's
// Algolia-backed search. These are public search-only credentials embedded in
// Untappd's web pages.
type AlgoliaConfig struct {
	AppID        string
	SearchKey    string
	BeerIndex    string
	BreweryIndex string
}

type algoliaClient struct {
	config AlgoliaConfig
	client *http.Client
}

func newAlgoliaClient(config AlgoliaConfig) *algoliaClient {
	return &algoliaClient{
		config: config,
		client: &http.Client{Timeout: algoliaTimeout},
	}
}

// algoliaBeerHit maps the fields of an Untappd "beer" index record.
//
//nolint:tagliatelle // field names are dictated by Algolia's snake_case API
type algoliaBeerHit struct {
	BID          uint64  `json:"bid"`
	BeerName     string  `json:"beer_name"`
	BeerABV      float64 `json:"beer_abv"`
	BeerIBU      float64 `json:"beer_ibu"`
	BeerLabel    string  `json:"beer_label"`
	BeerLabelHD  string  `json:"beer_label_hd"`
	TypeName     string  `json:"type_name"`
	RatingScore  float64 `json:"rating_score"`
	Homebrew     int     `json:"homebrew"`
	BreweryID    uint64  `json:"brewery_id"`
	BreweryName  string  `json:"brewery_name"`
	BreweryLabel string  `json:"brewery_label"`
}

// algoliaBreweryHit maps the fields of an Untappd "brewery" index record.
//
//nolint:tagliatelle // field names are dictated by Algolia's snake_case API
type algoliaBreweryHit struct {
	BreweryID      uint64 `json:"brewery_id"`
	BreweryName    string `json:"brewery_name"`
	BreweryImage   string `json:"brewery_image"`
	BreweryAddress string `json:"brewery_address"`
	BreweryCity    string `json:"brewery_city"`
	BreweryState   string `json:"brewery_state"`
	BreweryCountry string `json:"brewery_country"`
}

func (a *algoliaClient) searchBeers(query string) ([]algoliaBeerHit, error) {
	var response struct {
		Hits []algoliaBeerHit `json:"hits"`
	}

	err := a.search(a.config.BeerIndex, query, &response)
	if err != nil {
		return nil, err
	}

	return response.Hits, nil
}

func (a *algoliaClient) searchBreweries(query string) ([]algoliaBreweryHit, error) {
	var response struct {
		Hits []algoliaBreweryHit `json:"hits"`
	}

	err := a.search(a.config.BreweryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	return response.Hits, nil
}

func (a *algoliaClient) search(index, query string, out any) error {
	body, err := json.Marshal(map[string]string{"params": "query=" + url.QueryEscape(query)})
	if err != nil {
		return fmt.Errorf("marshalling algolia request: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s-dsn.algolia.net/1/indexes/%s/query", a.config.AppID, index)

	ctx, cancel := context.WithTimeout(context.Background(), algoliaTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building algolia request: %w", err)
	}

	request.Header.Set("X-Algolia-Application-Id", a.config.AppID)
	request.Header.Set("X-Algolia-Api-Key", a.config.SearchKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("performing algolia request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d for index %q", ErrAlgolia, response.StatusCode, index)
	}

	err = json.NewDecoder(response.Body).Decode(out)
	if err != nil {
		return fmt.Errorf("decoding algolia response: %w", err)
	}

	return nil
}
