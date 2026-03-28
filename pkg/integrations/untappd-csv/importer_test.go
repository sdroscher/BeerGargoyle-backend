package untappdcsv_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	untappdcsv "droscher.com/BeerGargoyle/pkg/integrations/untappd-csv"
	"droscher.com/BeerGargoyle/pkg/model"
)

const csvHeader = "beer_name,brewery_name,beer_type,beer_abv,beer_ibu,comment,venue_name,venue_city,venue_state,venue_country,venue_lat,venue_lng,rating_score,created_at,checkin_url,beer_url,brewery_url,brewery_country,brewery_city,brewery_state,flavor_profiles,purchase_venue,serving_type,checkin_id,bid,brewery_id,photo_url,global_rating_score,global_weighted_rating_score,tagged_friends,total_toasts,total_comments"

// csvRow builds a CSV data row with values at the correct column positions.
// Column indices match the csvHeader above (32 total columns).
func csvRow(beerName, breweryName, servingType, createdAt, checkinID, bid, rating, comment string) string {
	fields := make([]string, 32)
	fields[0] = beerName
	fields[1] = breweryName
	fields[2] = "Stout - Imperial"
	fields[3] = "10"
	fields[4] = "80"
	fields[5] = comment
	fields[12] = rating
	fields[13] = createdAt
	fields[22] = servingType
	fields[23] = checkinID
	fields[24] = bid
	fields[25] = "123"

	return strings.Join(fields, ",")
}

func makeRows(servingType string, n int) []untappdcsv.Row {
	rows := make([]untappdcsv.Row, n)

	for idx := range rows {
		rows[idx] = untappdcsv.Row{
			BeerName:    "Beer",
			BreweryName: "Brewery",
			ServingType: servingType,
			CreatedAt:   time.Date(2024, 1, idx+1, 0, 0, 0, 0, time.UTC),
			CheckinID:   uint64(idx) + 1, //nolint:gosec // idx is always small and positive
			BeerID:      100,
			RatingScore: 4.0,
		}
	}

	return rows
}

func TestParse_BasicRow(t *testing.T) {
	csv := csvHeader + "\n" + csvRow("Juicy AF", "Boombox", "Can", "2024-03-01 12:00:00", "999", "12345", "4.5", "great beer")
	rows, err := untappdcsv.Parse(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Juicy AF", rows[0].BeerName)
	assert.Equal(t, "Boombox", rows[0].BreweryName)
	assert.Equal(t, "Can", rows[0].ServingType)
	assert.Equal(t, uint64(999), rows[0].CheckinID)
	assert.Equal(t, uint64(12345), rows[0].BeerID)
	assert.InDelta(t, 4.5, rows[0].RatingScore, 0.001)
	assert.Equal(t, "great beer", rows[0].Comment)
	assert.Equal(t, time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC), rows[0].CreatedAt)
}

func TestParse_StripsUTF8BOM(t *testing.T) {
	// Prepend UTF-8 BOM to the CSV
	csv := "\xef\xbb\xbf" + csvHeader + "\n" + csvRow("IPA", "Test", "Bottle", "2024-01-01 00:00:00", "1", "2", "3.5", "")
	rows, err := untappdcsv.Parse(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "IPA", rows[0].BeerName)
}

func TestParse_SkipsMalformedRows(t *testing.T) {
	// Row with invalid created_at date
	badRow := "Beer,Brewery,Stout,10,80,,,,,,,,3.5,not-a-date,,,,,,,,,Can,123,456,789,,4.0,4.0,,0,0"
	csv := csvHeader + "\n" + badRow
	rows, err := untappdcsv.Parse(strings.NewReader(csv))
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParse_MissingRequiredColumn(t *testing.T) {
	data := "beer_name,brewery_name\nJuicy AF,Boombox\n"
	_, err := untappdcsv.Parse(strings.NewReader(data))
	require.ErrorIs(t, err, untappdcsv.ErrMissingColumn)
}

func TestBuildActivities_FiltersBeforeWatermark(t *testing.T) {
	watermark := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	rows := []untappdcsv.Row{
		{BeerName: "Old Beer", BreweryName: "Brewery", ServingType: "Can", CreatedAt: time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC), CheckinID: 1},
		{BeerName: "New Beer", BreweryName: "Brewery", ServingType: "Can", CreatedAt: time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC), CheckinID: 2},
	}
	beerIDByName := map[string]uint{
		untappdcsv.NameKey("New Beer", "Brewery"): 5,
	}
	activeEntryIDByBeerID := map[uint]uint{5: 10}
	anyEntryIDByBeerID := map[uint]uint{5: 10}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, watermark, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	assert.Len(t, acts, 1)
	assert.Empty(t, updates)
	assert.Equal(t, 1, skipped)
	assert.Equal(t, uint64(2), *acts[0].UntappdCheckinID)
}

func TestBuildActivities_FiltersWrongServingType(t *testing.T) {
	// Give all rows a matched entry so the only skip reason is serving type.
	beerIDByName := map[string]uint{untappdcsv.NameKey("", ""): 1}
	activeEntryIDByBeerID := map[uint]uint{1: 10}
	anyEntryIDByBeerID := map[uint]uint{1: 10}

	rows := []untappdcsv.Row{
		{ServingType: "Draft", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1},
		{ServingType: "Cask", CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), CheckinID: 2},
		{ServingType: "Can", CreatedAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), CheckinID: 3},
		{ServingType: "Bottle", CreatedAt: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), CheckinID: 4},
		{ServingType: "", CreatedAt: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), CheckinID: 5},
	}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	assert.Len(t, acts, 2)
	assert.Empty(t, updates)
	assert.Equal(t, 3, skipped)
}

func TestBuildActivities_SkipsAlreadyImported(t *testing.T) {
	rows := makeRows("Can", 3)
	existing := map[uint64]struct{}{2: {}}
	// makeRows uses BeerName "Beer", BreweryName "Brewery", BeerID 100.
	beerIDByExtID := map[uint64]uint{100: 5}
	activeEntryIDByBeerID := map[uint]uint{5: 10}
	anyEntryIDByBeerID := map[uint]uint{5: 10}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, existing, beerIDByExtID, nil, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	assert.Len(t, acts, 2)
	assert.Empty(t, updates)
	assert.Equal(t, 1, skipped)
}

func TestBuildActivities_MatchesByExternalID(t *testing.T) {
	rows := []untappdcsv.Row{
		{BeerName: "X", BreweryName: "Y", ServingType: "Can", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1, BeerID: 99},
	}
	beerIDByExtID := map[uint64]uint{99: 10}
	activeEntryIDByBeerID := map[uint]uint{10: 50}
	anyEntryIDByBeerID := map[uint]uint{10: 50}

	acts, _, _ := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, beerIDByExtID, nil, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	require.Len(t, acts, 1)
	require.NotNil(t, acts[0].CellarEntryID)
	assert.Equal(t, uint(50), *acts[0].CellarEntryID)
}

func TestBuildActivities_MatchesByNameFallback(t *testing.T) {
	rows := []untappdcsv.Row{
		{BeerName: "Juicy AF", BreweryName: "Boombox", ServingType: "Bottle", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1, BeerID: 0},
	}
	beerIDByName := map[string]uint{
		untappdcsv.NameKey("Juicy AF", "Boombox"): 7,
	}
	activeEntryIDByBeerID := map[uint]uint{7: 42}
	anyEntryIDByBeerID := map[uint]uint{7: 42}

	acts, _, _ := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	require.Len(t, acts, 1)
	require.NotNil(t, acts[0].CellarEntryID)
	assert.Equal(t, uint(42), *acts[0].CellarEntryID)
}

func TestBuildActivities_NoMatchIsSkipped(t *testing.T) {
	rows := []untappdcsv.Row{
		{BeerName: "Unknown", BreweryName: "Unknown", ServingType: "Can", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1},
	}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, nil, nil, nil, nil)
	assert.Empty(t, acts)
	assert.Empty(t, updates)
	assert.Equal(t, 1, skipped)
}

func TestBuildActivities_SetsFieldsCorrectly(t *testing.T) {
	row := untappdcsv.Row{
		BeerName:    "Test Beer",
		BreweryName: "Test Brewery",
		ServingType: "Can",
		CreatedAt:   time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
		CheckinID:   12345,
		RatingScore: 4.25,
		Comment:     "Very tasty",
	}
	beerIDByName := map[string]uint{untappdcsv.NameKey("Test Beer", "Test Brewery"): 1}
	activeEntryIDByBeerID := map[uint]uint{1: 99}
	anyEntryIDByBeerID := map[uint]uint{1: 99}

	acts, _, _ := untappdcsv.BuildActivities([]untappdcsv.Row{row}, 7, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	require.Len(t, acts, 1)

	act := acts[0]
	assert.Equal(t, uint(7), act.CellarID)
	assert.Equal(t, model.ActivityTypeBeerConsumed, act.ActivityType)
	assert.Equal(t, int64(1), act.Quantity)
	assert.Equal(t, row.CreatedAt, act.OccurredAt)
	assert.Equal(t, uint64(12345), *act.UntappdCheckinID)
	assert.InDelta(t, 4.25, *act.Rating, 0.001)
	assert.Equal(t, "Very tasty", *act.Note)
}

func TestBuildActivities_ZeroRatingNotSet(t *testing.T) {
	beerIDByName := map[string]uint{untappdcsv.NameKey("", ""): 1}
	activeEntryIDByBeerID := map[uint]uint{1: 10}
	anyEntryIDByBeerID := map[uint]uint{1: 10}
	rows := []untappdcsv.Row{
		{ServingType: "Can", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1, RatingScore: 0},
	}

	acts, _, _ := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	require.Len(t, acts, 1)
	assert.Nil(t, acts[0].Rating)
}

func TestBuildActivities_EmptyCommentNotSet(t *testing.T) {
	beerIDByName := map[string]uint{untappdcsv.NameKey("", ""): 1}
	activeEntryIDByBeerID := map[uint]uint{1: 10}
	anyEntryIDByBeerID := map[uint]uint{1: 10}
	rows := []untappdcsv.Row{
		{ServingType: "Can", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1, Comment: ""},
	}

	acts, _, _ := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, nil)
	require.Len(t, acts, 1)
	assert.Nil(t, acts[0].Note)
}

func TestBuildActivities_UpdatesExistingConsumedActivity(t *testing.T) {
	checkinTime := time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC)
	// Consumed activity was created ~3 days after the actual drinking date (within window).
	consumedAt := time.Date(2024, 3, 18, 10, 0, 0, 0, time.UTC)

	rows := []untappdcsv.Row{
		{BeerName: "Stout", BreweryName: "Dark Brewery", ServingType: "Can", CreatedAt: checkinTime, CheckinID: 555, RatingScore: 4.5, Comment: "Rich"},
	}
	beerIDByName := map[string]uint{untappdcsv.NameKey("Stout", "Dark Brewery"): 9}
	anyEntryIDByBeerID := map[uint]uint{9: 20}
	// No active entry (beer was consumed and entry deleted).
	consumedByEntryID := map[uint][]model.ConsumedActivitySummary{
		20: {{ID: 77, EntryID: 20, OccurredAt: consumedAt}},
	}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, nil, consumedByEntryID)
	assert.Empty(t, acts)
	assert.Equal(t, 0, skipped)
	require.Len(t, updates, 1)
	assert.Equal(t, uint(77), updates[0].ActivityID)
	assert.Equal(t, uint(20), updates[0].EntryID)
	assert.Equal(t, uint64(555), updates[0].CheckinID)
	require.NotNil(t, updates[0].Rating)
	assert.InDelta(t, 4.5, *updates[0].Rating, 0.001)
	require.NotNil(t, updates[0].Note)
	assert.Equal(t, "Rich", *updates[0].Note)
}

func TestBuildActivities_NoConsumedMatchOutsideWindow(t *testing.T) {
	// 40 days apart — outside the 30-day window.
	checkinTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	consumedAt := time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC)

	rows := []untappdcsv.Row{
		{BeerName: "Beer", BreweryName: "Brewery", ServingType: "Can", CreatedAt: checkinTime, CheckinID: 1, BeerID: 100},
	}
	beerIDByExtID := map[uint64]uint{100: 5}
	anyEntryIDByBeerID := map[uint]uint{5: 10}
	consumedByEntryID := map[uint][]model.ConsumedActivitySummary{
		10: {{ID: 99, EntryID: 10, OccurredAt: consumedAt}},
	}

	// No active entry, so skipped after failing the window check.
	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, beerIDByExtID, nil, anyEntryIDByBeerID, nil, consumedByEntryID)
	assert.Empty(t, acts)
	assert.Empty(t, updates)
	assert.Equal(t, 1, skipped)
}

func TestBuildActivities_UpdatePreferredOverInsert(t *testing.T) {
	// Beer has both an active entry AND a matching consumed activity — update should win.
	checkinTime := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	consumedAt := checkinTime.Add(24 * time.Hour)

	rows := []untappdcsv.Row{
		{BeerName: "IPA", BreweryName: "Hop Farm", ServingType: "Can", CreatedAt: checkinTime, CheckinID: 42},
	}
	beerIDByName := map[string]uint{untappdcsv.NameKey("IPA", "Hop Farm"): 3}
	anyEntryIDByBeerID := map[uint]uint{3: 7}
	activeEntryIDByBeerID := map[uint]uint{3: 7}
	consumedByEntryID := map[uint][]model.ConsumedActivitySummary{
		7: {{ID: 55, EntryID: 7, OccurredAt: consumedAt}},
	}

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID, consumedByEntryID)
	assert.Empty(t, acts)
	assert.Equal(t, 0, skipped)
	require.Len(t, updates, 1)
	assert.Equal(t, uint(55), updates[0].ActivityID)
}

func TestBuildActivities_SkipsDeletedEntryWithoutConsumedActivity(t *testing.T) {
	rows := []untappdcsv.Row{
		{BeerName: "Gone Beer", BreweryName: "Closed Brewery", ServingType: "Can", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), CheckinID: 1},
	}
	beerIDByName := map[string]uint{untappdcsv.NameKey("Gone Beer", "Closed Brewery"): 8}
	anyEntryIDByBeerID := map[uint]uint{8: 30}
	// Entry 30 has no consumed activities.

	acts, updates, skipped := untappdcsv.BuildActivities(rows, 1, time.Time{}, nil, nil, beerIDByName, anyEntryIDByBeerID, nil, nil)
	assert.Empty(t, acts)
	assert.Empty(t, updates)
	assert.Equal(t, 1, skipped)
}

func TestNameKey_CaseInsensitive(t *testing.T) {
	assert.Equal(t, untappdcsv.NameKey("Juicy AF", "Boombox"), untappdcsv.NameKey("juicy af", "boombox"))
}
