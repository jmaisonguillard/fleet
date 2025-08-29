package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

// NginxIntegrationTestSuite tests the nginx integration end-to-end
type NginxIntegrationTestSuite struct {
	suite.Suite
	tempDir string
}

func (suite *NginxIntegrationTestSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "fleet-nginx-integration-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *NginxIntegrationTestSuite) TearDownTest() {
	// Clean up temp directory
	os.RemoveAll(suite.tempDir)
}

// TestNginxVolumeMount_FileExistsBeforeDockerCompose tests that nginx.conf is created before Docker Compose references it
func (suite *NginxIntegrationTestSuite) TestNginxVolumeMount_FileExistsBeforeDockerCompose() {
	// Given: A config that requires nginx proxy
	config := &Config{
		Project: "test-app",
		Services: []Service{
			{
				Name:   "web",
				Image:  "nginx:latest",
				Domain: "web.test",
				Port:   8080,
				Ports:  []string{"8080:80"},
			},
			{
				Name:   "api",
				Image:  "node:latest",
				Port:   3000,
				Ports:  []string{"3000:3000"},
			},
		},
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(suite.tempDir)

	// When: Generating docker-compose with nginx proxy
	compose := generateDockerCompose(config)

	// Then: nginx.conf should be created and accessible before docker-compose.yml
	nginxConfigPath := filepath.Join(suite.tempDir, ".fleet", "nginx.conf")
	
	// Verify the file was created
	info, err := os.Stat(nginxConfigPath)
	suite.NoError(err, "nginx.conf should exist")
	suite.False(info.IsDir(), "nginx.conf should be a file, not a directory")
	suite.Equal(os.FileMode(0644), info.Mode().Perm(), "nginx.conf should have correct permissions")

	// Verify the nginx-proxy service exists
	nginxService, exists := compose.Services["nginx-proxy"]
	suite.True(exists, "nginx-proxy service should exist")

	// Verify the volume mount uses absolute path
	suite.Len(nginxService.Volumes, 1, "Should have one volume mount")
	volumeMount := nginxService.Volumes[0]
	
	// The volume mount should use absolute path
	suite.Contains(volumeMount, nginxConfigPath, "Volume mount should use absolute path to nginx.conf")
	suite.Contains(volumeMount, ":/etc/nginx/nginx.conf:ro", "Should mount to /etc/nginx/nginx.conf as read-only")

	// Verify nginx.conf content
	content, err := os.ReadFile(nginxConfigPath)
	suite.NoError(err, "Should be able to read nginx.conf")
	nginxContent := string(content)
	
	// Check for expected upstream definitions
	suite.Contains(nginxContent, "upstream web_backend", "Should contain web upstream")
	suite.Contains(nginxContent, "server web:80", "Should contain web server")
	suite.Contains(nginxContent, "upstream api_backend", "Should contain api upstream")
	suite.Contains(nginxContent, "server api:3000", "Should contain api server")
	
	// Check for virtual hosts
	suite.Contains(nginxContent, "server_name web.test", "Should contain web.test server")
	suite.Contains(nginxContent, "server_name api.test", "Should contain api.test server")
}

// TestNginxVolumeMount_DockerComposeYAML tests the generated docker-compose.yml file
func (suite *NginxIntegrationTestSuite) TestNginxVolumeMount_DockerComposeYAML() {
	// Given: A config with nginx requirements
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "frontend",
				Image:  "node:alpine",
				Domain: "app.test",
				Port:   3000,
			},
		},
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(suite.tempDir)

	// When: Generating and writing docker-compose
	compose := generateDockerCompose(config)
	composeFile := filepath.Join(suite.tempDir, "docker-compose.yml")
	err := writeDockerCompose(compose, composeFile)
	suite.NoError(err, "Should write docker-compose.yml successfully")

	// Then: Parse the YAML file to verify structure
	yamlContent, err := os.ReadFile(composeFile)
	suite.NoError(err, "Should read docker-compose.yml")

	var parsedCompose map[string]interface{}
	err = yaml.Unmarshal(yamlContent, &parsedCompose)
	suite.NoError(err, "Should parse YAML successfully")

	// Verify nginx-proxy service in YAML
	services, ok := parsedCompose["services"].(map[string]interface{})
	suite.True(ok, "Should have services section")
	
	nginxProxy, ok := services["nginx-proxy"].(map[string]interface{})
	suite.True(ok, "Should have nginx-proxy service")
	
	// Check volumes in the parsed YAML
	volumes, ok := nginxProxy["volumes"].([]interface{})
	suite.True(ok, "nginx-proxy should have volumes")
	suite.Len(volumes, 1, "Should have one volume")
	
	volumeStr, ok := volumes[0].(string)
	suite.True(ok, "Volume should be a string")
	
	// Verify the volume path is absolute
	parts := strings.Split(volumeStr, ":")
	suite.Len(parts, 3, "Volume should be in source:dest:mode format")
	
	sourcePath := parts[0]
	suite.True(filepath.IsAbs(sourcePath), "Source path should be absolute")
	suite.Contains(sourcePath, "nginx.conf", "Source should reference nginx.conf")
	
	// Verify the file exists at the specified path
	_, err = os.Stat(sourcePath)
	suite.NoError(err, "nginx.conf should exist at the specified absolute path")
}

// TestNginxVolumeMount_NoDirectoryCreation tests that Docker doesn't create a directory for nginx.conf
func (suite *NginxIntegrationTestSuite) TestNginxVolumeMount_NoDirectoryCreation() {
	// Given: A config requiring nginx
	config := &Config{
		Project: "test-app",
		Services: []Service{
			{
				Name:   "backend",
				Image:  "python:3.9",
				Domain: "api.test",
				Port:   8000,
			},
		},
	}

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(suite.tempDir)

	// When: Generating docker-compose
	compose := generateDockerCompose(config)

	// Then: Verify nginx.conf is a file, not a directory
	fleetDir := filepath.Join(suite.tempDir, ".fleet")
	nginxConfigPath := filepath.Join(fleetDir, "nginx.conf")
	
	// Check that .fleet exists and is a directory
	fleetInfo, err := os.Stat(fleetDir)
	suite.NoError(err, ".fleet directory should exist")
	suite.True(fleetInfo.IsDir(), ".fleet should be a directory")
	
	// Check that nginx.conf exists and is a file
	nginxInfo, err := os.Stat(nginxConfigPath)
	suite.NoError(err, "nginx.conf should exist")
	suite.False(nginxInfo.IsDir(), "nginx.conf MUST be a file, not a directory")
	suite.True(nginxInfo.Mode().IsRegular(), "nginx.conf should be a regular file")
	
	// Verify the volume mount in compose references this file
	nginxService := compose.Services["nginx-proxy"]
	suite.Len(nginxService.Volumes, 1)
	suite.Contains(nginxService.Volumes[0], nginxConfigPath)
}

func TestNginxIntegrationSuite(t *testing.T) {
	suite.Run(t, new(NginxIntegrationTestSuite))
}