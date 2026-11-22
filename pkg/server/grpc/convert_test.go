package grpc_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server/grpc"
)

type ConvertTestSuite struct {
	suite.Suite
}

func TestConvertTestSuite(t *testing.T) {
	suite.Run(t, new(ConvertTestSuite))
}

func (suite *ConvertTestSuite) TestBeerStyleFromModel_AccuratelyConvertsWithFullDetails() {
	// Arrange - Create a complete BeerStyle model with nested BJCP Style, Category, and Family
	beerStyle := model.BeerStyle{
		Model: gorm.Model{
			ID: 42,
		},
		Name:        "American IPA",
		BJCPStyleID: "21A",
		BJCPStyle: model.BeerStyleBJCP{
			BJCPID: "21A",
			Name:   "American IPA",
			Category: model.BeerCategoryBJCP{
				Model: gorm.Model{ID: 21},
				Name:  "IPA",
			},
			CategoryID: 21,
			Family: model.BeerStyleFamily{
				Model: gorm.Model{ID: 3},
				Name:  "Hoppy",
			},
			FamilyID: 3,
		},
	}

	// Act
	result := grpc.BeerStyleFromModel(beerStyle)

	// Assert - Verify all fields are converted correctly
	suite.Require().NotNil(result)
	suite.Equal(uint64(42), result.Id)
	suite.Equal("American IPA", result.Name)

	suite.Require().NotNil(result.BjcpStyle)
	suite.Equal("21A", result.BjcpStyle.Id)
	suite.Equal("American IPA", result.BjcpStyle.Name)
	suite.Equal("21. IPA", result.BjcpStyle.Category)

	suite.Require().NotNil(result.BjcpStyle.Family)
	suite.Equal(uint64(3), result.BjcpStyle.Family.Id)
	suite.Equal("Hoppy", result.BjcpStyle.Family.Name)
}

func (suite *ConvertTestSuite) TestBeerStyleFromModel_HandlesComplexCategory() {
	// Arrange - Test with a different category to ensure the format is correct
	beerStyle := model.BeerStyle{
		Model: gorm.Model{
			ID: 10,
		},
		Name:        "Belgian Dubbel",
		BJCPStyleID: "26B",
		BJCPStyle: model.BeerStyleBJCP{
			BJCPID: "26B",
			Name:   "Belgian Dubbel",
			Category: model.BeerCategoryBJCP{
				Model: gorm.Model{ID: 26},
				Name:  "Monastic Ale",
			},
			CategoryID: 26,
			Family: model.BeerStyleFamily{
				Model: gorm.Model{ID: 5},
				Name:  "Malty",
			},
			FamilyID: 5,
		},
	}

	// Act
	result := grpc.BeerStyleFromModel(beerStyle)

	// Assert - Verify category formatting with ID. Name pattern
	suite.Require().NotNil(result)
	suite.Equal("26. Monastic Ale", result.BjcpStyle.Category)
	suite.Equal("Belgian Dubbel", result.Name)
	suite.Equal("Malty", result.BjcpStyle.Family.Name)
}

func (suite *ConvertTestSuite) TestYearInReviewFromModel_MapsHighlights() {
	strongAbv := 11.5
	gentleAbv := 2.8
	highRating := 4.7
	lowRating := 2.1
	yir := &model.YearInReview{
		Year:       2026,
		CellarID:   7,
		BiggestDay: &model.DayCount{Date: "2026-06-14", Count: 9},
		MostHadBeer: &model.YearInReviewBeer{
			BeerID: 1, Name: "Most", BreweryName: "BrewA", StyleName: "IPA", Count: 12,
		},
		StrongestBeer: &model.YearInReviewBeer{
			BeerID: 2, Name: "Strong", BreweryName: "BrewB", StyleName: "Imperial Stout", ABV: &strongAbv, Count: 1,
		},
		GentlestBeer: &model.YearInReviewBeer{
			BeerID: 3, Name: "Gentle", BreweryName: "BrewC", StyleName: "Berliner Weisse", ABV: &gentleAbv, Count: 2,
		},
		HighestRatedBeer: &model.YearInReviewBeer{
			BeerID: 4, Name: "Best", BreweryName: "BrewD", StyleName: "Saison", ExternalRating: &highRating, Count: 3,
		},
		LowestRatedBeer: &model.YearInReviewBeer{
			BeerID: 5, Name: "Worst", BreweryName: "BrewE", StyleName: "Lager", ExternalRating: &lowRating, Count: 4,
		},
	}

	result := grpc.YearInReviewFromModel(yir)

	suite.Require().NotNil(result.BiggestDay)
	suite.Equal("2026-06-14", result.BiggestDay.Date)
	suite.Equal(int64(9), result.BiggestDay.Count)
	suite.Require().NotNil(result.MostHadBeer)
	suite.Equal(int64(12), result.MostHadBeer.Count)
	suite.Equal(uint64(1), result.MostHadBeer.BeerId)
	suite.Equal("Most", result.MostHadBeer.Name)
	suite.Equal("BrewA", result.MostHadBeer.BreweryName)
	suite.Equal("IPA", result.MostHadBeer.StyleName)
	suite.Require().NotNil(result.StrongestBeer)
	suite.Require().NotNil(result.StrongestBeer.Abv)
	suite.InDelta(11.5, *result.StrongestBeer.Abv, 0.0001)
	suite.Require().NotNil(result.GentlestBeer)
	suite.Require().NotNil(result.GentlestBeer.Abv)
	suite.InDelta(2.8, *result.GentlestBeer.Abv, 0.0001)
	suite.Require().NotNil(result.HighestRatedBeer)
	suite.Require().NotNil(result.HighestRatedBeer.ExternalRating)
	suite.InDelta(4.7, *result.HighestRatedBeer.ExternalRating, 0.0001)
	suite.Require().NotNil(result.LowestRatedBeer)
	suite.Require().NotNil(result.LowestRatedBeer.ExternalRating)
	suite.InDelta(2.1, *result.LowestRatedBeer.ExternalRating, 0.0001)
}

func (suite *ConvertTestSuite) TestYearInReviewFromModel_OmitsAbsentHighlights() {
	result := grpc.YearInReviewFromModel(&model.YearInReview{Year: 2026})

	suite.Nil(result.BiggestDay)
	suite.Nil(result.MostHadBeer)
	suite.Nil(result.StrongestBeer)
	suite.Nil(result.GentlestBeer)
	suite.Nil(result.HighestRatedBeer)
	suite.Nil(result.LowestRatedBeer)
}
