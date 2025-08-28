package main

import (
	"fmt"
	"strconv"
)

// Laravel Reverb WebSocket server configuration
type ReverbConfig struct {
	Host      string
	Port      int
	AppID     string
	AppKey    string
	AppSecret string
}

// addReverbService adds Laravel Reverb WebSocket server to the compose file
func addReverbService(compose *DockerCompose, svc *Service, config *Config) {
	// Only add Reverb if explicitly requested and service uses Laravel
	if !svc.Reverb {
		return
	}
	
	// Check if this is a Laravel application (either explicitly set or detected)
	if svc.Framework != "laravel" && svc.Framework != "lumen" {
		// Don't add Reverb for non-Laravel applications
		return
	}
	
	// Service name for Reverb (singleton pattern like mailpit)
	reverbServiceName := "reverb"
	
	// Check if Reverb service already exists
	if _, exists := compose.Services[reverbServiceName]; exists {
		// Service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, reverbServiceName) {
				appService.DependsOn = append(appService.DependsOn, reverbServiceName)
			}
			// Add Reverb environment variables
			addReverbEnvVars(&appService, svc)
			compose.Services[svc.Name] = appService
		}
		return
	}
	
	// Get PHP version from runtime to match Reverb container
	phpVersion := "8.3" // Default
	if svc.Runtime != "" {
		_, version := parsePHPRuntime(svc.Runtime)
		if version != "" {
			phpVersion = version
		}
	}
	
	// Create the Reverb service
	reverbService := DockerService{
		Image:    fmt.Sprintf("php:%s-cli", phpVersion),
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
		WorkingDir: "/app",
	}
	
	// Mount the Laravel application code
	if svc.Folder != "" {
		// Share the same code volume as the main application
		reverbService.Volumes = append(reverbService.Volumes, fmt.Sprintf("./%s:/app", svc.Folder))
	}
	
	// Configure Reverb environment
	configureReverbService(&reverbService, svc)
	
	// Set the command to run Reverb
	reverbService.Command = "sh -c 'if [ -f /app/artisan ]; then php /app/artisan reverb:start --host=0.0.0.0 --port=8080 --hostname=reverb; else echo \"Laravel artisan not found. Ensure Laravel is installed.\"; sleep infinity; fi'"
	
	// Add health check
	reverbService.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
	
	// Add the service to compose
	compose.Services[reverbServiceName] = reverbService
	
	// Update app service to depend on Reverb
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, reverbServiceName) {
			appService.DependsOn = append(appService.DependsOn, reverbServiceName)
		}
		
		// Add Reverb environment variables to the app
		addReverbEnvVars(&appService, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureReverbService configures the Reverb service environment
func configureReverbService(service *DockerService, svc *Service) {
	// Default Reverb configuration
	port := 8080
	if svc.ReverbPort > 0 {
		port = svc.ReverbPort
	}
	
	host := "0.0.0.0"
	if svc.ReverbHost != "" {
		host = svc.ReverbHost
	}
	
	// App credentials (generate defaults if not provided)
	appId := svc.ReverbAppId
	if appId == "" {
		appId = "fleet-app"
	}
	
	appKey := svc.ReverbAppKey
	if appKey == "" {
		appKey = "fleet-app-key"
	}
	
	appSecret := svc.ReverbAppSecret
	if appSecret == "" {
		appSecret = "fleet-app-secret"
	}
	
	// Set Reverb environment variables
	service.Environment["REVERB_HOST"] = host
	service.Environment["REVERB_PORT"] = strconv.Itoa(port)
	service.Environment["REVERB_APP_ID"] = appId
	service.Environment["REVERB_APP_KEY"] = appKey
	service.Environment["REVERB_APP_SECRET"] = appSecret
	
	// Laravel broadcasting configuration
	service.Environment["BROADCAST_DRIVER"] = "reverb"
	service.Environment["BROADCAST_CONNECTION"] = "reverb"
	
	// Additional Reverb settings
	service.Environment["REVERB_MAX_REQUEST_SIZE"] = "10000"
	service.Environment["REVERB_SCALING_ENABLED"] = "false"
	service.Environment["REVERB_PULSE_INGEST_ENABLED"] = "false"
}

// addReverbEnvVars adds Reverb environment variables to the application service
func addReverbEnvVars(service *DockerService, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	// Default port
	port := 8080
	if svc.ReverbPort > 0 {
		port = svc.ReverbPort
	}
	
	// App credentials (use same defaults as in configureReverbService)
	appId := svc.ReverbAppId
	if appId == "" {
		appId = "fleet-app"
	}
	
	appKey := svc.ReverbAppKey
	if appKey == "" {
		appKey = "fleet-app-key"
	}
	
	appSecret := svc.ReverbAppSecret
	if appSecret == "" {
		appSecret = "fleet-app-secret"
	}
	
	// WebSocket connection settings
	service.Environment["REVERB_HOST"] = "reverb"
	service.Environment["REVERB_PORT"] = strconv.Itoa(port)
	service.Environment["REVERB_APP_ID"] = appId
	service.Environment["REVERB_APP_KEY"] = appKey
	service.Environment["REVERB_APP_SECRET"] = appSecret
	
	// Laravel Echo/Broadcasting configuration
	service.Environment["BROADCAST_DRIVER"] = "reverb"
	service.Environment["BROADCAST_CONNECTION"] = "reverb"
	
	// Vite/Frontend configuration for WebSocket connections
	service.Environment["VITE_REVERB_APP_ID"] = appId
	service.Environment["VITE_REVERB_APP_KEY"] = appKey
	service.Environment["VITE_REVERB_HOST"] = "reverb"
	service.Environment["VITE_REVERB_PORT"] = strconv.Itoa(port)
	service.Environment["VITE_REVERB_SCHEME"] = "http"
	
	// Echo server configuration
	service.Environment["REVERB_SCHEME"] = "http"
	service.Environment["REVERB_SERVER_HOST"] = "reverb"
	service.Environment["REVERB_SERVER_PORT"] = strconv.Itoa(port)
}