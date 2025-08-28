package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

// DNSTestSuite tests the DNS functionality
type DNSTestSuite struct {
	suite.Suite
	tempDir string
}

func (suite *DNSTestSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "fleet-dns-test-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *DNSTestSuite) TearDownTest() {
	// Clean up temp directory
	os.RemoveAll(suite.tempDir)
}

func (suite *DNSTestSuite) TestGetScriptPath() {
	// Create mock scripts in temp directory
	scriptsDir := filepath.Join(suite.tempDir, "scripts")
	err := os.MkdirAll(scriptsDir, 0755)
	suite.Require().NoError(err)

	// Create appropriate script based on OS
	var scriptName string
	if runtime.GOOS == "windows" {
		scriptName = "setup-dns.ps1"
	} else {
		scriptName = "setup-dns.sh"
	}
	
	scriptPath := filepath.Join(scriptsDir, scriptName)
	err = os.WriteFile(scriptPath, []byte("#!/bin/bash\n# Test script"), 0755)
	suite.Require().NoError(err)

	// Save current dir and change to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(suite.tempDir)
	
	// Test that getScriptPath finds the script
	path := getScriptPath()
	suite.NotEmpty(path, "Script path should be found")
	suite.Contains(path, scriptName)
}

func (suite *DNSTestSuite) TestGetScriptPathNotFound() {
	// Save current dir and change to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(suite.tempDir)
	
	// Test that getScriptPath returns empty when no script exists
	path := getScriptPath()
	suite.Empty(path, "Script path should be empty when not found")
}

func (suite *DNSTestSuite) TestTestDNSResolution() {
	// This test would normally require a running DNS server
	// For unit testing, we'll just verify the function exists and handles failures gracefully
	
	// Test with a domain that definitely won't resolve
	result := testDNSResolution("nonexistent.invalid")
	suite.False(result, "Non-existent domain should not resolve")
}

func (suite *DNSTestSuite) TestDNSUsagePrinting() {
	// Test that printDNSUsage doesn't panic
	// This is a smoke test to ensure the function works
	suite.NotPanics(func() {
		// Temporarily redirect stdout to prevent test output pollution
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()
		
		tmpfile, err := os.CreateTemp("", "stdout")
		if err == nil {
			os.Stdout = tmpfile
			defer tmpfile.Close()
			defer os.Remove(tmpfile.Name())
		}
		
		printDNSUsage()
	})
}

func (suite *DNSTestSuite) TestCheckPort53() {
	// Test that checkPort53 doesn't panic and handles different OS
	suite.NotPanics(func() {
		// Temporarily redirect stdout to prevent test output pollution
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()
		
		tmpfile, err := os.CreateTemp("", "stdout")
		if err == nil {
			os.Stdout = tmpfile
			defer tmpfile.Close()
			defer os.Remove(tmpfile.Name())
		}
		
		checkPort53()
	})
}

func (suite *DNSTestSuite) TestScriptPathSelection() {
	// Test that the correct script name is selected based on OS
	
	// Create both scripts in temp directory
	scriptsDir := filepath.Join(suite.tempDir, "scripts")
	err := os.MkdirAll(scriptsDir, 0755)
	suite.Require().NoError(err)
	
	// Create both script files
	shScript := filepath.Join(scriptsDir, "setup-dns.sh")
	err = os.WriteFile(shScript, []byte("#!/bin/bash\n"), 0755)
	suite.Require().NoError(err)
	
	psScript := filepath.Join(scriptsDir, "setup-dns.ps1")
	err = os.WriteFile(psScript, []byte("# PowerShell\n"), 0755)
	suite.Require().NoError(err)
	
	// Save current dir and change to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(suite.tempDir)
	
	// Test that getScriptPath selects the right one for current OS
	path := getScriptPath()
	suite.NotEmpty(path)
	
	if runtime.GOOS == "windows" {
		suite.Contains(path, "setup-dns.ps1")
	} else {
		suite.Contains(path, "setup-dns.sh")
	}
}

// Mock DNS resolution for testing
func (suite *DNSTestSuite) TestDNSResolutionLogic() {
	// Test various DNS resolution scenarios
	testCases := []struct {
		name     string
		domain   string
		expected bool
	}{
		{
			name:     "Localhost should resolve",
			domain:   "localhost",
			expected: true, // localhost typically resolves to 127.0.0.1
		},
		{
			name:     "Invalid domain should not resolve",
			domain:   "this-domain-definitely-does-not-exist.invalid",
			expected: false,
		},
		{
			name:     "Empty domain should not resolve",
			domain:   "",
			expected: false,
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := testDNSResolution(tc.domain)
			if tc.expected {
				suite.True(result, "Expected %s to resolve", tc.domain)
			} else {
				suite.False(result, "Expected %s not to resolve", tc.domain)
			}
		})
	}
}

func (suite *DNSTestSuite) TestCrossPlatformScriptDetection() {
	// Ensure the script detection works correctly for different platforms
	
	scriptsDir := filepath.Join(suite.tempDir, "scripts")
	err := os.MkdirAll(scriptsDir, 0755)
	suite.Require().NoError(err)
	
	// Create platform-specific script
	var expectedScript string
	switch runtime.GOOS {
	case "windows":
		expectedScript = "setup-dns.ps1"
	default:
		expectedScript = "setup-dns.sh"
	}
	
	scriptPath := filepath.Join(scriptsDir, expectedScript)
	err = os.WriteFile(scriptPath, []byte("test"), 0755)
	suite.Require().NoError(err)
	
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(suite.tempDir)
	
	// Verify correct script is found
	path := getScriptPath()
	suite.Contains(path, expectedScript)
}

func TestDNSSuite(t *testing.T) {
	suite.Run(t, new(DNSTestSuite))
}