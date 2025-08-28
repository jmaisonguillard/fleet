package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SharedServiceNamerSuite struct {
	suite.Suite
	namer *SharedServiceNamer
}

func TestSharedServiceNamerSuite(t *testing.T) {
	suite.Run(t, new(SharedServiceNamerSuite))
}

func (suite *SharedServiceNamerSuite) SetupTest() {
	suite.namer = NewSharedServiceNamer()
}

func (suite *SharedServiceNamerSuite) TestGetServiceName() {
	tests := []struct {
		name        string
		serviceType string
		version     string
		expected    string
	}{
		{
			name:        "MySQL with version",
			serviceType: "mysql",
			version:     "8.0",
			expected:    "mysql-80",
		},
		{
			name:        "PostgreSQL with version",
			serviceType: "postgres",
			version:     "15",
			expected:    "postgres-15",
		},
		{
			name:        "Redis with dots in version",
			serviceType: "redis",
			version:     "7.2",
			expected:    "redis-72",
		},
		{
			name:        "Service with latest version",
			serviceType: "mongodb",
			version:     "latest",
			expected:    "mongodb-latest",
		},
		{
			name:        "Service with no version",
			serviceType: "mariadb",
			version:     "",
			expected:    "mariadb-latest",
		},
		{
			name:        "Singleton service (mailpit)",
			serviceType: "mailpit",
			version:     "1.20",
			expected:    "mailpit",
		},
		{
			name:        "Singleton service (reverb)",
			serviceType: "reverb",
			version:     "any",
			expected:    "reverb",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := suite.namer.GetServiceName(tt.serviceType, tt.version)
			suite.Equal(tt.expected, result)
		})
	}
}

func (suite *SharedServiceNamerSuite) TestGetUniqueServiceName() {
	// First registration should return base name
	name1 := suite.namer.GetUniqueServiceName("mysql")
	suite.Equal("mysql", name1)
	
	// Second registration should append -2
	name2 := suite.namer.GetUniqueServiceName("mysql")
	suite.Equal("mysql-2", name2)
	
	// Third registration should append -3
	name3 := suite.namer.GetUniqueServiceName("mysql")
	suite.Equal("mysql-3", name3)
}

func (suite *SharedServiceNamerSuite) TestHasCollision() {
	// Register a name
	suite.namer.RegisterName("postgres-15")
	
	// Check collision
	suite.True(suite.namer.HasCollision("postgres-15"))
	suite.False(suite.namer.HasCollision("postgres-16"))
}

func (suite *SharedServiceNamerSuite) TestCleanVersion() {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "1.2.3"},
		{"V2.0", "2.0"},
		{"latest", ""},
		{"default", ""},
		{"", ""},
		{"RELEASE.2024-01-01", "2024"},
		{"3.0", "3.0"},
	}

	for _, tt := range tests {
		result := suite.namer.cleanVersion(tt.input)
		suite.Equal(tt.expected, result, "Input: %s", tt.input)
	}
}

func (suite *SharedServiceNamerSuite) TestStandardizeServiceType() {
	tests := []struct {
		input    string
		expected string
	}{
		{"PostgreSQL", "postgres"},
		{"postgresql", "postgres"},
		{"MySQL", "mysql"},
		{"MariaDB", "mariadb"},
		{"mongo", "mongodb"},
		{"MongoDB", "mongodb"},
		{"Redis", "redis"},
		{"memcache", "memcached"},
		{"mail", "mailpit"},
		{"email", "mailpit"},
		{"MinIO", "minio"},
	}

	for _, tt := range tests {
		result := suite.namer.StandardizeServiceType(tt.input)
		suite.Equal(tt.expected, result, "Input: %s", tt.input)
	}
}

func (suite *SharedServiceNamerSuite) TestGetRegisteredNames() {
	// Register some names
	suite.namer.RegisterName("mysql-80")
	suite.namer.RegisterName("postgres-15")
	suite.namer.RegisterName("redis-72")
	
	names := suite.namer.GetRegisteredNames()
	suite.Len(names, 3)
	suite.Contains(names, "mysql-80")
	suite.Contains(names, "postgres-15")
	suite.Contains(names, "redis-72")
}

func (suite *SharedServiceNamerSuite) TestGlobalNamer() {
	// Test global convenience functions
	name := GetSharedServiceName("mysql", "8.0")
	suite.NotEmpty(name)
	
	RegisterServiceName("test-service")
	suite.True(GlobalServiceNamer.HasCollision("test-service"))
	
	uniqueName := GetUniqueServiceName("test-service")
	suite.Equal("test-service-2", uniqueName)
}