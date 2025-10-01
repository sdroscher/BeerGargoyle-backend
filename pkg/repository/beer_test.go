package repository_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
)

type BeerTestSuite struct {
	RepositorySuite
}

func TestBeerTestSuite(t *testing.T) {
	suite.Run(t, new(BeerTestSuite))
}

func (suite *BeerTestSuite) TestAddBeer_AddsBeer() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "beers" ("created_at","updated_at","deleted_at","name","description","image_url","brewery_id","style_id","abv","ibu","external_id","external_source","external_rating") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT ("name","brewery_id") DO UPDATE SET "updated_at"=$14,"deleted_at"="excluded"."deleted_at","name"="excluded"."name","description"="excluded"."description","image_url"="excluded"."image_url","brewery_id"="excluded"."brewery_id","style_id"="excluded"."style_id","abv"="excluded"."abv","ibu"="excluded"."ibu","external_id"="excluded"."external_id","external_source"="excluded"."external_source","external_rating"="excluded"."external_rating" RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Precious Bet", "Peach Saison with Brett", "", uint(10), uint(2), 8.2, 18, 4557393, "untappd-web", 4.0, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(1)))
	suite.mock.ExpectCommit()

	beer := model.Beer{
		Name:           "Precious Bet",
		Description:    "Peach Saison with Brett",
		BreweryID:      10,
		StyleID:        2,
		ABV:            pointy.Float64(8.2),
		IBU:            pointy.Uint64(18),
		ExternalSource: pointy.String("untappd-web"),
		ExternalID:     pointy.Uint64(4557393),
		ExternalRating: pointy.Float64(4.0),
	}
	result, err := suite.repository.AddBeer(context.Background(), beer)
	suite.Require().NoError(err)
	suite.NotNil(result)
}

func (suite *BeerTestSuite) TestFindBreweryByExternalSource_FindsBrewery() {
	suite.mock.ExpectQuery(`^SELECT (.+) FROM "breweries" WHERE \(external_id \= \$1 AND external_source \= \$2\) (.+)`).
		WithArgs(256500, "untappd-web", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uint(1), "Paronomastic Brewing"))

	brewery, err := suite.repository.FindBreweryByExternalSource(context.Background(), 256500, "untappd-web")
	suite.Require().NoError(err)
	suite.NotNil(brewery)
	suite.Equal(uint(1), brewery.ID)
	suite.Equal("Paronomastic Brewing", brewery.Name)
}

func (suite *BeerTestSuite) TestFindBreweryByExternalSource_ReturnsErrorWhenNoRecords() {
	suite.mock.ExpectQuery("^SELECT (.+)").WillReturnError(gorm.ErrRecordNotFound)

	brewery, err := suite.repository.FindBreweryByExternalSource(context.Background(), 100, "integration")
	suite.Require().ErrorIs(err, repository.ErrBreweryNotFound)
	suite.Nil(brewery)
	suite.Equal(1, suite.observedLogs.Len())

	errorLog := suite.observedLogs.All()[0]
	suite.Len(errorLog.Context, 4)
	suite.Equal("record not found", errorLog.ContextMap()["error"])
}

func (suite *BeerTestSuite) TestAddBeerStyle_AddsBeer() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "beer_styles" ("created_at","updated_at","deleted_at","name") VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "New Style!").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(1)))
	suite.mock.ExpectCommit()

	style, err := suite.repository.AddBeerStyle(context.Background(), "New Style!")
	suite.Require().NoError(err)
	suite.NotNil(style)
	suite.Equal(uint(1), style.ID)
	suite.Equal("New Style!", style.Name)
}

func (suite *BeerTestSuite) TestGetBeerFormats_GetsFormats() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "beer_formats" WHERE "beer_formats"."deleted_at" IS NULL`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "package", "size_metric", "size_imperial"}).
			AddRow(uint(2), "Bottle", 650, 22).AddRow(uint(1), "Can", 355.0, 12.0))

	formats, err := suite.repository.GetBeerFormats(context.Background())
	suite.Require().NoError(err)
	suite.NotNil(formats)
	suite.Len(formats, 2)
	suite.Equal("Bottle", formats[0].Package)
	suite.InDelta(650.0, formats[0].SizeMetric, 0.1)
	suite.InDelta(22.0, formats[0].SizeImperial, 0.1)
	suite.Equal("Can", formats[1].Package)
	suite.InDelta(355.0, formats[1].SizeMetric, 0.1)
	suite.InDelta(12.0, formats[1].SizeImperial, 0.1)
}
