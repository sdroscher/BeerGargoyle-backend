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
