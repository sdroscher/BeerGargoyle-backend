package untappdcsv

import (
	"time"

	"droscher.com/BeerGargoyle/pkg/model"
)

const matchWindow = 30 * 24 * time.Hour

// isAllowedServingType reports whether a serving type should be imported.
func isAllowedServingType(t string) bool {
	return t == "Can" || t == "Bottle"
}

// BuildActivities converts parsed CSV rows into Activity records to insert and updates to apply.
// It is a pure function — all matching state is passed in, no DB calls are made.
//
//   - watermark: rows with created_at strictly before the watermark are skipped.
//   - existingCheckinIDs: checkin IDs already in the DB; matching rows are skipped.
//   - beerIDByExternalID: maps Untappd beer ID → internal Beer.ID.
//   - beerIDByName: maps NameKey(beerName, breweryName) → internal Beer.ID.
//   - anyEntryIDByBeerID: maps internal Beer.ID → CellarEntry.ID (active + soft-deleted).
//   - activeEntryIDByBeerID: maps internal Beer.ID → CellarEntry.ID (active entries only).
//   - consumedByEntryID: maps CellarEntry.ID → unlinked beer_consumed activities for that entry.
//
// For each eligible row the logic is:
//  1. Match beer via external ID or name; skip if unmatched.
//  2. If a matching consumed activity exists within matchWindow → emit an update.
//  3. Otherwise, if an active cellar entry exists → emit a new activity to insert.
//  4. Otherwise → skip.
//
// Returns activities to insert, updates to apply, and the count of skipped rows.
func BuildActivities(
	rows []Row,
	cellarID uint,
	watermark time.Time,
	existingCheckinIDs map[uint64]struct{},
	beerIDByExternalID map[uint64]uint,
	beerIDByName map[string]uint,
	anyEntryIDByBeerID map[uint]uint,
	activeEntryIDByBeerID map[uint]uint,
	consumedByEntryID map[uint][]model.ConsumedActivitySummary,
) ([]*model.Activity, []model.ActivityUntappdUpdate, int) {
	activities := make([]*model.Activity, 0, len(rows))
	updates := make([]model.ActivityUntappdUpdate, 0)
	skipped := 0

	for _, row := range rows {
		if row.CreatedAt.Before(watermark) {
			skipped++

			continue
		}

		if !isAllowedServingType(row.ServingType) {
			skipped++

			continue
		}

		if _, ok := existingCheckinIDs[row.CheckinID]; ok {
			skipped++

			continue
		}

		beerID, matched := beerIDByExternalID[row.BeerID]
		if !matched {
			beerID, matched = beerIDByName[NameKey(row.BeerName, row.BreweryName)]
		}

		if !matched {
			skipped++

			continue
		}

		// Try to correlate with an existing consumed activity (e.g., from backfill for a deleted entry).
		if entryID, ok := anyEntryIDByBeerID[beerID]; ok {
			if summaries, ok := consumedByEntryID[entryID]; ok {
				if best := findBestConsumedMatch(row.CreatedAt, summaries); best != nil {
					updates = append(updates, buildUpdate(row, best))

					continue
				}
			}
		}

		// No existing activity to update — insert a new one if there's an active entry.
		if entryID, ok := activeEntryIDByBeerID[beerID]; ok {
			activities = append(activities, buildActivity(row, cellarID, entryID))

			continue
		}

		skipped++
	}

	return activities, updates, skipped
}

func findBestConsumedMatch(checkinTime time.Time, summaries []model.ConsumedActivitySummary) *model.ConsumedActivitySummary {
	var best *model.ConsumedActivitySummary
	var bestDiff time.Duration

	for idx := range summaries {
		diff := checkinTime.Sub(summaries[idx].OccurredAt)
		if diff < 0 {
			diff = -diff
		}

		if diff <= matchWindow && (best == nil || diff < bestDiff) {
			best = &summaries[idx]
			bestDiff = diff
		}
	}

	return best
}

func buildUpdate(row Row, best *model.ConsumedActivitySummary) model.ActivityUntappdUpdate {
	update := model.ActivityUntappdUpdate{
		ActivityID: best.ID,
		EntryID:    best.EntryID,
		CheckinID:  row.CheckinID,
	}

	if row.RatingScore > 0 {
		rating := row.RatingScore
		update.Rating = &rating
	}

	if row.Comment != "" {
		note := row.Comment
		update.Note = &note
	}

	return update
}

func buildActivity(row Row, cellarID uint, entryID uint) *model.Activity {
	checkinID := row.CheckinID

	act := &model.Activity{
		CellarID:         cellarID,
		CellarEntryID:    &entryID,
		ActivityType:     model.ActivityTypeBeerConsumed,
		Quantity:         1,
		OccurredAt:       row.CreatedAt,
		UntappdCheckinID: &checkinID,
	}

	if row.Comment != "" {
		note := row.Comment
		act.Note = &note
	}

	if row.RatingScore > 0 {
		rating := row.RatingScore
		act.Rating = &rating
	}

	return act
}
