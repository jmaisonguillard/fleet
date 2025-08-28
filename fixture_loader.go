package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/configs/* testdata/compose/*
var testDataFS embed.FS

// FixtureLoader provides utilities for loading test fixtures
type FixtureLoader struct {
	t       *testing.T
	tempDir string
}

// NewFixtureLoader creates a new fixture loader
func NewFixtureLoader(t *testing.T) *FixtureLoader {
	tempDir, err := os.MkdirTemp("", "fleet-fixture-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	return &FixtureLoader{
		t:       t,
		tempDir: tempDir,
	}
}

// Cleanup removes temporary files
func (fl *FixtureLoader) Cleanup() {
	os.RemoveAll(fl.tempDir)
}

// LoadFixture loads a fixture file from testdata
func (fl *FixtureLoader) LoadFixture(path string) string {
	fullPath := filepath.Join("testdata", path)
	data, err := testDataFS.ReadFile(fullPath)
	if err != nil {
		fl.t.Fatalf("Failed to load fixture %s: %v", path, err)
	}
	return string(data)
}

// LoadFixtureBytes loads a fixture file as bytes
func (fl *FixtureLoader) LoadFixtureBytes(path string) []byte {
	fullPath := filepath.Join("testdata", path)
	data, err := testDataFS.ReadFile(fullPath)
	if err != nil {
		fl.t.Fatalf("Failed to load fixture %s: %v", path, err)
	}
	return data
}

// CopyFixtureToTemp copies a fixture to a temp file and returns the path
func (fl *FixtureLoader) CopyFixtureToTemp(fixturePath, tempName string) string {
	content := fl.LoadFixture(fixturePath)
	tempPath := filepath.Join(fl.tempDir, tempName)
	
	dir := filepath.Dir(tempPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fl.t.Fatalf("Failed to create directory: %v", err)
	}
	
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		fl.t.Fatalf("Failed to write temp file: %v", err)
	}
	
	return tempPath
}

// LoadConfig loads a config fixture and returns it as a Config struct
func (fl *FixtureLoader) LoadConfig(name string) *Config {
	_ = fl.CopyFixtureToTemp(filepath.Join("configs", name), "fleet.toml")
	
	// Change to temp directory for loading
	oldDir, _ := os.Getwd()
	os.Chdir(fl.tempDir)
	defer os.Chdir(oldDir)
	
	config, err := loadConfig("fleet.toml")
	if err != nil {
		fl.t.Fatalf("Failed to load config fixture %s: %v", name, err)
	}
	
	return config
}

// TempDir returns the temporary directory path
func (fl *FixtureLoader) TempDir() string {
	return fl.tempDir
}

// CreateTempFile creates a temporary file with content
func (fl *FixtureLoader) CreateTempFile(name, content string) string {
	path := filepath.Join(fl.tempDir, name)
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		fl.t.Fatalf("Failed to create directory: %v", err)
	}
	
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fl.t.Fatalf("Failed to create file: %v", err)
	}
	
	return path
}

// ListFixtures returns all fixtures matching a pattern
func (fl *FixtureLoader) ListFixtures(pattern string) []string {
	var fixtures []string
	fl.walkFixtures("testdata", pattern, &fixtures)
	return fixtures
}

// walkFixtures recursively walks the fixture directory
func (fl *FixtureLoader) walkFixtures(dir, pattern string, fixtures *[]string) {
	entries, err := testDataFS.ReadDir(dir)
	if err != nil {
		return
	}
	
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			fl.walkFixtures(path, pattern, fixtures)
		} else {
			relativePath := strings.TrimPrefix(path, "testdata/")
			matched, _ := filepath.Match(pattern, entry.Name())
			if matched {
				*fixtures = append(*fixtures, relativePath)
			}
		}
	}
}

// FixtureTestCase represents a test case with fixture data
type FixtureTestCase struct {
	Name        string
	FixturePath string
	Setup       func(*FixtureLoader)
	Test        func(*FixtureLoader, string) // passes fixture content
}

// RunFixtureTests runs a set of fixture-based tests
func RunFixtureTests(t *testing.T, testCases []FixtureTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			loader := NewFixtureLoader(t)
			defer loader.Cleanup()
			
			if tc.Setup != nil {
				tc.Setup(loader)
			}
			
			content := loader.LoadFixture(tc.FixturePath)
			tc.Test(loader, content)
		})
	}
}

// CompareWithFixture compares content with a fixture file
func (fl *FixtureLoader) CompareWithFixture(actual, fixturePath string) bool {
	expected := fl.LoadFixture(fixturePath)
	return strings.TrimSpace(actual) == strings.TrimSpace(expected)
}

// AssertFixtureEquals asserts that content equals a fixture
func (fl *FixtureLoader) AssertFixtureEquals(actual, fixturePath string) {
	expected := fl.LoadFixture(fixturePath)
	if strings.TrimSpace(actual) != strings.TrimSpace(expected) {
		fl.t.Errorf("Content does not match fixture %s\nExpected:\n%s\nActual:\n%s",
			fixturePath, expected, actual)
	}
}

// WriteFixture writes content to a fixture file (useful for updating fixtures)
func WriteFixture(path string, content []byte) error {
	fullPath := filepath.Join("testdata", path)
	dir := filepath.Dir(fullPath)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	return os.WriteFile(fullPath, content, 0644)
}

// CopyReader copies from a reader to a writer
func CopyReader(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, src)
	return err
}