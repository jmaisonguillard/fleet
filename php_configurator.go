package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PHPConfigurator manages PHP service configuration
type PHPConfigurator struct {
	// Framework detection
	frameworkDetectors map[string]FrameworkDetector
	
	// Nginx config generators
	nginxGenerators map[string]NginxGenerator
	
	// Xdebug configuration
	xdebugConfig XdebugConfig
	
	// PHP version management
	supportedVersions map[string]string
	defaultVersion    string
}

// FrameworkDetector detects if a framework is present in a folder
type FrameworkDetector func(folder string) bool

// NginxGenerator generates nginx configuration for a framework
type NginxGenerator func(phpServiceName string) string

// XdebugConfig holds Xdebug configuration settings
type XdebugConfig struct {
	DefaultPort int
	Mode        string
	Trigger     string
}

// NewPHPConfigurator creates a new PHP configurator
func NewPHPConfigurator() *PHPConfigurator {
	pc := &PHPConfigurator{
		frameworkDetectors: make(map[string]FrameworkDetector),
		nginxGenerators:    make(map[string]NginxGenerator),
		xdebugConfig: XdebugConfig{
			DefaultPort: 9003,
			Mode:        "develop,debug,coverage",
			Trigger:     "yes",
		},
		supportedVersions: map[string]string{
			"7.4":     "php:7.4-fpm-alpine",
			"8.0":     "php:8.0-fpm-alpine",
			"8.1":     "php:8.1-fpm-alpine",
			"8.2":     "php:8.2-fpm-alpine",
			"8.3":     "php:8.3-fpm-alpine",
			"8.4":     "php:8.4-fpm-alpine",
			"latest":  "php:8.4-fpm-alpine",
			"default": "php:8.4-fpm-alpine",
		},
		defaultVersion: "8.4",
	}
	
	// Register framework detectors
	pc.registerFrameworkDetectors()
	
	// Register nginx generators
	pc.registerNginxGenerators()
	
	return pc
}

// registerFrameworkDetectors registers all framework detection functions
func (pc *PHPConfigurator) registerFrameworkDetectors() {
	// Laravel
	pc.frameworkDetectors["laravel"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		artisanPath := filepath.Join(folder, "artisan")
		composerPath := filepath.Join(folder, "composer.json")
		
		if fileExists(artisanPath) && fileExists(composerPath) {
			if content, err := os.ReadFile(composerPath); err == nil {
				return strings.Contains(string(content), "laravel/framework")
			}
		}
		return false
	}
	
	// Lumen (check before Laravel since it also has artisan)
	pc.frameworkDetectors["lumen"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		artisanPath := filepath.Join(folder, "artisan")
		composerPath := filepath.Join(folder, "composer.json")
		
		if fileExists(artisanPath) && fileExists(composerPath) {
			if content, err := os.ReadFile(composerPath); err == nil {
				return strings.Contains(string(content), "laravel/lumen-framework")
			}
		}
		return false
	}
	
	// Symfony
	pc.frameworkDetectors["symfony"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		return fileExists(filepath.Join(folder, "symfony.lock")) ||
			fileExists(filepath.Join(folder, "bin/console"))
	}
	
	// WordPress
	pc.frameworkDetectors["wordpress"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		return fileExists(filepath.Join(folder, "wp-config.php")) ||
			fileExists(filepath.Join(folder, "wp-config-sample.php")) ||
			fileExists(filepath.Join(folder, "wp-load.php"))
	}
	
	// Drupal
	pc.frameworkDetectors["drupal"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		indexPath := filepath.Join(folder, "index.php")
		if fileExists(indexPath) {
			if content, err := os.ReadFile(indexPath); err == nil {
				return strings.Contains(string(content), "Drupal")
			}
		}
		return false
	}
	
	// CodeIgniter
	pc.frameworkDetectors["codeigniter"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		return fileExists(filepath.Join(folder, "system/core/CodeIgniter.php")) ||
			fileExists(filepath.Join(folder, "spark"))
	}
	
	// Slim
	pc.frameworkDetectors["slim"] = func(folder string) bool {
		if folder == "" {
			return false
		}
		composerPath := filepath.Join(folder, "composer.json")
		if fileExists(composerPath) {
			if content, err := os.ReadFile(composerPath); err == nil {
				return strings.Contains(string(content), "slim/slim")
			}
		}
		return false
	}
}

// registerNginxGenerators registers all nginx config generators
func (pc *PHPConfigurator) registerNginxGenerators() {
	// Use existing generators but wrap them in the new structure
	pc.nginxGenerators["laravel"] = generateLaravelNginxConfig
	pc.nginxGenerators["lumen"] = generateLaravelNginxConfig // Same as Laravel
	pc.nginxGenerators["symfony"] = generateSymfonyNginxConfig
	pc.nginxGenerators["wordpress"] = generateWordPressNginxConfig
	pc.nginxGenerators["drupal"] = generateDrupalNginxConfig
	pc.nginxGenerators["codeigniter"] = generateCodeIgniterNginxConfig
	pc.nginxGenerators["slim"] = generateSlimNginxConfig
	pc.nginxGenerators["default"] = generateNginxPHPConfigWithService
}

// DetectFramework detects the PHP framework in the given folder
func (pc *PHPConfigurator) DetectFramework(folder string) string {
	if folder == "" {
		return ""
	}
	
	// Check detectors in priority order
	frameworkOrder := []string{"lumen", "laravel", "symfony", "wordpress", "drupal", "codeigniter", "slim"}
	
	for _, framework := range frameworkOrder {
		if detector, exists := pc.frameworkDetectors[framework]; exists {
			if detector(folder) {
				return framework
			}
		}
	}
	
	return ""
}

// ParseRuntime parses the PHP runtime string and returns language and version
func (pc *PHPConfigurator) ParseRuntime(runtime string) (string, string) {
	if runtime == "" || !strings.HasPrefix(runtime, "php") {
		return "", ""
	}
	
	parts := strings.Split(runtime, ":")
	if len(parts) == 1 {
		// Just "php" - use default version
		return "php", pc.defaultVersion
	}
	
	// "php:8.2" format
	return parts[0], parts[1]
}

// GetPHPImage returns the appropriate PHP-FPM image for the version
func (pc *PHPConfigurator) GetPHPImage(version string) string {
	if version == "" {
		version = pc.defaultVersion
	}
	
	if image, ok := pc.supportedVersions[version]; ok {
		return image
	}
	
	// If specific version not found, try to construct it
	if matched, _ := regexp.MatchString(`^\d+\.\d+$`, version); matched {
		return fmt.Sprintf("php:%s-fpm-alpine", version)
	}
	
	// Fallback to default
	return pc.supportedVersions["default"]
}

// GenerateNginxConfig generates nginx configuration for a service
func (pc *PHPConfigurator) GenerateNginxConfig(serviceName, framework string) string {
	phpServiceName := fmt.Sprintf("%s-php", serviceName)
	
	// Use framework-specific generator if available
	if generator, exists := pc.nginxGenerators[strings.ToLower(framework)]; exists {
		return generator(phpServiceName)
	}
	
	// Fallback to default
	if defaultGenerator, exists := pc.nginxGenerators["default"]; exists {
		return defaultGenerator(phpServiceName)
	}
	
	// Ultimate fallback (shouldn't happen)
	return generateNginxPHPConfigWithService(phpServiceName)
}

// ConfigureXdebug returns Xdebug configuration for a PHP service
func (pc *PHPConfigurator) ConfigureXdebug(svc *Service) XdebugSettings {
	debugPort := pc.xdebugConfig.DefaultPort
	if svc.DebugPort > 0 {
		debugPort = svc.DebugPort
	}
	
	return XdebugSettings{
		Port:       debugPort,
		Mode:       pc.xdebugConfig.Mode,
		Trigger:    pc.xdebugConfig.Trigger,
		ClientHost: "host.docker.internal",
		ServerName: svc.Name,
		LogPath:    "/tmp/xdebug.log",
	}
}

// XdebugSettings holds processed Xdebug configuration
type XdebugSettings struct {
	Port       int
	Mode       string
	Trigger    string
	ClientHost string
	ServerName string
	LogPath    string
}

// ApplyToService applies Xdebug settings to a Docker service
func (xs *XdebugSettings) ApplyToService(phpService *DockerService) {
	// Xdebug 3.x environment variables
	phpService.Environment["XDEBUG_MODE"] = xs.Mode
	phpService.Environment["XDEBUG_CONFIG"] = fmt.Sprintf("client_host=%s client_port=%d", xs.ClientHost, xs.Port)
	phpService.Environment["XDEBUG_SESSION"] = "1"
	phpService.Environment["PHP_IDE_CONFIG"] = fmt.Sprintf("serverName=%s", xs.ServerName)
	phpService.Environment["XDEBUG_TRIGGER"] = xs.Trigger
	
	// Composer environment variables
	phpService.Environment["COMPOSER_ALLOW_SUPERUSER"] = "1"
	phpService.Environment["COMPOSER_HOME"] = "/var/www/.composer"
	
	// Install Xdebug command
	installCmd := xs.generateInstallCommand()
	phpService.Command = installCmd
	
	// Add host.docker.internal for Linux
	if phpService.ExtraHosts == nil {
		phpService.ExtraHosts = []string{}
	}
	phpService.ExtraHosts = append(phpService.ExtraHosts, "host.docker.internal:host-gateway")
}

// generateInstallCommand generates the Xdebug installation command
func (xs *XdebugSettings) generateInstallCommand() string {
	return fmt.Sprintf(`sh -c "
		# Install Composer if not present
		if ! command -v composer >/dev/null 2>&1; then
			echo 'Installing Composer...';
			curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer && \
			chmod +x /usr/local/bin/composer && \
			echo 'Composer installed successfully';
		fi;
		
		# Install Xdebug if not present
		if ! php -m | grep -q xdebug; then
			echo 'Installing Xdebug...';
			apk add --no-cache $PHPIZE_DEPS && \
			pecl install xdebug && \
			docker-php-ext-enable xdebug && \
			echo 'xdebug.mode=%s' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.client_host=%s' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.client_port=%d' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.start_with_request=%s' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini && \
			echo 'xdebug.log=%s' >> /usr/local/etc/php/conf.d/docker-php-ext-xdebug.ini;
		fi;
		
		php-fpm
	"`, xs.Mode, xs.ClientHost, xs.Port, xs.Trigger, xs.LogPath)
}

// BuildPHPService builds a complete PHP-FPM service configuration
func (pc *PHPConfigurator) BuildPHPService(svc *Service) *DockerService {
	lang, version := pc.ParseRuntime(svc.Runtime)
	if lang != "php" {
		return nil
	}
	
	phpImage := pc.GetPHPImage(version)
	
	// Create PHP-FPM service
	phpService := &DockerService{
		Image:    phpImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: map[string]string{
			"PHP_FPM_USER":  "www-data",
			"PHP_FPM_GROUP": "www-data",
		},
	}
	
	// Mount folder
	if svc.Folder != "" {
		phpService.Volumes = append(phpService.Volumes, fmt.Sprintf("../%s:/var/www/html", svc.Folder))
	}
	
	// Detect and configure framework
	framework := svc.Framework
	if framework == "" {
		framework = pc.DetectFramework(svc.Folder)
	}
	
	// Add framework-specific environment variables
	pc.configureFrameworkEnvironment(phpService, framework)
	
	// Configure Xdebug if enabled
	if svc.Debug {
		xdebugSettings := pc.ConfigureXdebug(svc)
		xdebugSettings.ApplyToService(phpService)
	} else {
		// Install Composer by default for all PHP containers
		pc.installComposer(phpService)
	}
	
	// Add custom environment variables
	if svc.Environment != nil {
		for k, v := range svc.Environment {
			phpService.Environment[k] = v
		}
	}
	
	// Add health check
	phpService.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", "php-fpm-healthcheck || exit 1"},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
	
	return phpService
}

// configureFrameworkEnvironment adds framework-specific environment variables
func (pc *PHPConfigurator) configureFrameworkEnvironment(phpService *DockerService, framework string) {
	switch strings.ToLower(framework) {
	case "laravel", "lumen":
		phpService.Environment["LARAVEL_ENV"] = "production"
		phpService.Environment["APP_ENV"] = "production"
	case "symfony":
		phpService.Environment["APP_ENV"] = "prod"
		phpService.Environment["APP_DEBUG"] = "0"
	case "wordpress":
		phpService.Environment["WP_ENV"] = "production"
	}
}

// WriteNginxConfig writes the nginx configuration file for a PHP service
func (pc *PHPConfigurator) WriteNginxConfig(serviceName, framework string) (string, error) {
	configPath := filepath.Join(".fleet", fmt.Sprintf("%s-nginx.conf", serviceName))
	
	// Get framework-specific config or fallback to generic
	if framework == "" {
		framework = "default"
	}
	
	config := pc.GenerateNginxConfig(serviceName, framework)
	
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", fmt.Errorf("failed to write nginx PHP config: %w", err)
	}
	
	return configPath, nil
}

// GetSupportedVersions returns all supported PHP versions
func (pc *PHPConfigurator) GetSupportedVersions() []string {
	versions := make([]string, 0, len(pc.supportedVersions))
	for version := range pc.supportedVersions {
		if version != "latest" && version != "default" {
			versions = append(versions, version)
		}
	}
	return versions
}

// GetSupportedFrameworks returns all supported PHP frameworks
func (pc *PHPConfigurator) GetSupportedFrameworks() []string {
	frameworks := make([]string, 0, len(pc.frameworkDetectors))
	for framework := range pc.frameworkDetectors {
		frameworks = append(frameworks, framework)
	}
	return frameworks
}

// installComposer configures the PHP service to install Composer
func (pc *PHPConfigurator) installComposer(phpService *DockerService) {
	// Command to install Composer and then start PHP-FPM
	installCmd := `sh -c "
		if ! command -v composer >/dev/null 2>&1; then
			echo 'Installing Composer...';
			curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer && \
			chmod +x /usr/local/bin/composer && \
			echo 'Composer installed successfully';
		fi;
		php-fpm
	"`
	
	phpService.Command = installCmd
	
	// Add Composer environment variables
	phpService.Environment["COMPOSER_ALLOW_SUPERUSER"] = "1"
	phpService.Environment["COMPOSER_HOME"] = "/var/www/.composer"
}