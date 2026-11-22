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

const vancouverZone = "America/Vancouver"

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
// order, each returning empty rows. The first query is constrained to the given year bounds.
// Extend this helper whenever GetYearInReview gains a query.
func (suite *ActivityRepoTestSuite) expectYearInReviewQueries(start, end time.Time) {
	suite.mock.ExpectQuery(regexp.QuoteMeta("FROM activities a")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(sqlmock.NewRows([]string{"beers_consumed"}))

	for _, fragment := range []string{
		"COALESCE(SUM(quantity), 0) FROM activities",
		"GROUP BY bs.name",
		"GROUP BY bc.name",
		"GROUP BY br.name",
	} {
		suite.mock.ExpectQuery(regexp.QuoteMeta(fragment)).WillReturnRows(sqlmock.NewRows([]string{"name"}))
	}

	suite.mock.ExpectQuery(regexp.QuoteMeta("AT TIME ZONE")).
		WithArgs(vancouverZone, sqlmock.AnyArg(), sqlmock.AnyArg(), timeEqualArg{start}, timeEqualArg{end}).
		WillReturnRows(sqlmock.NewRows([]string{"month", "count"}))
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
