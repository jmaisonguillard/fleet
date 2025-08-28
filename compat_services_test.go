package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CompatServicesTestSuite struct {
	suite.Suite
}

func (suite *CompatServicesTestSuite) TestParseCompatType() {
	testCases := []struct {
		name          string
		input         string
		expectType    string
		expectVersion string
	}{
		{"MinIO with year version", "minio:2024", "minio", "2024"},
		{"MinIO 2023", "minio:2023", "minio", "2023"},
		{"MinIO without version", "minio", "minio", "2024"}, // Should use default
		{"Case insensitive", "MINIO:2023", "minio", "2023"},
		{"Empty string", "", "", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			compatType, version := parseCompatType(tc.input)
			suite.Equal(tc.expectType, compatType)
			suite.Equal(tc.expectVersion, version)
		})
	}
}

func (suite *CompatServicesTestSuite) TestGetCompatImage() {
	testCases := []struct {
		name       string
		compatType string
		version    string
		expected   string
	}{
		{"MinIO 2024", "minio", "2024", "minio/minio:RELEASE.2024-01-16T16-07-38Z"},
		{"MinIO 2023", "minio", "2023", "minio/minio:RELEASE.2023-12-20T01-00-02Z"},
		{"MinIO latest", "minio", "latest", "minio/minio:latest"},
		{"MinIO unknown version", "minio", "999", "minio/minio:RELEASE.2024-01-16T16-07-38Z"}, // Falls back to default
		{"Unknown compat type", "unknown", "1.0", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getCompatImage(tc.compatType, tc.version)
			suite.Equal(tc.expected, image)
		})
	}
}

func (suite *CompatServicesTestSuite) TestGetSharedCompatServiceName() {
	testCases := []struct {
		name       string
		compatType string
		version    string
		expected   string
	}{
		{"MinIO 2024", "minio", "2024", "minio-2024"},
		{"MinIO 2023", "minio", "2023", "minio-2023"},
		{"MinIO with RELEASE version", "minio", "RELEASE.2024-01-16T16-07-38Z", "minio-2024"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			serviceName := getSharedCompatServiceName(tc.compatType, tc.version)
			suite.Equal(tc.expected, serviceName)
		})
	}
}

func (suite *CompatServicesTestSuite) TestSharedContainerSameVersion() {
	// Test that multiple services using the same MinIO version share the container
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:   "api1",
				Image:  "node:18",
				Compat: "minio:2024",
			},
			{
				Name:   "api2",
				Image:  "python:3.9",
				Compat: "minio:2024", // Same version
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
	addCompatService(compose, &config.Services[0], config)

	// Add second service
	compose.Services["api2"] = DockerService{Image: "python:3.9"}
	addCompatService(compose, &config.Services[1], config)

	// Should only have one MinIO container
	_, exists := compose.Services["minio-2024"]
	suite.True(exists, "Shared MinIO container should exist")

	// Count MinIO services
	minioCount := 0
	for name := range compose.Services {
		if name == "minio-2024" {
			minioCount++
		}
	}
	suite.Equal(1, minioCount, "Should only have one MinIO container for same version")

	// Both app services should depend on the same MinIO service
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Contains(api1Service.DependsOn, "minio-2024")
	suite.Contains(api2Service.DependsOn, "minio-2024")
}

func (suite *CompatServicesTestSuite) TestSeparateContainersDifferentVersions() {
	// Test that different versions create separate containers
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:   "api1",
				Image:  "node:18",
				Compat: "minio:2024",
			},
			{
				Name:   "api2",
				Image:  "node:18",
				Compat: "minio:2023", // Different version
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
	addCompatService(compose, &config.Services[0], config)
	compose.Services["api2"] = DockerService{Image: "node:18"}
	addCompatService(compose, &config.Services[1], config)

	// Should have two MinIO containers
	_, exists1 := compose.Services["minio-2024"]
	_, exists2 := compose.Services["minio-2023"]
	suite.True(exists1, "MinIO 2024 container should exist")
	suite.True(exists2, "MinIO 2023 container should exist")
}

func (suite *CompatServicesTestSuite) TestConfigureMinIOService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:            "myapp",
		Compat:          "minio:2024",
		CompatAccessKey: "myaccesskey",
		CompatSecretKey: "mysecretkey",
		CompatRegion:    "eu-west-1",
	}

	configureMinIOService(service, svc, "minio-2024")

	// Check volume
	suite.Contains(service.Volumes, "minio-2024-data:/data")

	// Check environment variables
	suite.Equal("myaccesskey", service.Environment["MINIO_ROOT_USER"])
	suite.Equal("mysecretkey", service.Environment["MINIO_ROOT_PASSWORD"])
	suite.Equal("eu-west-1", service.Environment["MINIO_REGION"])
	suite.Equal("on", service.Environment["MINIO_BROWSER"])

	// Check command
	suite.Equal("server /data --console-address :9001", service.Command)

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "curl")
	suite.Contains(service.HealthCheck.Test, "http://localhost:9000/minio/health/live")
}

func (suite *CompatServicesTestSuite) TestConfigureMinIOServiceDefaults() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:   "myapp",
		Compat: "minio:2024",
	}

	configureMinIOService(service, svc, "minio-2024")

	// Check default credentials
	suite.Equal("minioadmin", service.Environment["MINIO_ROOT_USER"])
	suite.Equal("minioadmin", service.Environment["MINIO_ROOT_PASSWORD"])
}

func (suite *CompatServicesTestSuite) TestAddMinIOEnvVars() {
	service := &DockerService{
		Environment: make(map[string]string),
	}
	svc := &Service{
		Name:            "testapp",
		CompatAccessKey: "testkey",
		CompatSecretKey: "testsecret",
		CompatRegion:    "eu-west-1",
	}

	addCompatEnvVars(service, "minio", "minio-2024", svc)

	// Check S3-compatible environment variables
	suite.Equal("http://minio-2024:9000", service.Environment["S3_ENDPOINT"])
	suite.Equal("http://minio-2024:9000", service.Environment["S3_ENDPOINT_URL"])
	suite.Equal("http://minio-2024:9000", service.Environment["AWS_ENDPOINT_URL_S3"])
	
	// Check MinIO specific
	suite.Equal("http://minio-2024:9000", service.Environment["MINIO_ENDPOINT"])
	suite.Equal("http://minio-2024:9001", service.Environment["MINIO_CONSOLE_URL"])
	
	// Check credentials
	suite.Equal("testkey", service.Environment["AWS_ACCESS_KEY_ID"])
	suite.Equal("testsecret", service.Environment["AWS_SECRET_ACCESS_KEY"])
	suite.Equal("testkey", service.Environment["MINIO_ACCESS_KEY"])
	suite.Equal("testsecret", service.Environment["MINIO_SECRET_KEY"])
	
	// Check region
	suite.Equal("eu-west-1", service.Environment["AWS_DEFAULT_REGION"])
	suite.Equal("eu-west-1", service.Environment["AWS_REGION"])
	
	// Check S3 settings
	suite.Equal("true", service.Environment["S3_USE_PATH_STYLE"])
	suite.Equal("true", service.Environment["AWS_S3_FORCE_PATH_STYLE"])
}

func (suite *CompatServicesTestSuite) TestAddMinIOEnvVarsDefaults() {
	service := &DockerService{
		Environment: make(map[string]string),
	}
	svc := &Service{
		Name: "testapp",
	}

	addCompatEnvVars(service, "minio", "minio-2024", svc)

	// Check default credentials
	suite.Equal("minioadmin", service.Environment["AWS_ACCESS_KEY_ID"])
	suite.Equal("minioadmin", service.Environment["AWS_SECRET_ACCESS_KEY"])
	
	// Check default region
	suite.Equal("us-east-1", service.Environment["AWS_DEFAULT_REGION"])
}

func (suite *CompatServicesTestSuite) TestIntegrationComposeGenerationWithMinIO() {
	config := &Config{
		Project: "testproject",
		Services: []Service{
			{
				Name:            "storage-app",
				Image:           "node:18",
				Port:            3000,
				Compat:          "minio:2024",
				CompatAccessKey: "appkey",
				CompatSecretKey: "appsecret",
			},
			{
				Name:            "backup-app",
				Image:           "python:3.9",
				Port:            8000,
				Compat:          "minio:2024", // Same MinIO version
				CompatAccessKey: "appkey",
				CompatSecretKey: "appsecret",
			},
			{
				Name:   "legacy-app",
				Image:  "ruby:3.0",
				Port:   4000,
				Compat: "minio:2023", // Different MinIO version
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that MinIO services were created
	_, minio2024Exists := compose.Services["minio-2024"]
	_, minio2023Exists := compose.Services["minio-2023"]

	suite.True(minio2024Exists, "MinIO 2024 service should be created")
	suite.True(minio2023Exists, "MinIO 2023 service should be created")

	// Check dependencies
	storageService := compose.Services["storage-app"]
	backupService := compose.Services["backup-app"]
	legacyService := compose.Services["legacy-app"]

	suite.Contains(storageService.DependsOn, "minio-2024", "Storage app should depend on MinIO 2024")
	suite.Contains(backupService.DependsOn, "minio-2024", "Backup app should depend on MinIO 2024")
	suite.Contains(legacyService.DependsOn, "minio-2023", "Legacy app should depend on MinIO 2023")

	// Check that volumes are created
	_, minio2024VolExists := compose.Volumes["minio-2024-data"]
	suite.True(minio2024VolExists, "MinIO 2024 volume should be created")

	_, minio2023VolExists := compose.Volumes["minio-2023-data"]
	suite.True(minio2023VolExists, "MinIO 2023 volume should be created")
}

func (suite *CompatServicesTestSuite) TestServiceWithoutCompat() {
	// Test that services without compat configuration work correctly
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				// No compat specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Should only have the web service
	suite.NotNil(compose.Services["web"])

	// Should not have any MinIO services
	for name := range compose.Services {
		suite.NotContains(name, "minio")
	}
}

func (suite *CompatServicesTestSuite) TestMinIOWithFullStack() {
	// Test complete setup with MinIO, database, cache, and search
	config := &Config{
		Project: "fullstack",
		Services: []Service{
			{
				Name:            "app",
				Image:           "node:18",
				Port:            3000,
				Database:        "postgres:15",
				Cache:           "redis:7.2",
				Search:          "meilisearch:1.6",
				Compat:          "minio:2024",
				CompatAccessKey: "appkey",
				CompatSecretKey: "appsecret",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check all supporting services exist
	_, dbExists := compose.Services["postgres-15"]
	_, cacheExists := compose.Services["redis-72"]
	_, searchExists := compose.Services["meilisearch-16"]
	_, minioExists := compose.Services["minio-2024"]

	suite.True(dbExists, "Database service should exist")
	suite.True(cacheExists, "Cache service should exist")
	suite.True(searchExists, "Search service should exist")
	suite.True(minioExists, "MinIO service should exist")

	// Check app dependencies
	appService := compose.Services["app"]
	suite.Contains(appService.DependsOn, "postgres-15")
	suite.Contains(appService.DependsOn, "redis-72")
	suite.Contains(appService.DependsOn, "meilisearch-16")
	suite.Contains(appService.DependsOn, "minio-2024")
}

func (suite *CompatServicesTestSuite) TestMinIOCommandParsing() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:   "test",
		Compat: "minio",
	}

	configureMinIOService(service, svc, "minio-2024")

	// Verify command is properly formatted
	suite.Equal("server /data --console-address :9001", service.Command)
	
	// Verify command doesn't contain newlines or extra spaces
	suite.False(strings.Contains(service.Command, "\n"))
	suite.False(strings.Contains(service.Command, "  "))
}

func TestCompatServicesSuite(t *testing.T) {
	suite.Run(t, new(CompatServicesTestSuite))
}