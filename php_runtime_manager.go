package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PHPService represents a PHP service configuration
type PHPService struct {
	Name          string
	Runtime       string
	Version       string
	Framework     string
	Folder        string
	ContainerName string
}

// PHPRuntimeManager manages PHP services and their runtime configurations
type PHPRuntimeManager struct {
	config     *Config
	services   []PHPService
	hasComposer map[string]bool
}

// NewPHPRuntimeManager creates a new PHP runtime manager
func NewPHPRuntimeManager(config *Config) *PHPRuntimeManager {
	manager := &PHPRuntimeManager{
		config:      config,
		services:    []PHPService{},
		hasComposer: make(map[string]bool),
	}
	manager.detectPHPServices()
	return manager
}

// detectPHPServices identifies all PHP services in the configuration
func (m *PHPRuntimeManager) detectPHPServices() {
	for _, svc := range m.config.Services {
		if !strings.HasPrefix(svc.Runtime, "php") {
			continue
		}
		
		_, version := parsePHPRuntime(svc.Runtime)
		
		phpService := PHPService{
			Name:          svc.Name,
			Runtime:       svc.Runtime,
			Version:       version,
			Framework:     svc.Framework,
			Folder:        svc.Folder,
			ContainerName: m.getPHPContainerName(&svc),
		}
		
		// Check for composer.json in service folder
		if svc.Folder != "" {
			composerPath := filepath.Join(svc.Folder, "composer.json")
			m.hasComposer[svc.Name] = fileExists(composerPath)
		}
		
		m.services = append(m.services, phpService)
	}
}

// getPHPContainerName returns the container name for a PHP service
func (m *PHPRuntimeManager) getPHPContainerName(svc *Service) string {
	// PHP-FPM containers are named {service}-php
	return fmt.Sprintf("%s-%s-php", m.config.Project, svc.Name)
}

// GetPHPServices returns all PHP services
func (m *PHPRuntimeManager) GetPHPServices() []PHPService {
	return m.services
}

// GetDefaultPHPService returns the default (first) PHP service
func (m *PHPRuntimeManager) GetDefaultPHPService() *PHPService {
	if len(m.services) == 0 {
		return nil
	}
	return &m.services[0]
}

// GetPHPServiceByName returns a PHP service by name
func (m *PHPRuntimeManager) GetPHPServiceByName(name string) *PHPService {
	for _, svc := range m.services {
		if svc.Name == name {
			return &svc
		}
	}
	return nil
}

// HasPHPServices returns true if there are any PHP services
func (m *PHPRuntimeManager) HasPHPServices() bool {
	return len(m.services) > 0
}

// ShouldRunComposerInstall checks if composer install should run for a service
func (m *PHPRuntimeManager) ShouldRunComposerInstall(serviceName string) bool {
	// Check if service has composer.json
	if !m.hasComposer[serviceName] {
		return false
	}
	
	// Get the service
	var folder string
	for _, svc := range m.config.Services {
		if svc.Name == serviceName {
			folder = svc.Folder
			break
		}
	}
	
	if folder == "" {
		return false
	}
	
	// Check if vendor directory exists
	vendorPath := filepath.Join(folder, "vendor")
	if _, err := os.Stat(vendorPath); err == nil {
		// vendor directory exists, don't run composer install
		return false
	}
	
	return true
}

// GetServicesNeedingComposerInstall returns services that need composer install
func (m *PHPRuntimeManager) GetServicesNeedingComposerInstall() []PHPService {
	var services []PHPService
	
	for _, svc := range m.services {
		if m.ShouldRunComposerInstall(svc.Name) {
			services = append(services, svc)
		}
	}
	
	return services
}

// GetServiceFolder returns the folder path for a service
func (m *PHPRuntimeManager) GetServiceFolder(serviceName string) string {
	for _, svc := range m.config.Services {
		if svc.Name == serviceName {
			return svc.Folder
		}
	}
	return ""
}

// DetectFramework detects the PHP framework for a service
func (m *PHPRuntimeManager) DetectFramework(service *PHPService) string {
	if service.Framework != "" {
		return service.Framework
	}
	
	if service.Folder == "" {
		return ""
	}
	
	// Use PHPConfigurator's framework detection
	configurator := NewPHPConfigurator()
	framework := configurator.DetectFramework(service.Folder)
	
	return framework
}

// GetAvailableCommands returns available commands based on framework
func (m *PHPRuntimeManager) GetAvailableCommands(service *PHPService) []string {
	commands := []string{"composer", "php"}
	
	framework := m.DetectFramework(service)
	
	switch framework {
	case "laravel", "lumen":
		commands = append(commands, "artisan")
	case "symfony":
		commands = append(commands, "console")
	}
	
	return commands
}

// RunComposerInstall runs composer install for a service
func (m *PHPRuntimeManager) RunComposerInstall(service *PHPService) error {
	if service == nil {
		return fmt.Errorf("no PHP service provided")
	}
	
	fmt.Printf("📦 Running composer install for service '%s'...\n", service.Name)
	
	// Build docker exec command
	args := []string{
		"exec",
		"-w", "/app",
		service.ContainerName,
		"composer", "install", "--no-interaction", "--prefer-dist",
	}
	
	return runDocker(args)
}

