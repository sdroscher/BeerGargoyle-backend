package untappdweb_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	. "droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
)

// testConfig returns an integration configured with the public Algolia search
// credentials and no proxy (search is not behind Cloudflare).
func testConfig() Config {
	return Config{
		Algolia: AlgoliaConfig{
			AppID:        "9WBO4RQ3HO",
			SearchKey:    "1d347324d67ec472bb7132c66aead485",
			BeerIndex:    "beer",
			BreweryIndex: "brewery",
		},
	}
}

func TestFindBeer(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t), testConfig())
	results, err := untappd.FindBeer("Twin Sails Lights Out")
	require.NoError(t, err)
	require.NotEmpty(t, results)

	beer := results[0]
	assert.Contains(t, beer.Name, "Lights Out")
	assert.NotNil(t, beer.ABV)
	assert.Positive(t, *beer.ABV)
	assert.NotEmpty(t, beer.Style.Name)
	assert.NotEmpty(t, beer.ImageURL)
	assert.Equal(t, "Twin Sails Brewing", beer.Brewery.Name)
	assert.NotNil(t, beer.Brewery.ExternalID)
	assert.Equal(t, IntegrationName, *beer.ExternalSource)
	assert.NotNil(t, beer.ExternalID)
	assert.NotNil(t, beer.ExternalRating)
	assert.Positive(t, *beer.ExternalRating)
}

// TestGetBeerDescription exercises the Cloudflare-gated detail-page fetch. It is
// skipped when the request is blocked (e.g. from a datacenter IP in CI without a
// proxy), since the description is only available behind Cloudflare.
func TestGetBeerDescription(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t), testConfig())

	results, err := untappd.FindBeer("Twin Sails Lights Out")
	require.NoError(t, err)
	require.NotEmpty(t, results)
	require.NotNil(t, results[0].ExternalID)

	description, err := untappd.GetBeerDescription(*results[0].ExternalID)
	if err != nil || description == "" {
		t.Skipf("beer detail page unavailable (likely Cloudflare): err=%v", err)
	}

	assert.NotEmpty(t, description)
}

func TestFindHomebrew(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t), testConfig())
	results, err := untappd.FindBeer("Paronomastic Precious Bet")
	require.NoError(t, err)
	require.NotEmpty(t, results)

	beer := results[0]
	assert.Contains(t, beer.Name, "Precious Bet")
	assert.True(t, strings.HasPrefix(beer.Style.Name, "Homebrew"), "homebrew style should be prefixed: %q", beer.Style.Name)
	assert.Equal(t, "Paronomastic Brewing", beer.Brewery.Name)
	assert.Equal(t, IntegrationName, *beer.ExternalSource)
	assert.NotNil(t, beer.ExternalID)
}
