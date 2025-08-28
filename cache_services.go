package main

import (
	"fmt"
	"strings"
)

// Cache service configuration
type CacheConfig struct {
	Type     string // redis, memcached
	Version  string
	Password string // For Redis auth
	MaxMemory string // For memory limits
}

// Supported cache service versions
var supportedCacheVersions = map[string]map[string]string{
	"redis": {
		"6.0":     "redis:6.0-alpine",
		"6.2":     "redis:6.2-alpine",
		"7.0":     "redis:7.0-alpine",
		"7.2":     "redis:7.2-alpine",
		"7.4":     "redis:7.4-alpine",
		"latest":  "redis:alpine",
		"default": "redis:7.2-alpine",
	},
	"memcached": {
		"1.6":     "memcached:1.6-alpine",
		"1.6.21":  "memcached:1.6.21-alpine",
		"1.6.22":  "memcached:1.6.22-alpine",
		"1.6.23":  "memcached:1.6.23-alpine",
		"latest":  "memcached:alpine",
		"default": "memcached:1.6-alpine",
	},
}

// parseCacheType parses cache type and version from a string like "redis:7.2"
func parseCacheType(cacheString string) (cacheType string, version string) {
	if cacheString == "" {
		return "", ""
	}
	
	parts := strings.Split(cacheString, ":")
	cacheType = strings.ToLower(parts[0])
	
	if len(parts) > 1 {
		version = parts[1]
	} else {
		// Use default version if not specified
		if versions, ok := supportedCacheVersions[cacheType]; ok {
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
	
	return cacheType, version
}

// getCacheImage returns the appropriate Docker image for a cache service
func getCacheImage(cacheType, version string) string {
	cacheType = strings.ToLower(cacheType)
	
	if versions, ok := supportedCacheVersions[cacheType]; ok {
		if image, ok := versions[version]; ok {
			return image
		}
		// Fallback to default
		return versions["default"]
	}
	
	// Unknown cache type
	return ""
}

// getSharedCacheServiceName returns a shared service name for a cache type and version
func getSharedCacheServiceName(cacheType, version string) string {
	// Normalize the service name: redis-72, memcached-16, etc.
	cleanVersion := strings.ReplaceAll(version, ".", "")
	return fmt.Sprintf("%s-%s", cacheType, cleanVersion)
}

// addCacheService adds or reuses a cache service in the compose file
func addCacheService(compose *DockerCompose, svc *Service, config *Config) {
	if svc.Cache == "" {
		return
	}
	
	cacheType, version := parseCacheType(svc.Cache)
	if cacheType == "" {
		return
	}
	
	// Get the shared service name
	cacheServiceName := getSharedCacheServiceName(cacheType, version)
	
	// Check if this cache service already exists
	if _, exists := compose.Services[cacheServiceName]; exists {
		// Service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, cacheServiceName) {
				appService.DependsOn = append(appService.DependsOn, cacheServiceName)
				compose.Services[svc.Name] = appService
			}
		}
		return
	}
	
	// Create the cache service
	cacheImage := getCacheImage(cacheType, version)
	if cacheImage == "" {
		return
	}
	
	cacheService := DockerService{
		Image:    cacheImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
	}
	
	// Configure based on cache type
	switch cacheType {
	case "redis":
		configureRedisService(&cacheService, svc, cacheServiceName)
	case "memcached":
		configureMemcachedService(&cacheService, svc, cacheServiceName)
	}
	
	// Add the service to compose
	compose.Services[cacheServiceName] = cacheService
	
	// Update app service to depend on cache
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, cacheServiceName) {
			appService.DependsOn = append(appService.DependsOn, cacheServiceName)
		}
		
		// Add cache connection environment variables to the app
		addCacheEnvVars(&appService, cacheType, cacheServiceName, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureRedisService configures a Redis service
func configureRedisService(service *DockerService, svc *Service, cacheServiceName string) {
	// Data volume for persistence (optional for cache, but good to have)
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/data", cacheServiceName))
	
	// Add password if specified (use the existing Password field for backward compatibility)
	// Or use the new CachePassword field
	password := svc.CachePassword
	if password == "" && svc.Cache != "" && strings.HasPrefix(svc.Cache, "redis") {
		// For backward compatibility, check if Password field is set and this is likely a Redis service
		if svc.Password != "" && svc.Image == "" {
			password = svc.Password
		}
	}
	
	if password != "" {
		service.Command = fmt.Sprintf("redis-server --requirepass %s --appendonly yes", password)
	} else {
		service.Command = "redis-server --appendonly yes"
	}
	
	// Set max memory if specified
	if svc.CacheMaxMemory != "" {
		if service.Command != "" {
			service.Command += fmt.Sprintf(" --maxmemory %s --maxmemory-policy allkeys-lru", svc.CacheMaxMemory)
		}
	}
	
	// Health check
	if password != "" {
		service.HealthCheck = &HealthCheckYAML{
			Test:     []string{"CMD", "redis-cli", "-a", password, "ping"},
			Interval: "30s",
			Timeout:  "3s",
			Retries:  3,
		}
	} else {
		service.HealthCheck = &HealthCheckYAML{
			Test:     []string{"CMD", "redis-cli", "ping"},
			Interval: "30s",
			Timeout:  "3s",
			Retries:  3,
		}
	}
}

// configureMemcachedService configures a Memcached service
func configureMemcachedService(service *DockerService, svc *Service, cacheServiceName string) {
	// Memcached doesn't use persistent storage by design
	// But we can set memory limits and connection limits
	
	memoryLimit := "64"  // Default 64MB
	if svc.CacheMaxMemory != "" {
		// Parse memory limit - memcached expects MB as integer
		memoryLimit = strings.TrimSuffix(svc.CacheMaxMemory, "m")
		memoryLimit = strings.TrimSuffix(memoryLimit, "M")
	}
	
	// Configure memcached with memory limit and max connections
	service.Command = fmt.Sprintf("memcached -m %s -c 1024", memoryLimit)
	
	// Health check using stats command
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", "echo 'stats' | nc localhost 11211 | grep -q 'STAT'"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
}

// addCacheEnvVars adds cache connection environment variables to the app service
func addCacheEnvVars(service *DockerService, cacheType, cacheServiceName string, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	// Add standard cache environment variables
	switch cacheType {
	case "redis":
		service.Environment["REDIS_HOST"] = cacheServiceName
		service.Environment["REDIS_PORT"] = "6379"
		
		// Handle password
		password := svc.CachePassword
		if password == "" && svc.Password != "" && svc.Cache != "" && strings.HasPrefix(svc.Cache, "redis") {
			password = svc.Password
		}
		
		if password != "" {
			service.Environment["REDIS_PASSWORD"] = password
			service.Environment["REDIS_URL"] = fmt.Sprintf("redis://:%s@%s:6379/0", password, cacheServiceName)
		} else {
			service.Environment["REDIS_URL"] = fmt.Sprintf("redis://%s:6379/0", cacheServiceName)
		}
		
		// Laravel/common framework variables
		service.Environment["CACHE_DRIVER"] = "redis"
		service.Environment["SESSION_DRIVER"] = "redis"
		service.Environment["QUEUE_CONNECTION"] = "redis"
		
	case "memcached":
		service.Environment["MEMCACHED_HOST"] = cacheServiceName
		service.Environment["MEMCACHED_PORT"] = "11211"
		service.Environment["MEMCACHED_URL"] = fmt.Sprintf("%s:11211", cacheServiceName)
		
		// Laravel/common framework variables
		service.Environment["CACHE_DRIVER"] = "memcached"
		service.Environment["SESSION_DRIVER"] = "memcached"
	}
}