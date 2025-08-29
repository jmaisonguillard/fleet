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
	Name           string `toml:"name" yaml:"name" json:"name"`
	Runtime        string `toml:"runtime" yaml:"runtime" json:"runtime"`
	Framework      string `toml:"framework" yaml:"framework" json:"framework"`
	Folder         string `toml:"folder" yaml:"folder" json:"folder"`
	Image          string `toml:"image" yaml:"image" json:"image"`
	BuildCommand   string `toml:"build_command" yaml:"build_command" json:"build_command"`
	PackageManager string `toml:"package_manager" yaml:"package_manager" json:"package_manager"`
}

// NodeService represents a detected Node.js service
type NodeService struct {
	Name           string
	ContainerName  string
	Framework      string
	Folder         string
	PackageManager string
}

func main() {
	// Parse flags
	serviceFlag := flag.String("service", "", "Specify which service to use")
	versionFlag := flag.Bool("version", false, "Show version")
	helpFlag := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("fleet-node v%s\n", version)
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

	// Find Node.js services
	nodeServices := detectNodeServices(config)
	if len(nodeServices) == 0 {
		fmt.Fprintf(os.Stderr, "No Node.js services found in fleet configuration\n")
		os.Exit(1)
	}

	// Select service
	var selectedService *NodeService
	if *serviceFlag != "" {
		for _, svc := range nodeServices {
			if svc.Name == *serviceFlag {
				selectedService = &svc
				break
			}
		}
		if selectedService == nil {
			fmt.Fprintf(os.Stderr, "Service '%s' not found or is not a Node.js service\n", *serviceFlag)
			os.Exit(1)
		}
	} else {
		selectedService = &nodeServices[0]
		if len(nodeServices) > 1 {
			fmt.Printf("Multiple Node.js services found. Using '%s'. Use --service flag to specify.\n", selectedService.Name)
		}
	}

	// Execute command
	switch command {
	case "npm":
		executeNPM(selectedService, args)
	case "yarn":
		executeYarn(selectedService, args)
	case "pnpm":
		executePNPM(selectedService, args)
	case "node":
		executeNode(selectedService, args)
	case "npx":
		executeNPX(selectedService, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("fleet-node - Node.js CLI tool for Fleet")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Println("Usage: fleet-node [--service=<name>] <command> [args...]")
	fmt.Println("\nCommands:")
	fmt.Println("  npm [args...]        Run npm commands")
	fmt.Println("  yarn [args...]       Run yarn commands")
	fmt.Println("  pnpm [args...]       Run pnpm commands")
	fmt.Println("  node [args...]       Run Node.js scripts")
	fmt.Println("  npx [args...]        Run npx commands")
	fmt.Println("\nFlags:")
	fmt.Println("  --service=<name>     Specify which service to use (for multi-service projects)")
	fmt.Println("  --version            Show version")
	fmt.Println("  --help               Show this help")
	fmt.Println("\nExamples:")
	fmt.Println("  fleet-node npm install")
	fmt.Println("  fleet-node npm run build")
	fmt.Println("  fleet-node yarn add express")
	fmt.Println("  fleet-node node -v")
	fmt.Println("  fleet-node npx create-react-app my-app")
	fmt.Println("  fleet-node --service=api npm start")
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

func detectNodeServices(config *Config) []NodeService {
	var services []NodeService
	
	// Docker Compose uses "fleet" as the default project name
	projectName := "fleet"
	
	for _, svc := range config.Services {
		if strings.HasPrefix(svc.Runtime, "node") {
			// Container naming follows pattern: fleet-{service}-1
			nodeSvc := NodeService{
				Name:          svc.Name,
				ContainerName: fmt.Sprintf("%s-%s-1", projectName, svc.Name),
				Framework:     svc.Framework,
				Folder:        svc.Folder,
				PackageManager: svc.PackageManager,
			}
			
			// Auto-detect package manager if not specified
			if nodeSvc.PackageManager == "" && nodeSvc.Folder != "" {
				nodeSvc.PackageManager = detectPackageManager(nodeSvc.Folder)
			}
			
			// Auto-detect framework if not specified
			if nodeSvc.Framework == "" && nodeSvc.Folder != "" {
				nodeSvc.Framework = detectFramework(nodeSvc.Folder)
			}
			
			services = append(services, nodeSvc)
		}
	}
	
	return services
}

func detectPackageManager(folder string) string {
	// Check for lock files
	if _, err := os.Stat(filepath.Join(folder, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(folder, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(folder, "package-lock.json")); err == nil {
		return "npm"
	}
	return "npm" // Default
}

func detectFramework(folder string) string {
	// Read package.json to detect framework
	packagePath := filepath.Join(folder, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return ""
	}
	
	content := string(data)
	
	// Check for common frameworks
	if strings.Contains(content, "\"next\"") {
		return "nextjs"
	}
	if strings.Contains(content, "\"nuxt\"") {
		return "nuxt"
	}
	if strings.Contains(content, "\"@angular/core\"") {
		return "angular"
	}
	if strings.Contains(content, "\"express\"") {
		return "express"
	}
	if strings.Contains(content, "\"react\"") {
		return "react"
	}
	if strings.Contains(content, "\"vue\"") {
		return "vue"
	}
	
	return ""
}

func executeNPM(service *NodeService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/app",
	}
	
	// Add TTY if available and not just checking version/help
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "npm")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executeYarn(service *NodeService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/app",
	}
	
	// Add TTY if available
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "yarn")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executePNPM(service *NodeService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/app",
	}
	
	// Add TTY if available
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "pnpm")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executeNode(service *NodeService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/app",
	}
	
	// Add TTY if available
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "node")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

func executeNPX(service *NodeService, args []string) {
	dockerArgs := []string{
		"exec",
		"-w", "/app",
	}
	
	// Add TTY if available
	if isTerminal() && !isInfoCommand(args) {
		dockerArgs = append(dockerArgs, "-it")
	}
	
	dockerArgs = append(dockerArgs, service.ContainerName, "npx")
	dockerArgs = append(dockerArgs, args...)
	
	runDockerCommand(dockerArgs)
}

// isInfoCommand checks if the command is just for information (doesn't need TTY)
func isInfoCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	infoCommands := []string{"--version", "-V", "-v", "--help", "-h", "list", "about"}
	for _, cmd := range infoCommands {
		if args[0] == cmd {
			return true
		}
	}
	return false
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