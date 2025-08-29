package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PHPFrameworksTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *PHPFrameworksTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
	// Change to temp directory to isolate test
	originalDir, _ := os.Getwd()
	os.Chdir(suite.helper.tempDir)
	suite.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
}

func (suite *PHPFrameworksTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *PHPFrameworksTestSuite) TestDetectLaravel() {
	// Create Laravel project structure
	projectDir := suite.helper.TempDir()
	
	// Create artisan file (Laravel signature)
	suite.helper.CreateFile("artisan", "#!/usr/bin/env php\n<?php\n// Laravel artisan")
	
	// Create composer.json with Laravel
	composerContent := `{
		"require": {
			"laravel/framework": "^10.0"
		}
	}`
	suite.helper.CreateFile("composer.json", composerContent)
	
	framework := detectPHPFramework(projectDir)
	suite.Equal("laravel", framework)
}

func (suite *PHPFrameworksTestSuite) TestDetectSymfony() {
	projectDir := suite.helper.TempDir()
	
	// Create Symfony signature files
	suite.helper.CreateFile("symfony.lock", "{}")
	suite.helper.CreateFile("bin/console", "#!/usr/bin/env php\n<?php\n// Symfony console")
	
	framework := detectPHPFramework(projectDir)
	suite.Equal("symfony", framework)
}

func (suite *PHPFrameworksTestSuite) TestDetectWordPress() {
	projectDir := suite.helper.TempDir()
	
	// Create WordPress signature files
	suite.helper.CreateFile("wp-config.php", "<?php\n// WordPress config")
	suite.helper.CreateFile("wp-load.php", "<?php\n// WordPress loader")
	
	framework := detectPHPFramework(projectDir)
	suite.Equal("wordpress", framework)
}

func (suite *PHPFrameworksTestSuite) TestDetectNoFramework() {
	projectDir := suite.helper.TempDir()
	
	// Just a plain PHP file
	suite.helper.CreateFile("index.php", "<?php\necho 'Hello World';")
	
	framework := detectPHPFramework(projectDir)
	suite.Equal("", framework)
}

func (suite *PHPFrameworksTestSuite) TestGetNginxConfigForFramework() {
	testCases := []struct {
		framework string
		checks    []string
	}{
		{
			"laravel",
			[]string{
				"root /var/www/html/public",
				"try_files $uri $uri/ /index.php?$query_string",
				"fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name",
			},
		},
		{
			"symfony",
			[]string{
				"root /var/www/html/public",
				"try_files $uri /index.php$is_args$args",
				"fastcgi_param DOCUMENT_ROOT $realpath_root",
			},
		},
		{
			"wordpress",
			[]string{
				"root /var/www/html",
				"try_files $uri $uri/ /index.php?$args",
				"location ~* ^/wp-admin/",
				"location ~* /(?:uploads|files)/.*\\.php$",
			},
		},
		{
			"drupal",
			[]string{
				"root /var/www/html",
				"location ~ ^/sites/.*/files/styles/",
				"location ~ ^(/[a-z\\-]+)?/system/files/",
			},
		},
		{
			"codeigniter",
			[]string{
				"root /var/www/html/public",
				"try_files $uri $uri/ /index.php?/$request_uri",
				"location ~* ^/(system|application|spark|tests|vendor)/",
			},
		},
		{
			"slim",
			[]string{
				"root /var/www/html/public",
				"try_files $uri /index.php$is_args$args",
			},
		},
	}
	
	for _, tc := range testCases {
		config := getNginxConfigForFramework("test-service", tc.framework)
		
		for _, check := range tc.checks {
			suite.Contains(config, check, "Framework %s should contain: %s", tc.framework, check)
		}
		
		// All configs should have PHP-FPM pass
		suite.Contains(config, "fastcgi_pass test-service-php:9000")
	}
}

func (suite *PHPFrameworksTestSuite) TestFrameworkEnvironmentVariables() {
	// Test Laravel environment
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:      "web",
				Image:     "nginx:alpine",
				Runtime:   "php",
				Framework: "laravel",
			},
		},
	}
	
	compose := &DockerCompose{
		Version:  "3.8",
		Services: make(map[string]DockerService),
	}
	
	compose.Services["web"] = DockerService{}
	addPHPFPMService(compose, &config.Services[0], config)
	
	phpService := compose.Services["web-php"]
	suite.Equal("production", phpService.Environment["LARAVEL_ENV"])
	suite.Equal("production", phpService.Environment["APP_ENV"])
	
	// Test Symfony environment
	config.Services[0].Framework = "symfony"
	compose.Services = make(map[string]DockerService)
	compose.Services["web"] = DockerService{}
	addPHPFPMService(compose, &config.Services[0], config)
	
	phpService = compose.Services["web-php"]
	suite.Equal("prod", phpService.Environment["APP_ENV"])
	suite.Equal("0", phpService.Environment["APP_DEBUG"])
	
	// Test WordPress environment
	config.Services[0].Framework = "wordpress"
	compose.Services = make(map[string]DockerService)
	compose.Services["web"] = DockerService{}
	addPHPFPMService(compose, &config.Services[0], config)
	
	phpService = compose.Services["web-php"]
	suite.Equal("production", phpService.Environment["WP_ENV"])
}

func (suite *PHPFrameworksTestSuite) TestWriteNginxPHPConfigWithFramework() {
	// Ensure .fleet directory exists
	os.MkdirAll(".fleet", 0755)
	defer os.RemoveAll(".fleet")
	
	// Test with Laravel framework
	configPath, err := writeNginxPHPConfig("test-app", "laravel")
	suite.NoError(err)
	suite.Equal(filepath.Join(".fleet", "test-app-nginx.conf"), configPath)
	
	// Read and verify the config
	content, err := os.ReadFile(configPath)
	suite.NoError(err)
	suite.Contains(string(content), "root /var/www/html/public")
	suite.Contains(string(content), "Laravel")
	
	// Test with no framework (generic)
	configPath, err = writeNginxPHPConfig("generic-app", "")
	suite.NoError(err)
	
	content, err = os.ReadFile(configPath)
	suite.NoError(err)
	suite.Contains(string(content), "root /var/www/html")
	suite.NotContains(string(content), "/public") // Generic doesn't use public folder
}

func (suite *PHPFrameworksTestSuite) TestAutoDetectionIntegration() {
	// Create a mock Laravel project
	projectDir := suite.helper.TempDir()
	suite.helper.CreateFile("artisan", "#!/usr/bin/env php")
	suite.helper.CreateFile("composer.json", `{"require": {"laravel/framework": "^10.0"}}`)
	
	config := &Config{
		Project: "auto-detect",
		Services: []Service{
			{
				Name:    "app",
				Image:   "nginx:alpine",
				Runtime: "php",
				Folder:  projectDir,
				// No framework specified - should auto-detect
			},
		},
	}
	
	// Simulate compose generation
	compose := generateDockerCompose(config)
	
	// Check PHP-FPM service was created with Laravel settings
	phpService, exists := compose.Services["app-php"]
	suite.True(exists)
	suite.Equal("production", phpService.Environment["LARAVEL_ENV"])
}

func TestPHPFrameworksSuite(t *testing.T) {
	suite.Run(t, new(PHPFrameworksTestSuite))
}