package main

import (
	"fmt"
	"testing"
	
	"github.com/stretchr/testify/suite"
)

type XdebugSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *XdebugSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *XdebugSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *XdebugSuite) TestXdebugEnabled() {
	// Test that Xdebug is enabled when debug flag is set
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:    "api",
				Image:   "nginx",
				Runtime: "php:8.3",
				Folder:  "api",
				Debug:   true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// PHP-FPM service should have Xdebug configured
	phpService := compose.Services["api-php"]
	suite.Assert().NotNil(phpService, "PHP service should exist")
	
	// Check Xdebug environment variables
	suite.Assert().Equal("develop,debug,coverage", phpService.Environment["XDEBUG_MODE"])
	suite.Assert().Contains(phpService.Environment["XDEBUG_CONFIG"], "client_host=host.docker.internal")
	suite.Assert().Contains(phpService.Environment["XDEBUG_CONFIG"], "client_port=9003")
	suite.Assert().Equal("1", phpService.Environment["XDEBUG_SESSION"])
	suite.Assert().Equal("yes", phpService.Environment["XDEBUG_TRIGGER"])
	suite.Assert().Equal("serverName=api", phpService.Environment["PHP_IDE_CONFIG"])
	
	// Check that Xdebug installation command is present
	suite.Assert().Contains(phpService.Command, "xdebug")
	suite.Assert().Contains(phpService.Command, "pecl install xdebug")
	
	// Check extra hosts for Linux compatibility
	suite.Assert().Contains(phpService.ExtraHosts, "host.docker.internal:host-gateway")
}

func (suite *XdebugSuite) TestXdebugWithCustomPort() {
	// Test Xdebug with custom debug port
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "web",
				Image:     "nginx",
				Runtime:   "php:8.2",
				Folder:    "web",
				Debug:     true,
				DebugPort: 9005,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	phpService := compose.Services["web-php"]
	
	// Check custom port configuration
	suite.Assert().Contains(phpService.Environment["XDEBUG_CONFIG"], "client_port=9005")
	suite.Assert().Contains(phpService.Command, "xdebug.client_port=9005")
}

func (suite *XdebugSuite) TestXdebugNotEnabledByDefault() {
	// Test that Xdebug is not enabled when debug flag is false
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:    "api",
				Image:   "nginx",
				Runtime: "php:8.3",
				Folder:  "api",
				Debug:   false,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	phpService := compose.Services["api-php"]
	
	// Xdebug environment variables should not be set
	suite.Assert().Empty(phpService.Environment["XDEBUG_MODE"])
	suite.Assert().Empty(phpService.Environment["XDEBUG_CONFIG"])
	suite.Assert().Empty(phpService.Environment["XDEBUG_SESSION"])
	suite.Assert().Empty(phpService.Environment["XDEBUG_TRIGGER"])
	
	// Command should not contain Xdebug installation
	suite.Assert().NotContains(phpService.Command, "xdebug")
}

func (suite *XdebugSuite) TestXdebugWithLaravelFramework() {
	// Test Xdebug with Laravel framework
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "laravel-app",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "laravel",
				Folder:    "app",
				Debug:     true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	phpService := compose.Services["laravel-app-php"]
	
	// Should have both Laravel and Xdebug configuration
	suite.Assert().Equal("production", phpService.Environment["APP_ENV"])
	suite.Assert().Equal("develop,debug,coverage", phpService.Environment["XDEBUG_MODE"])
	suite.Assert().Equal("serverName=laravel-app", phpService.Environment["PHP_IDE_CONFIG"])
}

func (suite *XdebugSuite) TestXdebugInstallationCommand() {
	// Test that Xdebug installation command is properly formatted
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Folder:    "api",
				Debug:     true,
				DebugPort: 9004,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	phpService := compose.Services["api-php"]
	
	// Check that the command includes proper Xdebug configuration
	expectedConfigs := []string{
		"xdebug.mode=develop,debug,coverage",
		"xdebug.client_host=host.docker.internal",
		fmt.Sprintf("xdebug.client_port=%d", 9004),
		"xdebug.start_with_request=yes",
		"xdebug.log=/tmp/xdebug.log",
	}
	
	for _, config := range expectedConfigs {
		suite.Assert().Contains(phpService.Command, config, "Command should contain Xdebug config: "+config)
	}
}

func (suite *XdebugSuite) TestXdebugWithMultiplePHPServices() {
	// Test that Xdebug can be enabled for multiple PHP services
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Folder:    "api",
				Debug:     true,
				DebugPort: 9003,
			},
			{
				Name:      "admin",
				Image:     "nginx",
				Runtime:   "php:8.2",
				Folder:    "admin",
				Debug:     true,
				DebugPort: 9004,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Check API PHP service
	apiPhpService := compose.Services["api-php"]
	suite.Assert().Contains(apiPhpService.Environment["XDEBUG_CONFIG"], "client_port=9003")
	suite.Assert().Equal("serverName=api", apiPhpService.Environment["PHP_IDE_CONFIG"])
	
	// Check Admin PHP service
	adminPhpService := compose.Services["admin-php"]
	suite.Assert().Contains(adminPhpService.Environment["XDEBUG_CONFIG"], "client_port=9004")
	suite.Assert().Equal("serverName=admin", adminPhpService.Environment["PHP_IDE_CONFIG"])
	
	// Both should have host.docker.internal configured
	suite.Assert().Contains(apiPhpService.ExtraHosts, "host.docker.internal:host-gateway")
	suite.Assert().Contains(adminPhpService.ExtraHosts, "host.docker.internal:host-gateway")
}

func TestXdebugSuite(t *testing.T) {
	suite.Run(t, new(XdebugSuite))
}