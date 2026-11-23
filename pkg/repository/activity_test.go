package repository_test

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

const (
	vancouverZone = "America/Vancouver"
	columnCount   = "count"
)

type ActivityRepoTestSuite struct {
	RepositorySuite
}

func TestActivityRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ActivityRepoTestSuite))
}

func (suite *ActivityRepoTestSuite) TearDownTest() {
	suite.NoError(suite.mock.ExpectationsWereMet())
}

// timeEqualArg matches a time.Time argument by instant, regardless of location.
type timeEqualArg struct {
	want time.Time
}

func (a timeEqualArg) Match(value driver.Value) bool {
	got, ok := value.(time.Time)

	return ok && got.Equal(a.want)
}

// expectYearInReviewQueries registers an expectation for every query GetYearInReview runs, in
// order, each returning empty rows. Extend this helper whenever GetYearInReview gains a query.
func (suite *ActivityRepoTestSuite) expectYearInReviewQueries(start, end time.Time) {
	suite.expectYearInReviewQueriesWithHighlights(start, end,
		sqlmock.NewRows([]string{"beer_id"}), sqlmock.NewRows([]string{"date", columnCount}))
}

// expectYearInReviewQueriesWithHighlights is expectYearInReviewQueries with caller-supplied rows
// for the beer tally and biggest-day queries. Every query must be bound to the local-zone year.
func (suite *ActivityRepoTestSuite) expectYearInReviewQueriesWithHighlights(
	start, end time.Time, tallies, biggestDay *sqlmock.Rows,
) {
	// Leading bind args before the year bounds: cellar_id (+ activity type) or, for the
	// time-zone queries, the zone name first.
	// "Different beers" must count distinct beers (not cellar entries): the same beer in two formats
	// counts once. Joined entries may be soft-deleted, so ce is deliberately not filtered.
	suite.mock.ExpectQuery(regexp.QuoteMeta("COUNT(DISTINCT ce.beer_id)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(sqlmock.NewRows([]string{"beers_consumed"}))

	suite.mock.ExpectQuery(regexp.QuoteMeta("COALESCE(SUM(quantity), 0) FROM activities")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}))

	for _, fragment := range []string{"GROUP BY bs.name", "GROUP BY bc.name", "GROUP BY br.name"} {
		suite.mock.ExpectQuery(regexp.QuoteMeta(fragment)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"name"}))
	}

	suite.mock.ExpectQuery(regexp.QuoteMeta("EXTRACT(MONTH FROM occurred_at AT TIME ZONE")).
		WithArgs(vancouverZone, sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(sqlmock.NewRows([]string{"month", columnCount}))

	suite.mock.ExpectQuery(regexp.QuoteMeta("MAX(a.occurred_at)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(tallies)

	suite.mock.ExpectQuery(regexp.QuoteMeta("to_char(occurred_at AT TIME ZONE")).
		WithArgs(vancouverZone, sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(biggestDay)
}

// A 31 Dec 23:30 Vancouver pour is already 1 Jan UTC but must fall inside the local 2026 window.
func (suite *ActivityRepoTestSuite) TestGetYearInReview_UsesLocalYearBoundaries() {
	loc, err := time.LoadLocation(vancouverZone)
	suite.Require().NoError(err)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, loc)
	suite.expectYearInReviewQueries(start, end)

	_, err = suite.repository.GetYearInReview(context.Background(), 1, 2026, loc)
	suite.Require().NoError(err)
}

func (suite *ActivityRepoTestSuite) TestGetYearInReview_ByMonthBucketsInLocalZone() {
	loc, err := time.LoadLocation(vancouverZone)
	suite.Require().NoError(err)

	suite.expectYearInReviewQueries(time.Date(2026, 1, 1, 0, 0, 0, 0, loc), time.Date(2027, 1, 1, 0, 0, 0, 0, loc))

	_, err = suite.repository.GetYearInReview(context.Background(), 1, 2026, loc)
	suite.Require().NoError(err)
}

func (suite *ActivityRepoTestSuite) TestGetYearInReview_PopulatesHighlights() {
	loc, err := time.LoadLocation(vancouverZone)
	suite.Require().NoError(err)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, loc)

	columns := []string{"beer_id", "name", "brewery_name", "style_name", "abv", "external_rating", "count", "last_consumed_at"}
	last := time.Date(2026, 6, 14, 20, 0, 0, 0, time.UTC)
	tallies := sqlmock.NewRows(columns).
		AddRow(1, "Heady Topper", "Alchemist", "IPA", 8.0, 4.5, 5, last).
		AddRow(2, "Table Beer", "Fonta Flora", "Table", 2.8, 3.2, 2, last).
		AddRow(3, "Mystery Lager", "Nobody", "Lager", nil, nil, 1, last)
	biggestDay := sqlmock.NewRows([]string{"date", columnCount}).AddRow("2026-06-14", 7)

	suite.expectYearInReviewQueriesWithHighlights(start, end, tallies, biggestDay)

	result, err := suite.repository.GetYearInReview(context.Background(), 1, 2026, loc)
	suite.Require().NoError(err)

	suite.Require().NotNil(result.BiggestDay)
	suite.Equal("2026-06-14", result.BiggestDay.Date)
	suite.Equal(int64(7), result.BiggestDay.Count)
	suite.Require().NotNil(result.MostHadBeer)
	suite.Equal("Heady Topper", result.MostHadBeer.Name)
	suite.Equal(int64(5), result.MostHadBeer.Count)
	suite.Require().NotNil(result.StrongestBeer)
	suite.InDelta(8.0, *result.StrongestBeer.ABV, 0.001)
	suite.Require().NotNil(result.GentlestBeer)
	suite.Equal("Table Beer", result.GentlestBeer.Name)
	suite.Require().NotNil(result.HighestRatedBeer)
	suite.InDelta(4.5, *result.HighestRatedBeer.ExternalRating, 0.001)
	suite.Require().NotNil(result.LowestRatedBeer)
	suite.Equal("Table Beer", result.LowestRatedBeer.Name)
}

func (suite *ActivityRepoTestSuite) TestGetYearInReview_NoConsumptionHasNoHighlights() {
	loc, err := time.LoadLocation(vancouverZone)
	suite.Require().NoError(err)

	suite.expectYearInReviewQueries(time.Date(2026, 1, 1, 0, 0, 0, 0, loc), time.Date(2027, 1, 1, 0, 0, 0, 0, loc))

	result, err := suite.repository.GetYearInReview(context.Background(), 1, 2026, loc)
	suite.Require().NoError(err)

	suite.Nil(result.BiggestDay)
	suite.Nil(result.MostHadBeer)
	suite.Nil(result.StrongestBeer)
	suite.Nil(result.GentlestBeer)
	suite.Nil(result.HighestRatedBeer)
	suite.Nil(result.LowestRatedBeer)
	suite.Zero(result.BeersConsumed)
}
