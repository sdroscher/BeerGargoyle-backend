package cmd

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
)

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
		logger.Fatal("error connecting to database")
	}
	defer repo.Close()

	db := repo.DB

	var entries []model.CellarEntry

	err = db.Unscoped().Find(&entries).Error
	if err != nil {
		return fmt.Errorf("querying cellar entries: %w", err)
	}

	var created, skipped int

	for i := range entries {
		entryCreated, entrySkipped, err := backfillEntry(db, &entries[i])
		if err != nil {
			return err
		}

		created += entryCreated
		skipped += entrySkipped
	}

	logger.Info("backfill complete",
		zap.Int("created", created),
		zap.Int("skipped", skipped),
		zap.Int("entries_processed", len(entries)),
	)
	fmt.Printf("Done: %d activities created, %d already existed.\n", created, skipped)

	return nil
}

// backfillEntry creates missing Activity records for a single CellarEntry.
// Returns the number of records created and skipped.
func backfillEntry(db *gorm.DB, entry *model.CellarEntry) (int, int, error) {
	entryID := entry.ID

	occurredAt := entry.CreatedAt
	if entry.DateAdded != nil {
		occurredAt = *entry.DateAdded
	}

	qty := entry.Quantity
	if qty <= 0 {
		qty = 1
	}

	addedCreated, addedSkipped, err := backfillActivity(db, model.Activity{
		CellarID:      entry.CellarID,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerAdded,
		Quantity:      qty,
		OccurredAt:    occurredAt,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("BEER_ADDED for entry %d: %w", entryID, err)
	}

	if !entry.DeletedAt.Valid {
		return addedCreated, addedSkipped, nil
	}

	consumedCreated, consumedSkipped, err := backfillActivity(db, model.Activity{
		CellarID:      entry.CellarID,
		CellarEntryID: &entryID,
		ActivityType:  model.ActivityTypeBeerConsumed,
		Quantity:      1, // quantity is 0 in DB at soft-delete time; best guess is 1
		OccurredAt:    entry.DeletedAt.Time,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("BEER_CONSUMED for entry %d: %w", entryID, err)
	}

	return addedCreated + consumedCreated, addedSkipped + consumedSkipped, nil
}

// backfillActivity inserts act if no Activity with the same CellarEntryID and ActivityType exists.
func backfillActivity(db *gorm.DB, act model.Activity) (int, int, error) {
	var existing model.Activity

	lookupErr := db.Where("cellar_entry_id = ? AND activity_type = ?", act.CellarEntryID, act.ActivityType).
		First(&existing).Error

	switch {
	case errors.Is(lookupErr, gorm.ErrRecordNotFound):
		createErr := db.Create(&act).Error
		if createErr != nil {
			return 0, 0, createErr
		}

		return 1, 0, nil
	case lookupErr != nil:
		return 0, 0, lookupErr
	default:
		return 0, 1, nil
	}
}
