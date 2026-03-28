package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/bufbuild/connect-go"
	"gorm.io/gorm"

	api "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	untappdcsv "droscher.com/BeerGargoyle/pkg/integrations/untappd-csv"
	"droscher.com/BeerGargoyle/pkg/model"
	grpcconv "droscher.com/BeerGargoyle/pkg/server/grpc"
)

const defaultActivityPageSize = 25

var (
	ErrEntryNotFound   = errors.New("cellar entry not found")
	ErrInvalidQuantity = errors.New("invalid quantity")
)

// RecordConsumption marks units of a cellar entry as consumed, decrements quantity,
// and soft-deletes the entry if quantity reaches zero.
func (c *CellarServer) RecordConsumption(ctx context.Context, request *connect.Request[api.RecordConsumptionRequest]) (*connect.Response[api.RecordConsumptionResponse], error) {
	entryID := uint(request.Msg.GetCellarEntryId())

	entry, err := c.cellarRepository.GetCellarEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("%w: id %d", ErrEntryNotFound, entryID))
		}

		return nil, err
	}

	_, err = c.requireCellarOwner(ctx, entry.CellarID)
	if err != nil {
		return nil, err
	}

	qty := request.Msg.GetQuantity()
	if qty <= 0 {
		qty = 1
	}

	if qty > entry.Quantity {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%w: cannot consume %d, only %d available", ErrInvalidQuantity, qty, entry.Quantity))
	}

	occurredAt := time.Now().UTC()
	if request.Msg.GetConsumedAt() != nil {
		occurredAt = request.Msg.GetConsumedAt().AsTime()
	}

	activity := &model.Activity{
		CellarID:      entry.CellarID,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      qty,
		OccurredAt:    occurredAt,
		Note:          request.Msg.Note,
		Rating:        request.Msg.Rating,
	}

	entry.Quantity -= qty

	created, err := c.cellarRepository.ConsumeEntry(ctx, entry, activity)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.RecordConsumptionResponse{
		Consumption: grpcconv.ActivityEventFromModel(created, entry),
	}), nil
}

// GetActivityFeed returns a paginated list of activity events for a cellar.
func (c *CellarServer) GetActivityFeed(ctx context.Context, request *connect.Request[api.GetActivityFeedRequest]) (*connect.Response[api.GetActivityFeedResponse], error) {
	cellarID := uint(request.Msg.GetCellarId())

	_, err := c.requireCellarOwner(ctx, cellarID)
	if err != nil {
		return nil, err
	}

	pageSize := int(request.Msg.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultActivityPageSize
	}

	page := int(request.Msg.GetPage())
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * pageSize

	var from, until *time.Time

	if request.Msg.GetFrom() != nil {
		t := request.Msg.GetFrom().AsTime()
		from = &t
	}

	if request.Msg.GetTo() != nil {
		t := request.Msg.GetTo().AsTime()
		until = &t
	}

	activities, total, err := c.activityRepository.GetFeed(ctx, cellarID, from, until, pageSize, offset)
	if err != nil {
		return nil, err
	}

	events := make([]*api.ActivityEvent, 0, len(activities))

	for _, act := range activities {
		events = append(events, grpcconv.ActivityEventFromModel(act, act.CellarEntry))
	}

	totalCount := min(total, math.MaxInt32)

	return connect.NewResponse(&api.GetActivityFeedResponse{
		Events:  events,
		Total:   int32(totalCount), //nolint:gosec // clamped to MaxInt32 above
		HasMore: int64(offset+pageSize) < total,
	}), nil
}

// ImportUntappdCheckins performs an incremental import of Untappd CSV checkins into the activity feed.
// Only "Can" and "Bottle" serving types are imported. Rows before the stored watermark are skipped.
func (c *CellarServer) ImportUntappdCheckins(ctx context.Context, request *connect.Request[api.ImportUntappdCheckinsRequest]) (*connect.Response[api.ImportUntappdCheckinsResponse], error) {
	cellarID := uint(request.Msg.GetCellarId())

	_, err := c.requireCellarOwner(ctx, cellarID)
	if err != nil {
		return nil, err
	}

	rows, err := untappdcsv.Parse(bytes.NewReader(request.Msg.GetCsvData()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid CSV: %w", err))
	}

	if len(rows) == 0 {
		return connect.NewResponse(&api.ImportUntappdCheckinsResponse{}), nil
	}

	state, existingIDs, entries, consumedByEntryID, err := c.loadImportState(ctx, cellarID)
	if err != nil {
		return nil, err
	}

	beerIDByExternalID, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID := buildBeerLookupMaps(entries)

	activities, updates, skipped := untappdcsv.BuildActivities(
		rows,
		cellarID,
		state.LastImportAt,
		existingIDs,
		beerIDByExternalID,
		beerIDByName,
		anyEntryIDByBeerID,
		activeEntryIDByBeerID,
		consumedByEntryID,
	)

	if len(activities) == 0 && len(updates) == 0 {
		return connect.NewResponse(&api.ImportUntappdCheckinsResponse{
			Skipped: int32(skipped), //nolint:gosec // skipped count fits in int32 for any realistic CSV
		}), nil
	}

	newWatermark := maxCreatedAt(rows, state.LastImportAt)

	imported, updated, err := c.activityRepository.ImportActivitiesWithState(ctx, activities, updates, state, newWatermark)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.ImportUntappdCheckinsResponse{
		Imported: int32(imported), //nolint:gosec // row count fits in int32 for any realistic CSV
		Updated:  int32(updated),  //nolint:gosec // row count fits in int32 for any realistic CSV
		Skipped:  int32(skipped),  //nolint:gosec // skipped count fits in int32 for any realistic CSV
	}), nil
}

func (c *CellarServer) loadImportState(ctx context.Context, cellarID uint) (*model.UntappdImportState, map[uint64]struct{}, []*model.CellarEntry, map[uint][]model.ConsumedActivitySummary, error) {
	state, err := c.activityRepository.GetOrInitImportState(ctx, cellarID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	existingIDs, err := c.activityRepository.GetUntappdCheckinIDsForCellar(ctx, cellarID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	entries, err := c.cellarRepository.GetAllEntriesForCellar(ctx, cellarID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	consumedByEntryID, err := c.activityRepository.GetConsumedActivitiesForCellar(ctx, cellarID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return state, existingIDs, entries, consumedByEntryID, nil
}

// buildBeerLookupMaps builds four in-memory maps used to match CSV rows to internal beer/entry IDs.
// Active entries win over soft-deleted ones because GetAllEntriesForCellar returns them first.
func buildBeerLookupMaps(entries []*model.CellarEntry) (map[uint64]uint, map[string]uint, map[uint]uint, map[uint]uint) {
	beerIDByExternalID := make(map[uint64]uint)
	beerIDByName := make(map[string]uint)
	anyEntryIDByBeerID := make(map[uint]uint)
	activeEntryIDByBeerID := make(map[uint]uint)

	for _, entry := range entries {
		beerID := entry.Beer.ID

		// Beer-recognition maps include all entries (active + soft-deleted) so we can identify
		// any beer the user has ever had, even if its entry is gone.
		if entry.Beer.ExternalSource != nil && *entry.Beer.ExternalSource == "untappd" && entry.Beer.ExternalID != nil {
			if _, exists := beerIDByExternalID[*entry.Beer.ExternalID]; !exists {
				beerIDByExternalID[*entry.Beer.ExternalID] = beerID
			}
		}

		key := untappdcsv.NameKey(entry.Beer.Name, entry.Beer.Brewery.Name)
		if _, exists := beerIDByName[key]; !exists {
			beerIDByName[key] = beerID
		}

		if _, exists := anyEntryIDByBeerID[beerID]; !exists {
			anyEntryIDByBeerID[beerID] = entry.ID
		}

		// Active-only entry map used for new inserts. Soft-deleted entries have already been
		// handled by the backfill; a matching consumed activity will be updated instead.
		if !entry.DeletedAt.Valid {
			if _, exists := activeEntryIDByBeerID[beerID]; !exists {
				activeEntryIDByBeerID[beerID] = entry.ID
			}
		}
	}

	return beerIDByExternalID, beerIDByName, anyEntryIDByBeerID, activeEntryIDByBeerID
}

// maxCreatedAt returns the latest CreatedAt among rows that are at or after afterTime.
// Using >= ensures rows with CreatedAt == watermark are included, matching the
// BuildActivities skip condition (rows strictly before the watermark are skipped).
func maxCreatedAt(rows []untappdcsv.Row, afterTime time.Time) time.Time {
	latest := afterTime

	for _, row := range rows {
		if !row.CreatedAt.Before(afterTime) && row.CreatedAt.After(latest) {
			latest = row.CreatedAt
		}
	}

	return latest
}

// GetYearInReview returns aggregate consumption and acquisition stats for a cellar year.
func (c *CellarServer) GetYearInReview(ctx context.Context, request *connect.Request[api.GetYearInReviewRequest]) (*connect.Response[api.GetYearInReviewResponse], error) {
	cellarID := uint(request.Msg.GetCellarId())

	_, err := c.requireCellarOwner(ctx, cellarID)
	if err != nil {
		return nil, err
	}

	year := int(request.Msg.GetYear())
	if year == 0 {
		year = time.Now().UTC().Year()
	}

	yir, err := c.activityRepository.GetYearInReview(ctx, cellarID, year)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetYearInReviewResponse{
		YearInReview: grpcconv.YearInReviewFromModel(yir),
	}), nil
}
