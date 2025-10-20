package repository_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
)

type CellarTestSuite struct {
	RepositorySuite
}

func TestCellarTestSuite(t *testing.T) {
	suite.Run(t, new(CellarTestSuite))
}

func (suite *CellarTestSuite) TearDownTest() {
	suite.NoError(suite.mock.ExpectationsWereMet())
}

func (suite *CellarTestSuite) TestAddCellar_Adds_Cellars() {
	locations := []string{"A", "B"}
	owner := model.User{Model: gorm.Model{ID: 100}}

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cellars" ("created_at","updated_at","deleted_at","name","description","owner_id") VALUES ($1,$2,$3,$4,$5,$6)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "test cellar", "cellar description", owner.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("10"))
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "location_in_cellars" ("created_at","updated_at","deleted_at","name","cellar_id") VALUES ($1,$2,$3,$4,$5),($6,$7,$8,$9,$10)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "A", 10, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "B", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1").AddRow("2"))
	suite.mock.ExpectCommit()

	result, err := suite.repository.AddCellar(context.Background(), "test cellar", "cellar description", locations, owner)
	suite.Require().NoError(err)
	suite.NotNil(result)

	suite.Equal(uint(10), result.ID)
	suite.Equal("test cellar", result.Name)
	suite.Equal("cellar description", result.Description)
	suite.Equal(uint(100), result.OwnerID)

	suite.Len(result.Locations, 2)
	suite.Equal("A", result.Locations[0].Name)
	suite.Equal(uint(10), result.Locations[0].CellarID)
	suite.Equal("B", result.Locations[1].Name)
	suite.Equal(uint(10), result.Locations[1].CellarID)
}

func (suite *CellarTestSuite) TestAddCellar_ReturnsError() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery("^INSERT INTO (.+)").WillReturnError(gorm.ErrInvalidData)
	suite.mock.ExpectRollback()

	locations := []string{"A", "B"}
	owner := model.User{Model: gorm.Model{ID: 100}}
	result, err := suite.repository.AddCellar(context.Background(), "test cellar", "cellar description", locations, owner)

	suite.Nil(result)
	suite.EqualError(err, "unsupported data")
}

func (suite *CellarTestSuite) TestGetAllCellars_GetCellars() {
	owner := model.User{Model: gorm.Model{ID: 100}}

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "cellars"."id","cellars"."created_at","cellars"."updated_at","cellars"."deleted_at","cellars"."name","cellars"."description","cellars"."owner_id","Owner"."id" AS "Owner__id","Owner"."created_at" AS "Owner__created_at","Owner"."updated_at" AS "Owner__updated_at","Owner"."deleted_at" AS "Owner__deleted_at","Owner"."uuid" AS "Owner__uuid","Owner"."username" AS "Owner__username","Owner"."first_name" AS "Owner__first_name","Owner"."last_name" AS "Owner__last_name","Owner"."email" AS "Owner__email","Owner"."untappd_user_name" AS "Owner__untappd_user_name" FROM "cellars" LEFT JOIN "users" "Owner" ON "cellars"."owner_id" = "Owner"."id" AND "Owner"."deleted_at" IS NULL WHERE owner_id = $1 AND "cellars"."deleted_at" IS NULL`)).
		WithArgs(100).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "description", "owner_id", "Owner__id", "Owner__username"}).
				AddRow(1, "my cellar", "my cellar description", 100, 100, "testuser"))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "location_in_cellars" WHERE "location_in_cellars"."cellar_id" = $1 AND "location_in_cellars"."deleted_at" IS NULL`)).
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "cellar_id"}).
				AddRow(1, "Loc A", 1).
				AddRow(2, "Loc B", 1))

	results, err := suite.repository.GetCellarsForUser(context.Background(), owner)
	suite.Require().NoError(err)
	suite.Len(results, 1)
	suite.Equal("my cellar", results[0].Name)
	suite.Equal(uint(100), results[0].OwnerID)
	suite.NotNil(results[0].Owner)
	suite.Equal("testuser", results[0].Owner.Username)
	suite.Len(results[0].Locations, 2)
	suite.Equal("Loc A", results[0].Locations[0].Name)
	suite.Equal("Loc B", results[0].Locations[1].Name)
}

func (suite *CellarTestSuite) TestGetCellarById_GetsCellar() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "cellars"."id","cellars"."created_at","cellars"."updated_at","cellars"."deleted_at","cellars"."name","cellars"."description","cellars"."owner_id","Owner"."id" AS "Owner__id","Owner"."created_at" AS "Owner__created_at","Owner"."updated_at" AS "Owner__updated_at","Owner"."deleted_at" AS "Owner__deleted_at","Owner"."uuid" AS "Owner__uuid","Owner"."username" AS "Owner__username","Owner"."first_name" AS "Owner__first_name","Owner"."last_name" AS "Owner__last_name","Owner"."email" AS "Owner__email","Owner"."untappd_user_name" AS "Owner__untappd_user_name" FROM "cellars" LEFT JOIN "users" "Owner" ON "cellars"."owner_id" = "Owner"."id" AND "Owner"."deleted_at" IS NULL WHERE "cellars"."id" = $1 AND "cellars"."deleted_at" IS NULL ORDER BY "cellars"."id" LIMIT $2`)).
		WithArgs(100, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "description", "owner_id", "Owner__id", "Owner__username"}).
				AddRow(1, "my cellar", "my cellar description", 100, 100, "testuser"))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "location_in_cellars" WHERE "location_in_cellars"."cellar_id" = $1 AND "location_in_cellars"."deleted_at" IS NULL`)).
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "cellar_id"}).
				AddRow(1, "Loc A", 1).
				AddRow(2, "Loc B", 1))

	result, err := suite.repository.GetCellarByID(context.Background(), 100)

	suite.Require().NoError(err)
	suite.Equal("my cellar", result.Name)
	suite.Equal(uint(100), result.OwnerID)
	suite.NotNil(result.Owner)
	suite.Equal("testuser", result.Owner.Username)
	suite.Len(result.Locations, 2)
	suite.Equal("Loc A", result.Locations[0].Name)
	suite.Equal("Loc B", result.Locations[1].Name)
}

func (suite *CellarTestSuite) TestGetCellarEntryByID_GetsCellarEntryWithAssociations() {
	suite.mock.ExpectQuery(`SELECT (.+) FROM "cellar_entries" LEFT JOIN "beers" "Beer" ON "cellar_entries"\."beer_id" \= "Beer"\."id" AND "Beer"\."deleted_at" IS NULL LEFT JOIN "location_in_cellars" "Location" ON "cellar_entries"\."location_id" \= "Location"\."id" AND "Location"\."deleted_at" IS NULL LEFT JOIN "beer_formats" "Format" ON "cellar_entries"\."format_id" \= "Format"\."id" AND "Format"\."deleted_at" IS NULL WHERE "cellar_entries"\."id" \= \$1  (.+)`).
		WithArgs(100, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "cellar_id", "Beer__name", "Location__name", "Format__package", "Format__size_metric"}).
				AddRow(100, 10, "Tasty Beer", "Shelf 1", "Can", "330"))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "cellars"."id","cellars"."created_at","cellars"."updated_at","cellars"."deleted_at","cellars"."name","cellars"."description","cellars"."owner_id","Owner"."id" AS "Owner__id","Owner"."created_at" AS "Owner__created_at","Owner"."updated_at" AS "Owner__updated_at","Owner"."deleted_at" AS "Owner__deleted_at","Owner"."uuid" AS "Owner__uuid","Owner"."username" AS "Owner__username","Owner"."first_name" AS "Owner__first_name","Owner"."last_name" AS "Owner__last_name","Owner"."email" AS "Owner__email","Owner"."untappd_user_name" AS "Owner__untappd_user_name" FROM "cellars" LEFT JOIN "users" "Owner" ON "cellars"."owner_id" = "Owner"."id" AND "Owner"."deleted_at" IS NULL WHERE "cellars"."id" = $1 AND "cellars"."deleted_at" IS NULL ORDER BY "cellars"."id" LIMIT $2`)).
		WithArgs(10, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "description", "owner_id", "Owner__id", "Owner__username"}).
				AddRow(10, "Test Cellar", "my cellar description", 100, 100, "testuser"))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "location_in_cellars" WHERE "location_in_cellars"."cellar_id" = $1 AND "location_in_cellars"."deleted_at" IS NULL`)).
		WithArgs(10).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "cellar_id"}).
				AddRow(1, "Shelf 1", 10).
				AddRow(2, "Shelf 2", 10))

	cellarEntry, err := suite.repository.GetCellarEntryByID(context.Background(), 100)

	suite.Require().NoError(err)
	suite.NotNil(cellarEntry)
	suite.Equal(uint(100), cellarEntry.ID)
	suite.Equal(uint(10), cellarEntry.CellarID)
	suite.Equal("Test Cellar", cellarEntry.Cellar.Name)
	suite.Equal("testuser", cellarEntry.Cellar.Owner.Username)
	suite.Equal("Tasty Beer", cellarEntry.Beer.Name)
	suite.Equal("Shelf 1", cellarEntry.Location.Name)
	suite.Equal("Can", cellarEntry.Format.Package)
	suite.InDelta(330.0, cellarEntry.Format.SizeMetric, 0.1)
	suite.Len(cellarEntry.Cellar.Locations, 2)
}

func (suite *CellarTestSuite) TestGetCellarStats_GetsCellarStats() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT sum(quantity) as beer_count, count(distinct ce.beer_id) as unique_count, sum(bf.size_metric*quantity) as total_volume, count(distinct b.brewery_id) as brewery_count, sum(case when had_before = true then 0 else 1 end) as untried_count, sum(case when special = true then 1 else 0 end) as special_count, avg(b.abv) as average_abv, avg(b.external_rating) as average_rating FROM cellar_entries as ce INNER JOIN beer_formats bf on bf.id = ce.format_id INNER JOIN beers b on b.id = ce.beer_id WHERE cellar_id = $1`)).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"beer_count", "unique_count", "total_volume", "brewery_count", "untried_count", "average_abv", "average_rating"}).
			AddRow(10, 5, 3550, 2, 1, 9.8, 4.25))

	cellarStats, err := suite.repository.GetCellarStats(context.Background(), 100)

	suite.Require().NoError(err)
	suite.NotNil(cellarStats)
	suite.Equal(uint(100), cellarStats.CellarID)
	suite.Equal(uint64(10), cellarStats.BeerCount)
	suite.Equal(uint64(5), cellarStats.UniqueCount)
	suite.InDelta(3550.0, cellarStats.TotalVolume, 0.1)
	suite.Equal(uint64(2), cellarStats.BreweryCount)
	suite.Equal(uint64(1), cellarStats.UntriedCount)
	suite.InDelta(9.8, cellarStats.AverageABV, 0.01)
	suite.InDelta(4.25, cellarStats.AverageRating, 0.001)
}

func (suite *CellarTestSuite) TestGetCellarStats_ReturnError() {
	suite.mock.ExpectQuery("^SELECT (.+)").WillReturnError(gorm.ErrRecordNotFound)

	cellarStats, err := suite.repository.GetCellarStats(context.Background(), 999)

	suite.Nil(cellarStats)
	suite.EqualError(err, "record not found")
}

func (suite *CellarTestSuite) TestAddBeerToCellar_AddsBeer() {
	beer := model.CellarEntry{
		CellarID:   1,
		BeerID:     100,
		Vintage:    pointy.Uint64(2011),
		Quantity:   1,
		LocationID: pointy.Uint(1),
		FormatID:   pointy.Uint(1),
		HadBefore:  false,
		Special:    false,
		Tags:       nil,
	}

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cellar_entries" ("created_at","updated_at","deleted_at","cellar_id","beer_id","vintage","quantity","location_id","format_id","had_before","date_added","drink_before","cellar_until","special") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 1, 100, 2011, 1, 1, 1, false, nil, nil, nil, false).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(10)))
	suite.mock.ExpectCommit()

	cellarEntry, err := suite.repository.AddBeerToCellar(context.Background(), beer)
	suite.Require().NoError(err)
	suite.NotNil(cellarEntry)
	suite.Equal(uint(10), cellarEntry.ID)
}

func (suite *CellarTestSuite) TestDeleteCellarEntry_SoftDeletesEntry() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "cellar_entries" SET "deleted_at"=$1 WHERE "cellar_entries"."id" = $2 AND "cellar_entries"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnResult(sqlmock.NewResult(1, 1))
	suite.mock.ExpectCommit()

	err := suite.repository.DeleteCellarEntry(context.Background(), 10)
	suite.NoError(err)
}

func (suite *CellarTestSuite) TestUpdateCellarEntry_UpdatesCellarEntry() {
	beer := model.CellarEntry{
		Model:      gorm.Model{ID: 10},
		CellarID:   1,
		BeerID:     100,
		Vintage:    pointy.Uint64(2012),
		Quantity:   2,
		LocationID: pointy.Uint(2),
		FormatID:   pointy.Uint(3),
		HadBefore:  true,
		Special:    false,
		Tags:       nil,
	}

	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "cellar_entries" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"cellar_id"=$4,"beer_id"=$5,"vintage"=$6,"quantity"=$7,"location_id"=$8,"format_id"=$9,"had_before"=$10,"date_added"=$11,"drink_before"=$12,"cellar_until"=$13,"special"=$14 WHERE "cellar_entries"."deleted_at" IS NULL AND "id" = $15`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 1, 100, 2012, 2, 2, 3, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, 10).
		WillReturnResult(sqlmock.NewResult(10, 1))
	suite.mock.ExpectCommit()

	updatedEntry, err := suite.repository.UpdateCellarEntry(context.Background(), &beer)
	suite.Require().NoError(err)
	suite.NotNil(updatedEntry)
	suite.Equal(int64(2), updatedEntry.Quantity)
}

func (suite *CellarTestSuite) TestGetCellarBeers_GetsBeers() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "cellar_entries"."id","cellar_entries"."created_at","cellar_entries"."updated_at","cellar_entries"."deleted_at","cellar_entries"."cellar_id","cellar_entries"."beer_id","cellar_entries"."vintage","cellar_entries"."quantity","cellar_entries"."location_id","cellar_entries"."format_id","cellar_entries"."had_before","cellar_entries"."date_added","cellar_entries"."drink_before","cellar_entries"."cellar_until","cellar_entries"."special","Beer"."id" AS "Beer__id","Beer"."created_at" AS "Beer__created_at","Beer"."updated_at" AS "Beer__updated_at","Beer"."deleted_at" AS "Beer__deleted_at","Beer"."name" AS "Beer__name","Beer"."description" AS "Beer__description","Beer"."image_url" AS "Beer__image_url","Beer"."brewery_id" AS "Beer__brewery_id","Beer"."style_id" AS "Beer__style_id","Beer"."abv" AS "Beer__abv","Beer"."ibu" AS "Beer__ibu","Beer"."external_id" AS "Beer__external_id","Beer"."external_source" AS "Beer__external_source","Beer"."external_rating" AS "Beer__external_rating","Location"."id" AS "Location__id","Location"."created_at" AS "Location__created_at","Location"."updated_at" AS "Location__updated_at","Location"."deleted_at" AS "Location__deleted_at","Location"."name" AS "Location__name","Location"."cellar_id" AS "Location__cellar_id","Format"."id" AS "Format__id","Format"."created_at" AS "Format__created_at","Format"."updated_at" AS "Format__updated_at","Format"."deleted_at" AS "Format__deleted_at","Format"."package" AS "Format__package","Format"."size_metric" AS "Format__size_metric","Format"."size_imperial" AS "Format__size_imperial","Cellar"."id" AS "Cellar__id","Cellar"."created_at" AS "Cellar__created_at","Cellar"."updated_at" AS "Cellar__updated_at","Cellar"."deleted_at" AS "Cellar__deleted_at","Cellar"."name" AS "Cellar__name","Cellar"."description" AS "Cellar__description","Cellar"."owner_id" AS "Cellar__owner_id" FROM "cellar_entries" LEFT JOIN "beers" "Beer" ON "cellar_entries"."beer_id" = "Beer"."id" AND "Beer"."deleted_at" IS NULL LEFT JOIN "location_in_cellars" "Location" ON "cellar_entries"."location_id" = "Location"."id" AND "Location"."deleted_at" IS NULL LEFT JOIN "beer_formats" "Format" ON "cellar_entries"."format_id" = "Format"."id" AND "Format"."deleted_at" IS NULL LEFT JOIN "cellars" "Cellar" ON "cellar_entries"."cellar_id" = "Cellar"."id" AND "Cellar"."deleted_at" IS NULL WHERE cellar_entries.cellar_id = $1 AND "cellar_entries"."deleted_at" IS NULL`)).
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "quantity", "Beer__name"}).
				AddRow(uint(10), 2, "Pannepot").
				AddRow(uint(11), 1, "Pannepeut"))
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cellar_entry_tags" WHERE "cellar_entry_tags"."cellar_entry_id" IN ($1,$2)`)).
		WithArgs(10, 11).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	beers, err := suite.repository.GetCellarBeers(context.Background(), 1)
	suite.Require().NoError(err)
	suite.NotNil(beers)
	suite.Len(beers, 2)
	suite.Equal("Pannepot", beers[0].Beer.Name)
	suite.Equal("Pannepeut", beers[1].Beer.Name)
}

func (suite *CellarTestSuite) TestFindBeerRecommendations_FindsRecommendations() {
	expectedDate := time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "cellar_entries"."id","cellar_entries"."created_at","cellar_entries"."updated_at","cellar_entries"."deleted_at","cellar_entries"."cellar_id","cellar_entries"."beer_id","cellar_entries"."vintage","cellar_entries"."quantity","cellar_entries"."location_id","cellar_entries"."format_id","cellar_entries"."had_before","cellar_entries"."date_added","cellar_entries"."drink_before","cellar_entries"."cellar_until","cellar_entries"."special","Beer"."id" AS "Beer__id","Beer"."created_at" AS "Beer__created_at","Beer"."updated_at" AS "Beer__updated_at","Beer"."deleted_at" AS "Beer__deleted_at","Beer"."name" AS "Beer__name","Beer"."description" AS "Beer__description","Beer"."image_url" AS "Beer__image_url","Beer"."brewery_id" AS "Beer__brewery_id","Beer"."style_id" AS "Beer__style_id","Beer"."abv" AS "Beer__abv","Beer"."ibu" AS "Beer__ibu","Beer"."external_id" AS "Beer__external_id","Beer"."external_source" AS "Beer__external_source","Beer"."external_rating" AS "Beer__external_rating","Location"."id" AS "Location__id","Location"."created_at" AS "Location__created_at","Location"."updated_at" AS "Location__updated_at","Location"."deleted_at" AS "Location__deleted_at","Location"."name" AS "Location__name","Location"."cellar_id" AS "Location__cellar_id","Format"."id" AS "Format__id","Format"."created_at" AS "Format__created_at","Format"."updated_at" AS "Format__updated_at","Format"."deleted_at" AS "Format__deleted_at","Format"."package" AS "Format__package","Format"."size_metric" AS "Format__size_metric","Format"."size_imperial" AS "Format__size_imperial","Cellar"."id" AS "Cellar__id","Cellar"."created_at" AS "Cellar__created_at","Cellar"."updated_at" AS "Cellar__updated_at","Cellar"."deleted_at" AS "Cellar__deleted_at","Cellar"."name" AS "Cellar__name","Cellar"."description" AS "Cellar__description","Cellar"."owner_id" AS "Cellar__owner_id" FROM "cellar_entries" LEFT JOIN "beers" "Beer" ON "cellar_entries"."beer_id" = "Beer"."id" AND "Beer"."deleted_at" IS NULL LEFT JOIN "location_in_cellars" "Location" ON "cellar_entries"."location_id" = "Location"."id" AND "Location"."deleted_at" IS NULL LEFT JOIN "beer_formats" "Format" ON "cellar_entries"."format_id" = "Format"."id" AND "Format"."deleted_at" IS NULL LEFT JOIN "cellars" "Cellar" ON "cellar_entries"."cellar_id" = "Cellar"."id" AND "Cellar"."deleted_at" IS NULL WHERE cellar_entries.cellar_id = $1 AND "Beer".brewery_id = $2 AND "Beer".abv >= $3 AND "Beer".ABV <= $4 AND "Beer".external_rating >= $5 AND "Beer".external_rating <= $6 AND "Format".size_metric >= $7 AND "Format".size_metric <= $8 AND special = $9 AND had_before = $10 AND "Beer".style_id = $11 AND drink_before < $12 AND quantity >= $13 AND vintage >= $14 AND vintage <= $15 AND cellar_entries.id IN (SELECT cellar_entry_id FROM cellar_entry_tags INNER JOIN tags ON tag_id = tags.id WHERE tag IN ($16,$17) GROUP BY cellar_entry_id HAVING COUNT(*) = $18) AND date_added < $19 AND "cellar_entries"."deleted_at" IS NULL`)).
		WithArgs(1, 1, 4.0, 20.0, 3.5, 5.0, 330, 375, false, false, 1, sqlmock.AnyArg(), 1, 2011, 2020, "dark fruits", "sweet", 2, expectedDate).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "quantity", "Beer__name"}).
				AddRow(uint(10), 2, "Pannepot").
				AddRow(uint(11), 1, "Pannepeut"))
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cellar_entry_tags" WHERE "cellar_entry_tags"."cellar_entry_id" IN ($1,$2)`)).
		WithArgs(10, 11).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	filter := api.CellarFilter{
		BreweryId:       pointy.Uint64(1),
		MinimumAbv:      pointy.Float64(4.0),
		MaximumAbv:      pointy.Float64(20.0),
		MinimumRating:   pointy.Float64(3.5),
		MaximumRating:   pointy.Float64(5.0),
		MinimumSize:     pointy.Int64(330),
		MaximumSize:     pointy.Int64(375),
		Special:         pointy.Bool(false),
		HadBefore:       pointy.Bool(false),
		StyleId:         pointy.Uint64(1),
		OverdueToDrink:  pointy.Bool(false),
		MinimumQuantity: pointy.Int64(1),
		MinimumVintage:  pointy.Uint64(2011),
		MaximumVintage:  pointy.Uint64(2020),
		Tags:            []string{"dark fruits", "sweet"},
		AddedBefore:     timestamppb.New(expectedDate),
	}

	beers, err := suite.repository.FindBeerRecommendations(context.Background(), 1, &filter)
	suite.Require().NoError(err)
	suite.NotNil(beers)
}

func (suite *CellarTestSuite) TestGetCellarBreweryNames() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT breweries.id,breweries.name FROM "breweries" INNER JOIN beers b on breweries.id = b.brewery_id INNER JOIN cellar_entries ce on b.id = ce.beer_id WHERE ce.cellar_id = $1 AND "breweries"."deleted_at" IS NULL ORDER BY breweries.name asc`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uint(1), "Brouwerij de Molen").
			AddRow(uint(10), "Fremont Brewing").
			AddRow(uint(20), "Temporal Artisan Ales"))

	names, err := suite.repository.GetCellarBreweryNames(context.Background(), 1)
	suite.Require().NoError(err)
	suite.Len(names, 3)
	suite.Equal("Brouwerij de Molen", names[0].Name)
	suite.Equal("Fremont Brewing", names[1].Name)
	suite.Equal("Temporal Artisan Ales", names[2].Name)
}

func (suite *CellarTestSuite) TestGetCellarStyles() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT beer_styles.id,beer_styles.name FROM "beer_styles" INNER JOIN beers b on beer_styles.id = b.style_id INNER JOIN cellar_entries ce on b.id = ce.beer_id WHERE ce.cellar_id = $1 AND "beer_styles"."deleted_at" IS NULL ORDER BY beer_styles.name`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uint(1), "Lambic - Kriek").
			AddRow(uint(2), "Stout - Imperial / Double Coffee"))

	styles, err := suite.repository.GetCellarStyles(context.Background(), 1)
	suite.Require().NoError(err)
	suite.Len(styles, 2)
	suite.Equal("Lambic - Kriek", styles[0].Name)
	suite.Equal("Stout - Imperial / Double Coffee", styles[1].Name)
}

func (suite *CellarTestSuite) TestSaveAdventCalendar() {
	startDate := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)

	calendar := model.AdventCalendar{
		CellarID:    1,
		Name:        "Christmas Calendar",
		Description: "An advent calendar for December",
		StartDate:   startDate,
		EndDate:     endDate,
		Beers: []model.AdventCalendarBeer{
			{
				CellarEntryID: 10,
				FilterID:      1,
				Day:           startDate,
				Revealed:      false,
				Filter: model.AdventCalendarFilter{
					MaximumRating: pointy.Float64(6.5),
				},
			},
		},
	}

	suite.mock.ExpectBegin()
	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "advent_calendars" ("created_at","updated_at","deleted_at","cellar_id","name","description","start_date","end_date") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, uint(1), "Christmas Calendar", "An advent calendar for December", startDate, endDate).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(5)))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "advent_calendar_filters" ("created_at","updated_at","deleted_at","brewery_id","minimum_abv","maximum_abv","style_id","minimum_vintage","maximum_vintage","overdue_to_drink","had_before","special","minimum_quantity","minimum_size","maximum_size","minimum_rating","maximum_rating","added_before") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) ON CONFLICT DO NOTHING RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 6.5, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(1)))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "advent_calendar_beers" ("created_at","updated_at","deleted_at","advent_calendar_id","cellar_entry_id","filter_id","day","revealed") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT ("id") DO UPDATE SET "advent_calendar_id"="excluded"."advent_calendar_id" RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, uint(5), uint(10), uint(1), startDate, false).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(1)))

	suite.mock.ExpectCommit()

	result, err := suite.repository.SaveAdventCalendar(context.Background(), calendar)

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Equal(uint(5), result.ID)
	suite.Equal("Christmas Calendar", result.Name)
	suite.Equal(uint(1), result.CellarID)
	suite.Len(result.Beers, 1)
}

func (suite *CellarTestSuite) TestGetAdventCalendarByID() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "advent_calendars" WHERE cellar_id = $1 AND "advent_calendars"."id" = $2 AND "advent_calendars"."deleted_at" IS NULL ORDER BY "advent_calendars"."id" LIMIT $3`)).
		WithArgs(1, 1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(1)))

	suite.expectReadFromAssociatedTables()

	calendar, err := suite.repository.GetAdventCalendarByID(context.Background(), 1, 1)
	suite.Require().NoError(err)
	suite.NotNil(calendar)
	suite.Equal(uint(1), calendar.ID)

	suite.Len(calendar.Beers, 1)
}

func (suite *CellarTestSuite) TestGetAdventCalendarForDate() {
	testDate := time.Date(2022, 9, 3, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "advent_calendars" WHERE cellar_id = $1 AND ($2 between start_date and end_date) AND "advent_calendars"."deleted_at" IS NULL ORDER BY "advent_calendars"."id" LIMIT $3`)).
		WithArgs(1, testDate, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(1)))

	suite.expectReadFromAssociatedTables()

	calendar, err := suite.repository.GetAdventCalendarForDate(context.Background(), 1, testDate)
	suite.Require().NoError(err)
	suite.NotNil(calendar)
	suite.Equal(uint(1), calendar.ID)

	suite.Len(calendar.Beers, 1)
}

func (suite *CellarTestSuite) TestGetAdventCalendarByName() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "advent_calendars" WHERE cellar_id = $1 AND name = $2 AND "advent_calendars"."deleted_at" IS NULL ORDER BY "advent_calendars"."id" LIMIT $3`)).
		WithArgs(1, "Exciting Advent Calendar", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(1)))

	suite.expectReadFromAssociatedTables()

	calendar, err := suite.repository.GetAdventCalendarByName(context.Background(), 1, "Exciting Advent Calendar")
	suite.Require().NoError(err)
	suite.NotNil(calendar)
	suite.Equal(uint(1), calendar.ID)

	suite.Len(calendar.Beers, 1)
}

func (suite *CellarTestSuite) expectReadFromAssociatedTables() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "advent_calendar_beers" WHERE "advent_calendar_beers"."advent_calendar_id" = $1 AND "advent_calendar_beers"."deleted_at" IS NULL`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cellar_entry_id", "advent_calendar_id"}).
			AddRow(uint(1), uint(10), uint(1)))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cellar_entries" WHERE "cellar_entries"."id" = $1`)).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(10)))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cellar_entry_tags" WHERE "cellar_entry_tags"."cellar_entry_id" = $1`)).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"cellar_entry_id"}).
			AddRow(uint(10)))
}

func (suite *CellarTestSuite) TestGetCellarRecommendationRanges() {
	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT min(b.abv) as minimum_abv, max(b.abv) as maximum_abv, min(bf.size_metric) as minimum_size, max(bf.size_metric) as maximum_size, min(ce.vintage) as minimum_vintage, max(ce.vintage) as maximum_vintage, round(min(b.external_rating), 2) as minimum_rating, round(max(b.external_rating), 2) as maximum_rating, min(ce.date_added) as oldest_added_date FROM cellar_entries ce INNER JOIN beers b on b.id = ce.beer_id INNER JOIN beer_formats bf on ce.format_id = bf.id WHERE ce.cellar_id = $1 LIMIT $2`)).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"minimum_abv", "maximum_abv", "minimum_size", "maximum_size", "minimum_vintage", "maximum_vintage", "minimum_rating", "maximum_rating", "oldest_added_date"}).
			AddRow(3.5, 12.0, 330, 750, 2018, 2023, 3.0, 5.0, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)))

	ranges, err := suite.repository.GetCellarRecommendationRanges(context.Background(), 1)

	suite.Require().NoError(err)
	suite.NotNil(ranges)
	suite.InDelta(3.5, ranges.MinimumAbv, 0.1)
	suite.InDelta(12.0, ranges.MaximumAbv, 0.1)
	suite.Equal(int64(330), ranges.MinimumSize)
	suite.Equal(int64(750), ranges.MaximumSize)
	suite.Equal(uint64(2018), ranges.MinimumVintage)
	suite.Equal(uint64(2023), ranges.MaximumVintage)
	suite.InDelta(3.0, ranges.MinimumRating, 0.1)
	suite.InDelta(5.0, ranges.MaximumRating, 0.1)
	suite.Equal(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), ranges.OldestAddedDate)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendar() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE advent_calendar_beers SET revealed = NOT revealed, updated_at = CURRENT_TIMESTAMP FROM advent_calendars WHERE advent_calendar_beers.advent_calendar_id = advent_calendars.id AND advent_calendar_id = $1 AND advent_calendars.cellar_id = $2 AND day = $3`)).
		WithArgs(1, 1, day).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := suite.repository.UpdateAdventCalendar(context.Background(), 1, 1, day)

	suite.NoError(err)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendar_Error() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE advent_calendar_beers SET revealed = NOT revealed, updated_at = CURRENT_TIMESTAMP FROM advent_calendars WHERE advent_calendar_beers.advent_calendar_id = advent_calendars.id AND advent_calendar_id = $1 AND advent_calendars.cellar_id = $2 AND day = $3`)).
		WithArgs(1, 1, day).
		WillReturnError(gorm.ErrRecordNotFound)

	err := suite.repository.UpdateAdventCalendar(context.Background(), 1, 1, day)

	suite.Require().Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendarEntry() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE advent_calendar_beers SET cellar_entry_id = $1, updated_at = CURRENT_TIMESTAMP, revealed = false FROM advent_calendars WHERE advent_calendar_beers.advent_calendar_id = advent_calendars.id AND advent_calendar_id = $2 AND advent_calendars.cellar_id = $3 AND day = $4`)).
		WithArgs(25, 1, 1, day).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := suite.repository.UpdateAdventCalendarEntry(context.Background(), 1, 1, day, 25)

	suite.NoError(err)
}

func (suite *CellarTestSuite) TestUpdateAdventCalendarEntry_Error() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectExec(regexp.QuoteMeta(`UPDATE advent_calendar_beers SET cellar_entry_id = $1, updated_at = CURRENT_TIMESTAMP, revealed = false FROM advent_calendars WHERE advent_calendar_beers.advent_calendar_id = advent_calendars.id AND advent_calendar_id = $2 AND advent_calendars.cellar_id = $3 AND day = $4`)).
		WithArgs(25, 1, 1, day).
		WillReturnError(errors.New("database error"))

	err := suite.repository.UpdateAdventCalendarEntry(context.Background(), 1, 1, day, 25)

	suite.Require().Error(err)
	suite.Contains(err.Error(), "database error")
}

func (suite *CellarTestSuite) TestDeleteAdventCalendar() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM advent_calendar_beers WHERE advent_calendar_id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 5))
	suite.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM advent_calendars WHERE id = $1 AND cellar_id = $2`)).
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	suite.mock.ExpectCommit()

	err := suite.repository.DeleteAdventCalendar(context.Background(), 1, 1)

	suite.NoError(err)
}

func (suite *CellarTestSuite) TestDeleteAdventCalendar_BeerDeletionError() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM advent_calendar_beers WHERE advent_calendar_id = $1`)).
		WithArgs(1).
		WillReturnError(errors.New("database error"))
	suite.mock.ExpectRollback()

	err := suite.repository.DeleteAdventCalendar(context.Background(), 1, 1)

	suite.ErrorContains(err, "database error")
}

func (suite *CellarTestSuite) TestDeleteAdventCalendar_CalendarDeletionError() {
	suite.mock.ExpectBegin()
	suite.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM advent_calendar_beers WHERE advent_calendar_id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 5))
	suite.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM advent_calendars WHERE id = $1 AND cellar_id = $2`)).
		WithArgs(1, 1).
		WillReturnError(errors.New("calendar deletion error"))
	suite.mock.ExpectRollback()

	err := suite.repository.DeleteAdventCalendar(context.Background(), 1, 1)

	suite.ErrorContains(err, "calendar deletion error")
}

func (suite *CellarTestSuite) TestGetAdventCalendarFilter() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "advent_calendar_filters"."id","advent_calendar_filters"."created_at","advent_calendar_filters"."updated_at","advent_calendar_filters"."deleted_at","advent_calendar_filters"."brewery_id","advent_calendar_filters"."minimum_abv","advent_calendar_filters"."maximum_abv","advent_calendar_filters"."style_id","advent_calendar_filters"."minimum_vintage","advent_calendar_filters"."maximum_vintage","advent_calendar_filters"."overdue_to_drink","advent_calendar_filters"."had_before","advent_calendar_filters"."special","advent_calendar_filters"."minimum_quantity","advent_calendar_filters"."minimum_size","advent_calendar_filters"."maximum_size","advent_calendar_filters"."minimum_rating","advent_calendar_filters"."maximum_rating","advent_calendar_filters"."added_before" FROM "advent_calendar_filters" JOIN advent_calendar_beers ON advent_calendar_beers.filter_id = advent_calendar_filters.id JOIN advent_calendars ON advent_calendars.id = advent_calendar_beers.advent_calendar_id WHERE advent_calendars.cellar_id = $1 AND advent_calendar_beers.advent_calendar_id = $2 AND advent_calendar_beers.day = $3 AND "advent_calendar_filters"."deleted_at" IS NULL ORDER BY "advent_calendar_filters"."id" LIMIT $4`)).
		WithArgs(1, 1, day, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "minimum_abv", "maximum_abv"}).
			AddRow(1, 5.0, 10.0))

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "advent_calendar_filter_tags" WHERE "advent_calendar_filter_tags"."advent_calendar_filter_id" = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	filter, err := suite.repository.GetAdventCalendarFilter(context.Background(), 1, 1, day)

	suite.Require().NoError(err)
	suite.NotNil(filter)
	suite.Equal(uint(1), filter.ID)
	suite.NotNil(filter.MinimumAbv)
	suite.InDelta(5.0, *filter.MinimumAbv, 0.1)
	suite.NotNil(filter.MaximumAbv)
	suite.InDelta(10.0, *filter.MaximumAbv, 0.1)
}

func (suite *CellarTestSuite) TestGetAdventCalendarFilter_NotFound() {
	day := time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC)

	suite.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "advent_calendar_filters"."id","advent_calendar_filters"."created_at","advent_calendar_filters"."updated_at","advent_calendar_filters"."deleted_at","advent_calendar_filters"."brewery_id","advent_calendar_filters"."minimum_abv","advent_calendar_filters"."maximum_abv","advent_calendar_filters"."style_id","advent_calendar_filters"."minimum_vintage","advent_calendar_filters"."maximum_vintage","advent_calendar_filters"."overdue_to_drink","advent_calendar_filters"."had_before","advent_calendar_filters"."special","advent_calendar_filters"."minimum_quantity","advent_calendar_filters"."minimum_size","advent_calendar_filters"."maximum_size","advent_calendar_filters"."minimum_rating","advent_calendar_filters"."maximum_rating","advent_calendar_filters"."added_before" FROM "advent_calendar_filters" JOIN advent_calendar_beers ON advent_calendar_beers.filter_id = advent_calendar_filters.id JOIN advent_calendars ON advent_calendars.id = advent_calendar_beers.advent_calendar_id WHERE advent_calendars.cellar_id = $1 AND advent_calendar_beers.advent_calendar_id = $2 AND advent_calendar_beers.day = $3 AND "advent_calendar_filters"."deleted_at" IS NULL ORDER BY "advent_calendar_filters"."id" LIMIT $4`)).
		WithArgs(1, 1, day, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	filter, err := suite.repository.GetAdventCalendarFilter(context.Background(), 1, 1, day)

	suite.Require().ErrorIs(err, gorm.ErrRecordNotFound)
	suite.Nil(filter)
}
