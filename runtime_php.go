package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

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
	// Use the PHPConfigurator to build the PHP service
	configurator := NewPHPConfigurator()
	phpService := configurator.BuildPHPService(svc)
	
	if phpService == nil {
		// Not a PHP service
		return
	}
	
	phpServiceName := fmt.Sprintf("%s-php", svc.Name)
	
	// Add the PHP service to compose
	compose.Services[phpServiceName] = *phpService

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
	// For backward compatibility, we still use service-specific PHP name
	// This will be updated when calling from compose.go with version info
	phpServiceName := fmt.Sprintf("%s-php", serviceName)
	
	return generateNginxPHPConfigWithService(phpServiceName)
}

// generateNginxPHPConfigWithVersion generates nginx config for specific PHP version
func generateNginxPHPConfigWithVersion(serviceName, phpVersion string) string {
	// Using per-service PHP containers for now
	phpServiceName := fmt.Sprintf("%s-php", serviceName)
	return generateNginxPHPConfigWithService(phpServiceName)
}

// generateNginxPHPConfigWithService generates nginx config with specific PHP service
func generateNginxPHPConfigWithService(phpServiceName string) string {
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
func writeNginxPHPConfig(serviceName string, framework string) (string, error) {
	// Use the PHPConfigurator to write nginx config
	configurator := NewPHPConfigurator()
	return configurator.WriteNginxConfig(serviceName, framework)
}

// writeNginxPHPConfigWithVersion writes nginx config with specific PHP version
func writeNginxPHPConfigWithVersion(serviceName, framework, _ string) (string, error) {
	// For compatibility, just use the standard method
	// The version is already handled in the runtime configuration
	return writeNginxPHPConfig(serviceName, framework)
}

// Helper function to write file
func writeFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// configureXdebug adds Xdebug configuration to PHP service
func configureXdebug(phpService *DockerService, svc *Service) {
	// Default Xdebug port
	debugPort := 9003
	if svc.DebugPort > 0 {
		debugPort = svc.DebugPort
	}
	
	// Use a PHP image with Xdebug pre-installed or add installation command
	// For simplicity, we'll use environment variables to configure Xdebug
	// In production, you might want to use a custom Dockerfile
	
	// Xdebug 3.x configuration (modern version)
	phpService.Environment["XDEBUG_MODE"] = "develop,debug,coverage"
	phpService.Environment["XDEBUG_CONFIG"] = fmt.Sprintf("client_host=host.docker.internal client_port=%d", debugPort)
	phpService.Environment["XDEBUG_SESSION"] = "1"
	
	// Additional Xdebug environment variables
	phpService.Environment["PHP_IDE_CONFIG"] = fmt.Sprintf("serverName=%s", svc.Name)
	phpService.Environment["XDEBUG_TRIGGER"] = "yes"
	
	// Install Xdebug via command if not present
	// This modifies the command to install Xdebug before starting PHP-FPM
	installCmd := `sh -c "
		if ! php -m | grep -q xdebug; then
			echo 'Installing Xdebug...';
			apk add --no-cache $PHPIZE_DEPS && \
			pecl install xdebug && \
			docker-php-ext-enable xdebug && \
			echo 'xdebug.mode=develop,debug,coverage' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.client_host=host.docker.internal' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.client_port=%d' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.start_with_request=yes' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.log=/tmp/xdebug.log' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini;
		fi;
		php-fpm
	"`
	
	phpService.Command = fmt.Sprintf(installCmd, debugPort)
	
	// Add host.docker.internal for Linux (it's automatic on Mac/Windows)
	if phpService.ExtraHosts == nil {
		phpService.ExtraHosts = []string{}
	}
	phpService.ExtraHosts = append(phpService.ExtraHosts, "host.docker.internal:host-gateway")
}