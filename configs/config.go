package configs

import (
	"errors"
	"os"
	"strings"

	"github.com/kkyr/fig"
	"go.uber.org/zap"
)

type DB struct {
	Host               string `validate:"required"`
	Port               int    `default:"5432"`
	User               string `default:"postgres"`
	Password           string `validate:"required"`
	Database           string `default:"postgres"`
	MaxIdleConnections int    `default:"10"`
	MaxOpenConnections int    `default:"10"`
}

type Server struct {
	Port int `default:"8080"`
}

type Integrations struct {
	Beer []string `default:"untappd_web"`
}

type Config struct {
	DB           DB
	Server       Server
	Integrations Integrations
	Auth         Auth
}

type Auth struct {
	SecretKey string
	Audience  string
	Domain    string
}

const envPrefix = "BEERGARGOYLE" // env prefix for env vars

var ErrConfiguration = errors.New("configuration error")

func GetConfig(configFileName string, logger *zap.Logger) (*Config, error) {
	config := Config{}
	homeDir, _ := os.UserHomeDir()

	logger.Info("Loading config", zap.String("file", configFileName))

	err := fig.Load(&config, fig.File(configFileName), fig.Dirs(".", homeDir), fig.UseEnv(envPrefix))
	if err != nil {
		if strings.Contains(err.Error(), "file not found") {
			logger.Warn("Could not find config file", zap.String("file", configFileName))

			err = fig.Load(&config, fig.IgnoreFile(), fig.UseEnv(envPrefix))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &config, nil
}
