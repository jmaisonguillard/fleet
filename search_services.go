package main

import (
	"fmt"
	"strings"
)

// Search service configuration
type SearchConfig struct {
	Type      string // meilisearch, typesense
	Version   string
	MasterKey string // For Meilisearch authentication
	APIKey    string // For Typesense authentication
}

// Supported search service versions
var supportedSearchVersions = map[string]map[string]string{
	"meilisearch": {
		"1.0":     "getmeili/meilisearch:v1.0",
		"1.1":     "getmeili/meilisearch:v1.1",
		"1.2":     "getmeili/meilisearch:v1.2",
		"1.3":     "getmeili/meilisearch:v1.3",
		"1.4":     "getmeili/meilisearch:v1.4",
		"1.5":     "getmeili/meilisearch:v1.5",
		"1.6":     "getmeili/meilisearch:v1.6",
		"latest":  "getmeili/meilisearch:latest",
		"default": "getmeili/meilisearch:v1.6",
	},
	"typesense": {
		"0.24":    "typesense/typesense:0.24.0",
		"0.25":    "typesense/typesense:0.25.2",
		"26.0":    "typesense/typesense:26.0",
		"27.0":    "typesense/typesense:27.0",
		"27.1":    "typesense/typesense:27.1",
		"latest":  "typesense/typesense:latest",
		"default": "typesense/typesense:27.1",
	},
}

// parseSearchType parses search type and version from a string like "meilisearch:1.6"
func parseSearchType(searchString string) (searchType string, version string) {
	if searchString == "" {
		return "", ""
	}
	
	parts := strings.Split(searchString, ":")
	searchType = strings.ToLower(parts[0])
	
	if len(parts) > 1 {
		version = parts[1]
	} else {
		// Use default version if not specified
		if versions, ok := supportedSearchVersions[searchType]; ok {
			version = versions["default"]
			// Extract just the version number
			if idx := strings.LastIndex(version, ":"); idx >= 0 {
				version = version[idx+1:]
			}
			// Remove v prefix for consistency
			version = strings.TrimPrefix(version, "v")
		}
	}
	
	return searchType, version
}

// getSearchImage returns the appropriate Docker image for a search service
func getSearchImage(searchType, version string) string {
	searchType = strings.ToLower(searchType)
	
	// Add v prefix for meilisearch versions if not present
	if searchType == "meilisearch" && version != "" && version != "latest" && !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	
	if versions, ok := supportedSearchVersions[searchType]; ok {
		// Try exact match first
		if image, ok := versions[version]; ok {
			return image
		}
		// Try without v prefix for meilisearch
		if searchType == "meilisearch" {
			versionNoV := strings.TrimPrefix(version, "v")
			if image, ok := versions[versionNoV]; ok {
				return image
			}
		}
		// Fallback to default
		return versions["default"]
	}
	
	// Unknown search type
	return ""
}

// getSharedSearchServiceName returns a shared service name for a search type and version
func getSharedSearchServiceName(searchType, version string) string {
	// Normalize the service name: meilisearch-16, typesense-271, etc.
	cleanVersion := strings.ReplaceAll(version, ".", "")
	cleanVersion = strings.TrimPrefix(cleanVersion, "v")
	return fmt.Sprintf("%s-%s", searchType, cleanVersion)
}

// addSearchService adds or reuses a search service in the compose file
func addSearchService(compose *DockerCompose, svc *Service, config *Config) {
	if svc.Search == "" {
		return
	}
	
	searchType, version := parseSearchType(svc.Search)
	if searchType == "" {
		return
	}
	
	// Get the shared service name
	searchServiceName := getSharedSearchServiceName(searchType, version)
	
	// Check if this search service already exists
	if _, exists := compose.Services[searchServiceName]; exists {
		// Service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, searchServiceName) {
				appService.DependsOn = append(appService.DependsOn, searchServiceName)
				compose.Services[svc.Name] = appService
			}
		}
		return
	}
	
	// Create the search service
	searchImage := getSearchImage(searchType, version)
	if searchImage == "" {
		return
	}
	
	searchService := DockerService{
		Image:    searchImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
	}
	
	// Configure based on search type
	switch searchType {
	case "meilisearch":
		configureMeilisearchService(&searchService, svc, searchServiceName)
	case "typesense":
		configureTypesenseService(&searchService, svc, searchServiceName)
	}
	
	// Add the service to compose
	compose.Services[searchServiceName] = searchService
	
	// Update app service to depend on search
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, searchServiceName) {
			appService.DependsOn = append(appService.DependsOn, searchServiceName)
		}
		
		// Add search connection environment variables to the app
		addSearchEnvVars(&appService, searchType, searchServiceName, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureMeilisearchService configures a Meilisearch service
func configureMeilisearchService(service *DockerService, svc *Service, searchServiceName string) {
	// Data volume for persistence
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/meili_data", searchServiceName))
	
	// Add master key if specified
	masterKey := svc.SearchApiKey
	if masterKey == "" && svc.SearchMasterKey != "" {
		masterKey = svc.SearchMasterKey
	}
	
	if masterKey != "" {
		service.Environment["MEILI_MASTER_KEY"] = masterKey
		// In production mode with master key
		service.Environment["MEILI_ENV"] = "production"
	} else {
		// Development mode without master key
		service.Environment["MEILI_ENV"] = "development"
	}
	
	// Set the HTTP address to bind to all interfaces
	service.Environment["MEILI_HTTP_ADDR"] = "0.0.0.0:7700"
	
	// Configure analytics (disabled by default for privacy)
	service.Environment["MEILI_NO_ANALYTICS"] = "true"
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:7700/health"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
}

// configureTypesenseService configures a Typesense service
func configureTypesenseService(service *DockerService, svc *Service, searchServiceName string) {
	// Data volume for persistence
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/data", searchServiceName))
	
	// API key is required for Typesense
	apiKey := svc.SearchApiKey
	if apiKey == "" {
		// Generate a default API key for development
		apiKey = "xyz123development"
	}
	
	// Typesense uses command line arguments for configuration
	args := []string{
		"--data-dir=/data",
		"--api-key=" + apiKey,
		"--enable-cors",
		"--listen-address=0.0.0.0",
		"--listen-port=8108",
	}
	
	service.Command = strings.Join(args, " ")
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8108/health"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
}

// addSearchEnvVars adds search connection environment variables to the app service
func addSearchEnvVars(service *DockerService, searchType, searchServiceName string, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	switch searchType {
	case "meilisearch":
		service.Environment["MEILISEARCH_HOST"] = fmt.Sprintf("http://%s:7700", searchServiceName)
		service.Environment["MEILISEARCH_URL"] = fmt.Sprintf("http://%s:7700", searchServiceName)
		
		// Handle master key/API key
		masterKey := svc.SearchApiKey
		if masterKey == "" && svc.SearchMasterKey != "" {
			masterKey = svc.SearchMasterKey
		}
		
		if masterKey != "" {
			service.Environment["MEILISEARCH_KEY"] = masterKey
			service.Environment["MEILISEARCH_MASTER_KEY"] = masterKey
		}
		
		// Common search engine environment variables
		service.Environment["SEARCH_ENGINE"] = "meilisearch"
		service.Environment["SEARCH_HOST"] = searchServiceName
		service.Environment["SEARCH_PORT"] = "7700"
		
	case "typesense":
		service.Environment["TYPESENSE_HOST"] = searchServiceName
		service.Environment["TYPESENSE_PORT"] = "8108"
		service.Environment["TYPESENSE_PROTOCOL"] = "http"
		service.Environment["TYPESENSE_URL"] = fmt.Sprintf("http://%s:8108", searchServiceName)
		
		// API key
		apiKey := svc.SearchApiKey
		if apiKey == "" {
			apiKey = "xyz123development"
		}
		service.Environment["TYPESENSE_API_KEY"] = apiKey
		
		// Common search engine environment variables
		service.Environment["SEARCH_ENGINE"] = "typesense"
		service.Environment["SEARCH_HOST"] = searchServiceName
		service.Environment["SEARCH_PORT"] = "8108"
	}
}