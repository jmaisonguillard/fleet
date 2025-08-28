package main

import (
	"fmt"
)

// CompatServiceProvider implements ServiceProvider for compatibility services (S3, etc)
type CompatServiceProvider struct {
	defaultVersions map[string]string
	supportedVersions map[string][]string
}

// NewCompatServiceProvider creates a new compatibility service provider
func NewCompatServiceProvider() *CompatServiceProvider {
	return &CompatServiceProvider{
		defaultVersions: map[string]string{
			"minio": "2024",
		},
		supportedVersions: map[string][]string{
			"minio": {"2023", "2024", "latest"},
		},
	}
}

// GetServiceName returns the container name for the compat service
func (p *CompatServiceProvider) GetServiceName(serviceType, version string) string {
	return getSharedCompatServiceName(serviceType, version)
}

// AddService adds the compat service to the Docker Compose configuration
func (p *CompatServiceProvider) AddService(compose *DockerCompose, svc *Service, config *Config) {
	addCompatService(compose, svc, config)
}

// ValidateConfig validates the compat service configuration
func (p *CompatServiceProvider) ValidateConfig(svc *Service) error {
	if svc.Compat == "" {
		return nil // No compat service configured, nothing to validate
	}
	
	compatType, version := parseCompatType(svc.Compat)
	if compatType == "" {
		return fmt.Errorf("invalid compat service type: %s", svc.Compat)
	}
	
	// Check if version is supported
	if version != "" {
		if supportedVersions, ok := p.supportedVersions[compatType]; ok {
			found := false
			for _, v := range supportedVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported %s version: %s", compatType, version)
			}
		}
	}
	
	return nil
}

// GetDefaultVersion returns the default version for the compat type
func (p *CompatServiceProvider) GetDefaultVersion() string {
	return "2024" // Default MinIO version
}

// GetSupportedVersions returns all supported versions
func (p *CompatServiceProvider) GetSupportedVersions() []string {
	var versions []string
	for compatType, compatVersions := range p.supportedVersions {
		for _, v := range compatVersions {
			versions = append(versions, fmt.Sprintf("%s:%s", compatType, v))
		}
	}
	return versions
}

// IsShared indicates if compat services use shared containers
func (p *CompatServiceProvider) IsShared() bool {
	return true
}

// GetEnvironmentVariables returns environment variables for dependent services
func (p *CompatServiceProvider) GetEnvironmentVariables(svc *Service, config *Config) map[string]string {
	if svc.Compat == "" {
		return nil
	}
	
	compatType, version := parseCompatType(svc.Compat)
	serviceName := p.GetServiceName(compatType, version)
	
	env := make(map[string]string)
	
	// Set compatibility service specific environment variables
	switch compatType {
	case "minio":
		// S3 compatible environment variables
		env["S3_ENDPOINT"] = fmt.Sprintf("http://%s:9000", serviceName)
		env["S3_REGION"] = getStringOrDefault(svc.CompatRegion, "us-east-1")
		env["S3_BUCKET"] = config.Project
		
		// AWS SDK compatible variables
		env["AWS_ENDPOINT"] = env["S3_ENDPOINT"]
		env["AWS_REGION"] = env["S3_REGION"]
		env["AWS_DEFAULT_REGION"] = env["S3_REGION"]
		env["AWS_BUCKET"] = env["S3_BUCKET"]
		
		// MinIO specific
		env["MINIO_ENDPOINT"] = env["S3_ENDPOINT"]
		env["MINIO_BUCKET"] = env["S3_BUCKET"]
		
		// Credentials
		if svc.CompatAccessKey != "" {
			env["AWS_ACCESS_KEY_ID"] = svc.CompatAccessKey
			env["S3_ACCESS_KEY"] = svc.CompatAccessKey
			env["MINIO_ACCESS_KEY"] = svc.CompatAccessKey
		}
		
		if svc.CompatSecretKey != "" {
			env["AWS_SECRET_ACCESS_KEY"] = svc.CompatSecretKey
			env["S3_SECRET_KEY"] = svc.CompatSecretKey
			env["MINIO_SECRET_KEY"] = svc.CompatSecretKey
		}
		
		// Storage URL for frameworks
		env["STORAGE_URL"] = env["S3_ENDPOINT"]
	}
	
	return env
}

// getStringOrDefault returns the value if not empty, otherwise returns the default
func getStringOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}