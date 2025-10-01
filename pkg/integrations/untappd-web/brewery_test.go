package untappdweb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	. "droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
)

func TestFindBrewery(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t))
	results, err := untappd.FindBrewery("Fremont Brewing")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Fremont Brewing", results[0].Name)
	assert.Equal(t, "Fremont Brewing was born of our love for our home and history as well as the desire to prove that beer made with the finest local ingredients – organic when possible --, is not the wave of the future but the doorway to beer's history. Starting a brewery in the midst of the Great Recession is clearly an act of passion. We invite you to come along with us and enjoy that passion -- because beer matters.", results[0].Description)
	assert.NotEmpty(t, results[0].ImageURL)
	assert.Equal(t, IntegrationName, *results[0].ExternalSource)
	assert.Equal(t, uint64(1508), *results[0].ExternalID)
	assert.InDelta(t, 4.038, *results[0].ExternalRating, 0.1)
	assert.Equal(t, "Seattle", results[0].Address.Locality)
	assert.NotNil(t, results[0].Address.Region)
	assert.Equal(t, "WA", *results[0].Address.Region)
	assert.NotNil(t, results[0].Address.StreetAddress)
	assert.Equal(t, "3409 Woodland Park Ave North", *results[0].Address.StreetAddress)
}
