package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bufbuild/connect-go"
	"gorm.io/gorm"

	api "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/pkg/model"
	grpcconv "droscher.com/BeerGargoyle/pkg/server/grpc"
)

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
