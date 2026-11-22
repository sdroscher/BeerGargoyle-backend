package server

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/model"
)

// adventRefillTimeout bounds one background refill run.
const adventRefillTimeout = 30 * time.Second

// startAdventRefill refills, in the background, every future advent calendar day scheduled with the
// given cellar entry. It returns immediately; ctx is only used for its values, because the request
// context is cancelled as soon as the response is sent.
func (c *CellarServer) startAdventRefill(ctx context.Context, cellarID uint64, entryID uint64) {
	detached := context.WithoutCancel(ctx)

	c.wg.Go(func() {
		// net/http recovers handler panics but not panics in goroutines a handler starts, so an
		// unrecovered panic here would take down the whole server.
		defer func() {
			if recovered := recover(); recovered != nil {
				c.logger.Error("advent calendar refill panicked",
					zap.Uint64("cellar_id", cellarID), zap.Uint64("cellar_entry_id", entryID), zap.Any("panic", recovered))
			}
		}()

		refillCtx, cancel := context.WithTimeout(detached, adventRefillTimeout)
		defer cancel()

		c.refillAdventDaysForEntry(refillCtx, cellarID, entryID)
	})
}

// refillAdventDaysForEntry replaces the entry in every advent calendar day after today. Failures are
// logged and never returned: a day that cannot be refilled keeps its old entry.
func (c *CellarServer) refillAdventDaysForEntry(ctx context.Context, cellarID uint64, entryID uint64) {
	today := truncateToDay(time.Now().UTC())

	days, err := c.cellarRepository.FindFutureAdventCalendarBeersForEntry(ctx, cellarID, entryID, today)
	if err != nil {
		c.logger.Error("failed to find advent calendar days to refill",
			zap.Uint64("cellar_id", cellarID), zap.Uint64("cellar_entry_id", entryID), zap.Error(err))

		return
	}

	for _, day := range days {
		c.refillAdventDay(ctx, cellarID, entryID, day)
	}
}

func (c *CellarServer) refillAdventDay(ctx context.Context, cellarID uint64, entryID uint64, day model.AdventCalendarBeer) {
	calendarID := uint64(day.AdventCalendarID)
	fields := []zap.Field{
		zap.Uint64("cellar_id", cellarID),
		zap.Uint64("calendar_id", calendarID),
		zap.Uint64("cellar_entry_id", entryID),
		zap.Time("day", day.Day),
	}

	calendar, err := c.cellarRepository.GetAdventCalendarByID(ctx, cellarID, calendarID)
	if err != nil {
		c.logger.Error("failed to load advent calendar for refill", append(fields, zap.Error(err))...)

		return
	}

	_, err = c.regenerateAdventDay(ctx, cellarID, calendar, day.Day)

	switch {
	case errors.Is(err, ErrCannotCreate):
		c.logger.Warn("no replacement beer found for advent calendar day; leaving it unchanged", append(fields, zap.Error(err))...)
	case err != nil:
		c.logger.Error("failed to refill advent calendar day", append(fields, zap.Error(err))...)
	}
}
