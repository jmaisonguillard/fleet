package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// FixtureLoaderTestSuite tests the fixture loader functionality
type FixtureLoaderTestSuite struct {
	suite.Suite
}

func TestFixtureLoaderSuite(t *testing.T) {
	suite.Run(t, new(FixtureLoaderTestSuite))
}

func (suite *FixtureLoaderTestSuite) TestLoadFixture() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// Test loading a basic config
	content := loader.LoadFixture("configs/basic.toml")
	suite.Contains(content, "project = \"test-app\"")
	suite.Contains(content, "name = \"web\"")
}

func (suite *FixtureLoaderTestSuite) TestCopyFixtureToTemp() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// Copy fixture to temp
	path := loader.CopyFixtureToTemp("configs/basic.toml", "test.toml")
	suite.FileExists(path)
	
	// Verify content
	content := loader.LoadFixture("configs/basic.toml")
	suite.NotEmpty(content)
}

func (suite *FixtureLoaderTestSuite) TestListFixtures() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// List all TOML fixtures
	fixtures := loader.ListFixtures("*.toml")
	suite.GreaterOrEqual(len(fixtures), 3) // We created at least 3 TOML fixtures
}

func (suite *FixtureLoaderTestSuite) TestFixtureTestCases() {
	testCases := []FixtureTestCase{
		{
			Name:        "BasicConfig",
			FixturePath: "configs/basic.toml",
			Test: func(loader *FixtureLoader, content string) {
				suite.Contains(content, "nginx:alpine")
				suite.Contains(content, "port = 8080")
			},
		},
		{
			Name:        "FullStackConfig",
			FixturePath: "configs/full-stack.toml",
			Test: func(loader *FixtureLoader, content string) {
				suite.Contains(content, "postgres:15")
				suite.Contains(content, "redis:7.2")
			},
		},
	}
	
	RunFixtureTests(suite.T(), testCases)
}

func (suite *FixtureLoaderTestSuite) TestCompareWithFixture() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// Load a fixture
	expected := loader.LoadFixture("configs/basic.toml")
	
	// Compare with same content
	suite.True(loader.CompareWithFixture(expected, "configs/basic.toml"))
	
	// Compare with different content
	suite.False(loader.CompareWithFixture("different content", "configs/basic.toml"))
}

func (suite *FixtureLoaderTestSuite) TestLoadConfig() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// Load config fixture
	config := loader.LoadConfig("basic.toml")
	suite.NotNil(config)
	suite.Equal("test-app", config.Project)
	suite.Len(config.Services, 1)
	suite.Equal("web", config.Services[0].Name)
}

func (suite *FixtureLoaderTestSuite) TestCreateTempFile() {
	loader := NewFixtureLoader(suite.T())
	defer loader.Cleanup()
	
	// Create temp file
	content := "test content"
	path := loader.CreateTempFile("test.txt", content)
	suite.FileExists(path)
	
	// Verify it's in temp directory
	suite.True(strings.HasPrefix(path, loader.TempDir()))
}