package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PHPVersion represents a PHP version configuration
type PHPVersion struct {
	Version string
	Image   string
}

// Supported PHP versions with their FPM images
var supportedPHPVersions = map[string]string{
	"7.4":     "php:7.4-fpm-alpine",
	"8.0":     "php:8.0-fpm-alpine",
	"8.1":     "php:8.1-fpm-alpine",
	"8.2":     "php:8.2-fpm-alpine",
	"8.3":     "php:8.3-fpm-alpine",
	"8.4":     "php:8.4-fpm-alpine",
	"latest":  "php:8.4-fpm-alpine", // Default to 8.4
	"default": "php:8.4-fpm-alpine", // Default to 8.4
}

// parsePHPRuntime parses the runtime string and returns version
// Examples: "php", "php:8.2", "php:7.4"
func parsePHPRuntime(runtime string) (string, string) {
	if runtime == "" || !strings.HasPrefix(runtime, "php") {
		return "", ""
	}

	parts := strings.Split(runtime, ":")
	if len(parts) == 1 {
		// Just "php" - use default version
		return "php", "8.4"
	}

	// "php:8.2" format
	return parts[0], parts[1]
}

// getPHPImage returns the appropriate PHP-FPM image for the version
func getPHPImage(version string) string {
	if version == "" {
		version = "8.4"
	}

	if image, ok := supportedPHPVersions[version]; ok {
		return image
	}

	// If specific version not found, try to construct it
	if matched, _ := regexp.MatchString(`^\d+\.\d+$`, version); matched {
		return fmt.Sprintf("php:%s-fpm-alpine", version)
	}

	// Fallback to default
	return supportedPHPVersions["default"]
}

// addPHPFPMService adds a PHP-FPM service for nginx services with PHP runtime
func addPHPFPMService(compose *DockerCompose, svc *Service, config *Config) {
	lang, version := parsePHPRuntime(svc.Runtime)
	if lang != "php" {
		return
	}

	phpServiceName := fmt.Sprintf("%s-php", svc.Name)
	phpImage := getPHPImage(version)

	// Create PHP-FPM service
	phpService := DockerService{
		Image:    phpImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: map[string]string{
			"PHP_FPM_USER":  "www-data",
			"PHP_FPM_GROUP": "www-data",
		},
	}

	// Mount the same folder as the nginx service
	if svc.Folder != "" {
		// PHP files need to be in the same location as nginx expects
		phpService.Volumes = append(phpService.Volumes, fmt.Sprintf("../%s:/var/www/html", svc.Folder))
	}

	// Add any custom environment variables
	if svc.Environment != nil {
		for k, v := range svc.Environment {
			phpService.Environment[k] = v
		}
	}

	// Add health check for PHP-FPM
	phpService.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", "php-fpm-healthcheck || exit 1"},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}

	// Add the PHP service to compose
	compose.Services[phpServiceName] = phpService

	// Update the nginx service to depend on PHP-FPM
	if nginxSvc, exists := compose.Services[svc.Name]; exists {
		if nginxSvc.DependsOn == nil {
			nginxSvc.DependsOn = []string{}
		}
		nginxSvc.DependsOn = append(nginxSvc.DependsOn, phpServiceName)
		compose.Services[svc.Name] = nginxSvc
	}
}

// generateNginxPHPConfig generates nginx configuration for PHP-FPM
func generateNginxPHPConfig(serviceName string) string {
	phpServiceName := fmt.Sprintf("%s-php", serviceName)
	
	return fmt.Sprintf(`server {
    listen 80;
    server_name _;
    
    root /var/www/html;
    index index.php index.html index.htm;
    
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        
        # Performance tweaks
        fastcgi_buffer_size 128k;
        fastcgi_buffers 256 16k;
        fastcgi_busy_buffers_size 256k;
    }
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # Cache static files
    location ~* \.(jpg|jpeg|gif|png|css|js|ico|xml)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
    
    # Deny access to hidden files
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }
}`, phpServiceName)
}

// writeNginxPHPConfig writes the nginx configuration for PHP
func writeNginxPHPConfig(serviceName string) (string, error) {
	configPath := filepath.Join(".fleet", fmt.Sprintf("%s-nginx.conf", serviceName))
	config := generateNginxPHPConfig(serviceName)
	
	if err := writeFile(configPath, []byte(config), 0644); err != nil {
		return "", fmt.Errorf("failed to write nginx PHP config: %w", err)
	}
	
	return configPath, nil
}

// Helper function to write file
func writeFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}