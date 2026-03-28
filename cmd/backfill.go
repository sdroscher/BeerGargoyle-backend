package cmd

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
)

const backfillBatchSize = 500

type BackfillCmd struct {
	ConfigFile string `default:".BeerGargoyle.toml" help:"Path to config file" short:"c"`
}

func (b *BackfillCmd) Run(_ *Context) error {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.DisableStacktrace = true

	logger, _ := logConfig.Build()
	defer logger.Sync() //nolint:errcheck

	conf, err := configs.GetConfig(b.ConfigFile, logger)
	if err != nil {
		logger.Error("error loading config", zap.Error(err))

		return err
	}

	repo, err := repository.Open(conf, logger)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer repo.Close()

	db := repo.DB

	var entries []model.CellarEntry

	err = db.Unscoped().Find(&entries).Error
	if err != nil {
		return fmt.Errorf("querying cellar entries: %w", err)
	}

	candidates := BuildCandidateActivities(entries)

	if len(candidates) == 0 {
		logger.Info("backfill complete: no candidates found")

		return nil
	}

	created, skipped, err := insertNewActivities(db, entries, candidates)
	if err != nil {
		return err
	}

	logger.Info("backfill complete",
		zap.Int64("created", created),
		zap.Int("skipped", skipped),
		zap.Int("entries_processed", len(entries)),
	)
	fmt.Printf("Done: %d activities created, %d already existed.\n", created, skipped)

	return nil
}

// BuildCandidateActivities builds the full set of activities that should exist for the given entries.
func BuildCandidateActivities(entries []model.CellarEntry) []model.Activity {
	candidates := make([]model.Activity, 0, len(entries))

	for i := range entries {
		entry := &entries[i]
		entryID := entry.ID

		occurredAt := entry.CreatedAt
		if entry.DateAdded != nil {
			occurredAt = *entry.DateAdded
		}

		qty := entry.Quantity
		if qty <= 0 {
			qty = 1
		}

		candidates = append(candidates, model.Activity{
			CellarID:      entry.CellarID,
			CellarEntryID: &entryID,
			ActivityType:  model.ActivityTypeBeerAdded,
			Quantity:      qty,
			OccurredAt:    occurredAt,
		})

		if entry.DeletedAt.Valid {
			candidates = append(candidates, model.Activity{
				CellarID:      entry.CellarID,
				CellarEntryID: &entryID,
				ActivityType:  model.ActivityTypeBeerConsumed,
				// quantity is 0 at soft-delete time so the original value is lost; 1 is a best guess and may under-count multi-unit consumptions
				Quantity:   1,
				OccurredAt: entry.DeletedAt.Time,
			})
		}
	}

	return candidates
}

// insertNewActivities fetches existing activities for the given entries, filters the candidates to
// only those not already present, and bulk-inserts the remainder. Returns created and skipped counts.
func insertNewActivities(db *gorm.DB, entries []model.CellarEntry, candidates []model.Activity) (int64, int, error) {
	entryIDs := make([]uint, 0, len(entries))
	for i := range entries {
		entryIDs = append(entryIDs, entries[i].ID)
	}

	var existing []model.Activity

	err := db.Select("cellar_entry_id, activity_type").
		Where("cellar_entry_id IN (?)", entryIDs).
		Find(&existing).Error
	if err != nil {
		return 0, 0, fmt.Errorf("querying existing activities: %w", err)
	}

	type key struct {
		cellarEntryID uint
		activityType  model.ActivityType
	}

	existingSet := make(map[key]struct{}, len(existing))

	for _, act := range existing {
		if act.CellarEntryID != nil {
			existingSet[key{*act.CellarEntryID, act.ActivityType}] = struct{}{}
		}
	}

	toInsert := make([]model.Activity, 0, len(candidates))

	for _, act := range candidates {
		if _, found := existingSet[key{*act.CellarEntryID, act.ActivityType}]; !found {
			toInsert = append(toInsert, act)
		}
	}

	skipped := len(candidates) - len(toInsert)

	if len(toInsert) == 0 {
		return 0, skipped, nil
	}

	var inserted int64

	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.CreateInBatches(toInsert, backfillBatchSize)
		if result.Error != nil {
			return fmt.Errorf("inserting activities: %w", result.Error)
		}

		inserted = result.RowsAffected

		return markHadBefore(tx, toInsert)
	})
	if err != nil {
		return 0, 0, err
	}

	return inserted, skipped, nil
}

// markHadBefore sets had_before = TRUE on cellar entries for any beer_consumed activities in the list.
func markHadBefore(db *gorm.DB, activities []model.Activity) error {
	entryIDs := make([]uint, 0, len(activities))

	for _, act := range activities {
		if act.ActivityType == model.ActivityTypeBeerConsumed && act.CellarEntryID != nil {
			entryIDs = append(entryIDs, *act.CellarEntryID)
		}
	}

	if len(entryIDs) == 0 {
		return nil
	}

	err := db.Unscoped().Model(&model.CellarEntry{}).
		Where("id IN ?", entryIDs).
		Update("had_before", true).Error
	if err != nil {
		return fmt.Errorf("updating had_before: %w", err)
	}

	return nil
}
