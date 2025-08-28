package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHelper provides common test utilities
type TestHelper struct {
	t       *testing.T
	tempDir string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir, err := os.MkdirTemp("", "fleet-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	return &TestHelper{
		t:       t,
		tempDir: tempDir,
	}
}

// Cleanup removes temporary files
func (h *TestHelper) Cleanup() {
	os.RemoveAll(h.tempDir)
}

// CreateFile creates a file with content in the temp directory
func (h *TestHelper) CreateFile(name, content string) string {
	path := filepath.Join(h.tempDir, name)
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.t.Fatalf("Failed to create directory: %v", err)
	}
	
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		h.t.Fatalf("Failed to create file: %v", err)
	}
	
	return path
}

// CreateExecutable creates an executable file in the temp directory
func (h *TestHelper) CreateExecutable(name, content string) string {
	path := h.CreateFile(name, content)
	
	if err := os.Chmod(path, 0755); err != nil {
		h.t.Fatalf("Failed to make file executable: %v", err)
	}
	
	return path
}

// TempDir returns the temporary directory path
func (h *TestHelper) TempDir() string {
	return h.tempDir
}

// SampleFleetConfig returns a sample fleet.toml configuration
func SampleFleetConfig() string {
	return `
project = "test-app"

[[services]]
name = "web"
image = "nginx:alpine"
port = 8080
folder = "./website"

[[services]]
name = "api"
image = "node:18"
port = 3000
needs = ["database"]
[services.env]
NODE_ENV = "development"
DATABASE_URL = "postgresql://postgres:changeme@database:5432/test-app"

[[services]]
name = "database"
image = "postgres:15"
port = 5432
password = "changeme"
volumes = ["db-data:/var/lib/postgresql/data"]
`
}

// SampleDockerCompose returns a sample docker-compose.yml
func SampleDockerCompose() string {
	return `version: "3.8"

services:
  web:
    image: nginx:alpine
    container_name: test-web
    ports:
      - "8080:80"
    volumes:
      - ./website:/usr/share/nginx/html
    networks:
      - test-network
    restart: unless-stopped

  api:
    image: node:18
    container_name: test-api
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgresql://postgres:changeme@database:5432/test-app
    depends_on:
      - database
    networks:
      - test-network
    restart: unless-stopped

  database:
    image: postgres:15
    container_name: test-database
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_PASSWORD=changeme
    volumes:
      - db-data:/var/lib/postgresql/data
    networks:
      - test-network
    restart: unless-stopped

networks:
  test-network:
    driver: bridge

volumes:
  db-data:
`
}

// MockDockerCommand creates a mock docker command for testing
func MockDockerCommand(output string, exitCode int) string {
	script := `#!/bin/bash
echo "` + output + `"
exit ` + string(rune(exitCode+'0')) + `
`
	return script
}