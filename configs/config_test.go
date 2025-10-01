package configs_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	"droscher.com/BeerGargoyle/configs"
)

type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (suite *ConfigTestSuite) TestGetConfig_GetsNamedFile() {
	logger := zaptest.NewLogger(suite.T())

	config, err := configs.GetConfig("testdata/config.toml", logger)

	suite.Require().NoError(err)
	suite.Equal("test.local", config.DB.Host)
	suite.Equal(1234, config.DB.Port)
	suite.Equal("testuser", config.DB.User)
	suite.Equal("test123", config.DB.Password)
	suite.Equal("testdb", config.DB.Database)
	suite.Equal(5, config.DB.MaxIdleConnections)
	suite.Equal(7, config.DB.MaxOpenConnections)
	suite.Equal(666, config.Server.Port)
	suite.Equal("audience", config.Auth.Audience)
	suite.Equal("domain", config.Auth.Domain)
	suite.Equal("secret", config.Auth.SecretKey)
	suite.Equal([]string{"untappd_web"}, config.Integrations.Beer)
}

func (suite *ConfigTestSuite) TestGetConfig_GetsEnv() {
	logger := zaptest.NewLogger(suite.T())

	suite.T().Setenv("BEERGARGOYLE_DB_HOST", "test.local")
	suite.T().Setenv("BEERGARGOYLE_DB_PORT", "1234")
	suite.T().Setenv("BEERGARGOYLE_DB_USER", "testuser")
	suite.T().Setenv("BEERGARGOYLE_DB_PASSWORD", "test123")
	suite.T().Setenv("BEERGARGOYLE_DB_DATABASE", "testdb")
	suite.T().Setenv("BEERGARGOYLE_DB_MAXIDLECONNECTIONS", "5")
	suite.T().Setenv("BEERGARGOYLE_DB_MAXOPENCONNECTIONS", "7")
	suite.T().Setenv("BEERGARGOYLE_SERVER_PORT", "666")
	suite.T().Setenv("BEERGARGOYLE_AUTH_AUDIENCE", "audience")
	suite.T().Setenv("BEERGARGOYLE_AUTH_DOMAIN", "domain")
	suite.T().Setenv("BEERGARGOYLE_AUTH_SECRETKEY", "secret")
	suite.T().Setenv("BEERGARGOYLE_INTEGRATIONS_BEER", "untappd_web")

	config, err := configs.GetConfig("", logger)

	suite.Require().NoError(err)
	suite.Equal("test.local", config.DB.Host)
	suite.Equal(1234, config.DB.Port)
	suite.Equal("testuser", config.DB.User)
	suite.Equal("test123", config.DB.Password)
	suite.Equal("testdb", config.DB.Database)
	suite.Equal(5, config.DB.MaxIdleConnections)
	suite.Equal(7, config.DB.MaxOpenConnections)
	suite.Equal(666, config.Server.Port)
	suite.Equal("audience", config.Auth.Audience)
	suite.Equal("domain", config.Auth.Domain)
	suite.Equal("secret", config.Auth.SecretKey)
	suite.Equal([]string{"untappd_web"}, config.Integrations.Beer)
}

func (suite *ConfigTestSuite) TestGetConfig_EnvOverridesFile() {
	logger := zaptest.NewLogger(suite.T())

	suite.T().Setenv("BEERGARGOYLE_DB_HOST", "env.local")
	suite.T().Setenv("BEERGARGOYLE_DB_USER", "envuser")
	suite.T().Setenv("BEERGARGOYLE_DB_PASSWORD", "env123")
	suite.T().Setenv("BEERGARGOYLE_AUTH_AUDIENCE", "envaudience")
	suite.T().Setenv("BEERGARGOYLE_AUTH_DOMAIN", "envdomain")
	suite.T().Setenv("BEERGARGOYLE_AUTH_SECRETKEY", "envsecret")
	suite.T().Setenv("BEERGARGOYLE_INTEGRATIONS_BEER", "envuntappd_web")

	config, err := configs.GetConfig("testdata/config.toml", logger)

	suite.Require().NoError(err)
	suite.Equal("env.local", config.DB.Host)
	suite.Equal(1234, config.DB.Port)
	suite.Equal("envuser", config.DB.User)
	suite.Equal("env123", config.DB.Password)
	suite.Equal("testdb", config.DB.Database)
	suite.Equal(5, config.DB.MaxIdleConnections)
	suite.Equal(7, config.DB.MaxOpenConnections)
	suite.Equal(666, config.Server.Port)
	suite.Equal("envaudience", config.Auth.Audience)
	suite.Equal("envdomain", config.Auth.Domain)
	suite.Equal("envsecret", config.Auth.SecretKey)
	suite.Equal([]string{"envuntappd_web"}, config.Integrations.Beer)
}

func (suite *ConfigTestSuite) TestGetConfig_MissingFileReturnsError() {
	logger := zaptest.NewLogger(suite.T())

	config, err := configs.GetConfig("testdata/missing.toml", logger)

	suite.Nil(config)
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestGetConfig_MissingValues() {
	logger := zaptest.NewLogger(suite.T())

	config, err := configs.GetConfig("", logger)

	suite.Nil(config)
	suite.EqualError(err, "DB.Host: required validation failed, DB.Password: required validation failed")
}
