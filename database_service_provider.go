package main

import (
	"fmt"
)

// DatabaseServiceProvider implements ServiceProvider for database services
type DatabaseServiceProvider struct {
	defaultVersions map[string]string
	supportedVersions map[string][]string
}

// NewDatabaseServiceProvider creates a new database service provider
func NewDatabaseServiceProvider() *DatabaseServiceProvider {
	return &DatabaseServiceProvider{
		defaultVersions: map[string]string{
			"mysql":    "8.0",
			"postgres": "15",
			"mongodb":  "7.0",
			"mariadb":  "11.1",
		},
		supportedVersions: map[string][]string{
			"mysql":    {"5.7", "8.0", "8.1", "8.2", "8.3"},
			"postgres": {"13", "14", "15", "16"},
			"mongodb":  {"5.0", "6.0", "7.0"},
			"mariadb":  {"10.6", "10.11", "11.0", "11.1", "11.2"},
		},
	}
}

// GetServiceName returns the container name for the database service
func (p *DatabaseServiceProvider) GetServiceName(serviceType, version string) string {
	return getSharedDatabaseServiceName(serviceType, version)
}

// AddService adds the database service to the Docker Compose configuration
func (p *DatabaseServiceProvider) AddService(compose *DockerCompose, svc *Service, config *Config) {
	addDatabaseService(compose, svc, config)
}

// ValidateConfig validates the database service configuration
func (p *DatabaseServiceProvider) ValidateConfig(svc *Service) error {
	if svc.Database == "" {
		return nil // No database configured, nothing to validate
	}
	
	dbType, version := parseDatabaseType(svc.Database)
	if dbType == "" {
		return fmt.Errorf("invalid database type: %s", svc.Database)
	}
	
	// Check if version is supported
	if version != "" {
		if supportedVersions, ok := p.supportedVersions[dbType]; ok {
			found := false
			for _, v := range supportedVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported %s version: %s", dbType, version)
			}
		}
	}
	
	return nil
}

// GetDefaultVersion returns the default version for the database type
func (p *DatabaseServiceProvider) GetDefaultVersion() string {
	return "varies" // Different defaults per database type
}

// GetSupportedVersions returns all supported versions
func (p *DatabaseServiceProvider) GetSupportedVersions() []string {
	var versions []string
	for dbType, dbVersions := range p.supportedVersions {
		for _, v := range dbVersions {
			versions = append(versions, fmt.Sprintf("%s:%s", dbType, v))
		}
	}
	return versions
}

// IsShared indicates if database services use shared containers
func (p *DatabaseServiceProvider) IsShared() bool {
	return true
}

// GetEnvironmentVariables returns environment variables for dependent services
func (p *DatabaseServiceProvider) GetEnvironmentVariables(svc *Service, config *Config) map[string]string {
	if svc.Database == "" {
		return nil
	}
	
	dbType, version := parseDatabaseType(svc.Database)
	serviceName := p.GetServiceName(dbType, version)
	
	env := make(map[string]string)
	
	// Set database connection environment variables
	switch dbType {
	case "mysql", "mariadb":
		env["DB_CONNECTION"] = "mysql"
		env["DB_HOST"] = serviceName
		env["DB_PORT"] = "3306"
		env["DB_DATABASE"] = getString(svc.DatabaseName, config.Project)
		env["DB_USERNAME"] = getString(svc.DatabaseUser, "root")
		if svc.DatabasePassword != "" {
			env["DB_PASSWORD"] = svc.DatabasePassword
		} else if svc.DatabaseRootPassword != "" {
			env["DB_PASSWORD"] = svc.DatabaseRootPassword
		}
		env["DATABASE_URL"] = fmt.Sprintf("mysql://%s:%s@%s:3306/%s",
			env["DB_USERNAME"], env["DB_PASSWORD"], serviceName, env["DB_DATABASE"])
	
	case "postgres":
		env["DB_CONNECTION"] = "pgsql"
		env["DB_HOST"] = serviceName
		env["DB_PORT"] = "5432"
		env["DB_DATABASE"] = getString(svc.DatabaseName, config.Project)
		env["DB_USERNAME"] = getString(svc.DatabaseUser, "postgres")
		if svc.DatabasePassword != "" {
			env["DB_PASSWORD"] = svc.DatabasePassword
		}
		env["DATABASE_URL"] = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s",
			env["DB_USERNAME"], env["DB_PASSWORD"], serviceName, env["DB_DATABASE"])
	
	case "mongodb":
		env["DB_CONNECTION"] = "mongodb"
		env["DB_HOST"] = serviceName
		env["DB_PORT"] = "27017"
		env["DB_DATABASE"] = getString(svc.DatabaseName, config.Project)
		if svc.DatabaseRootPassword != "" {
			env["MONGODB_URL"] = fmt.Sprintf("mongodb://root:%s@%s:27017/", 
				svc.DatabaseRootPassword, serviceName)
			env["DATABASE_URL"] = env["MONGODB_URL"]
		} else {
			env["MONGODB_URL"] = fmt.Sprintf("mongodb://%s:27017/%s", 
				serviceName, env["DB_DATABASE"])
			env["DATABASE_URL"] = env["MONGODB_URL"]
		}
	}
	
	return env
}

// getString returns the string value or default
func getString(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}