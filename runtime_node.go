package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// NodeVersion represents a Node.js version configuration
type NodeVersion struct {
	Version string
	Image   string
}

// Supported Node.js LTS versions with their Docker images
var supportedNodeVersions = map[string]string{
	"16":       "node:16-alpine",
	"16-alpine": "node:16-alpine",
	"18":       "node:18-alpine",
	"18-alpine": "node:18-alpine",
	"20":       "node:20-alpine",
	"20-alpine": "node:20-alpine",
	"22":       "node:22-alpine",
	"22-alpine": "node:22-alpine",
	"latest":   "node:20-alpine", // Default to Node.js 20 LTS
	"lts":      "node:20-alpine", // LTS alias
	"default":  "node:20-alpine", // Default to Node.js 20 LTS
}

// parseNodeRuntime parses the runtime string and returns language and version
// Examples: "node", "node:20", "node:18-alpine"
func parseNodeRuntime(runtime string) (string, string) {
	if runtime == "" || !strings.HasPrefix(runtime, "node") {
		return "", ""
	}

	parts := strings.Split(runtime, ":")
	if len(parts) == 1 {
		// Just "node" - use default version
		return "node", "20"
	}

	// "node:20" or "node:18-alpine" format
	return parts[0], parts[1]
}

// getNodeImage returns the appropriate Node.js Docker image for the version
func getNodeImage(version string) string {
	if version == "" {
		version = "20"
	}

	if image, ok := supportedNodeVersions[version]; ok {
		return image
	}

	// If specific version not found, try to construct it
	if matched, _ := regexp.MatchString(`^\d+$`, version); matched {
		return fmt.Sprintf("node:%s-alpine", version)
	}

	// Check if it's a full version like "20.11.0"
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, version); matched {
		return fmt.Sprintf("node:%s-alpine", version)
	}

	// Fallback to default
	return supportedNodeVersions["default"]
}

// detectPackageManager detects the package manager from lock files
func detectPackageManager(folder string) string {
	if folder == "" {
		return "npm"
	}

	// Check for lock files in order of preference
	if fileExists(filepath.Join(folder, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if fileExists(filepath.Join(folder, "yarn.lock")) {
		return "yarn"
	}
	if fileExists(filepath.Join(folder, "package-lock.json")) {
		return "npm"
	}

	// Default to npm
	return "npm"
}

// addNodeService adds a Node.js service to the Docker Compose configuration
func addNodeService(compose *DockerCompose, svc *Service, config *Config) {
	// Use the NodeConfigurator to build the Node service
	configurator := NewNodeConfigurator()
	nodeService := configurator.BuildNodeService(svc)
	
	if nodeService == nil {
		// Not a Node service
		return
	}
	
	// For standalone Node.js services, use the service name directly
	// For Node.js with nginx, create a separate container with -node suffix
	var nodeServiceName string
	if strings.Contains(strings.ToLower(svc.Image), "nginx") {
		nodeServiceName = fmt.Sprintf("%s-node", svc.Name)
	} else {
		nodeServiceName = svc.Name
	}
	
	// Add the Node service to compose
	compose.Services[nodeServiceName] = *nodeService

	// If this is nginx with Node.js runtime (build mode), update nginx to depend on build
	if strings.Contains(strings.ToLower(svc.Image), "nginx") {
		if nginxSvc, exists := compose.Services[svc.Name]; exists {
			if nginxSvc.DependsOn == nil {
				nginxSvc.DependsOn = []string{}
			}
			nginxSvc.DependsOn = append(nginxSvc.DependsOn, nodeServiceName)
			compose.Services[svc.Name] = nginxSvc
		}
	}
}

// getNodeStartCommand attempts to detect the start command from package.json
func getNodeStartCommand(folder string) string {
	if folder == "" {
		return "node index.js"
	}

	packageJsonPath := filepath.Join(folder, "package.json")
	if !fileExists(packageJsonPath) {
		// Try common entry points
		if fileExists(filepath.Join(folder, "server.js")) {
			return "node server.js"
		}
		if fileExists(filepath.Join(folder, "app.js")) {
			return "node app.js"
		}
		if fileExists(filepath.Join(folder, "index.js")) {
			return "node index.js"
		}
		return "node index.js"
	}

	// TODO: Parse package.json to get start script
	// For now, use package manager default
	pm := detectPackageManager(folder)
	switch pm {
	case "yarn":
		return "yarn start"
	case "pnpm":
		return "pnpm start"
	default:
		return "npm start"
	}
}

// getNodeBuildCommand attempts to detect the build command from package.json
func getNodeBuildCommand(folder string) string {
	if folder == "" {
		return ""
	}

	// TODO: Parse package.json to check if build script exists
	// For now, use package manager default
	pm := detectPackageManager(folder)
	switch pm {
	case "yarn":
		return "yarn build"
	case "pnpm":
		return "pnpm build"
	default:
		return "npm run build"
	}
}

// getNodePort attempts to detect the port from common environment variables or defaults
func getNodePort(svc *Service) int {
	// If port is explicitly configured, use it
	if svc.Port > 0 {
		return svc.Port
	}

	// Check common Node.js frameworks default ports
	framework := detectNodeFramework(svc.Folder)
	switch framework {
	case "nextjs":
		return 3000
	case "nuxt":
		return 3000
	case "express":
		return 3000
	case "angular":
		return 4200
	case "react":
		return 3000
	case "vue":
		return 8080
	default:
		return 3000
	}
}

// isNodeBuildMode determines if this is a build-only container
func isNodeBuildMode(svc *Service) bool {
	// If service has an image (like nginx) and Node.js runtime, it's build mode
	if svc.Image != "" && strings.HasPrefix(svc.Runtime, "node") {
		return true
	}
	
	// If service has explicit build command, it's build mode
	if svc.BuildCommand != "" {
		return true
	}
	
	return false
}