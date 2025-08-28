package main

import (
	"fmt"
)

// SearchServiceProvider implements ServiceProvider for search services
type SearchServiceProvider struct {
	defaultVersions map[string]string
	supportedVersions map[string][]string
}

// NewSearchServiceProvider creates a new search service provider
func NewSearchServiceProvider() *SearchServiceProvider {
	return &SearchServiceProvider{
		defaultVersions: map[string]string{
			"meilisearch": "1.6",
			"typesense":   "27.1",
		},
		supportedVersions: map[string][]string{
			"meilisearch": {"1.0", "1.1", "1.2", "1.3", "1.4", "1.5", "1.6"},
			"typesense":   {"0.24", "0.25", "26.0", "27.0", "27.1"},
		},
	}
}

// GetServiceName returns the container name for the search service
func (p *SearchServiceProvider) GetServiceName(serviceType, version string) string {
	return getSharedSearchServiceName(serviceType, version)
}

// AddService adds the search service to the Docker Compose configuration
func (p *SearchServiceProvider) AddService(compose *DockerCompose, svc *Service, config *Config) {
	addSearchService(compose, svc, config)
}

// ValidateConfig validates the search service configuration
func (p *SearchServiceProvider) ValidateConfig(svc *Service) error {
	if svc.Search == "" {
		return nil // No search configured, nothing to validate
	}
	
	searchType, version := parseSearchType(svc.Search)
	if searchType == "" {
		return fmt.Errorf("invalid search type: %s", svc.Search)
	}
	
	// Check if version is supported
	if version != "" {
		if supportedVersions, ok := p.supportedVersions[searchType]; ok {
			found := false
			for _, v := range supportedVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported %s version: %s", searchType, version)
			}
		}
	}
	
	// Validate required fields for specific search engines
	if searchType == "typesense" && svc.SearchApiKey == "" {
		return fmt.Errorf("typesense requires search_api_key to be set")
	}
	
	return nil
}

// GetDefaultVersion returns the default version for the search type
func (p *SearchServiceProvider) GetDefaultVersion() string {
	return "varies" // Different defaults per search type
}

// GetSupportedVersions returns all supported versions
func (p *SearchServiceProvider) GetSupportedVersions() []string {
	var versions []string
	for searchType, searchVersions := range p.supportedVersions {
		for _, v := range searchVersions {
			versions = append(versions, fmt.Sprintf("%s:%s", searchType, v))
		}
	}
	return versions
}

// IsShared indicates if search services use shared containers
func (p *SearchServiceProvider) IsShared() bool {
	return true
}

// GetEnvironmentVariables returns environment variables for dependent services
func (p *SearchServiceProvider) GetEnvironmentVariables(svc *Service, config *Config) map[string]string {
	if svc.Search == "" {
		return nil
	}
	
	searchType, version := parseSearchType(svc.Search)
	serviceName := p.GetServiceName(searchType, version)
	
	env := make(map[string]string)
	env["SEARCH_ENGINE"] = searchType
	
	// Set search engine specific environment variables
	switch searchType {
	case "meilisearch":
		env["MEILISEARCH_HOST"] = fmt.Sprintf("http://%s:7700", serviceName)
		env["MEILISEARCH_URL"] = env["MEILISEARCH_HOST"]
		if svc.SearchMasterKey != "" {
			env["MEILISEARCH_KEY"] = svc.SearchMasterKey
			env["MEILISEARCH_MASTER_KEY"] = svc.SearchMasterKey
		}
		
	case "typesense":
		env["TYPESENSE_HOST"] = serviceName
		env["TYPESENSE_PORT"] = "8108"
		env["TYPESENSE_URL"] = fmt.Sprintf("http://%s:8108", serviceName)
		env["TYPESENSE_API_KEY"] = svc.SearchApiKey
	}
	
	return env
}