package mocks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// DockerMock provides a configurable mock for Docker commands
type DockerMock struct {
	t            *testing.T
	tempDir      string
	responses    map[string]MockResponse
	history      []MockCall
	historyMutex sync.Mutex
	enabled      bool
}

// MockResponse defines a response for a specific command pattern
type MockResponse struct {
	Pattern  string // Command pattern to match
	Output   string // Output to return
	ExitCode int    // Exit code to return
	Callback func(args []string) (string, int) // Optional dynamic response
}

// MockCall records a call to the mock
type MockCall struct {
	Command string
	Args    []string
	WorkDir string
}

// NewDockerMock creates a new Docker mock
func NewDockerMock(t *testing.T) *DockerMock {
	tempDir, err := os.MkdirTemp("", "fleet-mock-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	return &DockerMock{
		t:         t,
		tempDir:   tempDir,
		responses: make(map[string]MockResponse),
		history:   []MockCall{},
		enabled:   false,
	}
}

// Enable activates the mock by adding it to PATH
func (dm *DockerMock) Enable() error {
	if dm.enabled {
		return nil
	}
	
	// Create mock docker executable
	dockerPath := filepath.Join(dm.tempDir, "docker")
	dockerScript := dm.generateMockScript()
	
	if err := os.WriteFile(dockerPath, []byte(dockerScript), 0755); err != nil {
		return fmt.Errorf("failed to create mock docker: %w", err)
	}
	
	// Create mock docker-compose executable
	composePath := filepath.Join(dm.tempDir, "docker-compose")
	if err := os.WriteFile(composePath, []byte(dockerScript), 0755); err != nil {
		return fmt.Errorf("failed to create mock docker-compose: %w", err)
	}
	
	// Add to PATH
	oldPath := os.Getenv("PATH")
	newPath := dm.tempDir + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)
	
	dm.enabled = true
	return nil
}

// Disable deactivates the mock
func (dm *DockerMock) Disable() {
	if !dm.enabled {
		return
	}
	
	// Restore original PATH
	path := os.Getenv("PATH")
	parts := strings.Split(path, string(os.PathListSeparator))
	var newParts []string
	for _, part := range parts {
		if part != dm.tempDir {
			newParts = append(newParts, part)
		}
	}
	os.Setenv("PATH", strings.Join(newParts, string(os.PathListSeparator)))
	
	dm.enabled = false
}

// Cleanup removes temporary files and restores PATH
func (dm *DockerMock) Cleanup() {
	dm.Disable()
	os.RemoveAll(dm.tempDir)
}

// AddResponse adds a mock response for a command pattern
func (dm *DockerMock) AddResponse(pattern, output string, exitCode int) {
	dm.responses[pattern] = MockResponse{
		Pattern:  pattern,
		Output:   output,
		ExitCode: exitCode,
	}
}

// AddDynamicResponse adds a mock response with a callback
func (dm *DockerMock) AddDynamicResponse(pattern string, callback func(args []string) (string, int)) {
	dm.responses[pattern] = MockResponse{
		Pattern:  pattern,
		Callback: callback,
	}
}

// SetDefaultResponses sets up common Docker command responses
func (dm *DockerMock) SetDefaultResponses() {
	// Docker version
	dm.AddResponse("version", "Docker version 24.0.0, build abcdef", 0)
	
	// Docker compose commands
	dm.AddResponse("compose up", "Creating network fleet-network\nStarting services...", 0)
	dm.AddResponse("compose down", "Stopping services...\nRemoving network fleet-network", 0)
	dm.AddResponse("compose ps", "NAME                STATUS\nweb                 running", 0)
	dm.AddResponse("compose restart", "Restarting services...", 0)
	dm.AddResponse("compose logs", "web_1  | Server started\napi_1  | Listening on port 3000", 0)
	
	// Docker ps
	dm.AddResponse("ps", "CONTAINER ID   IMAGE   COMMAND   STATUS", 0)
	
	// Docker network
	dm.AddResponse("network", "fleet-network\nbridge\nhost\nnone", 0)
}

// GetHistory returns the call history
func (dm *DockerMock) GetHistory() []MockCall {
	dm.historyMutex.Lock()
	defer dm.historyMutex.Unlock()
	
	history := make([]MockCall, len(dm.history))
	copy(history, dm.history)
	return history
}

// ClearHistory clears the call history
func (dm *DockerMock) ClearHistory() {
	dm.historyMutex.Lock()
	defer dm.historyMutex.Unlock()
	dm.history = []MockCall{}
}

// WasCalled returns true if a command matching the pattern was called
func (dm *DockerMock) WasCalled(pattern string) bool {
	history := dm.GetHistory()
	for _, call := range history {
		fullCmd := call.Command + " " + strings.Join(call.Args, " ")
		if strings.Contains(fullCmd, pattern) {
			return true
		}
	}
	return false
}

// CallCount returns the number of times a pattern was called
func (dm *DockerMock) CallCount(pattern string) int {
	count := 0
	history := dm.GetHistory()
	for _, call := range history {
		fullCmd := call.Command + " " + strings.Join(call.Args, " ")
		if strings.Contains(fullCmd, pattern) {
			count++
		}
	}
	return count
}

// AssertCalled asserts that a command was called
func (dm *DockerMock) AssertCalled(pattern string) {
	if !dm.WasCalled(pattern) {
		dm.t.Errorf("Expected command '%s' to be called, but it wasn't", pattern)
	}
}

// AssertNotCalled asserts that a command was not called
func (dm *DockerMock) AssertNotCalled(pattern string) {
	if dm.WasCalled(pattern) {
		dm.t.Errorf("Expected command '%s' not to be called, but it was", pattern)
	}
}

// AssertCallCount asserts the number of times a command was called
func (dm *DockerMock) AssertCallCount(pattern string, expected int) {
	actual := dm.CallCount(pattern)
	if actual != expected {
		dm.t.Errorf("Expected command '%s' to be called %d times, but was called %d times", 
			pattern, expected, actual)
	}
}

// generateMockScript generates the mock docker script
func (dm *DockerMock) generateMockScript() string {
	stateFile := filepath.Join(dm.tempDir, "docker-mock-state")
	
	return fmt.Sprintf(`#!/bin/bash
# Mock Docker Script

# Save command to state file for verification
echo "CMD:$0 $@" >> %s
echo "PWD:$(pwd)" >> %s

# Parse command
FULL_CMD="$@"

# Default response
OUTPUT=""
EXIT_CODE=0

# Check for docker compose commands
if [[ "$1" == "compose" ]]; then
    case "$2" in
        "up")
            echo "Creating network fleet-network"
            echo "Starting services..."
            exit 0
            ;;
        "down")
            echo "Stopping services..."
            echo "Removing network fleet-network"
            exit 0
            ;;
        "ps")
            echo "NAME                STATUS"
            echo "web                 running"
            exit 0
            ;;
        "restart")
            echo "Restarting services..."
            exit 0
            ;;
        "logs")
            echo "web_1  | Server started"
            echo "api_1  | Listening on port 3000"
            exit 0
            ;;
    esac
fi

# Check for other docker commands
case "$1" in
    "version")
        echo "Docker version 24.0.0, build abcdef"
        exit 0
        ;;
    "ps")
        echo "CONTAINER ID   IMAGE   COMMAND   STATUS"
        exit 0
        ;;
    "network")
        echo "fleet-network"
        echo "bridge"
        echo "host"
        echo "none"
        exit 0
        ;;
    "logs")
        echo "Container logs..."
        exit 0
        ;;
esac

# Default error
echo "Error: Unknown command: $@" >&2
exit 1
`, stateFile, stateFile)
}

// RunCommand runs a command and captures the mock interaction
func (dm *DockerMock) RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	
	// Record the call
	dm.historyMutex.Lock()
	dm.history = append(dm.history, MockCall{
		Command: command,
		Args:    args,
		WorkDir: cmd.Dir,
	})
	dm.historyMutex.Unlock()
	
	return string(output), err
}

// VerifyStateFile reads and verifies the state file
func (dm *DockerMock) VerifyStateFile() []string {
	stateFile := filepath.Join(dm.tempDir, "docker-mock-state")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return []string{}
	}
	
	lines := strings.Split(string(data), "\n")
	var commands []string
	for _, line := range lines {
		if strings.HasPrefix(line, "CMD:") {
			commands = append(commands, strings.TrimPrefix(line, "CMD:"))
		}
	}
	return commands
}