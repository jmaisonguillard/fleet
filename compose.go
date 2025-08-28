package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type DockerCompose struct {
	Version  string                    `yaml:"version"`
	Services map[string]DockerService  `yaml:"services"`
	Networks map[string]DockerNetwork  `yaml:"networks,omitempty"`
	Volumes  map[string]DockerVolume   `yaml:"volumes,omitempty"`
}

type DockerService struct {
	Image       string            `yaml:"image,omitempty"`
	Build       string            `yaml:"build,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	HealthCheck *HealthCheckYAML  `yaml:"healthcheck,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	ExtraHosts  []string          `yaml:"extra_hosts,omitempty"`
}

type HealthCheckYAML struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

type DockerNetwork struct {
	Driver string                 `yaml:"driver"`
	IPAM   *DockerNetworkIPAM     `yaml:"ipam,omitempty"`
}

type DockerNetworkIPAM struct {
	Config []DockerNetworkIPAMConfig `yaml:"config,omitempty"`
}

type DockerNetworkIPAMConfig struct {
	Subnet string `yaml:"subnet,omitempty"`
}

type DockerVolume struct {
	Driver string `yaml:"driver"`
}

func generateDockerCompose(config *Config) *DockerCompose {
	// Ensure .fleet directory exists for generated configs
	os.MkdirAll(".fleet", 0755)
	
	compose := &DockerCompose{
		Version:  "3.8",
		Services: make(map[string]DockerService),
		Networks: map[string]DockerNetwork{
			"fleet-network": {
				Driver: "bridge",
				IPAM: &DockerNetworkIPAM{
					Config: []DockerNetworkIPAMConfig{
						{Subnet: "172.28.0.0/16"},
					},
				},
			},
		},
		Volumes: make(map[string]DockerVolume),
	}

	// Track which volumes need to be created
	volumesNeeded := make(map[string]bool)

	for _, svc := range config.Services {
		service := DockerService{
			Networks: []string{"fleet-network"},
			Restart:  "unless-stopped",
		}

		// Handle image or build
		if svc.Image != "" {
			service.Image = svc.Image
		} else if svc.Build != "" {
			service.Build = svc.Build
		}

		// Handle ports - only expose if service has no domain
		// Services with domains are accessed through nginx proxy, not directly
		// Only expose ports if:
		// 1. No nginx proxy is needed (no services with domains/ports in the config), OR
		// 2. This specific service has no domain (neither explicit nor auto-generated)
		hasDomain := getDomainForService(&svc) != ""
		if !hasDomain {
			if svc.Port > 0 {
				service.Ports = []string{fmt.Sprintf("%d:%d", svc.Port, svc.Port)}
			} else if len(svc.Ports) > 0 {
				service.Ports = svc.Ports
			}
		}

		// Handle volumes
		if svc.Folder != "" {
			// If it's an nginx image with PHP runtime, set up for PHP
			if strings.Contains(strings.ToLower(svc.Image), "nginx") && strings.HasPrefix(svc.Runtime, "php") {
				// For PHP, nginx serves from /var/www/html
				service.Volumes = append(service.Volumes, fmt.Sprintf("../%s:/var/www/html", svc.Folder))
				
				// Auto-detect framework if not specified
				framework := svc.Framework
				if framework == "" {
					framework = detectPHPFramework(svc.Folder)
				}
				
				// Parse PHP version from runtime
				_, phpVersion := parsePHPRuntime(svc.Runtime)
				
				// Generate and mount PHP nginx config with version
				configPath, err := writeNginxPHPConfigWithVersion(svc.Name, framework, phpVersion)
				if err == nil {
					absPath, _ := filepath.Abs(configPath)
					service.Volumes = append(service.Volumes, fmt.Sprintf("%s:/etc/nginx/conf.d/default.conf:ro", absPath))
				}
			} else if strings.Contains(strings.ToLower(svc.Image), "nginx") {
				// Regular nginx service
				service.Volumes = append(service.Volumes, fmt.Sprintf("../%s:/usr/share/nginx/html", svc.Folder))
			} else {
				// For other images, map to /app
				service.Volumes = append(service.Volumes, fmt.Sprintf("../%s:/app", svc.Folder))
			}
		}

		// Handle named volumes
		for _, vol := range svc.Volumes {
			service.Volumes = append(service.Volumes, vol)
			// If it's a named volume (not a bind mount), track it
			if !strings.Contains(vol, "/") && !strings.Contains(vol, ".") {
				volName := strings.Split(vol, ":")[0]
				volumesNeeded[volName] = true
			}
		}

		// Handle environment variables
		if len(svc.Environment) > 0 {
			service.Environment = svc.Environment
		}

		// Handle special password field for databases
		if svc.Password != "" {
			if service.Environment == nil {
				service.Environment = make(map[string]string)
			}
			
			// Auto-detect database type and set appropriate password env var
			if strings.Contains(svc.Image, "postgres") {
				service.Environment["POSTGRES_PASSWORD"] = svc.Password
				service.Environment["POSTGRES_DB"] = config.Project
			} else if strings.Contains(svc.Image, "mysql") || strings.Contains(svc.Image, "mariadb") {
				service.Environment["MYSQL_ROOT_PASSWORD"] = svc.Password
				service.Environment["MYSQL_DATABASE"] = config.Project
			} else if strings.Contains(svc.Image, "mongo") {
				service.Environment["MONGO_INITDB_ROOT_USERNAME"] = "root"
				service.Environment["MONGO_INITDB_ROOT_PASSWORD"] = svc.Password
			} else if strings.Contains(svc.Image, "redis") {
				service.Command = fmt.Sprintf("redis-server --requirepass %s", svc.Password)
			}
		}

		// Handle dependencies
		if len(svc.Needs) > 0 {
			service.DependsOn = svc.Needs
		}
		
		// Add PHP-FPM dependency for nginx with PHP runtime
		if strings.Contains(strings.ToLower(svc.Image), "nginx") && strings.HasPrefix(svc.Runtime, "php") {
			phpServiceName := fmt.Sprintf("%s-php", svc.Name)
			found := false
			for _, dep := range service.DependsOn {
				if dep == phpServiceName {
					found = true
					break
				}
			}
			if !found {
				service.DependsOn = append(service.DependsOn, phpServiceName)
			}
		}

		// Handle command
		if svc.Command != "" {
			service.Command = svc.Command
		}

		// Handle health check
		if svc.HealthCheck.Test != "" {
			service.HealthCheck = &HealthCheckYAML{
				Test:     strings.Split(svc.HealthCheck.Test, " "),
				Interval: svc.HealthCheck.Interval,
				Timeout:  svc.HealthCheck.Timeout,
				Retries:  svc.HealthCheck.Retries,
			}
			
			// Set defaults if not specified
			if service.HealthCheck.Interval == "" {
				service.HealthCheck.Interval = "30s"
			}
			if service.HealthCheck.Timeout == "" {
				service.HealthCheck.Timeout = "10s"
			}
			if service.HealthCheck.Retries == 0 {
				service.HealthCheck.Retries = 3
			}
		}

		compose.Services[svc.Name] = service
		
		// Add PHP-FPM service if this nginx service has PHP runtime
		if strings.Contains(strings.ToLower(svc.Image), "nginx") && strings.HasPrefix(svc.Runtime, "php") {
			addPHPFPMService(compose, &svc, config)
		}
		
		// Add database service if specified
		if svc.Database != "" {
			addDatabaseService(compose, &svc, config)
		}
		
		// Add cache service if specified
		if svc.Cache != "" {
			addCacheService(compose, &svc, config)
		}
		
		// Add search service if specified
		if svc.Search != "" {
			addSearchService(compose, &svc, config)
		}
		
		// Add compatibility service if specified
		if svc.Compat != "" {
			addCompatService(compose, &svc, config)
		}
		
		// Add email service if specified
		if svc.Email != "" {
			addEmailService(compose, &svc, config)
		}
		
		// Add Laravel Reverb service if specified (for Laravel/Lumen apps)
		if svc.Reverb && (svc.Framework == "laravel" || svc.Framework == "lumen") {
			addReverbService(compose, &svc, config)
		}
	}

	// Create volume definitions
	for volName := range volumesNeeded {
		compose.Volumes[volName] = DockerVolume{
			Driver: "local",
		}
	}
	
	// Add database volumes
	for _, service := range compose.Services {
		for _, volume := range service.Volumes {
			// Check if it's a named volume (contains : but doesn't start with . or /)
			if strings.Contains(volume, ":") {
				parts := strings.Split(volume, ":")
				volName := parts[0]
				if !strings.HasPrefix(volName, ".") && !strings.HasPrefix(volName, "/") && strings.HasSuffix(volName, "-data") {
					compose.Volumes[volName] = DockerVolume{
						Driver: "local",
					}
				}
			}
		}
	}

	// If no volumes needed, remove the volumes section
	if len(compose.Volumes) == 0 {
		compose.Volumes = nil
	}

	// Add nginx proxy if needed
	addNginxProxyToCompose(compose, config)
	
	// Write PostgreSQL initialization scripts if needed
	writePostgresInitScripts(compose)

	return compose
}

func writePostgresInitScripts(compose *DockerCompose) {
	// Check all services for PostgreSQL init scripts in labels
	for _, service := range compose.Services {
		if service.Labels != nil {
			if script, ok := service.Labels["fleet.postgres.init.script"]; ok {
				if path, ok := service.Labels["fleet.postgres.init.path"]; ok {
					// Write the init script to the specified path
					os.WriteFile(path, []byte(script), 0644)
					// Remove the labels after writing (they're not needed in docker-compose.yml)
					delete(service.Labels, "fleet.postgres.init.script")
					delete(service.Labels, "fleet.postgres.init.path")
					// If labels map is empty, set it to nil
					if len(service.Labels) == 0 {
						service.Labels = nil
					}
				}
			}
		}
	}
}

func writeDockerCompose(compose *DockerCompose, filename string) error {
	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal docker-compose: %w", err)
	}

	// Add header comment
	header := "# Generated by Fleet CLI - DO NOT EDIT\n# Edit fleet.toml instead and regenerate\n\n"
	data = append([]byte(header), data...)

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}