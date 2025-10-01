package repository

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"

	"droscher.com/BeerGargoyle/configs"
)

type Repository struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

const (
	maxIdleTime = 5 * time.Minute
	maxLifetime = time.Hour
)

type Closer func(*gorm.DB)

func Open(conf *configs.Config, logger *zap.Logger) (*Repository, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		conf.DB.Host, conf.DB.User, conf.DB.Password, conf.DB.Database, conf.DB.Port)

	gormLogger := zapgorm2.New(logger)
	gormLogger.SetAsDefault()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(conf.DB.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(conf.DB.MaxOpenConnections)
	sqlDB.SetConnMaxIdleTime(maxIdleTime)
	sqlDB.SetConnMaxLifetime(maxLifetime)

	return &Repository{DB: db, Logger: logger}, err
}

func (r *Repository) Close() {
	sqlDB, err := r.DB.DB()
	if err != nil && sqlDB != nil {
		_ = sqlDB.Close()
	}
}
