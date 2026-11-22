package server_test

import (
	"context"
	"errors"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

const (
	refillCellarID    = 1
	consumedEntryID   = 10
	replacementID     = 30
	refillAfterWindow = 24 * time.Hour
)

// todayUTCMatcher matches the "after" argument handed to the repository lookup: today's UTC midnight.
func todayUTCMatcher() any {
	return mock.MatchedBy(func(after time.Time) bool {
		now := time.Now()

		return after.Location() == time.UTC &&
			after.Equal(after.Truncate(refillAfterWindow)) &&
			!after.After(now) &&
			now.Sub(after) <= refillAfterWindow+time.Second
	})
}

// futureDay returns UTC midnight n days from today.
func futureDay(n int) time.Time {
	return time.Now().UTC().Truncate(refillAfterWindow).AddDate(0, 0, n)
}

func entryWithQuantity(quantity int64) *model.CellarEntry {
	return &model.CellarEntry{
		Model:    gorm.Model{ID: consumedEntryID},
		CellarID: refillCellarID,
		Quantity: quantity,
		Beer:     model.Beer{Model: gorm.Model{ID: 5}, Name: "Lights Out"},
	}
}

// expectConsumption sets up the repository calls RecordConsumption makes for consuming one unit of an
// entry that has startQuantity units.
func (suite *ActivityTestSuite) expectConsumption(ctx context.Context, startQuantity int64) {
	entryID := uint(consumedEntryID)
	activity := &model.Activity{Model: gorm.Model{ID: 1}, CellarID: refillCellarID, CellarEntryID: &entryID}

	suite.cellarRepo.EXPECT().GetCellarEntryByID(ctx, entryID).Return(entryWithQuantity(startQuantity), nil)
	suite.cellarRepo.EXPECT().GetCellarByID(ctx, uint(refillCellarID)).Return(ownedCellar(), nil)
	suite.cellarRepo.EXPECT().ConsumeEntry(ctx, entryWithQuantity(startQuantity-1), mock.Anything).Return(activity, nil)
}

// expectRefillLookup returns the future days the repository reports for the consumed entry.
func (suite *ActivityTestSuite) expectRefillLookup(days ...model.AdventCalendarBeer) {
	suite.cellarRepo.EXPECT().
		FindFutureAdventCalendarBeersForEntry(mock.Anything, uint64(refillCellarID), uint64(consumedEntryID), todayUTCMatcher()).
		Return(days, nil)
}

// expectDayRefill sets up the calls needed to refill one calendar day. The day's filter has the given
// minimum ABV so that candidates can be told apart per calendar. When replacement is 0 no candidate is
// found and no update is expected.
func (suite *ActivityTestSuite) expectDayRefill(calendarID uint, day time.Time, minimumAbv float64, replacement uint) {
	calendar := &model.AdventCalendar{
		Model:    gorm.Model{ID: calendarID},
		CellarID: refillCellarID,
		Beers:    []model.AdventCalendarBeer{{CellarEntryID: consumedEntryID}, {CellarEntryID: 20}},
	}
	filter := &model.AdventCalendarFilter{MinimumAbv: new(minimumAbv)}

	var candidates []*model.CellarEntry
	if replacement != 0 {
		candidates = []*model.CellarEntry{{Model: gorm.Model{ID: replacement}, CellarID: refillCellarID, BeerID: 300}}
	}

	suite.cellarRepo.EXPECT().GetAdventCalendarByID(mock.Anything, uint64(refillCellarID), uint64(calendarID)).Return(calendar, nil)
	suite.cellarRepo.EXPECT().GetAdventCalendarFilter(mock.Anything, uint64(refillCellarID), uint64(calendarID), day).Return(filter, nil)
	suite.cellarRepo.EXPECT().
		FindBeerRecommendations(mock.Anything, uint64(refillCellarID), mock.MatchedBy(func(f *apiv1.CellarFilter) bool { return f.GetMinimumAbv() == minimumAbv })).
		Return(candidates, nil)

	if replacement != 0 {
		suite.cellarRepo.EXPECT().
			UpdateAdventCalendarEntry(mock.Anything, uint64(refillCellarID), uint64(calendarID), day, uint64(replacement)).
			Return(nil)
	}
}

func (suite *ActivityTestSuite) observedService() (*server.CellarServer, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)

	return server.NewCellarServer(suite.cellarRepo, nil, suite.activityRepo, zap.New(core)), logs
}

func consume(ctx context.Context, service *server.CellarServer) (*connect.Response[apiv1.RecordConsumptionResponse], error) {
	return service.RecordConsumption(ctx, connect.NewRequest(&apiv1.RecordConsumptionRequest{CellarEntryId: consumedEntryID, Quantity: 1}))
}

func (suite *ActivityTestSuite) TestRecordConsumption_PartialDoesNotLookUpAdvent() {
	ctx := ctxWithUser(1)
	suite.expectConsumption(ctx, 2)

	_, err := consume(ctx, suite.service)
	suite.service.Wait()

	suite.Require().NoError(err)
}

func (suite *ActivityTestSuite) TestRecordConsumption_LastBottleRefillsFutureDay() {
	ctx := ctxWithUser(1)
	day := futureDay(2)
	suite.expectConsumption(ctx, 1)
	suite.expectRefillLookup(model.AdventCalendarBeer{AdventCalendarID: 3, CellarEntryID: consumedEntryID, Day: day})
	suite.expectDayRefill(3, day, 5, replacementID)

	_, err := consume(ctx, suite.service)
	suite.service.Wait()

	suite.Require().NoError(err)
}

func (suite *ActivityTestSuite) TestRecordConsumption_RefillsEveryDayEvenWhenOneHasNoCandidate() {
	ctx := ctxWithUser(1)
	dayInFirstCalendar, dayInSecondCalendar := futureDay(2), futureDay(3)
	service, logs := suite.observedService()

	suite.expectConsumption(ctx, 1)
	suite.expectRefillLookup(
		model.AdventCalendarBeer{AdventCalendarID: 3, CellarEntryID: consumedEntryID, Day: dayInFirstCalendar},
		model.AdventCalendarBeer{AdventCalendarID: 4, CellarEntryID: consumedEntryID, Day: dayInSecondCalendar},
	)
	suite.expectDayRefill(3, dayInFirstCalendar, 5, 0) // no candidate: no update expected
	suite.expectDayRefill(4, dayInSecondCalendar, 7, replacementID)

	_, err := consume(ctx, service)
	service.Wait()

	suite.Require().NoError(err)

	warnings := logs.FilterLevelExact(zapcore.WarnLevel).All()
	suite.Require().Len(warnings, 1)
	suite.Equal(uint64(3), warnings[0].ContextMap()["calendar_id"])
}

func (suite *ActivityTestSuite) TestRecordConsumption_RefillLookupErrorStillSucceeds() {
	ctx := ctxWithUser(1)
	service, logs := suite.observedService()

	suite.expectConsumption(ctx, 1)
	suite.cellarRepo.EXPECT().
		FindFutureAdventCalendarBeersForEntry(mock.Anything, uint64(refillCellarID), uint64(consumedEntryID), todayUTCMatcher()).
		Return(nil, errors.New("lookup failed"))

	response, err := consume(ctx, service)
	service.Wait()

	suite.Require().NoError(err)
	suite.NotNil(response)
	suite.Equal(1, logs.FilterLevelExact(zapcore.ErrorLevel).Len())
}

func (suite *ActivityTestSuite) TestRecordConsumption_RefillSurvivesRequestCancel() {
	ctx, cancel := context.WithCancel(ctxWithUser(1))
	defer cancel()

	day := futureDay(2)
	released := make(chan struct{})

	var backgroundCtxErr error

	suite.expectConsumption(ctx, 1)
	suite.cellarRepo.EXPECT().
		FindFutureAdventCalendarBeersForEntry(mock.Anything, uint64(refillCellarID), uint64(consumedEntryID), todayUTCMatcher()).
		RunAndReturn(func(lookupCtx context.Context, _ uint64, _ uint64, _ time.Time) ([]model.AdventCalendarBeer, error) {
			<-released // hold the background work until the request context has been cancelled
			backgroundCtxErr = lookupCtx.Err()

			return []model.AdventCalendarBeer{{AdventCalendarID: 3, CellarEntryID: consumedEntryID, Day: day}}, nil
		})
	suite.expectDayRefill(3, day, 5, replacementID)

	_, err := consume(ctx, suite.service)
	suite.Require().NoError(err)

	cancel()
	close(released)
	suite.service.Wait()

	suite.NoError(backgroundCtxErr)
}
