package untappdweb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	. "droscher.com/BeerGargoyle/pkg/integrations/untappd-web"
)

func TestFindBeer(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t))
	results, err := untappd.FindBeer("Twin Sails Lights Out (2021)")
	require.NoError(t, err)
	assert.Len(t, results, 1)

	assert.Equal(t, "Lights Out (2021)", results[0].Name)
	assert.InDelta(t, 14.3, *results[0].ABV, 0.01)
	assert.Nil(t, results[0].IBU)
	assert.Equal(t, "Stout - Imperial / Double", results[0].Style.Name)
	assert.Contains(t, results[0].Description, "toasted coconut")
	assert.NotEmpty(t, results[0].ImageURL)
	assert.Equal(t, "Twin Sails Brewing", results[0].Brewery.Name)
	assert.Equal(t, "We're just a bunch of people who love beer that took a stab at this brewery thing. People take beer too seriously, we decided to do things differently.", results[0].Brewery.Description)
	assert.NotEmpty(t, results[0].Brewery.ImageURL)
	assert.Equal(t, "Port Moody Canada", results[0].Brewery.Address.Locality)
	assert.NotNil(t, results[0].Brewery.Address.Region)
	assert.Equal(t, "BC", *results[0].Brewery.Address.Region)
	assert.NotNil(t, results[0].Brewery.Address.StreetAddress)
	assert.Equal(t, "2821 Murray St", *results[0].Brewery.Address.StreetAddress)
	assert.Equal(t, IntegrationName, *results[0].ExternalSource)
	assert.Equal(t, uint64(4591477), *results[0].ExternalID)
	assert.Greater(t, *results[0].ExternalRating, 0.0)
}

func TestFindHomebrew(t *testing.T) {
	untappd := NewUntappedWebIntegration(zaptest.NewLogger(t))
	results, err := untappd.FindBeer("Paronomastic Precious Bet")
	require.NoError(t, err)
	assert.Len(t, results, 1)

	assert.Equal(t, "Precious Bet", results[0].Name)
	assert.InDelta(t, 8.2, *results[0].ABV, 0.01)
	assert.Equal(t, uint64(18), *results[0].IBU)
	assert.Equal(t, "Homebrew \u00a0|\u00a0 Farmhouse Ale - Saison", results[0].Style.Name)
	assert.Equal(t, "Saison aged on peaches ‘n Brett.", results[0].Description)
	assert.NotEmpty(t, results[0].ImageURL)
	assert.Equal(t, "Paronomastic Brewing", results[0].Brewery.Name)
	assert.Equal(t, IntegrationName, *results[0].ExternalSource)
	assert.Equal(t, uint64(4557393), *results[0].ExternalID)
	assert.Nil(t, results[0].ExternalRating)
}
