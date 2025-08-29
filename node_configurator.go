package main

import (
	"fmt"
	"strings"
)

// NodeConfigurator manages Node.js service configuration
type NodeConfigurator struct {
	// Framework detection
	frameworkDetectors map[string]FrameworkDetector
	
	// Supported versions
	supportedVersions map[string]string
	defaultVersion    string
}

// NewNodeConfigurator creates a new Node configurator
func NewNodeConfigurator() *NodeConfigurator {
	nc := &NodeConfigurator{
		frameworkDetectors: make(map[string]FrameworkDetector),
		supportedVersions: map[string]string{
			"16":       "node:16-alpine",
			"18":       "node:18-alpine",
			"20":       "node:20-alpine",
			"22":       "node:22-alpine",
			"latest":   "node:20-alpine",
			"lts":      "node:20-alpine",
			"default":  "node:20-alpine",
		},
		defaultVersion: "20",
	}
	
	// Register framework detectors
	nc.registerFrameworkDetectors()
	
	return nc
}

// registerFrameworkDetectors registers all framework detection functions
func (nc *NodeConfigurator) registerFrameworkDetectors() {
	// Express
	nc.frameworkDetectors["express"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && hasPackage(*pkg, "express")
	}
	
	// Next.js
	nc.frameworkDetectors["nextjs"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && hasPackage(*pkg, "next")
	}
	
	// Nuxt
	nc.frameworkDetectors["nuxt"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && (hasPackage(*pkg, "nuxt") || hasPackage(*pkg, "@nuxt/core"))
	}
	
	// Angular
	nc.frameworkDetectors["angular"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && hasPackage(*pkg, "@angular/core")
	}
	
	// React
	nc.frameworkDetectors["react"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && hasPackage(*pkg, "react") && !hasPackage(*pkg, "next")
	}
	
	// Vue
	nc.frameworkDetectors["vue"] = func(folder string) bool {
		pkg, _ := getPackageJSON(folder)
		return pkg != nil && hasPackage(*pkg, "vue") && !hasPackage(*pkg, "nuxt")
	}
}

// DetectFramework detects the Node.js framework in the given folder
func (nc *NodeConfigurator) DetectFramework(folder string) string {
	if folder == "" {
		return ""
	}
	
	// Use the detection function from node_frameworks.go
	return detectNodeFramework(folder)
}

// ParseRuntime parses the Node.js runtime string and returns language and version
func (nc *NodeConfigurator) ParseRuntime(runtime string) (string, string) {
	return parseNodeRuntime(runtime)
}

// GetNodeImage returns the appropriate Node.js image for the version
func (nc *NodeConfigurator) GetNodeImage(version string) string {
	return getNodeImage(version)
}

// BuildNodeService builds a complete Node.js service configuration
func (nc *NodeConfigurator) BuildNodeService(svc *Service) *DockerService {
	lang, version := nc.ParseRuntime(svc.Runtime)
	if lang != "node" {
		return nil
	}
	
	nodeImage := nc.GetNodeImage(version)
	
	// Determine if this is build mode or service mode
	isBuildMode := isNodeBuildMode(svc)
	
	// Create Node.js service
	nodeService := &DockerService{
		Image:    nodeImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: map[string]string{
			"NODE_ENV": nc.getNodeEnv(svc),
		},
	}
	
	// Set working directory
	workDir := "/app"
	nodeService.WorkingDir = workDir
	
	// Mount folder
	if svc.Folder != "" {
		nodeService.Volumes = append(nodeService.Volumes, fmt.Sprintf("../%s:%s", svc.Folder, workDir))
		
		// Add node_modules volume for better performance
		if !isBuildMode {
			volumeName := fmt.Sprintf("%s_node_modules", strings.ReplaceAll(svc.Name, "-", "_"))
			nodeService.Volumes = append(nodeService.Volumes, fmt.Sprintf("%s:%s/node_modules", volumeName, workDir))
		}
	}
	
	// Detect package manager
	packageManager := svc.PackageManager
	if packageManager == "" {
		packageManager = detectPackageManager(svc.Folder)
	}
	
	// Detect framework
	framework := svc.Framework
	if framework == "" {
		framework = nc.DetectFramework(svc.Folder)
	}
	
	// Configure based on mode
	if isBuildMode {
		// Build mode - one-time build container
		nc.configureBuildMode(nodeService, svc, packageManager, framework)
	} else {
		// Service mode - long-running container
		nc.configureServiceMode(nodeService, svc, packageManager, framework)
	}
	
	// Add custom environment variables
	if svc.Environment != nil {
		for k, v := range svc.Environment {
			nodeService.Environment[k] = v
		}
	}
	
	// Add custom volumes
	if len(svc.Volumes) > 0 {
		nodeService.Volumes = append(nodeService.Volumes, svc.Volumes...)
	}
	
	return nodeService
}

// configureBuildMode configures a Node.js container for build-only operations
func (nc *NodeConfigurator) configureBuildMode(nodeService *DockerService, svc *Service, packageManager string, framework string) {
	// Build containers don't restart
	nodeService.Restart = "no"
	
	// Determine build command
	buildCommand := svc.BuildCommand
	if buildCommand == "" {
		buildCommand = nc.getBuildCommand(svc.Folder, packageManager, framework)
	}
	
	// Create build script
	installCmd := nc.getInstallCommand(packageManager)
	buildScript := fmt.Sprintf(`sh -c "
		echo 'Installing dependencies with %s...';
		%s && \
		echo 'Building application...';
		%s && \
		echo 'Build completed successfully';
		if [ -d dist ]; then
			echo 'Copying dist folder...';
			cp -r dist/* /output/ 2>/dev/null || true;
		fi;
		if [ -d build ]; then
			echo 'Copying build folder...';
			cp -r build/* /output/ 2>/dev/null || true;
		fi;
	"`, packageManager, installCmd, buildCommand)
	
	nodeService.Command = buildScript
	
	// Add output volume for build artifacts
	nodeService.Volumes = append(nodeService.Volumes, "../.fleet/build-output:/output")
}

// configureServiceMode configures a Node.js container for long-running services
func (nc *NodeConfigurator) configureServiceMode(nodeService *DockerService, svc *Service, packageManager string, framework string) {
	// Determine start command
	startCommand := svc.Command
	if startCommand == "" {
		startCommand = nc.getStartCommand(svc.Folder, packageManager, framework, svc.NodeEnv == "development")
	}
	
	// Determine port
	port := getNodePort(svc)
	if port > 0 {
		nodeService.Environment["PORT"] = fmt.Sprintf("%d", port)
		
		// Only expose port if no domain (services with domains use nginx proxy)
		if svc.Domain == "" && svc.Port > 0 {
			nodeService.Ports = []string{fmt.Sprintf("%d:%d", svc.Port, port)}
		}
	}
	
	// Create startup script with dependency installation
	installCmd := nc.getInstallCommand(packageManager)
	startScript := fmt.Sprintf(`sh -c "
		echo 'Installing dependencies with %s...';
		%s && \
		echo 'Starting application...';
		%s
	"`, packageManager, installCmd, startCommand)
	
	nodeService.Command = startScript
	
	// Add health check for services
	nodeService.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD-SHELL", fmt.Sprintf("wget --no-verbose --tries=1 --spider http://localhost:%d/health || exit 1", port)},
		Interval: "30s",
		Timeout:  "5s",
		Retries:  3,
	}
}

// getNodeEnv returns the Node environment setting
func (nc *NodeConfigurator) getNodeEnv(svc *Service) string {
	if svc.NodeEnv != "" {
		return svc.NodeEnv
	}
	return "development"
}

// getInstallCommand returns the package installation command
func (nc *NodeConfigurator) getInstallCommand(packageManager string) string {
	switch packageManager {
	case "yarn":
		return "yarn install"
	case "pnpm":
		return "pnpm install"
	default:
		return "npm ci 2>/dev/null || npm install"
	}
}

// getBuildCommand returns the build command for the project
func (nc *NodeConfigurator) getBuildCommand(folder string, packageManager string, framework string) string {
	// First check package.json for build script
	if cmd := getBuildScriptFromPackageJSON(folder); cmd != "" {
		return nc.wrapWithPackageManager(cmd, packageManager)
	}
	
	// Use framework default
	switch packageManager {
	case "yarn":
		return "yarn build"
	case "pnpm":
		return "pnpm build"
	default:
		return "npm run build"
	}
}

// getStartCommand returns the start command for the project
func (nc *NodeConfigurator) getStartCommand(folder string, packageManager string, framework string, isDev bool) string {
	// First check package.json for start script
	if cmd := getStartScriptFromPackageJSON(folder); cmd != "" {
		return nc.wrapWithPackageManager(cmd, packageManager)
	}
	
	// Use framework-specific command
	return getFrameworkCommand(framework, packageManager, isDev)
}

// wrapWithPackageManager wraps a script command with the appropriate package manager
func (nc *NodeConfigurator) wrapWithPackageManager(cmd string, packageManager string) string {
	// If command already starts with npm/yarn/pnpm, return as-is
	if strings.HasPrefix(cmd, "npm ") || strings.HasPrefix(cmd, "yarn ") || strings.HasPrefix(cmd, "pnpm ") {
		return cmd
	}
	
	// If it's a node command, return as-is
	if strings.HasPrefix(cmd, "node ") {
		return cmd
	}
	
	// Otherwise, wrap with package manager run command
	switch packageManager {
	case "yarn":
		return "yarn " + cmd
	case "pnpm":
		return "pnpm " + cmd
	default:
		return "npm run " + cmd
	}
}

// GetSupportedVersions returns all supported Node.js versions
func (nc *NodeConfigurator) GetSupportedVersions() []string {
	versions := make([]string, 0, len(nc.supportedVersions))
	for version := range nc.supportedVersions {
		if version != "latest" && version != "default" && version != "lts" {
			versions = append(versions, version)
		}
	}
	return versions
}

// GetSupportedFrameworks returns all supported Node.js frameworks
func (nc *NodeConfigurator) GetSupportedFrameworks() []string {
	return SupportedNodeFrameworks
}