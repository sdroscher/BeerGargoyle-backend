package repository_test

import (
	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"

	"droscher.com/BeerGargoyle/pkg/repository"
)

type RepositorySuite struct {
	suite.Suite
	DB           *gorm.DB
	mock         sqlmock.Sqlmock
	observedLogs *observer.ObservedLogs
	repository   repository.Repository
}

func (suite *RepositorySuite) SetupTest() {
	var (
		db              *sql.DB
		err             error
		observedZapCore zapcore.Core
	)

	observedZapCore, suite.observedLogs = observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	db, suite.mock, err = sqlmock.New()
	suite.Require().NoError(err)

	gormLogger := zapgorm2.New(observedLogger)
	gormLogger.SetAsDefault()

	suite.DB, err = gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: gormLogger})
	suite.NoError(err)

	suite.repository = repository.Repository{DB: suite.DB, Logger: observedLogger}
}
