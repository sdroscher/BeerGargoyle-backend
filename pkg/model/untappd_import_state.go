package model

import (
	"time"

	"gorm.io/gorm"
)

// UntappdImportState tracks the incremental import watermark for Untappd CSV imports per cellar.
// Rows with created_at strictly before LastImportAt are skipped on subsequent imports.
type UntappdImportState struct {
	gorm.Model
	CellarID     uint      `gorm:"uniqueIndex"`
	LastImportAt time.Time // advance to max(created_at) after each successful import
}
