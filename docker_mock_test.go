package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// DockerMock provides a mock Docker command for testing
type DockerMock struct {
	t            *testing.T
	tempDir      string
	originalPath string
	responses    map[string]MockResponse
}

// MockResponse defines what a mock Docker command should return
type MockResponse struct {
	Output   string
	ExitCode int
}

// NewDockerMock creates a new Docker mock
func NewDockerMock(t *testing.T) *DockerMock {
	tempDir, err := os.MkdirTemp("", "docker-mock-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for mock: %v", err)
	}

	return &DockerMock{
		t:            t,
		tempDir:      tempDir,
		originalPath: os.Getenv("PATH"),
		responses:    make(map[string]MockResponse),
	}
}

// Setup creates the mock Docker executable and updates PATH
func (m *DockerMock) Setup() {
	// Create mock docker script
	dockerScript := filepath.Join(m.tempDir, "docker")
	
	// Create a shell script that will handle Docker commands
	scriptContent := `#!/bin/bash

# Mock Docker command for testing Fleet

# Capture all arguments
ARGS="$@"

# Default response
echo "Mock Docker: $ARGS"

# Handle specific commands
case "$1" in
    "compose")
        case "$3" in
            "up")
                echo "Creating network \"test-network\" with driver \"bridge\""
                echo "Creating test-container ..."
                echo "Creating test-container ... done"
                exit 0
                ;;
            "down")
                echo "Stopping test-container ..."
                echo "Stopping test-container ... done"
                echo "Removing test-container ..."
                echo "Removing test-container ... done"
                exit 0
                ;;
            "ps")
                echo "NAME                STATUS              PORTS"
                echo "test-container      Up 2 minutes        8080->80/tcp"
                exit 0
                ;;
            "restart")
                echo "Restarting test-container ... done"
                exit 0
                ;;
        esac
        ;;
    "ps")
        if [[ "$ARGS" == *"fleet-dnsmasq"* ]] || [[ "$ARGS" == *"dnsmasq"* ]]; then
            echo "NAMES     STATUS         PORTS"
            echo "dnsmasq   Up 5 minutes   127.0.0.1:53->53/tcp, 127.0.0.1:53->53/udp"
        else
            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES"
            echo "abc123         nginx     nginx     1h ago    Up 1h     80/tcp    test"
        fi
        exit 0
        ;;
    "logs")
        echo "2024-01-01 12:00:00 Container started"
        echo "2024-01-01 12:00:01 Listening on port 80"
        if [[ "$2" == "dnsmasq" ]] || [[ "$2" == "fleet-dnsmasq" ]]; then
            echo "dnsmasq[1]: query[A] test.test from 172.30.0.1"
            echo "dnsmasq[1]: config test.test is 127.0.0.1"
        fi
        exit 0
        ;;
    "version")
        echo "Docker version 24.0.0, build test"
        exit 0
        ;;
    *)
        echo "Mock Docker: Unhandled command: $@" >&2
        exit 0
        ;;
esac
`

	// Write the script
	if err := os.WriteFile(dockerScript, []byte(scriptContent), 0755); err != nil {
		m.t.Fatalf("Failed to create mock docker script: %v", err)
	}

	// Update PATH to include our mock directory first
	newPath := m.tempDir + string(os.PathListSeparator) + m.originalPath
	os.Setenv("PATH", newPath)
}

// AddResponse adds a specific response for a Docker command pattern
func (m *DockerMock) AddResponse(commandPattern string, response MockResponse) {
	m.responses[commandPattern] = response
}

// Cleanup restores the original PATH and removes temp files
func (m *DockerMock) Cleanup() {
	os.Setenv("PATH", m.originalPath)
	os.RemoveAll(m.tempDir)
}

// VerifyDockerCommand checks if Docker mock is being used
func (m *DockerMock) VerifyDockerCommand() bool {
	cmd := exec.Command("docker", "version")
	output, _ := cmd.CombinedOutput()
	return strings.Contains(string(output), "Mock Docker") || 
	       strings.Contains(string(output), "build test")
}

// CreateMockDockerCompose creates a mock docker-compose executable
func (m *DockerMock) CreateMockDockerCompose() {
	composeScript := filepath.Join(m.tempDir, "docker-compose")
	
	scriptContent := `#!/bin/bash
# Mock docker-compose for testing
echo "Mock docker-compose: $@"

case "$2" in
    "up")
        echo "Creating network..."
        echo "Starting services..."
        echo "Services started successfully"
        exit 0
        ;;
    "down")
        echo "Stopping services..."
        echo "Services stopped"
        exit 0
        ;;
    *)
        echo "docker-compose $@"
        exit 0
        ;;
esac
`

	if err := os.WriteFile(composeScript, []byte(scriptContent), 0755); err != nil {
		m.t.Fatalf("Failed to create mock docker-compose: %v", err)
	}
}

// MockDockerForTest sets up a complete Docker mock environment for a test
func MockDockerForTest(t *testing.T) *DockerMock {
	mock := NewDockerMock(t)
	mock.Setup()
	mock.CreateMockDockerCompose()
	return mock
}

// Helper function to check if we're in a test environment
func IsTestEnvironment() bool {
	for _, arg := range os.Args {
		if strings.HasSuffix(arg, ".test") || strings.Contains(arg, "-test.") {
			return true
		}
	}
	return false
}

// TestDockerMock verifies the mock itself works
func TestDockerMock(t *testing.T) {
	mock := MockDockerForTest(t)
	defer mock.Cleanup()

	// Test that mock is being used
	if !mock.VerifyDockerCommand() {
		t.Skip("Docker mock not properly set up")
	}

	// Test various Docker commands
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "docker version",
			args:     []string{"docker", "version"},
			expected: "build test",
		},
		{
			name:     "docker ps",
			args:     []string{"docker", "ps"},
			expected: "CONTAINER ID",
		},
		{
			name:     "docker compose up",
			args:     []string{"docker", "compose", "-f", "test.yml", "up"},
			expected: "Creating",
		},
		{
			name:     "docker logs",
			args:     []string{"docker", "logs", "test-container"},
			expected: "Container started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(tt.args[0], tt.args[1:]...)
			output, err := cmd.CombinedOutput()
			
			if err != nil && !strings.Contains(string(output), tt.expected) {
				t.Errorf("Command %v failed: %v\nOutput: %s", tt.args, err, output)
			}
			
			if !strings.Contains(string(output), tt.expected) {
				t.Errorf("Expected output to contain %q, got: %s", tt.expected, output)
			}
		})
	}
}

// MockExec allows us to mock exec.Command calls in tests
type MockExec struct {
	Commands [][]string
}

func (m *MockExec) RecordCommand(name string, args ...string) {
	cmd := append([]string{name}, args...)
	m.Commands = append(m.Commands, cmd)
}

func (m *MockExec) HasCommand(name string) bool {
	for _, cmd := range m.Commands {
		if len(cmd) > 0 && cmd[0] == name {
			return true
		}
	}
	return false
}

func (m *MockExec) GetLastCommand() []string {
	if len(m.Commands) == 0 {
		return nil
	}
	return m.Commands[len(m.Commands)-1]
}

// Helper to create a mock runDocker function for testing
func createMockRunDocker(t *testing.T) func([]string) error {
	return func(args []string) error {
		// Log the command for debugging
		t.Logf("Mock runDocker called with: %v", args)
		
		// Simulate success for most commands
		if len(args) > 0 {
			switch args[0] {
			case "compose":
				if len(args) > 2 && args[2] == "up" {
					fmt.Println("Mock: Starting services...")
					return nil
				}
				if len(args) > 2 && args[2] == "down" {
					fmt.Println("Mock: Stopping services...")
					return nil
				}
			case "ps":
				fmt.Println("Mock: Container running")
				return nil
			case "logs":
				fmt.Println("Mock: Showing logs...")
				return nil
			}
		}
		return nil
	}
}