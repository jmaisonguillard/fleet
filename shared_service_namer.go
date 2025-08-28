package main

import (
	"fmt"
	"strings"
)

// SharedServiceNamer provides unified naming for shared containers
type SharedServiceNamer struct {
	nameRegistry map[string]bool // Track used names to detect collisions
}

// NewSharedServiceNamer creates a new shared service namer
func NewSharedServiceNamer() *SharedServiceNamer {
	return &SharedServiceNamer{
		nameRegistry: make(map[string]bool),
	}
}

// GetServiceName generates a consistent service name based on type and version
func (n *SharedServiceNamer) GetServiceName(serviceType, version string) string {
	// Normalize the service type
	serviceType = strings.ToLower(serviceType)
	
	// Handle singleton services (no version in name)
	if n.isSingleton(serviceType) {
		return serviceType
	}
	
	// Clean version string
	version = n.cleanVersion(version)
	
	// Generate the service name
	var name string
	if version != "" && version != "latest" {
		// Replace dots with hyphens for Docker compatibility
		cleanVersion := strings.ReplaceAll(version, ".", "")
		name = fmt.Sprintf("%s-%s", serviceType, cleanVersion)
	} else {
		// Default version or latest
		name = fmt.Sprintf("%s-latest", serviceType)
	}
	
	// Register the name
	n.nameRegistry[name] = true
	
	return name
}

// GetUniqueServiceName ensures a unique name by appending a suffix if needed
func (n *SharedServiceNamer) GetUniqueServiceName(baseName string) string {
	// If name is not taken, use it
	if !n.nameRegistry[baseName] {
		n.nameRegistry[baseName] = true
		return baseName
	}
	
	// Find a unique name by appending numbers
	for i := 2; ; i++ {
		uniqueName := fmt.Sprintf("%s-%d", baseName, i)
		if !n.nameRegistry[uniqueName] {
			n.nameRegistry[uniqueName] = true
			return uniqueName
		}
	}
}

// HasCollision checks if a name is already registered
func (n *SharedServiceNamer) HasCollision(name string) bool {
	return n.nameRegistry[name]
}

// RegisterName manually registers a name to prevent collisions
func (n *SharedServiceNamer) RegisterName(name string) {
	n.nameRegistry[name] = true
}

// GetRegisteredNames returns all registered service names
func (n *SharedServiceNamer) GetRegisteredNames() []string {
	names := make([]string, 0, len(n.nameRegistry))
	for name := range n.nameRegistry {
		names = append(names, name)
	}
	return names
}

// isSingleton returns true if the service type should have only one instance
func (n *SharedServiceNamer) isSingleton(serviceType string) bool {
	singletons := map[string]bool{
		"mailpit":      true,
		"reverb":       true,
		"nginx-proxy":  true,
	}
	return singletons[serviceType]
}

// cleanVersion normalizes version strings
func (n *SharedServiceNamer) cleanVersion(version string) string {
	// Remove common prefixes
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	
	// Handle special versions
	if version == "" || version == "latest" || version == "default" {
		return ""
	}
	
	// For MinIO, handle RELEASE versions
	if strings.Contains(version, "RELEASE") {
		// Extract year from RELEASE.2024-01-01 format
		parts := strings.Split(version, ".")
		if len(parts) >= 2 && strings.HasPrefix(parts[1], "20") {
			return parts[1][:4] // Return year only
		}
	}
	
	return version
}

// StandardizeServiceType converts various service type names to standard forms
func (n *SharedServiceNamer) StandardizeServiceType(serviceType string) string {
	// Normalize to lowercase
	serviceType = strings.ToLower(serviceType)
	
	// Map aliases to standard names
	aliases := map[string]string{
		"postgresql": "postgres",
		"mariadb":    "mariadb",
		"mysql":      "mysql",
		"mongo":      "mongodb",
		"redis":      "redis",
		"memcache":   "memcached",
		"minio":      "minio",
		"mailpit":    "mailpit",
		"mail":       "mailpit",
		"email":      "mailpit",
	}
	
	if standard, ok := aliases[serviceType]; ok {
		return standard
	}
	
	return serviceType
}

// GlobalServiceNamer is the singleton instance
var GlobalServiceNamer = NewSharedServiceNamer()

// Convenience functions that use the global namer

// GetSharedServiceName generates a consistent shared service name
func GetSharedServiceName(serviceType, version string) string {
	return GlobalServiceNamer.GetServiceName(serviceType, version)
}

// GetUniqueServiceName ensures a unique service name
func GetUniqueServiceName(baseName string) string {
	return GlobalServiceNamer.GetUniqueServiceName(baseName)
}

// RegisterServiceName registers a name to prevent collisions
func RegisterServiceName(name string) {
	GlobalServiceNamer.RegisterName(name)
}