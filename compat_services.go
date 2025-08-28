package main

import (
	"fmt"
	"strings"
)

// Compatibility service configuration
type CompatConfig struct {
	Type        string // minio, localstack, azurite
	Version     string
	AccessKey   string // For MinIO/S3 authentication
	SecretKey   string // For MinIO/S3 authentication
	ConsolePort int    // For MinIO console
	Region      string // For AWS/S3 region emulation
}

// Supported compatibility service versions
var supportedCompatVersions = map[string]map[string]string{
	"minio": {
		"2023":    "minio/minio:RELEASE.2023-12-20T01-00-02Z",
		"2024":    "minio/minio:RELEASE.2024-01-16T16-07-38Z",
		"latest":  "minio/minio:latest",
		"default": "minio/minio:RELEASE.2024-01-16T16-07-38Z",
	},
}

// parseCompatType parses compatibility service type and version from a string like "minio:2024"
func parseCompatType(compatString string) (compatType string, version string) {
	if compatString == "" {
		return "", ""
	}
	
	parts := strings.Split(compatString, ":")
	compatType = strings.ToLower(parts[0])
	
	if len(parts) > 1 {
		version = parts[1]
	} else {
		// Use default version if not specified
		if versions, ok := supportedCompatVersions[compatType]; ok {
			version = versions["default"]
			// Extract just the version number
			if idx := strings.LastIndex(version, ":"); idx >= 0 {
				version = version[idx+1:]
			}
			// Remove RELEASE prefix for MinIO
			if strings.HasPrefix(version, "RELEASE.") {
				version = strings.Split(version[8:], "T")[0]
				// Convert date format to year
				if len(version) >= 4 {
					version = version[:4]
				}
			}
		}
	}
	
	return compatType, version
}

// getCompatImage returns the appropriate Docker image for a compatibility service
func getCompatImage(compatType, version string) string {
	compatType = strings.ToLower(compatType)
	
	// Handle MinIO special case with RELEASE tags
	if compatType == "minio" && version != "" && version != "latest" {
		// Check if version is just a year (e.g., "2024")
		if len(version) == 4 {
			if versions, ok := supportedCompatVersions[compatType]; ok {
				if image, ok := versions[version]; ok {
					return image
				}
			}
		}
		// Otherwise try to find exact match
		for _, img := range supportedCompatVersions[compatType] {
			if strings.Contains(img, version) {
				return img
			}
		}
	}
	
	if versions, ok := supportedCompatVersions[compatType]; ok {
		// Try exact match first
		if image, ok := versions[version]; ok {
			return image
		}
		// Fallback to default
		return versions["default"]
	}
	
	// Unknown compatibility type
	return ""
}

// getSharedCompatServiceName returns a shared service name for a compatibility type and version
func getSharedCompatServiceName(compatType, version string) string {
	// Normalize the service name: minio-2024, etc.
	cleanVersion := version
	
	// For MinIO with RELEASE versions, extract just the year
	if compatType == "minio" && strings.HasPrefix(version, "RELEASE.") {
		// Extract date part from RELEASE.2024-01-16T16-07-38Z
		parts := strings.Split(version[8:], "T")
		if len(parts) > 0 {
			datePart := parts[0]
			// Get just the year
			if len(datePart) >= 4 {
				cleanVersion = datePart[:4]
			}
		}
	}
	
	// Clean up version string
	cleanVersion = strings.ReplaceAll(cleanVersion, ".", "")
	cleanVersion = strings.ReplaceAll(cleanVersion, "-", "")
	
	// For MinIO, ensure we only use the year
	if compatType == "minio" && len(cleanVersion) > 4 {
		cleanVersion = cleanVersion[:4]
	}
	
	return fmt.Sprintf("%s-%s", compatType, cleanVersion)
}

// addCompatService adds or reuses a compatibility service in the compose file
func addCompatService(compose *DockerCompose, svc *Service, config *Config) {
	if svc.Compat == "" {
		return
	}
	
	compatType, version := parseCompatType(svc.Compat)
	if compatType == "" {
		return
	}
	
	// Get the shared service name
	compatServiceName := getSharedCompatServiceName(compatType, version)
	
	// Check if this compatibility service already exists
	if _, exists := compose.Services[compatServiceName]; exists {
		// Service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, compatServiceName) {
				appService.DependsOn = append(appService.DependsOn, compatServiceName)
				compose.Services[svc.Name] = appService
			}
		}
		return
	}
	
	// Create the compatibility service
	compatImage := getCompatImage(compatType, version)
	if compatImage == "" {
		return
	}
	
	compatService := DockerService{
		Image:    compatImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
	}
	
	// Configure based on compatibility type
	switch compatType {
	case "minio":
		configureMinIOService(&compatService, svc, compatServiceName)
	}
	
	// Add the service to compose
	compose.Services[compatServiceName] = compatService
	
	// Update app service to depend on compatibility service
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, compatServiceName) {
			appService.DependsOn = append(appService.DependsOn, compatServiceName)
		}
		
		// Add compatibility service environment variables to the app
		addCompatEnvVars(&appService, compatType, compatServiceName, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureMinIOService configures a MinIO S3-compatible service
func configureMinIOService(service *DockerService, svc *Service, compatServiceName string) {
	// Data volume for persistence
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/data", compatServiceName))
	
	// Set access and secret keys
	accessKey := svc.CompatAccessKey
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := svc.CompatSecretKey
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	
	service.Environment["MINIO_ROOT_USER"] = accessKey
	service.Environment["MINIO_ROOT_PASSWORD"] = secretKey
	
	// Set the region if specified
	if svc.CompatRegion != "" {
		service.Environment["MINIO_REGION"] = svc.CompatRegion
	}
	
	// Configure browser/console access
	service.Environment["MINIO_BROWSER"] = "on"
	
	// Command to start MinIO server
	service.Command = "server /data --console-address :9001"
	
	// Expose both API and Console ports internally (not to host)
	// API port: 9000, Console port: 9001
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "curl", "-f", "http://localhost:9000/minio/health/live"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
}


// addCompatEnvVars adds compatibility service environment variables to the app service
func addCompatEnvVars(service *DockerService, compatType, compatServiceName string, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	switch compatType {
	case "minio":
		// S3-compatible environment variables
		service.Environment["S3_ENDPOINT"] = fmt.Sprintf("http://%s:9000", compatServiceName)
		service.Environment["S3_ENDPOINT_URL"] = fmt.Sprintf("http://%s:9000", compatServiceName)
		service.Environment["AWS_ENDPOINT_URL_S3"] = fmt.Sprintf("http://%s:9000", compatServiceName)
		
		// MinIO specific
		service.Environment["MINIO_ENDPOINT"] = fmt.Sprintf("http://%s:9000", compatServiceName)
		service.Environment["MINIO_CONSOLE_URL"] = fmt.Sprintf("http://%s:9001", compatServiceName)
		
		// Access credentials
		accessKey := svc.CompatAccessKey
		if accessKey == "" {
			accessKey = "minioadmin"
		}
		secretKey := svc.CompatSecretKey
		if secretKey == "" {
			secretKey = "minioadmin"
		}
		
		service.Environment["AWS_ACCESS_KEY_ID"] = accessKey
		service.Environment["AWS_SECRET_ACCESS_KEY"] = secretKey
		service.Environment["MINIO_ACCESS_KEY"] = accessKey
		service.Environment["MINIO_SECRET_KEY"] = secretKey
		
		// Region
		region := svc.CompatRegion
		if region == "" {
			region = "us-east-1"
		}
		service.Environment["AWS_DEFAULT_REGION"] = region
		service.Environment["AWS_REGION"] = region
		
		// S3 settings
		service.Environment["S3_USE_PATH_STYLE"] = "true"
		service.Environment["AWS_S3_FORCE_PATH_STYLE"] = "true"
	}
}