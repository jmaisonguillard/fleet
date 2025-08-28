package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CacheServicesTestSuite struct {
	suite.Suite
}

func (suite *CacheServicesTestSuite) TestParseCacheType() {
	testCases := []struct {
		name          string
		input         string
		expectType    string
		expectVersion string
	}{
		{"Redis with version", "redis:7.2", "redis", "7.2"},
		{"Redis 7.0", "redis:7.0", "redis", "7.0"},
		{"Memcached with version", "memcached:1.6", "memcached", "1.6"},
		{"Memcached specific version", "memcached:1.6.23", "memcached", "1.6.23"},
		{"Redis without version", "redis", "redis", "7.2"}, // Should use default
		{"Memcached without version", "memcached", "memcached", "1.6"}, // Should use default
		{"Case insensitive", "REDIS:7.0", "redis", "7.0"},
		{"Empty string", "", "", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cacheType, version := parseCacheType(tc.input)
			suite.Equal(tc.expectType, cacheType)
			suite.Equal(tc.expectVersion, version)
		})
	}
}

func (suite *CacheServicesTestSuite) TestGetCacheImage() {
	testCases := []struct {
		name      string
		cacheType string
		version   string
		expected  string
	}{
		{"Redis 7.2", "redis", "7.2", "redis:7.2-alpine"},
		{"Redis 7.0", "redis", "7.0", "redis:7.0-alpine"},
		{"Redis 6.2", "redis", "6.2", "redis:6.2-alpine"},
		{"Memcached 1.6", "memcached", "1.6", "memcached:1.6-alpine"},
		{"Memcached 1.6.23", "memcached", "1.6.23", "memcached:1.6.23-alpine"},
		{"Redis unknown version", "redis", "999", "redis:7.2-alpine"}, // Falls back to default
		{"Unknown cache type", "unknown", "1.0", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getCacheImage(tc.cacheType, tc.version)
			suite.Equal(tc.expected, image)
		})
	}
}

func (suite *CacheServicesTestSuite) TestGetSharedCacheServiceName() {
	testCases := []struct {
		name      string
		cacheType string
		version   string
		expected  string
	}{
		{"Redis 7.2", "redis", "7.2", "redis-72"},
		{"Redis 7.0", "redis", "7.0", "redis-70"},
		{"Memcached 1.6", "memcached", "1.6", "memcached-16"},
		{"Memcached 1.6.23", "memcached", "1.6.23", "memcached-1623"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			serviceName := getSharedCacheServiceName(tc.cacheType, tc.version)
			suite.Equal(tc.expected, serviceName)
		})
	}
}

func (suite *CacheServicesTestSuite) TestSharedContainerSameVersion() {
	// Test that multiple services using the same cache version share the container
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "api1",
				Image: "node:18",
				Cache: "redis:7.2",
			},
			{
				Name:  "api2",
				Image: "python:3.9",
				Cache: "redis:7.2", // Same version
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	// Add first service
	compose.Services["api1"] = DockerService{Image: "node:18"}
	addCacheService(compose, &config.Services[0], config)

	// Add second service
	compose.Services["api2"] = DockerService{Image: "python:3.9"}
	addCacheService(compose, &config.Services[1], config)

	// Should only have one Redis container
	_, exists := compose.Services["redis-72"]
	suite.True(exists, "Shared Redis container should exist")

	// Count cache services
	cacheCount := 0
	for name := range compose.Services {
		if name == "redis-72" {
			cacheCount++
		}
	}
	suite.Equal(1, cacheCount, "Should only have one Redis container for same version")

	// Both app services should depend on the same cache
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Contains(api1Service.DependsOn, "redis-72")
	suite.Contains(api2Service.DependsOn, "redis-72")
}

func (suite *CacheServicesTestSuite) TestSeparateContainersDifferentVersions() {
	// Test that different versions create separate containers
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "api1",
				Image: "node:18",
				Cache: "redis:7.2",
			},
			{
				Name:  "api2",
				Image: "node:18",
				Cache: "redis:7.0", // Different version
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	// Add services
	compose.Services["api1"] = DockerService{Image: "node:18"}
	addCacheService(compose, &config.Services[0], config)
	compose.Services["api2"] = DockerService{Image: "node:18"}
	addCacheService(compose, &config.Services[1], config)

	// Should have two Redis containers
	_, exists1 := compose.Services["redis-72"]
	_, exists2 := compose.Services["redis-70"]
	suite.True(exists1, "Redis 7.2 container should exist")
	suite.True(exists2, "Redis 7.0 container should exist")
}

func (suite *CacheServicesTestSuite) TestConfigureRedisService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:          "myapp",
		Cache:         "redis:7.2",
		CachePassword: "secret123",
		CacheMaxMemory: "256m",
	}

	configureRedisService(service, svc, "redis-72")

	// Check volume
	suite.Contains(service.Volumes, "redis-72-data:/data")

	// Check command includes password and memory settings
	suite.Contains(service.Command, "--requirepass secret123")
	suite.Contains(service.Command, "--maxmemory 256m")
	suite.Contains(service.Command, "--maxmemory-policy allkeys-lru")
	suite.Contains(service.Command, "--appendonly yes")

	// Check health check with password
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "redis-cli")
	suite.Contains(service.HealthCheck.Test, "-a")
	suite.Contains(service.HealthCheck.Test, "secret123")
}

func (suite *CacheServicesTestSuite) TestConfigureRedisServiceNoPassword() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:  "myapp",
		Cache: "redis:7.2",
	}

	configureRedisService(service, svc, "redis-72")

	// Check command without password
	suite.Equal("redis-server --appendonly yes", service.Command)

	// Check health check without password
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "redis-cli")
	suite.Contains(service.HealthCheck.Test, "ping")
	suite.NotContains(service.HealthCheck.Test, "-a")
}

func (suite *CacheServicesTestSuite) TestConfigureMemcachedService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:           "myapp",
		Cache:          "memcached:1.6",
		CacheMaxMemory: "128m",
	}

	configureMemcachedService(service, svc, "memcached-16")

	// Check no volumes (memcached is memory-only)
	suite.Empty(service.Volumes)

	// Check command includes memory limit
	suite.Contains(service.Command, "-m 128")
	suite.Contains(service.Command, "-c 1024")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Equal("CMD-SHELL", service.HealthCheck.Test[0])
	suite.Contains(service.HealthCheck.Test[1], "stats")
	suite.Contains(service.HealthCheck.Test[1], "nc localhost 11211")
}

func (suite *CacheServicesTestSuite) TestAddCacheEnvVars() {
	testCases := []struct {
		name       string
		cacheType  string
		checkVars  map[string]string
		password   string
	}{
		{
			"Redis environment variables with password",
			"redis",
			map[string]string{
				"REDIS_HOST":       "redis-72",
				"REDIS_PORT":       "6379",
				"REDIS_PASSWORD":   "secret123",
				"CACHE_DRIVER":     "redis",
				"SESSION_DRIVER":   "redis",
				"QUEUE_CONNECTION": "redis",
			},
			"secret123",
		},
		{
			"Redis environment variables without password",
			"redis",
			map[string]string{
				"REDIS_HOST":       "redis-72",
				"REDIS_PORT":       "6379",
				"CACHE_DRIVER":     "redis",
				"SESSION_DRIVER":   "redis",
				"QUEUE_CONNECTION": "redis",
			},
			"",
		},
		{
			"Memcached environment variables",
			"memcached",
			map[string]string{
				"MEMCACHED_HOST": "memcached-16",
				"MEMCACHED_PORT": "11211",
				"CACHE_DRIVER":   "memcached",
				"SESSION_DRIVER": "memcached",
			},
			"",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			service := &DockerService{
				Environment: make(map[string]string),
			}
			svc := &Service{
				Name:          "testapp",
				CachePassword: tc.password,
			}

			// Get the service name based on cache type
			var cacheServiceName string
			if tc.cacheType == "redis" {
				cacheServiceName = "redis-72"
			} else {
				cacheServiceName = "memcached-16"
			}

			addCacheEnvVars(service, tc.cacheType, cacheServiceName, svc)

			for key, expectedValue := range tc.checkVars {
				suite.Equal(expectedValue, service.Environment[key], "Environment variable %s should be set correctly", key)
			}

			// Check URL is set
			if tc.cacheType == "redis" {
				suite.NotEmpty(service.Environment["REDIS_URL"], "REDIS_URL should be set")
				if tc.password != "" {
					suite.Contains(service.Environment["REDIS_URL"], tc.password)
				}
			} else if tc.cacheType == "memcached" {
				suite.NotEmpty(service.Environment["MEMCACHED_URL"], "MEMCACHED_URL should be set")
			}
		})
	}
}

func (suite *CacheServicesTestSuite) TestBackwardCompatibilityRedisPassword() {
	// Test backward compatibility with existing Password field for Redis
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:     "cache",
				Cache:    "redis:7.2",
				Password: "oldpassword", // Using old Password field
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	compose.Services["cache"] = DockerService{Image: "redis:7.2-alpine"}
	addCacheService(compose, &config.Services[0], config)

	redisService := compose.Services["redis-72"]
	suite.Contains(redisService.Command, "--requirepass oldpassword")
}

func (suite *CacheServicesTestSuite) TestIntegrationComposeGenerationWithCaches() {
	config := &Config{
		Project: "testproject",
		Services: []Service{
			{
				Name:          "web",
				Image:         "nginx:alpine",
				Port:          80,
				Cache:         "redis:7.2",
				CachePassword: "webpass",
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,
				Cache: "memcached:1.6",
			},
			{
				Name:          "worker",
				Image:         "python:3.9",
				Cache:         "redis:7.2", // Same Redis as web
				CachePassword: "webpass",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that cache services were created
	_, redisExists := compose.Services["redis-72"]
	_, memcachedExists := compose.Services["memcached-16"]

	suite.True(redisExists, "Redis service should be created")
	suite.True(memcachedExists, "Memcached service should be created")

	// Check dependencies
	webService := compose.Services["web"]
	apiService := compose.Services["api"]
	workerService := compose.Services["worker"]

	suite.Contains(webService.DependsOn, "redis-72", "Web service should depend on Redis")
	suite.Contains(apiService.DependsOn, "memcached-16", "API service should depend on Memcached")
	suite.Contains(workerService.DependsOn, "redis-72", "Worker service should depend on Redis")

	// Check that Redis volume is created
	_, redisVolExists := compose.Volumes["redis-72-data"]
	suite.True(redisVolExists, "Redis volume should be created")

	// Memcached shouldn't have a volume
	_, memcachedVolExists := compose.Volumes["memcached-16-data"]
	suite.False(memcachedVolExists, "Memcached should not have a volume")
}

func (suite *CacheServicesTestSuite) TestMixedCacheTypes() {
	// Test a complex scenario with multiple cache types
	config := &Config{
		Project: "complex",
		Services: []Service{
			{
				Name:  "app1",
				Image: "node:18",
				Cache: "redis:7.2",
			},
			{
				Name:  "app2",
				Image: "python:3.9",
				Cache: "memcached:1.6",
			},
			{
				Name:  "app3",
				Image: "ruby:3.0",
				Cache: "redis:7.2", // Shares with app1
			},
			{
				Name:  "app4",
				Image: "php:8.1",
				Cache: "redis:7.0", // Different Redis version
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check cache service count
	cacheServices := []string{"redis-72", "redis-70", "memcached-16"}
	for _, cacheName := range cacheServices {
		_, exists := compose.Services[cacheName]
		suite.True(exists, "Cache service %s should exist", cacheName)
	}

	// Verify app1 and app3 share the same Redis container
	app1Service := compose.Services["app1"]
	app3Service := compose.Services["app3"]
	suite.Contains(app1Service.DependsOn, "redis-72")
	suite.Contains(app3Service.DependsOn, "redis-72")

	// Verify app4 uses different Redis version
	app4Service := compose.Services["app4"]
	suite.Contains(app4Service.DependsOn, "redis-70")
}

func (suite *CacheServicesTestSuite) TestServiceWithoutCache() {
	// Test that services without cache configuration work correctly
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				// No cache specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Should only have the web service and nginx-proxy (if domains exist)
	suite.NotNil(compose.Services["web"])

	// Should not have any cache services
	for name := range compose.Services {
		suite.NotContains(name, "redis")
		suite.NotContains(name, "memcached")
	}
}

func TestCacheServicesSuite(t *testing.T) {
	suite.Run(t, new(CacheServicesTestSuite))
}