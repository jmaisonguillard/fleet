package main

// ServiceProvider defines the interface for all service providers
// This standardizes how different service types are handled in Fleet
type ServiceProvider interface {
	// GetServiceName returns the container name for the service
	GetServiceName(serviceType, version string) string
	
	// AddService adds the service to the Docker Compose configuration
	AddService(compose *DockerCompose, svc *Service, config *Config)
	
	// ValidateConfig validates the service configuration
	ValidateConfig(svc *Service) error
	
	// GetDefaultVersion returns the default version for the service type
	GetDefaultVersion() string
	
	// GetSupportedVersions returns all supported versions
	GetSupportedVersions() []string
	
	// IsShared indicates if this service type uses shared containers
	IsShared() bool
	
	// GetEnvironmentVariables returns environment variables for dependent services
	GetEnvironmentVariables(svc *Service, config *Config) map[string]string
}

// ServiceRegistry manages all registered service providers
type ServiceRegistry struct {
	providers map[string]ServiceProvider
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		providers: make(map[string]ServiceProvider),
	}
}

// Register adds a service provider to the registry
func (r *ServiceRegistry) Register(serviceType string, provider ServiceProvider) {
	r.providers[serviceType] = provider
}

// Get returns a service provider by type
func (r *ServiceRegistry) Get(serviceType string) (ServiceProvider, bool) {
	provider, exists := r.providers[serviceType]
	return provider, exists
}

// GetAll returns all registered providers
func (r *ServiceRegistry) GetAll() map[string]ServiceProvider {
	return r.providers
}

// DefaultServiceRegistry is the global service registry
var DefaultServiceRegistry = NewServiceRegistry()

// RegisterDefaultProviders registers all default service providers
func RegisterDefaultProviders() {
	DefaultServiceRegistry.Register("database", NewDatabaseServiceProvider())
	DefaultServiceRegistry.Register("cache", NewCacheServiceProvider())
	DefaultServiceRegistry.Register("search", NewSearchServiceProvider())
	DefaultServiceRegistry.Register("compat", NewCompatServiceProvider())
	DefaultServiceRegistry.Register("email", NewEmailServiceProvider())
}