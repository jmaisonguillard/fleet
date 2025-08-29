package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// SupportedNodeFrameworks lists all supported Node.js frameworks
var SupportedNodeFrameworks = []string{
	"express",
	"nextjs",
	"nuxt",
	"react",
	"vue",
	"angular",
	"fastify",
	"nestjs",
	"svelte",
	"remix",
}

// PackageJSON represents the structure of package.json
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Main            string            `json:"main"`
	Type            string            `json:"type"`
}

// detectNodeFramework auto-detects the Node.js framework from project files
func detectNodeFramework(folder string) string {
	if folder == "" {
		return ""
	}

	packageJsonPath := filepath.Join(folder, "package.json")
	if !fileExists(packageJsonPath) {
		return ""
	}

	// Read and parse package.json
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return ""
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// Check dependencies for framework detection
	// Priority order matters - check more specific frameworks first
	
	// Next.js
	if hasPackage(pkg, "next") {
		return "nextjs"
	}
	
	// Nuxt
	if hasPackage(pkg, "nuxt") || hasPackage(pkg, "@nuxt/core") {
		return "nuxt"
	}
	
	// Remix
	if hasPackage(pkg, "@remix-run/node") || hasPackage(pkg, "@remix-run/serve") {
		return "remix"
	}
	
	// Angular
	if hasPackage(pkg, "@angular/core") {
		return "angular"
	}
	
	// NestJS
	if hasPackage(pkg, "@nestjs/core") {
		return "nestjs"
	}
	
	// Express
	if hasPackage(pkg, "express") {
		return "express"
	}
	
	// Fastify
	if hasPackage(pkg, "fastify") {
		return "fastify"
	}
	
	// Svelte/SvelteKit
	if hasPackage(pkg, "@sveltejs/kit") || hasPackage(pkg, "svelte") {
		return "svelte"
	}
	
	// Vue (without Nuxt)
	if hasPackage(pkg, "vue") || hasPackage(pkg, "@vue/cli-service") {
		return "vue"
	}
	
	// React (without Next.js)
	if hasPackage(pkg, "react") {
		return "react"
	}

	return ""
}

// hasPackage checks if a package exists in dependencies or devDependencies
func hasPackage(pkg PackageJSON, packageName string) bool {
	if _, ok := pkg.Dependencies[packageName]; ok {
		return true
	}
	if _, ok := pkg.DevDependencies[packageName]; ok {
		return true
	}
	return false
}

// getPackageJSON reads and parses package.json from a folder
func getPackageJSON(folder string) (*PackageJSON, error) {
	if folder == "" {
		return nil, nil
	}

	packageJsonPath := filepath.Join(folder, "package.json")
	if !fileExists(packageJsonPath) {
		return nil, nil
	}

	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

// getStartScriptFromPackageJSON returns the start script from package.json
func getStartScriptFromPackageJSON(folder string) string {
	pkg, err := getPackageJSON(folder)
	if err != nil || pkg == nil {
		return ""
	}

	// Check for start script
	if startScript, ok := pkg.Scripts["start"]; ok {
		return startScript
	}

	// Check for dev script (common in development)
	if devScript, ok := pkg.Scripts["dev"]; ok {
		return devScript
	}

	// Check for serve script
	if serveScript, ok := pkg.Scripts["serve"]; ok {
		return serveScript
	}

	// Check main field
	if pkg.Main != "" {
		return "node " + pkg.Main
	}

	return ""
}

// getBuildScriptFromPackageJSON returns the build script from package.json
func getBuildScriptFromPackageJSON(folder string) string {
	pkg, err := getPackageJSON(folder)
	if err != nil || pkg == nil {
		return ""
	}

	// Check for build script
	if buildScript, ok := pkg.Scripts["build"]; ok {
		return buildScript
	}

	// Check for compile script
	if compileScript, ok := pkg.Scripts["compile"]; ok {
		return compileScript
	}

	return ""
}

// getPortFromPackageJSON attempts to detect port from package.json scripts
func getPortFromPackageJSON(folder string) int {
	pkg, err := getPackageJSON(folder)
	if err != nil || pkg == nil {
		return 0
	}

	// Check start script for port references
	if startScript, ok := pkg.Scripts["start"]; ok {
		// Look for PORT= or --port patterns
		if strings.Contains(startScript, "PORT=") {
			// Extract port number (basic implementation)
			parts := strings.Split(startScript, "PORT=")
			if len(parts) > 1 {
				portStr := strings.Fields(parts[1])[0]
				// TODO: Parse port number
				_ = portStr
			}
		}
	}

	return 0
}

// getFrameworkCommand returns the appropriate command for a framework
func getFrameworkCommand(framework string, pm string, isDev bool) string {
	runCmd := "npm run"
	if pm == "yarn" {
		runCmd = "yarn"
	} else if pm == "pnpm" {
		runCmd = "pnpm"
	}

	switch framework {
	case "nextjs":
		if isDev {
			return runCmd + " dev"
		}
		return runCmd + " start"
	case "nuxt":
		if isDev {
			return runCmd + " dev"
		}
		return runCmd + " start"
	case "angular":
		if isDev {
			return runCmd + " start"
		}
		return runCmd + " serve"
	case "react":
		if isDev {
			return runCmd + " start"
		}
		return runCmd + " serve"
	case "vue":
		if isDev {
			return runCmd + " serve"
		}
		return runCmd + " preview"
	case "svelte":
		if isDev {
			return runCmd + " dev"
		}
		return runCmd + " preview"
	case "express", "fastify", "nestjs":
		if isDev {
			return runCmd + " dev"
		}
		return runCmd + " start"
	default:
		if isDev {
			return runCmd + " dev"
		}
		return runCmd + " start"
	}
}

// getFrameworkBuildOutput returns the build output directory for a framework
func getFrameworkBuildOutput(framework string, folder string) string {
	// Check package.json for custom build output
	// TODO: Parse build configuration from package.json
	
	// Default build output directories
	switch framework {
	case "nextjs":
		return ".next"
	case "nuxt":
		return ".nuxt"
	case "angular":
		return "dist"
	case "react":
		return "build"
	case "vue":
		return "dist"
	case "svelte":
		return "build"
	default:
		return "dist"
	}
}

// isStaticSiteFramework returns true if the framework generates static files
func isStaticSiteFramework(framework string) bool {
	switch framework {
	case "react", "vue", "angular", "svelte":
		return true
	case "nextjs", "nuxt", "remix":
		// These can be static or server-rendered
		return false
	default:
		return false
	}
}