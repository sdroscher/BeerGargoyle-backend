package server_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

type ActivityTestSuite struct {
	suite.Suite
	activityRepo *mocks.ActivityRepository
	cellarRepo   *mocks.CellarRepository
	service      *server.CellarServer
}

func TestActivityTestSuite(t *testing.T) {
	suite.Run(t, new(ActivityTestSuite))
}

func (suite *ActivityTestSuite) SetupTest() {
	suite.activityRepo = mocks.NewActivityRepository(suite.T())
	suite.cellarRepo = mocks.NewCellarRepository(suite.T())
	suite.service = server.NewCellarServer(suite.cellarRepo, nil, suite.activityRepo, zap.NewNop())
}

func (suite *ActivityTestSuite) TestRecordConsumption_Success() {
	ctx := ctxWithUser(1)
	format := &model.BeerFormat{Package: "Can", SizeMetric: 473}
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 4,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
		Format:   format,
	}
	entryID := uint(10)
	activity := &model.Activity{
		Model:         gorm.Model{ID: 1},
		CellarID:      1,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      2,
	}
	activityMatcher := mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 2
	})
	entryAfterDecrement := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 2,
		Beer:     entry.Beer,
		Format:   format,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, entryAfterDecrement, activityMatcher).Return(activity, nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      2,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Consumption)
	suite.Equal(apiv1.ActivityEventType_ACTIVITY_EVENT_TYPE_BEER_CONSUMED, resp.Msg.Consumption.Type)
	suite.Equal(int64(2), resp.Msg.Consumption.Quantity)
	suite.NotNil(resp.Msg.Consumption.CellarBeer)
}

func (suite *ActivityTestSuite) TestRecordConsumption_LastBottle() {
	ctx := ctxWithUser(1)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 1,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}
	entryID := uint(10)
	activity := &model.Activity{
		Model:         gorm.Model{ID: 1},
		CellarID:      1,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      1,
	}

	activityMatcher := mock.MatchedBy(func(act *model.Activity) bool {
		return act.CellarID == 1 &&
			act.CellarEntryID != nil && *act.CellarEntryID == 10 &&
			act.ActivityType == model.ActivityTypeBeerConsumed &&
			act.Quantity == 1
	})
	entryAfterDecrement := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 0,
		Beer:     entry.Beer,
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, entryAfterDecrement, activityMatcher).Return(activity, nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      1,
	}))

	suite.Require().NoError(err)
	suite.NotNil(resp.Msg.Consumption)
	suite.NotNil(resp.Msg.Consumption.CellarBeer)
}

func (suite *ActivityTestSuite) TestGetActivityFeed_Basic() {
	ctx := ctxWithUser(1)
	entryID := uint(10)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 3,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}
	activities := []*model.Activity{
		{
			Model:         gorm.Model{ID: 1},
			CellarID:      1,
			CellarEntryID: &entryID,
			ActivityType:  model.ActivityTypeBeerConsumed,
			Quantity:      1,
			CellarEntry:   entry,
		},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetFeed(ctx, uint(1), (*time.Time)(nil), (*time.Time)(nil), 25, 0).
		Return(activities, int64(1), nil)

	resp, err := suite.service.GetActivityFeed(ctx, connect.NewRequest(&apiv1.GetActivityFeedRequest{
		CellarId: 1,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(1), resp.Msg.Total)
	suite.False(resp.Msg.HasMore)
	suite.Len(resp.Msg.Events, 1)
	suite.Equal(apiv1.ActivityEventType_ACTIVITY_EVENT_TYPE_BEER_CONSUMED, resp.Msg.Events[0].Type)
}

func (suite *ActivityTestSuite) TestGetActivityFeed_Pagination() {
	ctx := ctxWithUser(1)
	entryID := uint(10)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF"},
	}

	activities := make([]*model.Activity, 0, 5)

	for range 5 {
		activities = append(activities, &model.Activity{
			CellarID:      1,
			CellarEntryID: &entryID,
			ActivityType:  model.ActivityTypeBeerConsumed,
			Quantity:      1,
			CellarEntry:   entry,
		})
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetFeed(ctx, uint(1), (*time.Time)(nil), (*time.Time)(nil), 5, 5).
		Return(activities, int64(15), nil)

	resp, err := suite.service.GetActivityFeed(ctx, connect.NewRequest(&apiv1.GetActivityFeedRequest{
		CellarId: 1,
		PageSize: 5,
		Page:     2,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(15), resp.Msg.Total)
	suite.True(resp.Msg.HasMore)
	suite.Len(resp.Msg.Events, 5)
}

func (suite *ActivityTestSuite) TestRecordConsumption_ConsumeEntryFails() {
	ctx := ctxWithUser(1)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 3,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}},
	}
	dbErr := errors.New("db error")

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, mock.Anything, mock.Anything).Return(nil, dbErr)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      1,
	}))

	suite.Require().Error(err)
	suite.Nil(resp)
}

func (suite *ActivityTestSuite) TestRecordConsumption_ExceedsQuantity() {
	ctx := ctxWithUser(1)
	entry := &model.CellarEntry{
		Model:    gorm.Model{ID: 10},
		CellarID: 1,
		Quantity: 2,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}},
	}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, uint(10)).Return(entry, nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	resp, err := suite.service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{
		CellarEntryId: 10,
		Quantity:      5,
	}))

	suite.Require().Error(err)
	suite.Require().ErrorIs(err, server.ErrInvalidQuantity)
	suite.Nil(resp)
}

func (suite *ActivityTestSuite) TestGetYearInReview_ExplicitYear() {
	ctx := ctxWithUser(1)
	yir := &model.YearInReview{
		Year:          2024,
		CellarID:      1,
		BeersConsumed: 42,
		UniqueBeers:   15,
		TotalVolumeMl: 19866,
		AverageRating: 4.1,
		BeersAdded:    60,
		TopStyles: []model.NameCount{
			{Name: "American IPA", Count: 18},
			{Name: "Imperial Stout", Count: 10},
		},
		TopCategories: []model.NameCount{
			{Name: "IPA", Count: 28},
			{Name: "Dark Beer", Count: 10},
		},
		TopBreweries: []model.NameCount{
			{Name: "Boombox", Count: 20},
		},
		ByMonth: []model.MonthlyCount{
			{Month: 1, Count: 3},
			{Month: 6, Count: 12},
		},
	}

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetYearInReview(ctx, uint(1), 2024).Return(yir, nil)

	resp, err := suite.service.GetYearInReview(ctx, connect.NewRequest(&apiv1.GetYearInReviewRequest{
		CellarId: 1,
		Year:     2024,
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(2024), resp.Msg.YearInReview.Year)
	suite.Equal(int64(42), resp.Msg.YearInReview.BeersConsumed)
	suite.Equal(int64(15), resp.Msg.YearInReview.UniqueBeers)
	suite.Equal(int64(60), resp.Msg.YearInReview.BeersAdded)
	suite.InDelta(4.1, resp.Msg.YearInReview.AverageRating, 0.001)
	suite.Len(resp.Msg.YearInReview.TopStyles, 2)
	suite.Equal("American IPA", resp.Msg.YearInReview.TopStyles[0].Name)
	suite.Len(resp.Msg.YearInReview.TopCategories, 2)
	suite.Equal("IPA", resp.Msg.YearInReview.TopCategories[0].Name)
	suite.Len(resp.Msg.YearInReview.TopBreweries, 1)
	suite.Len(resp.Msg.YearInReview.ByMonth, 2)
}

func (suite *ActivityTestSuite) TestGetYearInReview_DefaultsToCurrentYear() {
	ctx := ctxWithUser(1)
	currentYear := time.Now().UTC().Year()

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetYearInReview(ctx, uint(1), currentYear).
		Return(&model.YearInReview{Year: currentYear, CellarID: 1}, nil)

	resp, err := suite.service.GetYearInReview(ctx, connect.NewRequest(&apiv1.GetYearInReviewRequest{
		CellarId: 1,
	}))

	suite.Require().NoError(err)
	suite.Greater(resp.Msg.YearInReview.Year, int32(2000))
}

// ── ImportUntappdCheckins ─────────────────────────────────────────────────────

const untappdCSVHeader = "beer_name,brewery_name,beer_type,beer_abv,beer_ibu,comment,venue_name,venue_city,venue_state,venue_country,venue_lat,venue_lng,rating_score,created_at,checkin_url,beer_url,brewery_url,brewery_country,brewery_city,brewery_state,flavor_profiles,purchase_venue,serving_type,checkin_id,bid,brewery_id,photo_url,global_rating_score,global_weighted_rating_score,tagged_friends,total_toasts,total_comments"

// untappdCSVRow builds a CSV data row with values at the correct column positions (32 total columns).
func untappdCSVRow(beerName, breweryName, servingType, createdAt, checkinID, bid string) string {
	fields := make([]string, 32)
	fields[0] = beerName
	fields[1] = breweryName
	fields[2] = "Stout"
	fields[3] = "10"
	fields[4] = "80"
	fields[13] = createdAt
	fields[22] = servingType
	fields[23] = checkinID
	fields[24] = bid
	fields[25] = "123"

	return strings.Join(fields, ",")
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_Success() {
	ctx := ctxWithUser(1)

	extID := uint64(14534)
	beer := model.Beer{Model: gorm.Model{ID: 5}, Name: "Juicy AF", ExternalID: &extID, ExternalSource: strPtr("untappd"), Brewery: model.Brewery{Name: "Boombox"}}
	entryID := uint(10)
	entries := []*model.CellarEntry{
		{Model: gorm.Model{ID: entryID}, CellarID: 1, BeerID: 5, Beer: beer},
	}

	state := &model.UntappdImportState{CellarID: 1, LastImportAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}

	csvData := untappdCSVHeader + "\n" + untappdCSVRow("Juicy AF", "Boombox", "Can", "2024-03-01 12:00:00", "9501642", "14534")

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetOrInitImportState(ctx, uint(1)).Return(state, nil)
	suite.activityRepo.EXPECT().GetUntappdCheckinIDsForCellar(ctx, uint(1)).Return(map[uint64]struct{}{}, nil)
	suite.cellarRepo.EXPECT().GetAllEntriesForCellar(ctx, uint(1)).Return(entries, nil)
	suite.activityRepo.EXPECT().GetConsumedActivitiesForCellar(ctx, uint(1)).Return(nil, nil)
	suite.activityRepo.EXPECT().ImportActivitiesWithState(ctx, mock.Anything, mock.Anything, state, mock.Anything).Return(int64(1), int64(0), nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(csvData),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(1), resp.Msg.Imported)
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_AllSkippedWrongServingType() {
	ctx := ctxWithUser(1)

	state := &model.UntappdImportState{CellarID: 1, LastImportAt: time.Time{}}

	csvData := untappdCSVHeader + "\n" +
		untappdCSVRow("Draft Beer", "Brewery", "Draft", "2024-03-01 12:00:00", "1", "100") + "\n" +
		untappdCSVRow("Nitro Beer", "Brewery", "Nitro", "2024-03-02 12:00:00", "2", "101")

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetOrInitImportState(ctx, uint(1)).Return(state, nil)
	suite.activityRepo.EXPECT().GetUntappdCheckinIDsForCellar(ctx, uint(1)).Return(map[uint64]struct{}{}, nil)
	suite.cellarRepo.EXPECT().GetAllEntriesForCellar(ctx, uint(1)).Return(nil, nil)
	suite.activityRepo.EXPECT().GetConsumedActivitiesForCellar(ctx, uint(1)).Return(nil, nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(csvData),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(0), resp.Msg.Imported)
	suite.Equal(int32(2), resp.Msg.Skipped)
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_InvalidCSV() {
	ctx := ctxWithUser(1)

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte("not,a,valid,csv,header\nrow"),
	}))

	suite.Require().Error(err)
	suite.Nil(resp)
	suite.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_EmptyCSV() {
	ctx := ctxWithUser(1)

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(untappdCSVHeader),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(0), resp.Msg.Imported)
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_MatchesByNameFallback() {
	ctx := ctxWithUser(1)

	beer := model.Beer{Model: gorm.Model{ID: 7}, Name: "Juicy AF", Brewery: model.Brewery{Name: "Boombox"}}
	entries := []*model.CellarEntry{
		{Model: gorm.Model{ID: 20}, CellarID: 1, BeerID: 7, Beer: beer},
	}

	state := &model.UntappdImportState{CellarID: 1, LastImportAt: time.Time{}}

	// Row has bid=0 so will fall back to name matching
	csvData := untappdCSVHeader + "\n" + untappdCSVRow("Juicy AF", "Boombox", "Bottle", "2024-03-01 12:00:00", "9999", "0")

	var capturedActivities []*model.Activity
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetOrInitImportState(ctx, uint(1)).Return(state, nil)
	suite.activityRepo.EXPECT().GetUntappdCheckinIDsForCellar(ctx, uint(1)).Return(map[uint64]struct{}{}, nil)
	suite.cellarRepo.EXPECT().GetAllEntriesForCellar(ctx, uint(1)).Return(entries, nil)
	suite.activityRepo.EXPECT().GetConsumedActivitiesForCellar(ctx, uint(1)).Return(nil, nil)
	suite.activityRepo.EXPECT().ImportActivitiesWithState(ctx, mock.MatchedBy(func(acts []*model.Activity) bool {
		capturedActivities = acts

		return len(acts) == 1
	}), mock.Anything, state, mock.Anything).Return(int64(1), int64(0), nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(csvData),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(1), resp.Msg.Imported)
	suite.Require().NotEmpty(capturedActivities)
	suite.Equal(uint(20), *capturedActivities[0].CellarEntryID)
}

func strPtr(s string) *string { return &s }

func (suite *ActivityTestSuite) TestImportUntappdCheckins_WithBOM() {
	ctx := ctxWithUser(1)
	state := &model.UntappdImportState{CellarID: 1, LastImportAt: time.Time{}}

	extID := uint64(22222)
	beer := model.Beer{Model: gorm.Model{ID: 3}, Name: "Some Beer", ExternalID: &extID, ExternalSource: strPtr("untappd"), Brewery: model.Brewery{Name: "Brewery"}}
	entries := []*model.CellarEntry{
		{Model: gorm.Model{ID: 99}, CellarID: 1, BeerID: 3, Beer: beer},
	}

	// Prepend UTF-8 BOM
	csvData := "\xef\xbb\xbf" + untappdCSVHeader + "\n" + untappdCSVRow("Some Beer", "Brewery", "Can", "2024-03-01 12:00:00", "11111", "22222")

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetOrInitImportState(ctx, uint(1)).Return(state, nil)
	suite.activityRepo.EXPECT().GetUntappdCheckinIDsForCellar(ctx, uint(1)).Return(map[uint64]struct{}{}, nil)
	suite.cellarRepo.EXPECT().GetAllEntriesForCellar(ctx, uint(1)).Return(entries, nil)
	suite.activityRepo.EXPECT().GetConsumedActivitiesForCellar(ctx, uint(1)).Return(nil, nil)
	suite.activityRepo.EXPECT().ImportActivitiesWithState(ctx, mock.Anything, mock.Anything, state, mock.Anything).Return(int64(1), int64(0), nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(csvData),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(1), resp.Msg.Imported)
}

func (suite *ActivityTestSuite) TestImportUntappdCheckins_SkipsBeforeWatermark() {
	ctx := ctxWithUser(1)
	watermark := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	state := &model.UntappdImportState{CellarID: 1, LastImportAt: watermark}

	// Both rows are before the watermark
	csvData := untappdCSVHeader + "\n" +
		untappdCSVRow("Old Beer", "Brewery", "Can", "2024-05-01 12:00:00", "1", "100") + "\n" +
		untappdCSVRow("Old Beer 2", "Brewery", "Bottle", "2024-05-15 12:00:00", "2", "101")

	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(1)).Return(ownedCellar(), nil)
	suite.activityRepo.EXPECT().GetOrInitImportState(ctx, uint(1)).Return(state, nil)
	suite.activityRepo.EXPECT().GetUntappdCheckinIDsForCellar(ctx, uint(1)).Return(map[uint64]struct{}{}, nil)
	suite.cellarRepo.EXPECT().GetAllEntriesForCellar(ctx, uint(1)).Return(nil, nil)
	suite.activityRepo.EXPECT().GetConsumedActivitiesForCellar(ctx, uint(1)).Return(nil, nil)

	resp, err := suite.service.ImportUntappdCheckins(ctx, connect.NewRequest(&apiv1.ImportUntappdCheckinsRequest{
		CellarId: 1,
		CsvData:  []byte(csvData),
	}))

	suite.Require().NoError(err)
	suite.Equal(int32(0), resp.Msg.Imported)
	suite.Equal(int32(2), resp.Msg.Skipped)
}
