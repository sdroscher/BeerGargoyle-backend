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
	Beer    []string `default:"untappd_web"`
	Untappd Untappd
}

// Untappd configures the Untappd web-scraping beer search backend.
type Untappd struct {
	// Proxy is an optional outbound proxy for scraping requests, e.g.
	// "http://user:pass@host:port". Cloudflare serves a managed challenge to
	// datacenter IPs (such as Fly.io), so a residential/ISP proxy is required
	// in production. Leave empty to scrape directly (works from residential
	// IPs). Takes precedence over ScraperAPIKey.
	Proxy string
	// ScraperAPIKey, when set and Proxy is empty, routes scraping through
	// ScraperAPI's proxy endpoint (https://www.scraperapi.com), which rotates
	// residential IPs and solves Cloudflare challenges.
	ScraperAPIKey string
	// ProxyInsecureSkipVerify disables TLS certificate verification for the
	// proxy. Auto-enabled for ScraperAPI (its proxy mode performs TLS
	// interception). Only set this for a generic Proxy that requires it.
	ProxyInsecureSkipVerify bool
	// Algolia* configure the Untappd search backend, which is powered by
	// Algolia and is NOT behind Cloudflare (so search needs no proxy). The
	// defaults are the public search-only credentials embedded in Untappd's web
	// pages; override via env vars if Untappd ever rotates them.
	AlgoliaAppID        string `default:"9WBO4RQ3HO"`
	AlgoliaSearchKey    string `default:"1d347324d67ec472bb7132c66aead485"`
	AlgoliaBeerIndex    string `default:"beer"`
	AlgoliaBreweryIndex string `default:"brewery"`
}

const scraperAPIProxyHost = "proxy-server.scraperapi.com:8001"

// ResolveProxy returns the effective proxy URL for Untappd scraping and whether
// TLS verification must be skipped. An empty URL means scrape directly.
func (u Untappd) ResolveProxy() (string, bool) {
	if u.Proxy != "" {
		return u.Proxy, u.ProxyInsecureSkipVerify
	}

	if u.ScraperAPIKey != "" {
		return "http://scraperapi:" + u.ScraperAPIKey + "@" + scraperAPIProxyHost, true
	}

	return "", false
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
