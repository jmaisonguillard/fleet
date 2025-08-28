package main

import (
	"testing"
	
	"github.com/stretchr/testify/suite"
)

type ReverbServiceSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *ReverbServiceSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *ReverbServiceSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *ReverbServiceSuite) TestReverbServiceCreation() {
	// Test that Reverb service is created for Laravel apps
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "laravel",
				Folder:    "api",
				Reverb:    true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Check that reverb service was created
	suite.Assert().Contains(compose.Services, "reverb", "Reverb service should be created")
	
	reverbService := compose.Services["reverb"]
	
	// Check reverb service configuration
	suite.Assert().Equal("php:8.3-cli", reverbService.Image)
	suite.Assert().Contains(reverbService.Command, "reverb:start")
	suite.Assert().Contains(reverbService.Volumes, "./api:/app")
	suite.Assert().Equal("/app", reverbService.WorkingDir)
	
	// Check environment variables
	suite.Assert().Equal("fleet-app", reverbService.Environment["REVERB_APP_ID"])
	suite.Assert().Equal("fleet-app-key", reverbService.Environment["REVERB_APP_KEY"])
	suite.Assert().Equal("fleet-app-secret", reverbService.Environment["REVERB_APP_SECRET"])
	suite.Assert().Equal("reverb", reverbService.Environment["BROADCAST_DRIVER"])
	
	// Check that API service depends on reverb
	apiService := compose.Services["api"]
	suite.Assert().Contains(apiService.DependsOn, "reverb")
}

func (suite *ReverbServiceSuite) TestReverbServiceWithCustomConfig() {
	// Test Reverb with custom configuration
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:            "web",
				Image:           "nginx",
				Runtime:         "php:8.2",
				Framework:       "laravel",
				Folder:          "web",
				Reverb:          true,
				ReverbPort:      8888,
				ReverbAppId:     "custom-app",
				ReverbAppKey:    "custom-key",
				ReverbAppSecret: "custom-secret",
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	reverbService := compose.Services["reverb"]
	
	// Check custom configuration
	suite.Assert().Equal("8888", reverbService.Environment["REVERB_PORT"])
	suite.Assert().Equal("custom-app", reverbService.Environment["REVERB_APP_ID"])
	suite.Assert().Equal("custom-key", reverbService.Environment["REVERB_APP_KEY"])
	suite.Assert().Equal("custom-secret", reverbService.Environment["REVERB_APP_SECRET"])
	
	// Check that web service has matching environment variables
	webService := compose.Services["web"]
	suite.Assert().Equal("custom-app", webService.Environment["REVERB_APP_ID"])
	suite.Assert().Equal("custom-key", webService.Environment["REVERB_APP_KEY"])
	suite.Assert().Equal("8888", webService.Environment["REVERB_PORT"])
}

func (suite *ReverbServiceSuite) TestReverbNotCreatedForNonLaravelApps() {
	// Test that Reverb is not created for non-Laravel apps
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "symfony",
				Folder:    "api",
				Reverb:    true, // Even though true, should not create for Symfony
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Reverb should not be created for non-Laravel apps
	suite.Assert().NotContains(compose.Services, "reverb", "Reverb should not be created for non-Laravel apps")
}

func (suite *ReverbServiceSuite) TestReverbSingletonPattern() {
	// Test that only one Reverb service is created even with multiple Laravel services
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api1",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "laravel",
				Folder:    "api1",
				Reverb:    true,
			},
			{
				Name:      "api2",
				Image:     "nginx",
				Runtime:   "php:8.2",
				Framework: "laravel",
				Folder:    "api2",
				Reverb:    true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Should only have one reverb service
	reverbCount := 0
	for name := range compose.Services {
		if name == "reverb" {
			reverbCount++
		}
	}
	suite.Assert().Equal(1, reverbCount, "Should only have one Reverb service")
	
	// Both services should depend on the same reverb service
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Assert().Contains(api1Service.DependsOn, "reverb")
	suite.Assert().Contains(api2Service.DependsOn, "reverb")
}

func (suite *ReverbServiceSuite) TestReverbHealthCheck() {
	// Test that Reverb service has proper health check
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "api",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "laravel",
				Folder:    "api",
				Reverb:    true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	reverbService := compose.Services["reverb"]
	suite.Assert().NotNil(reverbService.HealthCheck, "Reverb should have health check")
	suite.Assert().Contains(reverbService.HealthCheck.Test[0], "CMD-SHELL")
	suite.Assert().Contains(reverbService.HealthCheck.Test[1], "curl")
	suite.Assert().Contains(reverbService.HealthCheck.Test[1], "localhost:8080/health")
}

func (suite *ReverbServiceSuite) TestReverbEnvironmentVariables() {
	// Test that app service receives correct Reverb environment variables
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:      "web",
				Image:     "nginx",
				Runtime:   "php:8.3",
				Framework: "laravel",
				Folder:    "web",
				Reverb:    true,
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	webService := compose.Services["web"]
	
	// Check broadcasting-related environment variables
	suite.Assert().Equal("reverb", webService.Environment["BROADCAST_DRIVER"])
	suite.Assert().Equal("reverb", webService.Environment["BROADCAST_CONNECTION"])
	
	// Check Vite-related environment variables for frontend
	suite.Assert().Equal("fleet-app", webService.Environment["VITE_REVERB_APP_ID"])
	suite.Assert().Equal("fleet-app-key", webService.Environment["VITE_REVERB_APP_KEY"])
	suite.Assert().Equal("reverb", webService.Environment["VITE_REVERB_HOST"])
	suite.Assert().Equal("8080", webService.Environment["VITE_REVERB_PORT"])
	suite.Assert().Equal("http", webService.Environment["VITE_REVERB_SCHEME"])
}

func TestReverbServiceSuite(t *testing.T) {
	suite.Run(t, new(ReverbServiceSuite))
}