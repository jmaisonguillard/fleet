package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PHPRuntimeTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *PHPRuntimeTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *PHPRuntimeTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *PHPRuntimeTestSuite) TestParsePHPRuntime() {
	testCases := []struct {
		runtime      string
		expectedLang string
		expectedVer  string
	}{
		{"php", "php", "8.4"},
		{"php:8.2", "php", "8.2"},
		{"php:7.4", "php", "7.4"},
		{"php:8.4", "php", "8.4"},
		{"python", "", ""},
		{"", "", ""},
		{"ruby:3.0", "", ""},
	}

	for _, tc := range testCases {
		lang, version := parsePHPRuntime(tc.runtime)
		suite.Equal(tc.expectedLang, lang, "Runtime: %s", tc.runtime)
		suite.Equal(tc.expectedVer, version, "Runtime: %s", tc.runtime)
	}
}

func (suite *PHPRuntimeTestSuite) TestGetPHPImage() {
	testCases := []struct {
		version  string
		expected string
	}{
		{"8.4", "php:8.4-fpm-alpine"},
		{"8.3", "php:8.3-fpm-alpine"},
		{"8.2", "php:8.2-fpm-alpine"},
		{"8.1", "php:8.1-fpm-alpine"},
		{"8.0", "php:8.0-fpm-alpine"},
		{"7.4", "php:7.4-fpm-alpine"},
		{"latest", "php:8.4-fpm-alpine"},
		{"default", "php:8.4-fpm-alpine"},
		{"", "php:8.4-fpm-alpine"},
		{"9.0", "php:9.0-fpm-alpine"}, // Future version
		{"invalid", "php:8.4-fpm-alpine"}, // Invalid defaults to 8.4
	}

	for _, tc := range testCases {
		result := getPHPImage(tc.version)
		suite.Equal(tc.expected, result, "Version: %s", tc.version)
	}
}

func (suite *PHPRuntimeTestSuite) TestAddPHPFPMService() {
	config := &Config{
		Project: "test-app",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:alpine",
				Port:    80,
				Runtime: "php:8.2",
				Folder:  "./app",
				Environment: map[string]string{
					"APP_ENV": "production",
				},
			},
		},
	}

	compose := &DockerCompose{
		Version:  "3.8",
		Services: make(map[string]DockerService),
		Networks: make(map[string]DockerNetwork),
	}

	// Add the nginx service first
	compose.Services["web"] = DockerService{
		Image:     "nginx:alpine",
		Networks:  []string{"fleet-network"},
		DependsOn: []string{},
	}

	// Add PHP-FPM service
	addPHPFPMService(compose, &config.Services[0], config)

	// Check PHP service was created
	phpService, exists := compose.Services["web-php"]
	suite.True(exists, "PHP-FPM service should be created")
	suite.Equal("php:8.2-fpm-alpine", phpService.Image)
	suite.Contains(phpService.Networks, "fleet-network")
	suite.Equal("www-data", phpService.Environment["PHP_FPM_USER"])
	suite.Equal("www-data", phpService.Environment["PHP_FPM_GROUP"])
	suite.Equal("production", phpService.Environment["APP_ENV"])

	// Check volumes
	suite.Contains(phpService.Volumes, ".././app:/var/www/html")

	// Check health check
	suite.NotNil(phpService.HealthCheck)
	suite.Contains(phpService.HealthCheck.Test, "CMD-SHELL")

	// Check nginx depends on PHP
	webService := compose.Services["web"]
	suite.Contains(webService.DependsOn, "web-php")
}

func (suite *PHPRuntimeTestSuite) TestGenerateNginxPHPConfig() {
	config := generateNginxPHPConfig("myapp")
	
	// Check key PHP-FPM configurations
	suite.Contains(config, "fastcgi_pass myapp-php:9000")
	suite.Contains(config, "index.php")
	suite.Contains(config, "location ~ \\.php$")
	suite.Contains(config, "fastcgi_param SCRIPT_FILENAME")
	suite.Contains(config, "try_files $uri $uri/ /index.php?$query_string")
}

func (suite *PHPRuntimeTestSuite) TestPHPIntegrationInCompose() {
	config := &Config{
		Project: "php-app",
		Services: []Service{
			{
				Name:    "website",
				Image:   "nginx:alpine",
				Port:    80,
				Domain:  "php.test",
				Runtime: "php:8.3",
				Folder:  "./src",
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,
				Runtime: "node", // Should be ignored
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that PHP-FPM service was created for nginx+php
	_, hasPhpService := compose.Services["website-php"]
	suite.True(hasPhpService, "PHP-FPM service should be created for nginx with runtime=php")

	// Check that no PHP service was created for non-nginx service
	_, hasApiPhp := compose.Services["api-php"]
	suite.False(hasApiPhp, "PHP-FPM service should not be created for non-nginx services")
}

func (suite *PHPRuntimeTestSuite) TestNginxVolumeWithPHP() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "web",
				Image:   "nginx:alpine",
				Runtime: "php",
				Folder:  "./public",
			},
		},
	}

	compose := generateDockerCompose(config)
	webService := compose.Services["web"]

	// Check nginx uses /var/www/html for PHP
	hasCorrectVolume := false
	for _, vol := range webService.Volumes {
		if strings.Contains(vol, "/var/www/html") {
			hasCorrectVolume = true
			break
		}
	}
	suite.True(hasCorrectVolume, "nginx with PHP should mount to /var/www/html")
}

func TestPHPRuntimeSuite(t *testing.T) {
	suite.Run(t, new(PHPRuntimeTestSuite))
}