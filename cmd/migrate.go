package cmd

import (
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
)

type MigrateCmd struct {
	ConfigFile string `default:".BeerGargoyle.toml" help:"Path to config file" short:"c"`
}

func (m *MigrateCmd) Run(_ *Context) error {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.DisableStacktrace = true

	logger, _ := logConfig.Build()
	defer logger.Sync() //nolint:errcheck // we don't care about logger sync errors

	conf, err := configs.GetConfig(m.ConfigFile, logger)
	if err != nil {
		logger.Error("error loading config", zap.Error(err))

		return err
	}

	repo, err := repository.Open(conf, logger)
	if err != nil {
		logger.Fatal("error connecting to database")
	}
	defer repo.Close()

	err = repo.DB.AutoMigrate(
		&model.Address{}, &model.Brewery{},
		&model.BeerStyle{}, &model.BeerFormat{}, &model.Beer{},
		&model.User{},
		&model.Cellar{}, &model.LocationInCellar{}, &model.CellarEntry{},
		&model.AdventCalendar{}, &model.AdventCalendarBeer{}, &model.AdventCalendarFilter{})
	if err != nil {
		return err
	}

	return nil
}
