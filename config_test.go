package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests the configuration parsing functionality
type ConfigTestSuite struct {
	suite.Suite
	tempDir string
}

func (suite *ConfigTestSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "fleet-test-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *ConfigTestSuite) TearDownTest() {
	// Clean up temp directory
	os.RemoveAll(suite.tempDir)
}

func (suite *ConfigTestSuite) TestLoadConfigSuccess() {
	// Create a valid config file
	configContent := `
project = "test-app"

[[services]]
name = "web"
image = "nginx:alpine"
port = 8080
folder = "./website"

[[services]]
name = "database"
image = "postgres:15"
port = 5432
password = "testpass"
volumes = ["db-data:/var/lib/postgresql/data"]
`
	configFile := filepath.Join(suite.tempDir, "fleet.toml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	// Load the config
	config, err := loadConfig(configFile)
	suite.Require().NoError(err)

	// Assert config values
	suite.Equal("test-app", config.Project)
	suite.Len(config.Services, 2)
	
	// Check first service
	suite.Equal("web", config.Services[0].Name)
	suite.Equal("nginx:alpine", config.Services[0].Image)
	suite.Equal(8080, config.Services[0].Port)
	suite.Equal("./website", config.Services[0].Folder)

	// Check second service
	suite.Equal("database", config.Services[1].Name)
	suite.Equal("postgres:15", config.Services[1].Image)
	suite.Equal(5432, config.Services[1].Port)
	suite.Equal("testpass", config.Services[1].Password)
	suite.Len(config.Services[1].Volumes, 1)
}

func (suite *ConfigTestSuite) TestLoadConfigFileNotFound() {
	nonExistentFile := filepath.Join(suite.tempDir, "nonexistent.toml")
	
	_, err := loadConfig(nonExistentFile)
	suite.Error(err)
	suite.Contains(err.Error(), "no such file")
}

func (suite *ConfigTestSuite) TestLoadConfigInvalidTOML() {
	// Create an invalid TOML file
	configContent := `
project = "test-app
[[services]]
name = 
`
	configFile := filepath.Join(suite.tempDir, "invalid.toml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	_, err = loadConfig(configFile)
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestLoadConfigWithEnvironmentVariables() {
	configContent := `
project = "test-app"

[[services]]
name = "api"
image = "node:18"
port = 3000
[services.env]
NODE_ENV = "production"
DATABASE_URL = "postgresql://localhost/test"
`
	configFile := filepath.Join(suite.tempDir, "fleet.toml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	config, err := loadConfig(configFile)
	suite.Require().NoError(err)

	suite.Equal("api", config.Services[0].Name)
	suite.Equal("production", config.Services[0].Environment["NODE_ENV"])
	suite.Equal("postgresql://localhost/test", config.Services[0].Environment["DATABASE_URL"])
}

func (suite *ConfigTestSuite) TestLoadConfigWithDependencies() {
	configContent := `
project = "test-app"

[[services]]
name = "database"
image = "postgres:15"
port = 5432

[[services]]
name = "api"
image = "node:18"
port = 3000
needs = ["database"]
`
	configFile := filepath.Join(suite.tempDir, "fleet.toml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	suite.Require().NoError(err)

	config, err := loadConfig(configFile)
	suite.Require().NoError(err)

	suite.Len(config.Services[1].Needs, 1)
	suite.Equal("database", config.Services[1].Needs[0])
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}