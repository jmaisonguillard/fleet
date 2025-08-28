package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DatabaseServicesTestSuite struct {
	suite.Suite
}

func (suite *DatabaseServicesTestSuite) TestParseDatabaseType() {
	testCases := []struct {
		name        string
		input       string
		expectType  string
		expectVersion string
	}{
		{"MySQL with version", "mysql:8.0", "mysql", "8.0"},
		{"PostgreSQL with version", "postgres:15", "postgres", "15"},
		{"MongoDB with version", "mongodb:6.0", "mongodb", "6.0"},
		{"MariaDB with version", "mariadb:10.11", "mariadb", "10.11"},
		{"MySQL without version", "mysql", "mysql", "8.0"}, // Should use default
		{"PostgreSQL without version", "postgres", "postgres", "15"}, // Should use default
		{"Case insensitive", "MySQL:8.0", "mysql", "8.0"},
		{"Empty string", "", "", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			dbType, version := parseDatabaseType(tc.input)
			suite.Equal(tc.expectType, dbType)
			suite.Equal(tc.expectVersion, version)
		})
	}
}

func (suite *DatabaseServicesTestSuite) TestGetDatabaseImage() {
	testCases := []struct {
		name     string
		dbType   string
		version  string
		expected string
	}{
		{"MySQL 8.0", "mysql", "8.0", "mysql:8.0"},
		{"MySQL 8.3", "mysql", "8.3", "mysql:8.3"},
		{"PostgreSQL 15", "postgres", "15", "postgres:15-alpine"},
		{"MongoDB 6.0", "mongodb", "6.0", "mongo:6.0"},
		{"MariaDB 10.11", "mariadb", "10.11", "mariadb:10.11"},
		{"MySQL unknown version", "mysql", "999", "mysql:8.0"}, // Falls back to default
		{"Unknown database", "unknown", "1.0", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getDatabaseImage(tc.dbType, tc.version)
			suite.Equal(tc.expected, image)
		})
	}
}

func (suite *DatabaseServicesTestSuite) TestGetSharedDatabaseServiceName() {
	testCases := []struct {
		name     string
		dbType   string
		version  string
		expected string
	}{
		{"MySQL 8.0", "mysql", "8.0", "mysql-80"},
		{"PostgreSQL 15", "postgres", "15", "postgres-15"},
		{"MongoDB 6.0", "mongodb", "6.0", "mongodb-60"},
		{"MariaDB 10.11", "mariadb", "10.11", "mariadb-1011"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			serviceName := getSharedDatabaseServiceName(tc.dbType, tc.version)
			suite.Equal(tc.expected, serviceName)
		})
	}
}

func (suite *DatabaseServicesTestSuite) TestSharedContainerSameVersion() {
	// Test that multiple services using the same database version share the container
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:     "app1",
				Image:    "node:18",
				Database: "mysql:8.0",
			},
			{
				Name:     "app2",
				Image:    "node:18",
				Database: "mysql:8.0", // Same version
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	// Add first service
	compose.Services["app1"] = DockerService{Image: "node:18"}
	addDatabaseService(compose, &config.Services[0], config)

	// Add second service
	compose.Services["app2"] = DockerService{Image: "node:18"}
	addDatabaseService(compose, &config.Services[1], config)

	// Should only have one MySQL container
	_, exists := compose.Services["mysql-80"]
	suite.True(exists, "Shared MySQL container should exist")

	// Count database services
	dbCount := 0
	for name := range compose.Services {
		if name == "mysql-80" {
			dbCount++
		}
	}
	suite.Equal(1, dbCount, "Should only have one MySQL container for same version")

	// Both app services should depend on the same database
	app1Service := compose.Services["app1"]
	app2Service := compose.Services["app2"]
	suite.Contains(app1Service.DependsOn, "mysql-80")
	suite.Contains(app2Service.DependsOn, "mysql-80")
}

func (suite *DatabaseServicesTestSuite) TestSeparateContainersDifferentVersions() {
	// Test that different versions create separate containers
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:     "app1",
				Image:    "node:18",
				Database: "mysql:8.0",
			},
			{
				Name:     "app2",
				Image:    "node:18",
				Database: "mysql:8.3", // Different version
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	// Add services
	compose.Services["app1"] = DockerService{Image: "node:18"}
	addDatabaseService(compose, &config.Services[0], config)
	compose.Services["app2"] = DockerService{Image: "node:18"}
	addDatabaseService(compose, &config.Services[1], config)

	// Should have two MySQL containers
	_, exists1 := compose.Services["mysql-80"]
	_, exists2 := compose.Services["mysql-83"]
	suite.True(exists1, "MySQL 8.0 container should exist")
	suite.True(exists2, "MySQL 8.3 container should exist")
}

func (suite *DatabaseServicesTestSuite) TestConfigureMySQLService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:                 "myapp",
		DatabaseName:         "appdb",
		DatabaseUser:         "appuser",
		DatabasePassword:     "secret123",
		DatabaseRootPassword: "rootsecret",
	}

	configureMySQLService(service, svc, "mysql-80")

	// Check environment variables
	suite.Equal("rootsecret", service.Environment["MYSQL_ROOT_PASSWORD"])
	suite.Equal("appdb", service.Environment["MYSQL_DATABASE"])
	suite.Equal("appuser", service.Environment["MYSQL_USER"])
	suite.Equal("secret123", service.Environment["MYSQL_PASSWORD"])

	// Check volume
	suite.Contains(service.Volumes, "mysql-80-data:/var/lib/mysql")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "mysqladmin")
}

func (suite *DatabaseServicesTestSuite) TestConfigurePostgresService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:             "myapp",
		DatabaseName:     "appdb",
		DatabaseUser:     "appuser",
		DatabasePassword: "secret123",
	}

	configurePostgresService(service, svc, "postgres-15")

	// Check environment variables
	suite.Equal("appdb", service.Environment["POSTGRES_DB"])
	suite.Equal("appuser", service.Environment["POSTGRES_USER"])
	suite.Equal("secret123", service.Environment["POSTGRES_PASSWORD"])

	// Check volume
	suite.Contains(service.Volumes, "postgres-15-data:/var/lib/postgresql/data")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Equal("CMD-SHELL", service.HealthCheck.Test[0])
	suite.Contains(service.HealthCheck.Test[1], "pg_isready")
}

func (suite *DatabaseServicesTestSuite) TestConfigureMongoDBService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:             "myapp",
		DatabaseName:     "appdb",
		DatabaseUser:     "admin",
		DatabasePassword: "secret123",
	}

	configureMongoDBService(service, svc, "mongodb-60")

	// Check environment variables
	suite.Equal("appdb", service.Environment["MONGO_INITDB_DATABASE"])
	suite.Equal("admin", service.Environment["MONGO_INITDB_ROOT_USERNAME"])
	suite.Equal("secret123", service.Environment["MONGO_INITDB_ROOT_PASSWORD"])

	// Check volume
	suite.Contains(service.Volumes, "mongodb-60-data:/data/db")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "mongosh")
}

func (suite *DatabaseServicesTestSuite) TestConfigureMariaDBService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:                 "myapp",
		DatabaseName:         "appdb",
		DatabaseUser:         "appuser",
		DatabasePassword:     "secret123",
		DatabaseRootPassword: "rootsecret",
	}

	configureMariaDBService(service, svc, "mariadb-1011")

	// Check environment variables
	suite.Equal("rootsecret", service.Environment["MARIADB_ROOT_PASSWORD"])
	suite.Equal("appdb", service.Environment["MARIADB_DATABASE"])
	suite.Equal("appuser", service.Environment["MARIADB_USER"])
	suite.Equal("secret123", service.Environment["MARIADB_PASSWORD"])

	// Check volume
	suite.Contains(service.Volumes, "mariadb-1011-data:/var/lib/mysql")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "healthcheck.sh")
}

func (suite *DatabaseServicesTestSuite) TestAddDatabaseEnvVars() {
	testCases := []struct {
		name      string
		dbType    string
		checkVars map[string]string
	}{
		{
			"MySQL environment variables",
			"mysql",
			map[string]string{
				"DB_CONNECTION": "mysql",
				"DB_HOST":       "mysql-80",
				"DB_PORT":       "3306",
				"DB_DATABASE":   "testapp",
				"DB_USERNAME":   "testapp",
				"DB_PASSWORD":   "password",
			},
		},
		{
			"PostgreSQL environment variables",
			"postgres",
			map[string]string{
				"DB_CONNECTION": "pgsql",
				"DB_HOST":       "postgres-15",
				"DB_PORT":       "5432",
				"DB_DATABASE":   "testapp",
				"DB_USERNAME":   "testapp",
				"DB_PASSWORD":   "password",
			},
		},
		{
			"MongoDB environment variables",
			"mongodb",
			map[string]string{
				"MONGO_HOST":     "mongodb-60",
				"MONGO_PORT":     "27017",
				"MONGO_DB":       "testapp",
				"MONGO_USER":     "admin",
				"MONGO_PASSWORD": "password",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			service := &DockerService{
				Environment: make(map[string]string),
			}
			svc := &Service{
				Name: "testapp",
			}

			// Use default versions for the test
			versions := map[string]string{
				"mysql": "8.0",
				"postgres": "15",
				"mongodb": "6.0",
			}
			dbServiceName := getSharedDatabaseServiceName(tc.dbType, versions[tc.dbType])
			addDatabaseEnvVars(service, tc.dbType, dbServiceName, svc)

			for key, expectedValue := range tc.checkVars {
				suite.Equal(expectedValue, service.Environment[key], "Environment variable %s should be set correctly", key)
			}

			// Check DATABASE_URL or MONGODB_URI is set
			if tc.dbType == "mongodb" {
				suite.NotEmpty(service.Environment["MONGODB_URI"], "MONGODB_URI should be set")
			} else {
				suite.NotEmpty(service.Environment["DATABASE_URL"], "DATABASE_URL should be set")
			}
		})
	}
}

func (suite *DatabaseServicesTestSuite) TestIntegrationComposeGenerationWithDatabases() {
	config := &Config{
		Project: "testproject",
		Services: []Service{
			{
				Name:     "web",
				Image:    "nginx:alpine",
				Port:     80,
				Database: "mysql:8.0",
			},
			{
				Name:     "api",
				Image:    "node:18",
				Port:     3000,
				Database: "postgres:15",
			},
			{
				Name:     "worker",
				Image:    "python:3.9",
				Database: "mongodb:6.0",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that all database services were created
	_, mysqlExists := compose.Services["mysql-80"]
	_, postgresExists := compose.Services["postgres-15"]
	_, mongoExists := compose.Services["mongodb-60"]

	suite.True(mysqlExists, "MySQL service should be created")
	suite.True(postgresExists, "PostgreSQL service should be created")
	suite.True(mongoExists, "MongoDB service should be created")

	// Check dependencies
	webService := compose.Services["web"]
	apiService := compose.Services["api"]
	workerService := compose.Services["worker"]

	suite.Contains(webService.DependsOn, "mysql-80", "Web service should depend on MySQL")
	suite.Contains(apiService.DependsOn, "postgres-15", "API service should depend on PostgreSQL")
	suite.Contains(workerService.DependsOn, "mongodb-60", "Worker service should depend on MongoDB")

	// Check that database volumes are created
	_, mysqlVolExists := compose.Volumes["mysql-80-data"]
	_, postgresVolExists := compose.Volumes["postgres-15-data"]
	_, mongoVolExists := compose.Volumes["mongodb-60-data"]

	suite.True(mysqlVolExists, "MySQL volume should be created")
	suite.True(postgresVolExists, "PostgreSQL volume should be created")
	suite.True(mongoVolExists, "MongoDB volume should be created")
}

func (suite *DatabaseServicesTestSuite) TestMixedDatabaseTypes() {
	// Test a complex scenario with multiple database types
	config := &Config{
		Project: "complex",
		Services: []Service{
			{
				Name:     "app1",
				Image:    "node:18",
				Database: "mysql:8.0",
			},
			{
				Name:     "app2",
				Image:    "python:3.9",
				Database: "postgres:15",
			},
			{
				Name:     "app3",
				Image:    "ruby:3.0",
				Database: "mysql:8.0", // Shares with app1
			},
			{
				Name:     "app4",
				Image:    "php:8.1",
				Database: "mariadb:10.11",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check database service count
	dbServices := []string{"mysql-80", "postgres-15", "mariadb-1011"}
	for _, dbName := range dbServices {
		_, exists := compose.Services[dbName]
		suite.True(exists, "Database service %s should exist", dbName)
	}

	// Verify app1 and app3 share the same MySQL container
	app1Service := compose.Services["app1"]
	app3Service := compose.Services["app3"]
	suite.Contains(app1Service.DependsOn, "mysql-80")
	suite.Contains(app3Service.DependsOn, "mysql-80")
}

func (suite *DatabaseServicesTestSuite) TestDatabaseConfigurationDefaults() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name: "myapp",
		// No database credentials specified - should use defaults
	}

	configureMySQLService(service, svc, "mysql-80")

	// Check default values are used
	suite.Equal("rootpassword", service.Environment["MYSQL_ROOT_PASSWORD"])
	suite.Equal("myapp", service.Environment["MYSQL_DATABASE"])
	suite.Equal("myapp", service.Environment["MYSQL_USER"])
	suite.Equal("password", service.Environment["MYSQL_PASSWORD"])
}

func (suite *DatabaseServicesTestSuite) TestServiceWithoutDatabase() {
	// Test that services without database configuration work correctly
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				// No database specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Should only have the web service and nginx-proxy
	suite.NotNil(compose.Services["web"])
	
	// Should not have any database services
	for name := range compose.Services {
		suite.NotContains(name, "mysql")
		suite.NotContains(name, "postgres")
		suite.NotContains(name, "mongodb")
		suite.NotContains(name, "mariadb")
	}
}

func TestDatabaseServicesSuite(t *testing.T) {
	suite.Run(t, new(DatabaseServicesTestSuite))
}