package untappdweb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	. "droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
)

func TestFindBrewery(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t), testConfig())
	results, err := untappd.FindBrewery("Fremont Brewing")

	require.NoError(t, err)
	require.NotEmpty(t, results)

	brewery := results[0]
	assert.Equal(t, "Fremont Brewing", brewery.Name)
	assert.NotEmpty(t, brewery.ImageURL)
	assert.Equal(t, IntegrationName, *brewery.ExternalSource)
	assert.Equal(t, uint64(1508), *brewery.ExternalID)
	assert.Equal(t, "Seattle", brewery.Address.Locality)
	assert.NotNil(t, brewery.Address.Region)
	assert.Equal(t, "WA", *brewery.Address.Region)
	assert.NotNil(t, brewery.Address.StreetAddress)
	assert.Equal(t, "3409 Woodland Park Ave North", *brewery.Address.StreetAddress)
}

// TestGetBreweryDetails exercises the Cloudflare-gated detail-page fetch. It is
// skipped when the request is blocked (e.g. from a datacenter IP in CI without a
// proxy), since the description is only available behind Cloudflare.
func TestGetBreweryDetails(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t), testConfig())

	brewery, err := untappd.GetBreweryDetails(1508)
	if err != nil || brewery.Name == "" {
		t.Skipf("brewery detail page unavailable (likely Cloudflare): err=%v", err)
	}

	assert.Equal(t, "Fremont Brewing", brewery.Name)
	assert.NotEmpty(t, brewery.Description)
	assert.NotNil(t, brewery.ExternalRating)
}
