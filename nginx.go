package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// NginxConfig represents the nginx configuration
type NginxConfig struct {
	Services []ServiceWithDomain
}

// ServiceWithDomain represents a service with domain configuration
type ServiceWithDomain struct {
	Name   string
	Domain string
	Port   int
}

// shouldAddNginxProxy checks if we need to add nginx proxy
func shouldAddNginxProxy(config *Config) bool {
	for _, svc := range config.Services {
		if svc.Domain != "" || svc.Port > 0 {
			return true
		}
	}
	return false
}

// getDomainForService returns the domain for a service
func getDomainForService(svc *Service) string {
	if svc.Domain != "" {
		return svc.Domain
	}
	// Auto-generate domain as {service-name}.test
	if svc.Port > 0 {
		return fmt.Sprintf("%s.test", svc.Name)
	}
	return ""
}

// generateNginxConfig generates nginx configuration from fleet config
func generateNginxConfig(config *Config) (string, error) {
	// Read the template
	tmplContent, err := templatesFS.ReadFile("templates/nginx/nginx.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read nginx template: %w", err)
	}

	// Parse template
	tmpl, err := template.New("nginx").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse nginx template: %w", err)
	}

	// Prepare services with domains
	services := []ServiceWithDomain{}
	for _, svc := range config.Services {
		domain := getDomainForService(&svc)
		if domain != "" {
			port := svc.Port
			if port == 0 && len(svc.Ports) > 0 {
				// Extract port from first port mapping
				// Format can be: "8080:80", "127.0.0.1:8080:80", or "8080:80/tcp"
				parts := strings.Split(svc.Ports[0], ":")
				containerPort := parts[len(parts)-1]
				// Remove protocol suffix if present (e.g., "80/tcp" -> "80")
				if idx := strings.Index(containerPort, "/"); idx > 0 {
					containerPort = containerPort[:idx]
				}
				fmt.Sscanf(containerPort, "%d", &port)
			}
			
			services = append(services, ServiceWithDomain{
				Name:   svc.Name,
				Domain: domain,
				Port:   port,
			})
		}
	}

	// Execute template
	var buf bytes.Buffer
	nginxConfig := NginxConfig{Services: services}
	if err := tmpl.Execute(&buf, nginxConfig); err != nil {
		return "", fmt.Errorf("failed to execute nginx template: %w", err)
	}

	return buf.String(), nil
}

// writeNginxConfig writes nginx configuration to file
func writeNginxConfig(config *Config, filename string) error {
	nginxConf, err := generateNginxConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, []byte(nginxConf), 0644); err != nil {
		return fmt.Errorf("failed to write nginx config: %w", err)
	}

	return nil
}

// addNginxProxyToCompose adds nginx proxy service to docker-compose
func addNginxProxyToCompose(compose *DockerCompose, config *Config) {
	if !shouldAddNginxProxy(config) {
		return
	}

	// Get current working directory for absolute paths
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Warning: failed to get working directory: %v\n", err)
		return
	}

	// Create .fleet directory if it doesn't exist
	fleetDir := filepath.Join(cwd, ".fleet")
	if err := os.MkdirAll(fleetDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create .fleet directory: %v\n", err)
		return
	}

	// Create the nginx config file path with absolute path
	nginxConfigPath := filepath.Join(fleetDir, "nginx.conf")
	
	// Write nginx config BEFORE creating docker service
	if err := writeNginxConfig(config, nginxConfigPath); err != nil {
		fmt.Printf("Warning: failed to write nginx config: %v\n", err)
		return
	}

	// Verify the file exists and is readable
	if _, err := os.Stat(nginxConfigPath); err != nil {
		fmt.Printf("Warning: nginx config file does not exist or is not accessible: %v\n", err)
		return
	}

	// Ensure file has proper permissions for Docker to read
	if err := os.Chmod(nginxConfigPath, 0644); err != nil {
		fmt.Printf("Warning: failed to set permissions on nginx config: %v\n", err)
		return
	}

	// Add nginx proxy service with absolute path for the volume mount
	nginxService := DockerService{
		Image:    "nginx:alpine",
		Ports:    []string{"80:80"},
		Volumes:  []string{fmt.Sprintf("%s:/etc/nginx/nginx.conf:ro", nginxConfigPath)},
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		DependsOn: []string{},
		HealthCheck: &HealthCheckYAML{
			Test:     []string{"CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost/health"},
			Interval: "30s",
			Timeout:  "3s",
			Retries:  3,
		},
	}

	// Add all services with domains as dependencies
	for _, svc := range config.Services {
		if getDomainForService(&svc) != "" {
			nginxService.DependsOn = append(nginxService.DependsOn, svc.Name)
		}
	}

	compose.Services["nginx-proxy"] = nginxService
}

// getDomainMappings returns all domain to IP mappings for hosts file
func getDomainMappings(config *Config) map[string]string {
	mappings := make(map[string]string)
	
	for _, svc := range config.Services {
		domain := getDomainForService(&svc)
		if domain != "" {
			// All domains point to localhost where nginx is listening
			mappings[domain] = "127.0.0.1"
		}
	}
	
	return mappings
}

// getHostsFilePath returns the path to the system hosts file
var getHostsFilePath = func() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "drivers", "etc", "hosts")
	default:
		return "/etc/hosts"
	}
}

// updateHostsFileWithDomains updates the hosts file with service domains
func updateHostsFileWithDomains(config *Config) error {
	mappings := getDomainMappings(config)
	if len(mappings) == 0 {
		return nil
	}

	hostsFile := getHostsFilePath()
	
	// Read current hosts file
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}
	inFleetSection := false
	
	// Process existing lines, removing old Fleet service entries
	for _, line := range lines {
		if strings.Contains(line, "# Fleet Services - START") {
			inFleetSection = true
			continue
		}
		if strings.Contains(line, "# Fleet Services - END") {
			inFleetSection = false
			continue
		}
		if !inFleetSection {
			newLines = append(newLines, line)
		}
	}

	// Add new Fleet service entries
	if len(mappings) > 0 {
		newLines = append(newLines, "# Fleet Services - START")
		for domain, ip := range mappings {
			newLines = append(newLines, fmt.Sprintf("%s %s", ip, domain))
		}
		newLines = append(newLines, "# Fleet Services - END")
	}

	// Write back to hosts file
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(hostsFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}

	return nil
}

// removeDomainsFromHostsFile removes Fleet service domains from hosts file
func removeDomainsFromHostsFile() error {
	hostsFile := getHostsFilePath()
	
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}
	inFleetSection := false
	
	for _, line := range lines {
		if strings.Contains(line, "# Fleet Services - START") {
			inFleetSection = true
			continue
		}
		if strings.Contains(line, "# Fleet Services - END") {
			inFleetSection = false
			continue
		}
		if !inFleetSection {
			newLines = append(newLines, line)
		}
	}

	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(hostsFile, []byte(newContent), 0644)
}