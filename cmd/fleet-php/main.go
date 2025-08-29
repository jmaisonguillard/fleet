package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const version = "1.0.0"

// Config represents fleet configuration (minimal subset needed)
type Config struct {
	Project  string    `toml:"project" yaml:"project" json:"project"`
	Services []Service `toml:"services" yaml:"services" json:"services"`
}

// Service represents a service configuration (minimal subset)
type Service struct {
	Name      string `toml:"name" yaml:"name" json:"name"`
	Runtime   string `toml:"runtime" yaml:"runtime" json:"runtime"`
	Framework string `toml:"framework" yaml:"framework" json:"framework"`
	Folder    string `toml:"folder" yaml:"folder" json:"folder"`
}

// PHPService represents a detected PHP service
type PHPService struct {
	Name          string
	ContainerName string
	Framework     string
	Folder        string
}

func main() {
	// Parse flags
	serviceFlag := flag.String("service", "", "Specify which service to use")
	versionFlag := flag.Bool("version", false, "Show version")
	helpFlag := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("fleet-php v%s\n", version)
		os.Exit(0)
	}

	if *helpFlag || flag.NArg() == 0 {
		printUsage()
		os.Exit(0)
	}

	// Get command and args
	command := flag.Arg(0)
	args := flag.Args()[1:]

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading fleet configuration: %v\n", err)
		os.Exit(1)
	}

	// Find PHP services
	phpServices := detectPHPServices(config)
	if len(phpServices) == 0 {
		fmt.Fprintf(os.Stderr, "No PHP services found in fleet configuration\n")
		os.Exit(1)
	}

	// Select service
	var selectedService *PHPService
	if *serviceFlag != "" {
		for _, svc := range phpServices {
			if svc.Name == *serviceFlag {
				selectedService = &svc
				break
			}
		}
		if selectedService == nil {
			fmt.Fprintf(os.Stderr, "Service '%s' not found or is not a PHP service\n", *serviceFlag)
			os.Exit(1)
		}
	} else {
		selectedService = &phpServices[0]
		if len(phpServices) > 1 {
			fmt.Printf("Multiple PHP services found. Using '%s'. Use --service flag to specify.\n", selectedService.Name)
		}
	}

	// Execute command
	switch command {
	case "composer":
		executeComposer(selectedService, args)
	case "php":
		executePHP(selectedService, args)
	case "artisan":
		if !isLaravelService(selectedService) {
			fmt.Fprintf(os.Stderr, "artisan command is only available for Laravel/Lumen projects\n")
			os.Exit(1)
		}
		executeArtisan(selectedService, args)
	case "console":
		if !isSymfonyService(selectedService) {
			fmt.Fprintf(os.Stderr, "console command is only available for Symfony projects\n")
			os.Exit(1)
		}
		executeConsole(selectedService, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("fleet-php - PHP CLI tool for Fleet")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Println("Usage: fleet-php [--service=<name>] <command> [args...]")
	fmt.Println("\nCommands:")
	fmt.Println("  composer [args...]   Run Composer commands")
	fmt.Println("  php [args...]        Run PHP scripts")
	fmt.Println("  artisan [args...]    Run Laravel Artisan commands (Laravel/Lumen only)")
	fmt.Println("  console [args...]    Run Symfony Console commands (Symfony only)")
	fmt.Println("\nFlags:")
	fmt.Println("  --service=<name>     Specify which service to use (for multi-service projects)")
	fmt.Println("  --version            Show version")
	fmt.Println("  --help               Show this help")
	fmt.Println("\nExamples:")
	fmt.Println("  fleet-php composer install")
	fmt.Println("  fleet-php composer require laravel/sanctum")
	fmt.Println("  fleet-php php -v")
	fmt.Println("  fleet-php artisan migrate")
	fmt.Println("  fleet-php --service=api composer update")
}

func loadConfig() (*Config, error) {
	// Try different config file formats
	configFiles := []string{"fleet.toml", "fleet.yaml", "fleet.yml", "fleet.json"}
	
	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			return loadConfigFile(file)
		}
	}
	
	return nil, fmt.Errorf("no fleet configuration file found")
}

func loadConfigFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	
	// Handle different config formats
	switch {
	case strings.HasSuffix(filename, ".toml"):
		if err := toml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse TOML: %w", err)
		}
	case strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml"):
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case strings.HasSuffix(filename, ".json"):
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", filename)
	}
	
	return config, nil
}

func detectPHPServices(config *Config) []PHPService {
	var services []PHPService
	
	// Docker Compose uses "fleet" as the default project name for Fleet
	// regardless of the config project name
	projectName := "fleet"
	
	for _, svc := range config.Services {
		if strings.HasPrefix(svc.Runtime, "php") {
			// Container naming follows pattern: fleet-{service}-1
			// PHP services don't have "-php" suffix in the container name
			phpSvc := PHPService{
				Name:          svc.Name,
				ContainerName: fmt.Sprintf("%s-%s-1", projectName, svc.Name),
				Framework:     svc.Framework,
				Folder:        svc.Folder,
			}
			
			// Auto-detect framework if not specified
			if phpSvc.Framework == "" && phpSvc.Folder != "" {
				phpSvc.Framework = detectFramework(phpSvc.Folder)
			}
			
			services = append(services, phpSvc)
		}
	}
	
	return services
}

func detectFramework(folder string) string {
	// Check for Laravel/Lumen
	artisanPath := filepath.Join(folder, "artisan")
	if _, err := os.Stat(artisanPath); err == nil {
		composerPath := filepath.Join(folder, "composer.json")
		if data, err := os.ReadFile(composerPath); err == nil {
			if strings.Contains(string(data), "laravel/lumen-framework") {
				return "lumen"
			}
			if strings.Contains(string(data), "laravel/framework") {
				return "laravel"
			}
		}
	}
	
	// Check for Symfony
	consolePath := filepath.Join(folder, "bin", "console")
	if _, err := os.Stat(consolePath); err == nil {
		return "symfony"
	}
	
	return ""
}

func isLaravelService(service *PHPService) bool {
	return service.Framework == "laravel" || service.Framework == "lumen"
}

func isSymfonyService(service *PHPService) bool {
	return service.Framework == "symfony"
}

func executeComposer(service *PHPService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/var/www/html",
	}
	
	// Add TTY if available and not just checking version/help
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "composer")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

// isInfoCommand checks if the command is just for information (doesn't need TTY)
func isInfoCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	infoCommands := []string{"--version", "-V", "-v", "--help", "-h", "list", "about", "-i", "--info"}
	for _, cmd := range infoCommands {
		if args[0] == cmd {
			return true
		}
	}
	return false
}

func executePHP(service *PHPService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/var/www/html",
	}
	
	// Add TTY if available and not just checking version/info
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "php")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executeArtisan(service *PHPService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/var/www/html",
	}
	
	// Add TTY if available
	if isTerminal() {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "php", "artisan")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executeConsole(service *PHPService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/var/www/html",
	}
	
	// Add TTY if available
	if isTerminal() {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "php", "bin/console")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func runDockerCommand(args []string) {
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running docker command: %v\n", err)
		os.Exit(1)
	}
}

func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}