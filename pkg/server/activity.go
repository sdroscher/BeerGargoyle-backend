package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/bufbuild/connect-go"
	"gorm.io/gorm"

	api "droscher.com/BeerGargoyle/generated/grpc/api/v1"
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

	created, err := c.activityRepository.CreateActivity(ctx, activity)
	if err != nil {
		return nil, err
	}

	entry.Quantity -= qty

	if entry.Quantity == 0 {
		err = c.cellarRepository.DeleteCellarEntry(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
	} else {
		_, updateErr := c.cellarRepository.UpdateCellarEntry(ctx, entry)
		if updateErr != nil {
			return nil, updateErr
		}
	}

	return connect.NewResponse(&api.RecordConsumptionResponse{
		Consumption: grpcconv.ActivityEventFromModel(created, entry),
	}), nil
}

// GetActivityFeed returns a paginated list of activity events for a cellar.
func (c *CellarServer) GetActivityFeed(ctx context.Context, request *connect.Request[api.GetActivityFeedRequest]) (*connect.Response[api.GetActivityFeedResponse], error) {
	cellarID := uint(request.Msg.GetCellarId())

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

// GetYearInReview returns aggregate consumption and acquisition stats for a cellar year.
func (c *CellarServer) GetYearInReview(ctx context.Context, request *connect.Request[api.GetYearInReviewRequest]) (*connect.Response[api.GetYearInReviewResponse], error) {
	year := int(request.Msg.GetYear())
	if year == 0 {
		year = time.Now().UTC().Year()
	}

	yir, err := c.activityRepository.GetYearInReview(ctx, uint(request.Msg.GetCellarId()), year)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetYearInReviewResponse{
		YearInReview: grpcconv.YearInReviewFromModel(yir),
	}), nil
}
