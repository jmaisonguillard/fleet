package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SearchServicesTestSuite struct {
	suite.Suite
}

func (suite *SearchServicesTestSuite) TestParseSearchType() {
	testCases := []struct {
		name          string
		input         string
		expectType    string
		expectVersion string
	}{
		{"Meilisearch with version", "meilisearch:1.6", "meilisearch", "1.6"},
		{"Meilisearch 1.5", "meilisearch:1.5", "meilisearch", "1.5"},
		{"Typesense with version", "typesense:27.1", "typesense", "27.1"},
		{"Typesense 26.0", "typesense:26.0", "typesense", "26.0"},
		{"Meilisearch without version", "meilisearch", "meilisearch", "1.6"}, // Should use default
		{"Typesense without version", "typesense", "typesense", "27.1"}, // Should use default
		{"Case insensitive", "MEILISEARCH:1.5", "meilisearch", "1.5"},
		{"Empty string", "", "", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			searchType, version := parseSearchType(tc.input)
			suite.Equal(tc.expectType, searchType)
			suite.Equal(tc.expectVersion, version)
		})
	}
}

func (suite *SearchServicesTestSuite) TestGetSearchImage() {
	testCases := []struct {
		name       string
		searchType string
		version    string
		expected   string
	}{
		{"Meilisearch 1.6", "meilisearch", "1.6", "getmeili/meilisearch:v1.6"},
		{"Meilisearch v1.5", "meilisearch", "v1.5", "getmeili/meilisearch:v1.5"},
		{"Meilisearch 1.4", "meilisearch", "1.4", "getmeili/meilisearch:v1.4"},
		{"Typesense 27.1", "typesense", "27.1", "typesense/typesense:27.1"},
		{"Typesense 26.0", "typesense", "26.0", "typesense/typesense:26.0"},
		{"Meilisearch unknown version", "meilisearch", "999", "getmeili/meilisearch:v1.6"}, // Falls back to default
		{"Unknown search type", "unknown", "1.0", ""},
		{"Meilisearch latest", "meilisearch", "latest", "getmeili/meilisearch:latest"},
		{"Typesense latest", "typesense", "latest", "typesense/typesense:latest"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getSearchImage(tc.searchType, tc.version)
			suite.Equal(tc.expected, image)
		})
	}
}

func (suite *SearchServicesTestSuite) TestGetSharedSearchServiceName() {
	testCases := []struct {
		name       string
		searchType string
		version    string
		expected   string
	}{
		{"Meilisearch 1.6", "meilisearch", "1.6", "meilisearch-16"},
		{"Meilisearch v1.5", "meilisearch", "v1.5", "meilisearch-15"},
		{"Typesense 27.1", "typesense", "27.1", "typesense-271"},
		{"Typesense 26.0", "typesense", "26.0", "typesense-260"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			serviceName := getSharedSearchServiceName(tc.searchType, tc.version)
			suite.Equal(tc.expected, serviceName)
		})
	}
}

func (suite *SearchServicesTestSuite) TestSharedContainerSameVersion() {
	// Test that multiple services using the same search version share the container
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:   "api1",
				Image:  "node:18",
				Search: "meilisearch:1.6",
			},
			{
				Name:   "api2",
				Image:  "python:3.9",
				Search: "meilisearch:1.6", // Same version
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
	addSearchService(compose, &config.Services[0], config)

	// Add second service
	compose.Services["api2"] = DockerService{Image: "python:3.9"}
	addSearchService(compose, &config.Services[1], config)

	// Should only have one Meilisearch container
	_, exists := compose.Services["meilisearch-16"]
	suite.True(exists, "Shared Meilisearch container should exist")

	// Count search services
	searchCount := 0
	for name := range compose.Services {
		if name == "meilisearch-16" {
			searchCount++
		}
	}
	suite.Equal(1, searchCount, "Should only have one Meilisearch container for same version")

	// Both app services should depend on the same search service
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Contains(api1Service.DependsOn, "meilisearch-16")
	suite.Contains(api2Service.DependsOn, "meilisearch-16")
}

func (suite *SearchServicesTestSuite) TestSeparateContainersDifferentVersions() {
	// Test that different versions create separate containers
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:   "api1",
				Image:  "node:18",
				Search: "meilisearch:1.6",
			},
			{
				Name:   "api2",
				Image:  "node:18",
				Search: "meilisearch:1.5", // Different version
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
	addSearchService(compose, &config.Services[0], config)
	compose.Services["api2"] = DockerService{Image: "node:18"}
	addSearchService(compose, &config.Services[1], config)

	// Should have two Meilisearch containers
	_, exists1 := compose.Services["meilisearch-16"]
	_, exists2 := compose.Services["meilisearch-15"]
	suite.True(exists1, "Meilisearch 1.6 container should exist")
	suite.True(exists2, "Meilisearch 1.5 container should exist")
}

func (suite *SearchServicesTestSuite) TestConfigureMeilisearchService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:            "myapp",
		Search:          "meilisearch:1.6",
		SearchMasterKey: "secure_master_key",
	}

	configureMeilisearchService(service, svc, "meilisearch-16")

	// Check volume
	suite.Contains(service.Volumes, "meilisearch-16-data:/meili_data")

	// Check environment variables
	suite.Equal("secure_master_key", service.Environment["MEILI_MASTER_KEY"])
	suite.Equal("production", service.Environment["MEILI_ENV"])
	suite.Equal("0.0.0.0:7700", service.Environment["MEILI_HTTP_ADDR"])
	suite.Equal("true", service.Environment["MEILI_NO_ANALYTICS"])

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "wget")
	suite.Contains(service.HealthCheck.Test, "http://localhost:7700/health")
}

func (suite *SearchServicesTestSuite) TestConfigureMeilisearchServiceNoAuth() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:   "myapp",
		Search: "meilisearch:1.6",
	}

	configureMeilisearchService(service, svc, "meilisearch-16")

	// Check development mode without master key
	suite.Equal("development", service.Environment["MEILI_ENV"])
	suite.Empty(service.Environment["MEILI_MASTER_KEY"])
}

func (suite *SearchServicesTestSuite) TestConfigureTypesenseService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:         "myapp",
		Search:       "typesense:27.1",
		SearchApiKey: "my_api_key",
	}

	configureTypesenseService(service, svc, "typesense-271")

	// Check volume
	suite.Contains(service.Volumes, "typesense-271-data:/data")

	// Check command includes API key
	suite.Contains(service.Command, "--api-key=my_api_key")
	suite.Contains(service.Command, "--data-dir=/data")
	suite.Contains(service.Command, "--enable-cors")
	suite.Contains(service.Command, "--listen-address=0.0.0.0")
	suite.Contains(service.Command, "--listen-port=8108")

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "wget")
	suite.Contains(service.HealthCheck.Test, "http://localhost:8108/health")
}

func (suite *SearchServicesTestSuite) TestConfigureTypesenseServiceDefaultKey() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:   "myapp",
		Search: "typesense:27.1",
	}

	configureTypesenseService(service, svc, "typesense-271")

	// Check default API key is set
	suite.Contains(service.Command, "--api-key=xyz123development")
}

func (suite *SearchServicesTestSuite) TestAddSearchEnvVars() {
	testCases := []struct {
		name       string
		searchType string
		checkVars  map[string]string
		apiKey     string
		masterKey  string
	}{
		{
			"Meilisearch environment variables with master key",
			"meilisearch",
			map[string]string{
				"MEILISEARCH_HOST":       "http://meilisearch-16:7700",
				"MEILISEARCH_URL":        "http://meilisearch-16:7700",
				"MEILISEARCH_KEY":        "master123",
				"MEILISEARCH_MASTER_KEY": "master123",
				"SEARCH_ENGINE":          "meilisearch",
				"SEARCH_HOST":            "meilisearch-16",
				"SEARCH_PORT":            "7700",
			},
			"",
			"master123",
		},
		{
			"Meilisearch environment variables without key",
			"meilisearch",
			map[string]string{
				"MEILISEARCH_HOST": "http://meilisearch-16:7700",
				"MEILISEARCH_URL":  "http://meilisearch-16:7700",
				"SEARCH_ENGINE":    "meilisearch",
				"SEARCH_HOST":      "meilisearch-16",
				"SEARCH_PORT":      "7700",
			},
			"",
			"",
		},
		{
			"Typesense environment variables",
			"typesense",
			map[string]string{
				"TYPESENSE_HOST":     "typesense-271",
				"TYPESENSE_PORT":     "8108",
				"TYPESENSE_PROTOCOL": "http",
				"TYPESENSE_URL":      "http://typesense-271:8108",
				"TYPESENSE_API_KEY":  "api_key_123",
				"SEARCH_ENGINE":      "typesense",
				"SEARCH_HOST":        "typesense-271",
				"SEARCH_PORT":        "8108",
			},
			"api_key_123",
			"",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			service := &DockerService{
				Environment: make(map[string]string),
			}
			svc := &Service{
				Name:            "testapp",
				SearchApiKey:    tc.apiKey,
				SearchMasterKey: tc.masterKey,
			}

			// Get the service name based on search type
			var searchServiceName string
			if tc.searchType == "meilisearch" {
				searchServiceName = "meilisearch-16"
			} else {
				searchServiceName = "typesense-271"
			}

			addSearchEnvVars(service, tc.searchType, searchServiceName, svc)

			for key, expectedValue := range tc.checkVars {
				suite.Equal(expectedValue, service.Environment[key], "Environment variable %s should be set correctly", key)
			}
		})
	}
}

func (suite *SearchServicesTestSuite) TestIntegrationComposeGenerationWithSearch() {
	config := &Config{
		Project: "testproject",
		Services: []Service{
			{
				Name:            "web",
				Image:           "nginx:alpine",
				Port:            80,
				Search:          "meilisearch:1.6",
				SearchMasterKey: "webkey",
			},
			{
				Name:         "api",
				Image:        "node:18",
				Port:         3000,
				Search:       "typesense:27.1",
				SearchApiKey: "apikey123",
			},
			{
				Name:            "indexer",
				Image:           "python:3.9",
				Search:          "meilisearch:1.6", // Same Meilisearch as web
				SearchMasterKey: "webkey",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that search services were created
	_, meilisearchExists := compose.Services["meilisearch-16"]
	_, typesenseExists := compose.Services["typesense-271"]

	suite.True(meilisearchExists, "Meilisearch service should be created")
	suite.True(typesenseExists, "Typesense service should be created")

	// Check dependencies
	webService := compose.Services["web"]
	apiService := compose.Services["api"]
	indexerService := compose.Services["indexer"]

	suite.Contains(webService.DependsOn, "meilisearch-16", "Web service should depend on Meilisearch")
	suite.Contains(apiService.DependsOn, "typesense-271", "API service should depend on Typesense")
	suite.Contains(indexerService.DependsOn, "meilisearch-16", "Indexer service should depend on Meilisearch")

	// Check that volumes are created
	_, meilisearchVolExists := compose.Volumes["meilisearch-16-data"]
	suite.True(meilisearchVolExists, "Meilisearch volume should be created")

	_, typesenseVolExists := compose.Volumes["typesense-271-data"]
	suite.True(typesenseVolExists, "Typesense volume should be created")
}

func (suite *SearchServicesTestSuite) TestMixedSearchTypes() {
	// Test a complex scenario with multiple search types
	config := &Config{
		Project: "complex",
		Services: []Service{
			{
				Name:   "app1",
				Image:  "node:18",
				Search: "meilisearch:1.6",
			},
			{
				Name:   "app2",
				Image:  "python:3.9",
				Search: "typesense:27.1",
			},
			{
				Name:   "app3",
				Image:  "ruby:3.0",
				Search: "meilisearch:1.6", // Shares with app1
			},
			{
				Name:   "app4",
				Image:  "php:8.1",
				Search: "meilisearch:1.5", // Different Meilisearch version
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check search service count
	searchServices := []string{"meilisearch-16", "meilisearch-15", "typesense-271"}
	for _, searchName := range searchServices {
		_, exists := compose.Services[searchName]
		suite.True(exists, "Search service %s should exist", searchName)
	}

	// Verify app1 and app3 share the same Meilisearch container
	app1Service := compose.Services["app1"]
	app3Service := compose.Services["app3"]
	suite.Contains(app1Service.DependsOn, "meilisearch-16")
	suite.Contains(app3Service.DependsOn, "meilisearch-16")

	// Verify app4 uses different Meilisearch version
	app4Service := compose.Services["app4"]
	suite.Contains(app4Service.DependsOn, "meilisearch-15")
}

func (suite *SearchServicesTestSuite) TestServiceWithoutSearch() {
	// Test that services without search configuration work correctly
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				// No search specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Should only have the web service
	suite.NotNil(compose.Services["web"])

	// Should not have any search services
	for name := range compose.Services {
		suite.NotContains(name, "meilisearch")
		suite.NotContains(name, "typesense")
	}
}

func (suite *SearchServicesTestSuite) TestSearchWithDatabaseAndCache() {
	// Test complete setup with search, database, and cache
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
				SearchMasterKey: "secret_key",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check all supporting services exist
	_, dbExists := compose.Services["postgres-15"]
	_, cacheExists := compose.Services["redis-72"]
	_, searchExists := compose.Services["meilisearch-16"]

	suite.True(dbExists, "Database service should exist")
	suite.True(cacheExists, "Cache service should exist")
	suite.True(searchExists, "Search service should exist")

	// Check app dependencies
	appService := compose.Services["app"]
	suite.Contains(appService.DependsOn, "postgres-15")
	suite.Contains(appService.DependsOn, "redis-72")
	suite.Contains(appService.DependsOn, "meilisearch-16")
}

func TestSearchServicesSuite(t *testing.T) {
	suite.Run(t, new(SearchServicesTestSuite))
}