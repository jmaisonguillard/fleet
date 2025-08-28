package main

import (
	"fmt"
	"strings"
)

// Database service configuration
type DatabaseConfig struct {
	Type     string // mysql, postgres, mongodb, mariadb
	Version  string
	Name     string // Database/schema name
	User     string
	Password string
	RootPassword string // For MySQL/MariaDB
	Port     int
}

// Supported database versions
var supportedDatabaseVersions = map[string]map[string]string{
	"mysql": {
		"5.7":     "mysql:5.7",
		"8.0":     "mysql:8.0",
		"8.1":     "mysql:8.1",
		"8.2":     "mysql:8.2",
		"8.3":     "mysql:8.3",
		"latest":  "mysql:latest",
		"default": "mysql:8.0",
	},
	"postgres": {
		"12":      "postgres:12-alpine",
		"13":      "postgres:13-alpine",
		"14":      "postgres:14-alpine",
		"15":      "postgres:15-alpine",
		"16":      "postgres:16-alpine",
		"latest":  "postgres:alpine",
		"default": "postgres:15-alpine",
	},
	"mongodb": {
		"4.4":     "mongo:4.4",
		"5.0":     "mongo:5.0",
		"6.0":     "mongo:6.0",
		"7.0":     "mongo:7.0",
		"latest":  "mongo:latest",
		"default": "mongo:6.0",
	},
	"mariadb": {
		"10.6":    "mariadb:10.6",
		"10.11":   "mariadb:10.11",
		"11.0":    "mariadb:11.0",
		"11.1":    "mariadb:11.1",
		"11.2":    "mariadb:11.2",
		"latest":  "mariadb:latest",
		"default": "mariadb:10.11",
	},
}

// parseDatabaseType parses database type and version from a string like "mysql:8.0"
func parseDatabaseType(dbString string) (dbType string, version string) {
	if dbString == "" {
		return "", ""
	}
	
	parts := strings.Split(dbString, ":")
	dbType = strings.ToLower(parts[0])
	
	if len(parts) > 1 {
		version = parts[1]
	} else {
		// Use default version if not specified
		if versions, ok := supportedDatabaseVersions[dbType]; ok {
			version = versions["default"]
			// Extract just the version number
			if idx := strings.LastIndex(version, ":"); idx >= 0 {
				version = version[idx+1:]
				// Remove -alpine or other suffixes for version number
				if idx := strings.Index(version, "-"); idx >= 0 {
					version = version[:idx]
				}
			}
		}
	}
	
	return dbType, version
}

// getDatabaseImage returns the appropriate Docker image for a database
func getDatabaseImage(dbType, version string) string {
	dbType = strings.ToLower(dbType)
	
	if versions, ok := supportedDatabaseVersions[dbType]; ok {
		if image, ok := versions[version]; ok {
			return image
		}
		// Fallback to default
		return versions["default"]
	}
	
	// Unknown database type
	return ""
}

// getSharedDatabaseServiceName returns a shared service name for a database type and version
func getSharedDatabaseServiceName(dbType, version string) string {
	// Normalize the service name: postgres-15, mysql-80, mongodb-60, etc.
	cleanVersion := strings.ReplaceAll(version, ".", "")
	return fmt.Sprintf("%s-%s", dbType, cleanVersion)
}

// addDatabaseService adds or reuses a database service in the compose file
func addDatabaseService(compose *DockerCompose, svc *Service, config *Config) {
	if svc.Database == "" {
		return
	}
	
	dbType, version := parseDatabaseType(svc.Database)
	if dbType == "" {
		return
	}
	
	// Get the shared service name
	dbServiceName := getSharedDatabaseServiceName(dbType, version)
	
	// Check if this database service already exists
	if _, exists := compose.Services[dbServiceName]; exists {
		// Service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, dbServiceName) {
				appService.DependsOn = append(appService.DependsOn, dbServiceName)
				compose.Services[svc.Name] = appService
			}
		}
		return
	}
	
	// Create the database service
	dbImage := getDatabaseImage(dbType, version)
	if dbImage == "" {
		return
	}
	
	dbService := DockerService{
		Image:    dbImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
	}
	
	// Configure based on database type
	switch dbType {
	case "mysql":
		configureMySQLService(&dbService, svc, dbServiceName)
	case "postgres":
		configurePostgresService(&dbService, svc, dbServiceName)
	case "mongodb":
		configureMongoDBService(&dbService, svc, dbServiceName)
	case "mariadb":
		configureMariaDBService(&dbService, svc, dbServiceName)
	}
	
	// Add the service to compose
	compose.Services[dbServiceName] = dbService
	
	// Update app service to depend on database
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, dbServiceName) {
			appService.DependsOn = append(appService.DependsOn, dbServiceName)
		}
		
		// Add database connection environment variables to the app
		addDatabaseEnvVars(&appService, dbType, dbServiceName, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureMySQLService configures a MySQL service
func configureMySQLService(service *DockerService, svc *Service, dbServiceName string) {
	// Data volume
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/var/lib/mysql", dbServiceName))
	
	// Environment variables
	service.Environment["MYSQL_ROOT_PASSWORD"] = getEnvOrDefault(svc.DatabaseRootPassword, "rootpassword")
	service.Environment["MYSQL_DATABASE"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
	service.Environment["MYSQL_USER"] = getEnvOrDefault(svc.DatabaseUser, svc.Name)
	service.Environment["MYSQL_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "mysqladmin", "ping", "-h", "localhost"},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
}

// configurePostgresService configures a PostgreSQL service
func configurePostgresService(service *DockerService, svc *Service, dbServiceName string) {
	// Data volume
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/var/lib/postgresql/data", dbServiceName))
	
	// Environment variables
	service.Environment["POSTGRES_DB"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
	service.Environment["POSTGRES_USER"] = getEnvOrDefault(svc.DatabaseUser, svc.Name)
	service.Environment["POSTGRES_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", "pg_isready -U " + service.Environment["POSTGRES_USER"]},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
}

// configureMongoDBService configures a MongoDB service
func configureMongoDBService(service *DockerService, svc *Service, dbServiceName string) {
	// Data volume
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/data/db", dbServiceName))
	
	// Environment variables
	service.Environment["MONGO_INITDB_DATABASE"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
	service.Environment["MONGO_INITDB_ROOT_USERNAME"] = getEnvOrDefault(svc.DatabaseUser, "admin")
	service.Environment["MONGO_INITDB_ROOT_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "mongosh", "--eval", "db.adminCommand('ping')"},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
}


// configureMariaDBService configures a MariaDB service
func configureMariaDBService(service *DockerService, svc *Service, dbServiceName string) {
	// Data volume
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/var/lib/mysql", dbServiceName))
	
	// Environment variables
	service.Environment["MARIADB_ROOT_PASSWORD"] = getEnvOrDefault(svc.DatabaseRootPassword, "rootpassword")
	service.Environment["MARIADB_DATABASE"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
	service.Environment["MARIADB_USER"] = getEnvOrDefault(svc.DatabaseUser, svc.Name)
	service.Environment["MARIADB_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "healthcheck.sh", "--connect", "--innodb_initialized"},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
}

// addDatabaseEnvVars adds database connection environment variables to the app service
func addDatabaseEnvVars(service *DockerService, dbType, dbServiceName string, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	// Add standard database environment variables
	switch dbType {
	case "mysql", "mariadb":
		service.Environment["DB_CONNECTION"] = dbType
		service.Environment["DB_HOST"] = dbServiceName
		service.Environment["DB_PORT"] = "3306"
		service.Environment["DB_DATABASE"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
		service.Environment["DB_USERNAME"] = getEnvOrDefault(svc.DatabaseUser, svc.Name)
		service.Environment["DB_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
		
		// Laravel specific
		service.Environment["DATABASE_URL"] = fmt.Sprintf("%s://%s:%s@%s:3306/%s",
			dbType,
			getEnvOrDefault(svc.DatabaseUser, svc.Name),
			getEnvOrDefault(svc.DatabasePassword, "password"),
			dbServiceName,
			getEnvOrDefault(svc.DatabaseName, svc.Name),
		)
		
	case "postgres":
		service.Environment["DB_CONNECTION"] = "pgsql"
		service.Environment["DB_HOST"] = dbServiceName
		service.Environment["DB_PORT"] = "5432"
		service.Environment["DB_DATABASE"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
		service.Environment["DB_USERNAME"] = getEnvOrDefault(svc.DatabaseUser, svc.Name)
		service.Environment["DB_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
		
		// Standard PostgreSQL URL
		service.Environment["DATABASE_URL"] = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s",
			getEnvOrDefault(svc.DatabaseUser, svc.Name),
			getEnvOrDefault(svc.DatabasePassword, "password"),
			dbServiceName,
			getEnvOrDefault(svc.DatabaseName, svc.Name),
		)
		
	case "mongodb":
		service.Environment["MONGO_HOST"] = dbServiceName
		service.Environment["MONGO_PORT"] = "27017"
		service.Environment["MONGO_DB"] = getEnvOrDefault(svc.DatabaseName, svc.Name)
		service.Environment["MONGO_USER"] = getEnvOrDefault(svc.DatabaseUser, "admin")
		service.Environment["MONGO_PASSWORD"] = getEnvOrDefault(svc.DatabasePassword, "password")
		
		// MongoDB connection string
		service.Environment["MONGODB_URI"] = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s",
			getEnvOrDefault(svc.DatabaseUser, "admin"),
			getEnvOrDefault(svc.DatabasePassword, "password"),
			dbServiceName,
			getEnvOrDefault(svc.DatabaseName, svc.Name),
		)
		
	}
}

// getEnvOrDefault returns the value if not empty, otherwise returns the default
func getEnvOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// containsString checks if a string slice contains a string
func containsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}