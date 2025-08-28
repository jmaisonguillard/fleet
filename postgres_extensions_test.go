package main

import (
	"strings"
	"testing"
	
	"github.com/stretchr/testify/suite"
)

type PostgresExtensionsSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *PostgresExtensionsSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *PostgresExtensionsSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *PostgresExtensionsSuite) TestPostGISExtension() {
	// Test PostGIS extension configuration
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:               "api",
				Image:              "nginx",
				Database:           "postgres:15",
				DatabaseExtensions: []string{"postgis"},
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Check that PostgreSQL service was created with PostGIS
	pgService := compose.Services["postgres-15"]
	suite.Assert().NotNil(pgService, "PostgreSQL service should exist")
	
	// Should use PostGIS image instead of regular PostgreSQL
	suite.Assert().Contains(pgService.Image, "postgis/postgis")
	suite.Assert().Contains(pgService.Image, "15-3.4")
	
	// Check that init script is mounted
	hasInitScript := false
	for _, volume := range pgService.Volumes {
		if strings.Contains(volume, "init.sql") {
			hasInitScript = true
			break
		}
	}
	suite.Assert().True(hasInitScript, "Should mount init script for extensions")
}

func (suite *PostgresExtensionsSuite) TestPgVectorExtension() {
	// Test pgvector extension configuration
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:               "search",
				Image:              "node:18",
				Database:           "postgres:16",
				DatabaseExtensions: []string{"pgvector"},
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	pgService := compose.Services["postgres-16"]
	
	// Should use pgvector image for vector operations
	suite.Assert().Contains(pgService.Image, "pgvector/pgvector")
	suite.Assert().Contains(pgService.Image, "pg16")
}

func (suite *PostgresExtensionsSuite) TestMultipleExtensions() {
	// Test multiple PostgreSQL extensions
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:     "geo-api",
				Image:    "python:3.11",
				Database: "postgres:14",
				DatabaseExtensions: []string{
					"postgis",
					"uuid-ossp",
					"hstore",
					"pg_trgm",
				},
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	pgService := compose.Services["postgres-14"]
	
	// Should use PostGIS image (most feature-rich)
	suite.Assert().Contains(pgService.Image, "postgis/postgis")
	
	// Should have init script for all extensions
	hasInitScript := false
	for _, volume := range pgService.Volumes {
		if strings.Contains(volume, "postgres-14-init.sql") {
			hasInitScript = true
			break
		}
	}
	suite.Assert().True(hasInitScript, "Should have init script for extensions")
}

func (suite *PostgresExtensionsSuite) TestPostgresWithoutExtensions() {
	// Test that PostgreSQL without extensions uses regular image
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:     "api",
				Image:    "nginx",
				Database: "postgres:15",
				// No extensions specified
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	pgService := compose.Services["postgres-15"]
	
	// Should use regular PostgreSQL image (alpine variant)
	suite.Assert().Equal("postgres:15-alpine", pgService.Image)
	suite.Assert().NotContains(pgService.Image, "postgis")
	suite.Assert().NotContains(pgService.Image, "pgvector")
}

func (suite *PostgresExtensionsSuite) TestPgRoutingExtension() {
	// Test pgrouting extension (requires PostGIS image)
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:               "routing",
				Image:              "node:18",
				Database:           "postgres:15",
				DatabaseExtensions: []string{"pgrouting"},
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	pgService := compose.Services["postgres-15"]
	
	// pgrouting requires PostGIS image
	suite.Assert().Contains(pgService.Image, "postgis/postgis")
}

func (suite *PostgresExtensionsSuite) TestSharedPostgresWithExtensions() {
	// Test that shared PostgreSQL service gets extensions from first service
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:               "api1",
				Image:              "nginx",
				Database:           "postgres:15",
				DatabaseExtensions: []string{"postgis", "uuid-ossp"},
			},
			{
				Name:     "api2",
				Image:    "nginx",
				Database: "postgres:15",
				// Second service uses same PostgreSQL version
			},
		},
	}
	
	compose := generateDockerCompose(config)
	
	// Should only have one PostgreSQL service
	pgCount := 0
	for name := range compose.Services {
		if strings.HasPrefix(name, "postgres-") {
			pgCount++
		}
	}
	suite.Assert().Equal(1, pgCount, "Should only have one PostgreSQL service")
	
	pgService := compose.Services["postgres-15"]
	
	// Should have PostGIS image from first service
	suite.Assert().Contains(pgService.Image, "postgis/postgis")
	
	// Both services should depend on the same PostgreSQL
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Assert().Contains(api1Service.DependsOn, "postgres-15")
	suite.Assert().Contains(api2Service.DependsOn, "postgres-15")
}

func (suite *PostgresExtensionsSuite) TestCommonExtensions() {
	// Test common PostgreSQL extensions
	extensions := []struct {
		name     string
		expected string
	}{
		{"uuid-ossp", `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`},
		{"hstore", "CREATE EXTENSION IF NOT EXISTS hstore"},
		{"pg_trgm", "CREATE EXTENSION IF NOT EXISTS pg_trgm"},
		{"btree_gin", "CREATE EXTENSION IF NOT EXISTS btree_gin"},
		{"btree_gist", "CREATE EXTENSION IF NOT EXISTS btree_gist"},
	}
	
	for _, ext := range extensions {
		script := generatePostgresInitScript([]string{ext.name})
		suite.Assert().Contains(script, ext.expected, 
			"Script should contain proper CREATE EXTENSION for "+ext.name)
	}
}

func (suite *PostgresExtensionsSuite) TestPostGISFullExtensions() {
	// Test that PostGIS enables multiple related extensions
	script := generatePostgresInitScript([]string{"postgis"})
	
	expectedExtensions := []string{
		"CREATE EXTENSION IF NOT EXISTS postgis",
		"CREATE EXTENSION IF NOT EXISTS postgis_topology",
		"CREATE EXTENSION IF NOT EXISTS fuzzystrmatch",
		"CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder",
	}
	
	for _, expected := range expectedExtensions {
		suite.Assert().Contains(script, expected, 
			"PostGIS should enable extension: "+expected)
	}
}

func (suite *PostgresExtensionsSuite) TestCustomExtension() {
	// Test that unknown extensions are handled gracefully
	script := generatePostgresInitScript([]string{"custom_extension"})
	
	suite.Assert().Contains(script, "CREATE EXTENSION IF NOT EXISTS custom_extension",
		"Should handle custom extensions")
}

func TestPostgresExtensionsSuite(t *testing.T) {
	suite.Run(t, new(PostgresExtensionsSuite))
}