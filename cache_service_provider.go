package main

import (
	"fmt"
)

// CacheServiceProvider implements ServiceProvider for cache services
type CacheServiceProvider struct {
	defaultVersions map[string]string
	supportedVersions map[string][]string
}

// NewCacheServiceProvider creates a new cache service provider
func NewCacheServiceProvider() *CacheServiceProvider {
	return &CacheServiceProvider{
		defaultVersions: map[string]string{
			"redis":     "7.2",
			"memcached": "1.6",
		},
		supportedVersions: map[string][]string{
			"redis":     {"6.0", "6.2", "7.0", "7.2", "7.4"},
			"memcached": {"1.6", "1.6.23"},
		},
	}
}

// GetServiceName returns the container name for the cache service
func (p *CacheServiceProvider) GetServiceName(serviceType, version string) string {
	return getSharedCacheServiceName(serviceType, version)
}

// AddService adds the cache service to the Docker Compose configuration
func (p *CacheServiceProvider) AddService(compose *DockerCompose, svc *Service, config *Config) {
	addCacheService(compose, svc, config)
}

// ValidateConfig validates the cache service configuration
func (p *CacheServiceProvider) ValidateConfig(svc *Service) error {
	if svc.Cache == "" {
		return nil // No cache configured, nothing to validate
	}
	
	cacheType, version := parseCacheType(svc.Cache)
	if cacheType == "" {
		return fmt.Errorf("invalid cache type: %s", svc.Cache)
	}
	
	// Check if version is supported
	if version != "" {
		if supportedVersions, ok := p.supportedVersions[cacheType]; ok {
			found := false
			for _, v := range supportedVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported %s version: %s", cacheType, version)
			}
		}
	}
	
	return nil
}

// GetDefaultVersion returns the default version for the cache type
func (p *CacheServiceProvider) GetDefaultVersion() string {
	return "varies" // Different defaults per cache type
}

// GetSupportedVersions returns all supported versions
func (p *CacheServiceProvider) GetSupportedVersions() []string {
	var versions []string
	for cacheType, cacheVersions := range p.supportedVersions {
		for _, v := range cacheVersions {
			versions = append(versions, fmt.Sprintf("%s:%s", cacheType, v))
		}
	}
	return versions
}

// IsShared indicates if cache services use shared containers
func (p *CacheServiceProvider) IsShared() bool {
	return true
}

// GetEnvironmentVariables returns environment variables for dependent services
func (p *CacheServiceProvider) GetEnvironmentVariables(svc *Service, config *Config) map[string]string {
	if svc.Cache == "" {
		return nil
	}
	
	cacheType, version := parseCacheType(svc.Cache)
	serviceName := p.GetServiceName(cacheType, version)
	
	env := make(map[string]string)
	
	// Set cache connection environment variables
	switch cacheType {
	case "redis":
		env["REDIS_HOST"] = serviceName
		env["REDIS_PORT"] = "6379"
		env["CACHE_DRIVER"] = "redis"
		if svc.CachePassword != "" {
			env["REDIS_PASSWORD"] = svc.CachePassword
			env["REDIS_URL"] = fmt.Sprintf("redis://:%s@%s:6379/0", svc.CachePassword, serviceName)
		} else {
			env["REDIS_URL"] = fmt.Sprintf("redis://%s:6379/0", serviceName)
		}
		
	case "memcached":
		env["MEMCACHED_HOST"] = serviceName
		env["MEMCACHED_PORT"] = "11211"
		env["CACHE_DRIVER"] = "memcached"
		env["MEMCACHED_URL"] = fmt.Sprintf("%s:11211", serviceName)
	}
	
	return env
}