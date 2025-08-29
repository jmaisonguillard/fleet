package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NodeRuntimeTestSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *NodeRuntimeTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
	// Change to temp directory for testing
	os.Chdir(suite.helper.TempDir())
}

func (suite *NodeRuntimeTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func TestNodeRuntimeSuite(t *testing.T) {
	suite.Run(t, new(NodeRuntimeTestSuite))
}

// TestParseNodeRuntime tests parsing of Node.js runtime strings
func (suite *NodeRuntimeTestSuite) TestParseNodeRuntime() {
	testCases := []struct {
		name            string
		runtime         string
		expectedLang    string
		expectedVersion string
	}{
		{
			name:            "Just node",
			runtime:         "node",
			expectedLang:    "node",
			expectedVersion: "20",
		},
		{
			name:            "Node with version",
			runtime:         "node:18",
			expectedLang:    "node",
			expectedVersion: "18",
		},
		{
			name:            "Node with alpine version",
			runtime:         "node:20-alpine",
			expectedLang:    "node",
			expectedVersion: "20-alpine",
		},
		{
			name:            "Node with full version",
			runtime:         "node:20.11.0",
			expectedLang:    "node",
			expectedVersion: "20.11.0",
		},
		{
			name:            "Empty runtime",
			runtime:         "",
			expectedLang:    "",
			expectedVersion: "",
		},
		{
			name:            "Non-node runtime",
			runtime:         "python:3.9",
			expectedLang:    "",
			expectedVersion: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			lang, version := parseNodeRuntime(tc.runtime)
			suite.Equal(tc.expectedLang, lang)
			suite.Equal(tc.expectedVersion, version)
		})
	}
}

// TestGetNodeImage tests Node.js Docker image selection
func (suite *NodeRuntimeTestSuite) TestGetNodeImage() {
	testCases := []struct {
		name          string
		version       string
		expectedImage string
	}{
		{
			name:          "Version 20",
			version:       "20",
			expectedImage: "node:20-alpine",
		},
		{
			name:          "Version 18",
			version:       "18",
			expectedImage: "node:18-alpine",
		},
		{
			name:          "Version 16",
			version:       "16",
			expectedImage: "node:16-alpine",
		},
		{
			name:          "LTS version",
			version:       "lts",
			expectedImage: "node:20-alpine",
		},
		{
			name:          "Latest version",
			version:       "latest",
			expectedImage: "node:20-alpine",
		},
		{
			name:          "Empty version (default)",
			version:       "",
			expectedImage: "node:20-alpine",
		},
		{
			name:          "Specific version",
			version:       "20.11.0",
			expectedImage: "node:20.11.0-alpine",
		},
		{
			name:          "Unknown version",
			version:       "99",
			expectedImage: "node:99-alpine",
		},
		{
			name:          "Invalid version",
			version:       "invalid",
			expectedImage: "node:20-alpine",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getNodeImage(tc.version)
			suite.Equal(tc.expectedImage, image)
		})
	}
}

// TestDetectPackageManager tests package manager detection
func (suite *NodeRuntimeTestSuite) TestDetectPackageManager() {
	suite.Run("Detect npm", func() {
		// Create package-lock.json
		suite.helper.CreateFile("test-npm/package-lock.json", "{}")
		pm := detectPackageManager("test-npm")
		suite.Equal("npm", pm)
	})

	suite.Run("Detect yarn", func() {
		// Create yarn.lock
		suite.helper.CreateFile("test-yarn/yarn.lock", "")
		pm := detectPackageManager("test-yarn")
		suite.Equal("yarn", pm)
	})

	suite.Run("Detect pnpm", func() {
		// Create pnpm-lock.yaml
		suite.helper.CreateFile("test-pnpm/pnpm-lock.yaml", "")
		pm := detectPackageManager("test-pnpm")
		suite.Equal("pnpm", pm)
	})

	suite.Run("Default to npm", func() {
		pm := detectPackageManager("nonexistent")
		suite.Equal("npm", pm)
	})

	suite.Run("Empty folder", func() {
		pm := detectPackageManager("")
		suite.Equal("npm", pm)
	})
}

// TestIsNodeBuildMode tests build mode detection
func (suite *NodeRuntimeTestSuite) TestIsNodeBuildMode() {
	testCases := []struct {
		name     string
		service  Service
		expected bool
	}{
		{
			name: "Service with nginx image and node runtime",
			service: Service{
				Image:   "nginx:alpine",
				Runtime: "node:20",
			},
			expected: true,
		},
		{
			name: "Service with build command",
			service: Service{
				BuildCommand: "npm run build",
			},
			expected: true,
		},
		{
			name: "Standalone Node service",
			service: Service{
				Runtime: "node:20",
			},
			expected: false,
		},
		{
			name: "Service without node runtime",
			service: Service{
				Image: "nginx:alpine",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := isNodeBuildMode(&tc.service)
			suite.Equal(tc.expected, result)
		})
	}
}

// TestGetNodePort tests port detection for Node.js services
func (suite *NodeRuntimeTestSuite) TestGetNodePort() {
	testCases := []struct {
		name         string
		service      Service
		expectedPort int
	}{
		{
			name: "Explicit port",
			service: Service{
				Port: 8080,
			},
			expectedPort: 8080,
		},
		{
			name: "Next.js default",
			service: Service{
				Folder: "nextjs-app",
			},
			expectedPort: 3000,
		},
		{
			name: "Angular default",
			service: Service{
				Folder: "angular-app",
			},
			expectedPort: 4200,
		},
		{
			name: "Vue default",
			service: Service{
				Folder: "vue-app",
			},
			expectedPort: 8080,
		},
		{
			name: "Default port",
			service: Service{},
			expectedPort: 3000,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create mock package.json for framework detection
			if tc.service.Folder != "" {
				var packageContent string
				if strings.Contains(tc.service.Folder, "nextjs") {
					packageContent = `{"dependencies": {"next": "14.0.0"}}`
				} else if strings.Contains(tc.service.Folder, "angular") {
					packageContent = `{"dependencies": {"@angular/core": "17.0.0"}}`
				} else if strings.Contains(tc.service.Folder, "vue") {
					packageContent = `{"dependencies": {"vue": "3.0.0"}}`
				}
				if packageContent != "" {
					suite.helper.CreateFile(filepath.Join(tc.service.Folder, "package.json"), packageContent)
				}
			}
			
			port := getNodePort(&tc.service)
			suite.Equal(tc.expectedPort, port)
		})
	}
}

// TestAddNodeService tests adding Node.js service to Docker Compose
func (suite *NodeRuntimeTestSuite) TestAddNodeService() {
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:    "api",
				Runtime: "node:20",
				Folder:  "api",
				Port:    3000,
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
	}

	// Create mock package.json
	suite.helper.CreateFile("api/package.json", `{
		"name": "api",
		"dependencies": {
			"express": "4.18.0"
		}
	}`)

	// Add Node service
	addNodeService(compose, &config.Services[0], config)

	// Check service was added
	service, exists := compose.Services["api"]
	suite.True(exists, "Node service should be added")
	suite.Contains(service.Image, "node")
	suite.Contains(service.Networks, "fleet-network")
}

// TestNodeIntegrationInCompose tests full integration with Docker Compose
func (suite *NodeRuntimeTestSuite) TestNodeIntegrationInCompose() {
	// Create test config with Node.js service
	configContent := `
project = "test-node"

[[services]]
name = "api"
runtime = "node:20"
folder = "api"
port = 3000

[[services]]
name = "frontend"
image = "nginx:alpine"
runtime = "node:20"
folder = "frontend"
build_command = "npm run build"
`

	// Write config file
	err := os.WriteFile("fleet.toml", []byte(configContent), 0644)
	suite.Require().NoError(err)

	// Create mock package.json files
	suite.helper.CreateFile("api/package.json", `{"dependencies": {"express": "4.18.0"}}`)
	suite.helper.CreateFile("frontend/package.json", `{"dependencies": {"react": "18.0.0"}}`)

	// Load config
	config, err := loadConfig("fleet.toml")
	suite.Require().NoError(err)

	// Generate Docker Compose
	compose := generateDockerCompose(config)
	suite.Require().NotNil(compose)

	// Check API service (service mode)
	apiService, exists := compose.Services["api"]
	suite.True(exists, "API service should exist")
	suite.Contains(apiService.Image, "node:20")
	suite.Contains(apiService.Volumes[0], "api:/app")

	// Check that node_modules volume is created for service mode
	hasNodeModulesVolume := false
	for _, volume := range apiService.Volumes {
		if strings.Contains(volume, "node_modules") {
			hasNodeModulesVolume = true
			break
		}
	}
	suite.True(hasNodeModulesVolume, "Should have node_modules volume for service mode")

	// Check nginx service exists
	nginxService, exists := compose.Services["frontend"]
	suite.True(exists, "Frontend nginx service should exist")
	suite.Equal("nginx:alpine", nginxService.Image)
}