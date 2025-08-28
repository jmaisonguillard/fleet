package mocks

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// DockerMockTestSuite tests the Docker mock functionality
type DockerMockTestSuite struct {
	suite.Suite
}

func TestDockerMockSuite(t *testing.T) {
	suite.Run(t, new(DockerMockTestSuite))
}

func (suite *DockerMockTestSuite) TestMockCreation() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	suite.NotNil(mock)
	suite.Empty(mock.GetHistory())
}

func (suite *DockerMockTestSuite) TestEnableDisable() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	// Enable mock
	err := mock.Enable()
	suite.NoError(err)
	suite.True(mock.enabled)
	
	// Disable mock
	mock.Disable()
	suite.False(mock.enabled)
}

func (suite *DockerMockTestSuite) TestDefaultResponses() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	mock.SetDefaultResponses()
	mock.Enable()
	
	// Test docker version
	cmd := exec.Command("docker", "version")
	output, err := cmd.CombinedOutput()
	suite.NoError(err)
	suite.Contains(string(output), "Docker version")
	
	// Test docker compose up
	cmd = exec.Command("docker", "compose", "up")
	output, err = cmd.CombinedOutput()
	suite.NoError(err)
	suite.Contains(string(output), "Creating network")
}

func (suite *DockerMockTestSuite) TestCallHistory() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	mock.SetDefaultResponses()
	mock.Enable()
	
	// Make some calls
	exec.Command("docker", "version").CombinedOutput()
	exec.Command("docker", "ps").CombinedOutput()
	
	// Check history via state file
	commands := mock.VerifyStateFile()
	suite.GreaterOrEqual(len(commands), 2)
}

func (suite *DockerMockTestSuite) TestWasCalled() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	mock.SetDefaultResponses()
	mock.Enable()
	
	// Run command
	output, err := mock.RunCommand("docker", "compose", "up", "-d")
	suite.NoError(err)
	suite.NotEmpty(output)
	
	// Verify it was called
	suite.True(mock.WasCalled("compose up"))
	suite.False(mock.WasCalled("compose down"))
}

func (suite *DockerMockTestSuite) TestCallCount() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	mock.SetDefaultResponses()
	
	// Run command multiple times
	mock.RunCommand("docker", "ps")
	mock.RunCommand("docker", "ps")
	mock.RunCommand("docker", "version")
	
	// Check call counts
	suite.Equal(2, mock.CallCount("ps"))
	suite.Equal(1, mock.CallCount("version"))
	suite.Equal(0, mock.CallCount("logs"))
}

func (suite *DockerMockTestSuite) TestAssertions() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	mock.SetDefaultResponses()
	
	// Run command
	mock.RunCommand("docker", "compose", "up")
	
	// Test assertions (these would fail the test if wrong)
	mock.AssertCalled("compose up")
	mock.AssertNotCalled("compose down")
	mock.AssertCallCount("compose up", 1)
}

func (suite *DockerMockTestSuite) TestAddCustomResponse() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	// Add custom response
	mock.AddResponse("custom", "Custom output", 0)
	
	// Add dynamic response
	mock.AddDynamicResponse("dynamic", func(args []string) (string, int) {
		return "Args: " + strings.Join(args, " "), 0
	})
	
	// Verify responses are registered
	suite.Len(mock.responses, 2)
}

func (suite *DockerMockTestSuite) TestClearHistory() {
	mock := NewDockerMock(suite.T())
	defer mock.Cleanup()
	
	// Add some history
	mock.RunCommand("docker", "ps")
	mock.RunCommand("docker", "version")
	suite.Len(mock.GetHistory(), 2)
	
	// Clear history
	mock.ClearHistory()
	suite.Empty(mock.GetHistory())
}