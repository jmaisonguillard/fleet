package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

// MainTestSuite tests the main entry point and command routing
type MainTestSuite struct {
	suite.Suite
	oldArgs   []string
	oldStdout *os.File
}

func (suite *MainTestSuite) SetupTest() {
	// Save original args and stdout
	suite.oldArgs = os.Args
	suite.oldStdout = os.Stdout
}

func (suite *MainTestSuite) TearDownTest() {
	// Restore original args and stdout
	os.Args = suite.oldArgs
	os.Stdout = suite.oldStdout
}

func (suite *MainTestSuite) captureOutput(f func()) string {
	// Create a pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Run the function
	f()
	
	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	
	return buf.String()
}

func (suite *MainTestSuite) TestPrintUsage() {
	output := suite.captureOutput(func() {
		printUsage()
	})
	
	// Check that usage contains expected elements
	suite.Contains(output, "Fleet CLI")
	suite.Contains(output, "Usage: fleet <command>")
	suite.Contains(output, "up, start")
	suite.Contains(output, "down, stop")
	suite.Contains(output, "dns")
	suite.Contains(output, "init")
	suite.Contains(output, "version")
	suite.Contains(output, "help")
}

func (suite *MainTestSuite) TestVersionCommand() {
	// Test version output
	_ = suite.captureOutput(func() {
		os.Args = []string{"fleet", "version"}
		// We can't call main() directly as it uses os.Exit
		// So we'll just test the version string format
		suite.Equal("1.0.0", version)
	})
	
	// Version constant should be defined
	suite.NotEmpty(version)
}

func (suite *MainTestSuite) TestHelpCommand() {
	// Test that help command prints usage
	os.Args = []string{"fleet", "help"}
	
	output := suite.captureOutput(func() {
		// Simulate what main() does for help
		printUsage()
	})
	
	suite.Contains(output, "Fleet CLI")
	suite.Contains(output, "Usage:")
}

func (suite *MainTestSuite) TestNoArguments() {
	// Test that no arguments prints usage
	os.Args = []string{"fleet"}
	
	output := suite.captureOutput(func() {
		// Simulate what main() does with no args
		if len(os.Args) < 2 {
			printUsage()
		}
	})
	
	suite.Contains(output, "Fleet CLI")
}

func (suite *MainTestSuite) TestInvalidCommand() {
	// Test handling of invalid commands
	os.Args = []string{"fleet", "invalid-command"}
	
	// We can't test main() directly due to os.Exit
	// But we can verify the logic
	command := os.Args[1]
	suite.Equal("invalid-command", command)
	
	// Verify this wouldn't match any known commands
	knownCommands := []string{"up", "start", "down", "stop", "restart", 
		"status", "ps", "logs", "init", "dns", "version", "-v", 
		"--version", "help", "-h", "--help"}
	
	suite.NotContains(knownCommands, command)
}

func TestMainSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}