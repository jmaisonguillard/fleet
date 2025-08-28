package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite tests full command flows with mocked Docker
type IntegrationTestSuite struct {
	suite.Suite
	helper     *TestHelper
	dockerMock *DockerMock
	originalDir string
	outputBuf   *bytes.Buffer
}

func (suite *IntegrationTestSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
	suite.dockerMock = MockDockerForTest(suite.T())
	suite.originalDir, _ = os.Getwd()
	suite.outputBuf = new(bytes.Buffer)
}

func (suite *IntegrationTestSuite) TearDownTest() {
	os.Chdir(suite.originalDir)
	suite.dockerMock.Cleanup()
	suite.helper.Cleanup()
}

func (suite *IntegrationTestSuite) TestFullFleetWorkflow() {
	// Change to temp directory
	os.Chdir(suite.helper.TempDir())
	
	// 1. Test fleet init
	suite.T().Log("Testing fleet init...")
	handleInit()
	suite.FileExists("fleet.toml")
	suite.FileExists("website/index.html")
	
	// 2. Test loading the created config
	config, err := loadConfig("fleet.toml")
	suite.NoError(err)
	suite.NotEmpty(config.Project)
	
	// 3. Test generating Docker Compose
	compose := generateDockerCompose(config)
	suite.NotNil(compose)
	suite.NotEmpty(compose.Services)
	
	// 4. Test writing Docker Compose file
	err = os.MkdirAll(".fleet", 0755)
	suite.NoError(err)
	
	composeFile := ".fleet/docker-compose.yml"
	err = writeDockerCompose(compose, composeFile)
	suite.NoError(err)
	suite.FileExists(composeFile)
	
	// 5. Test that mock Docker is working
	suite.True(suite.dockerMock.VerifyDockerCommand())
	
	// 6. Test running Docker commands with mock
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	output, err := cmd.CombinedOutput()
	suite.NoError(err)
	suite.Contains(string(output), "Creating")
}

func (suite *IntegrationTestSuite) TestDockerComposeOperations() {
	// Create a test config
	configContent := `
project = "integration-test"

[[services]]
name = "web"
image = "nginx:alpine"
port = 8080
`
	configPath := suite.helper.CreateFile("fleet.toml", configContent)
	
	// Load config
	config, err := loadConfig(configPath)
	suite.NoError(err)
	
	// Generate compose
	compose := generateDockerCompose(config)
	
	// Write compose file
	composeFile := filepath.Join(suite.helper.TempDir(), "docker-compose.yml")
	err = writeDockerCompose(compose, composeFile)
	suite.NoError(err)
	
	// Test Docker commands with mock
	tests := []struct {
		name    string
		command []string
		check   string
	}{
		{
			name:    "docker compose up",
			command: []string{"docker", "compose", "-f", composeFile, "up", "-d"},
			check:   "Creating",
		},
		{
			name:    "docker compose ps",
			command: []string{"docker", "compose", "-f", composeFile, "ps"},
			check:   "STATUS",
		},
		{
			name:    "docker compose down",
			command: []string{"docker", "compose", "-f", composeFile, "down"},
			check:   "Stopping",
		},
	}
	
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tt.command[0], tt.command[1:]...)
			output, err := cmd.CombinedOutput()
			
			if err != nil {
				t.Logf("Command failed: %v\nOutput: %s", err, output)
			}
			
			suite.Contains(string(output), tt.check)
		})
	}
}

func (suite *IntegrationTestSuite) TestDNSCommandsWithMock() {
	// Test DNS status with mock
	cmd := exec.Command("docker", "ps", "--filter", "name=dnsmasq")
	output, _ := cmd.CombinedOutput()
	suite.Contains(string(output), "dnsmasq")
	
	// Test DNS logs with mock
	cmd = exec.Command("docker", "logs", "dnsmasq", "--tail", "10")
	output, _ = cmd.CombinedOutput()
	suite.Contains(string(output), "dnsmasq[1]")
}

func (suite *IntegrationTestSuite) TestRunDockerWithMock() {
	// Test that runDocker would work with our mock
	suite.True(suite.dockerMock.VerifyDockerCommand())
	
	// Verify Docker version command
	cmd := exec.Command("docker", "version")
	output, err := cmd.CombinedOutput()
	suite.NoError(err)
	suite.Contains(string(output), "Docker version")
}

func (suite *IntegrationTestSuite) TestConfigToComposeFlow() {
	// Create a comprehensive config
	configContent := `
project = "test-app"

[[services]]
name = "frontend"
image = "nginx:alpine"
port = 80
folder = "./public"

[[services]]
name = "backend"
image = "node:18"
port = 3000
needs = ["database"]
[services.env]
NODE_ENV = "production"
DATABASE_URL = "postgresql://postgres:password@database:5432/app"

[[services]]
name = "database"
image = "postgres:15"
port = 5432
password = "password"
volumes = ["db-data:/var/lib/postgresql/data"]
`
	
	configPath := suite.helper.CreateFile("fleet.toml", configContent)
	
	// Load and process
	config, err := loadConfig(configPath)
	suite.NoError(err)
	suite.Len(config.Services, 3)
	
	// Generate Docker Compose
	compose := generateDockerCompose(config)
	suite.NotNil(compose)
	
	// Verify structure
	suite.Contains(compose.Services, "frontend")
	suite.Contains(compose.Services, "backend")
	suite.Contains(compose.Services, "database")
	
	// Check dependencies
	backend := compose.Services["backend"]
	suite.Contains(backend.DependsOn, "database")
	
	// Check environment
	suite.NotEmpty(backend.Environment)
	
	// Check volumes
	database := compose.Services["database"]
	suite.NotEmpty(database.Volumes)
}

func (suite *IntegrationTestSuite) TestErrorHandling() {
	// Test with non-existent config
	_, err := loadConfig("non-existent.toml")
	suite.Error(err)
	
	// Test with invalid YAML writing
	compose := &DockerCompose{}
	err = writeDockerCompose(compose, "/invalid/path/compose.yml")
	suite.Error(err)
}

func TestIntegrationSuite(t *testing.T) {
	// Only run integration tests if explicitly requested or in CI
	if os.Getenv("RUN_INTEGRATION") != "1" && !IsTestEnvironment() {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION=1 to run")
	}
	
	suite.Run(t, new(IntegrationTestSuite))
}

// BenchmarkDockerCompose benchmarks Docker Compose generation
func BenchmarkDockerComposeGeneration(b *testing.B) {
	config := Config{
		Project: "benchmark",
		Services: []Service{
			{Name: "web", Image: "nginx", Port: 80},
			{Name: "api", Image: "node", Port: 3000},
			{Name: "db", Image: "postgres", Port: 5432},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDockerCompose(&config)
	}
}

// BenchmarkConfigLoading benchmarks config file loading
func BenchmarkConfigLoading(b *testing.B) {
	// Create a temp config file
	tempDir, _ := os.MkdirTemp("", "bench-*")
	defer os.RemoveAll(tempDir)
	
	configPath := filepath.Join(tempDir, "fleet.toml")
	os.WriteFile(configPath, []byte(SampleFleetConfig()), 0644)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loadConfig(configPath)
	}
}

// TestDockerMockIntegration ensures our mock works correctly
func TestDockerMockIntegration(t *testing.T) {
	mock := MockDockerForTest(t)
	defer mock.Cleanup()
	
	// Ensure mock is in PATH
	if !mock.VerifyDockerCommand() {
		t.Fatal("Docker mock not properly configured")
	}
	
	// Test various Docker operations
	operations := []struct {
		name string
		args []string
		want string
	}{
		{"compose up", []string{"compose", "-f", "test.yml", "up"}, "Creating"},
		{"compose down", []string{"compose", "-f", "test.yml", "down"}, "Stopping"},
		{"ps", []string{"ps"}, "CONTAINER"},
		{"logs", []string{"logs", "test"}, "Container started"},
	}
	
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			cmd := exec.Command("docker", op.args...)
			output, _ := cmd.CombinedOutput()
			
			if !strings.Contains(string(output), op.want) {
				t.Errorf("Docker %s: expected %q in output, got: %s", 
					op.name, op.want, output)
			}
		})
	}
}