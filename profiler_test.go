package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProfilerTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *ProfilerTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
	// Change to temp directory for testing
	os.Chdir(suite.helper.TempDir())
}

func (suite *ProfilerTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func TestProfilerSuite(t *testing.T) {
	suite.Run(t, new(ProfilerTestSuite))
}

// TestProfileEnabled tests that profile mode is correctly configured
func (suite *ProfilerTestSuite) TestProfileEnabled() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:latest",
				Runtime: "php:8.4",
				Folder:  "app",
				Profile: true,
			},
		},
	}

	compose := generateDockerCompose(config)
	suite.Require().NotNil(compose)

	// Check PHP service exists
	phpService, exists := compose.Services["web-php"]
	suite.Require().True(exists, "PHP service should exist")

	// Check Xdebug mode includes profile
	xdebugMode := phpService.Environment["XDEBUG_MODE"]
	suite.Contains(xdebugMode, "profile", "Xdebug mode should include profile")

	// Check profiler output directory is set
	profilerDir := phpService.Environment["XDEBUG_PROFILER_OUTPUT_DIR"]
	suite.Equal("/var/www/profiles", profilerDir)

	// Check volume mount for profiles
	hasProfileVolume := false
	for _, volume := range phpService.Volumes {
		if strings.Contains(volume, ":/var/www/profiles") {
			hasProfileVolume = true
			break
		}
	}
	suite.True(hasProfileVolume, "Should have profile volume mount")
}

// TestProfileWithCustomOutput tests custom profile output directory
func (suite *ProfilerTestSuite) TestProfileWithCustomOutput() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:          "api",
				Image:         "nginx:latest",
				Runtime:       "php:8.3",
				Folder:        "api",
				Profile:       true,
				ProfileOutput: "custom/profiles",
			},
		},
	}

	compose := generateDockerCompose(config)
	suite.Require().NotNil(compose)

	phpService, exists := compose.Services["api-php"]
	suite.Require().True(exists)

	// Check custom volume mount
	hasCustomVolume := false
	for _, volume := range phpService.Volumes {
		if strings.Contains(volume, "custom/profiles:/var/www/profiles") {
			hasCustomVolume = true
			break
		}
	}
	suite.True(hasCustomVolume, "Should have custom profile volume mount")

	// Verify directory was created
	suite.DirExists("custom/profiles")
}

// TestProfileTriggerModes tests different profiler trigger modes
func (suite *ProfilerTestSuite) TestProfileTriggerModes() {
	testCases := []struct {
		name           string
		profileTrigger string
		expectedEnvs   map[string]string
	}{
		{
			name:           "Request trigger mode",
			profileTrigger: "request",
			expectedEnvs: map[string]string{
				"XDEBUG_PROFILER_ENABLE_TRIGGER": "1",
				"XDEBUG_TRIGGER_VALUE":            "PROFILE",
			},
		},
		{
			name:           "Always trigger mode",
			profileTrigger: "always",
			expectedEnvs: map[string]string{
				"XDEBUG_PROFILER_ENABLE": "1",
			},
		},
		{
			name:           "Default trigger mode (request)",
			profileTrigger: "",
			expectedEnvs: map[string]string{
				"XDEBUG_PROFILER_ENABLE_TRIGGER": "1",
				"XDEBUG_TRIGGER_VALUE":            "PROFILE",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := &Config{
				Project: "test",
				Services: []Service{
					{
						Name:           "web",
						Image:          "nginx:latest",
						Runtime:        "php:8.4",
						Folder:         "app",
						Profile:        true,
						ProfileTrigger: tc.profileTrigger,
					},
				},
			}

			compose := generateDockerCompose(config)
			phpService, exists := compose.Services["web-php"]
			suite.Require().True(exists)

			// Check expected environment variables
			for key, expectedValue := range tc.expectedEnvs {
				actualValue, exists := phpService.Environment[key]
				suite.True(exists, "Environment variable %s should exist", key)
				suite.Equal(expectedValue, actualValue, "Environment variable %s should match", key)
			}
		})
	}
}

// TestProfileWithDebug tests that both debug and profile can be enabled together
func (suite *ProfilerTestSuite) TestProfileWithDebug() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:      "web",
				Image:     "nginx:latest",
				Runtime:   "php:8.4",
				Folder:    "app",
				Debug:     true,
				DebugPort: 9003,
				Profile:   true,
			},
		},
	}

	compose := generateDockerCompose(config)
	phpService, exists := compose.Services["web-php"]
	suite.Require().True(exists)

	// Check Xdebug mode includes both debug and profile
	xdebugMode := phpService.Environment["XDEBUG_MODE"]
	suite.Contains(xdebugMode, "debug", "Xdebug mode should include debug")
	suite.Contains(xdebugMode, "profile", "Xdebug mode should include profile")
	suite.Contains(xdebugMode, "develop", "Xdebug mode should include develop")
	suite.Contains(xdebugMode, "coverage", "Xdebug mode should include coverage")

	// Check debug configuration is still present
	xdebugConfig := phpService.Environment["XDEBUG_CONFIG"]
	suite.Contains(xdebugConfig, "client_port=9003")
	suite.Contains(xdebugConfig, "client_host=host.docker.internal")

	// Check profiler configuration
	profilerDir := phpService.Environment["XDEBUG_PROFILER_OUTPUT_DIR"]
	suite.Equal("/var/www/profiles", profilerDir)
}

// TestProfileOnlyNoDebug tests profile-only mode without debug
func (suite *ProfilerTestSuite) TestProfileOnlyNoDebug() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:latest",
				Runtime: "php:8.4",
				Folder:  "app",
				Profile: true,
				Debug:   false,
			},
		},
	}

	compose := generateDockerCompose(config)
	phpService, exists := compose.Services["web-php"]
	suite.Require().True(exists)

	// Check Xdebug mode only has profile (and develop)
	xdebugMode := phpService.Environment["XDEBUG_MODE"]
	suite.Contains(xdebugMode, "profile", "Xdebug mode should include profile")
	suite.Contains(xdebugMode, "develop", "Xdebug mode should include develop")
	suite.NotContains(xdebugMode, "debug", "Xdebug mode should not include debug when debug is false")

	// Check Composer is still installed
	suite.Contains(phpService.Command, "composer")
	suite.Contains(phpService.Command, "xdebug")
}

// TestProfileDirectoryCreation tests that profile directories are created
func (suite *ProfilerTestSuite) TestProfileDirectoryCreation() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:latest",
				Runtime: "php:8.4",
				Folder:  "app",
				Profile: true,
			},
		},
	}

	generateDockerCompose(config)

	// Check that .fleet/profiles directory was created
	profileDir := filepath.Join(".fleet", "profiles")
	suite.DirExists(profileDir, "Profile directory should be created")
}

// TestMultipleServicesWithProfile tests multiple PHP services with profiling
func (suite *ProfilerTestSuite) TestMultipleServicesWithProfile() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:           "web",
				Image:          "nginx:latest",
				Runtime:        "php:8.4",
				Folder:         "frontend",
				Profile:        true,
				ProfileTrigger: "request",
			},
			{
				Name:           "api",
				Image:          "nginx:latest",
				Runtime:        "php:8.3",
				Folder:         "backend",
				Profile:        true,
				ProfileTrigger: "always",
				ProfileOutput:  "api-profiles",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check web-php service
	webPHP, exists := compose.Services["web-php"]
	suite.Require().True(exists)
	suite.Equal("1", webPHP.Environment["XDEBUG_PROFILER_ENABLE_TRIGGER"])
	
	// Check api-php service
	apiPHP, exists := compose.Services["api-php"]
	suite.Require().True(exists)
	suite.Equal("1", apiPHP.Environment["XDEBUG_PROFILER_ENABLE"])
	
	// Check different profile outputs
	webHasDefaultVolume := false
	for _, volume := range webPHP.Volumes {
		if strings.Contains(volume, ".fleet/profiles:/var/www/profiles") {
			webHasDefaultVolume = true
			break
		}
	}
	suite.True(webHasDefaultVolume)

	apiHasCustomVolume := false
	for _, volume := range apiPHP.Volumes {
		if strings.Contains(volume, "api-profiles:/var/www/profiles") {
			apiHasCustomVolume = true
			break
		}
	}
	suite.True(apiHasCustomVolume)
}

// TestNoProfileByDefault tests that profiling is not enabled by default
func (suite *ProfilerTestSuite) TestNoProfileByDefault() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:latest",
				Runtime: "php:8.4",
				Folder:  "app",
			},
		},
	}

	compose := generateDockerCompose(config)
	phpService, exists := compose.Services["web-php"]
	suite.Require().True(exists)

	// Check that profile is not in Xdebug mode
	xdebugMode, hasMode := phpService.Environment["XDEBUG_MODE"]
	if hasMode {
		suite.NotContains(xdebugMode, "profile", "Profile should not be enabled by default")
	}

	// Check no profiler environment variables
	suite.NotContains(phpService.Environment, "XDEBUG_PROFILER_OUTPUT_DIR")
	suite.NotContains(phpService.Environment, "XDEBUG_PROFILER_ENABLE")
	suite.NotContains(phpService.Environment, "XDEBUG_PROFILER_ENABLE_TRIGGER")

	// Check no profile volume
	for _, volume := range phpService.Volumes {
		suite.NotContains(volume, "profiles", "Should not have profile volume when profiling is disabled")
	}
}

// TestXdebugInstallCommand tests that Xdebug install command includes profiler config
func (suite *ProfilerTestSuite) TestXdebugInstallCommand() {
	configurator := NewPHPConfigurator()
	svc := &Service{
		Name:           "web",
		Runtime:        "php:8.4",
		Profile:        true,
		ProfileTrigger: "request",
	}

	xdebugSettings := configurator.ConfigureXdebug(svc)
	suite.True(xdebugSettings.ProfileEnabled)
	suite.Equal("request", xdebugSettings.ProfileTrigger)

	// Generate install command
	installCmd := xdebugSettings.generateInstallCommand()
	
	// Check that command includes profiler configuration
	suite.Contains(installCmd, "xdebug.output_dir=/var/www/profiles")
	suite.Contains(installCmd, "mkdir -p /var/www/profiles")
	suite.Contains(installCmd, "chmod 777 /var/www/profiles")
	suite.Contains(installCmd, "xdebug.trigger_value=PROFILE")
}