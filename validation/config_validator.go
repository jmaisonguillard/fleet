package validation

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ConfigValidator validates configuration completeness
type ConfigValidator struct {
	strict   bool // If true, warnings become errors
	warnings []string
	errors   []error
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(strict bool) *ConfigValidator {
	return &ConfigValidator{
		strict:   strict,
		warnings: []string{},
		errors:   []error{},
	}
}

// ServiceConfig represents a service configuration for validation
type ServiceConfig struct {
	Name        string
	Image       string
	Build       string
	Port        int
	Ports       []string
	Domain      string
	Folder      string
	Runtime     string
	Framework   string
	Database    string
	Cache       string
	Search      string
	Email       string
	Environment map[string]string
	Volumes     []string
	Needs       []string
}

// ValidateService validates a single service configuration
func (cv *ConfigValidator) ValidateService(svc *ServiceConfig) *Validator {
	validator := NewValidator()
	
	// Required fields
	if svc.Name == "" {
		validator.AddError(fmt.Errorf("service name is required"))
	} else if err := ValidateServiceName(svc.Name); err != nil {
		validator.AddError(fmt.Errorf("invalid service name: %v", err))
	}
	
	// Must have either image or build
	if svc.Image == "" && svc.Build == "" {
		validator.AddError(fmt.Errorf("service '%s' must specify either 'image' or 'build'", svc.Name))
	}
	
	// Validate image name if present
	if svc.Image != "" {
		if err := ValidateImageName(svc.Image); err != nil {
			validator.AddError(fmt.Errorf("service '%s': %v", svc.Name, err))
		}
	}
	
	// Validate build path if present
	if svc.Build != "" {
		if err := ValidatePath(svc.Build); err != nil {
			validator.AddError(fmt.Errorf("service '%s' build path: %v", svc.Name, err))
		}
	}
	
	// Validate port configuration
	if svc.Port > 0 {
		if err := ValidatePort(svc.Port); err != nil {
			validator.AddError(fmt.Errorf("service '%s': %v", svc.Name, err))
		}
	}
	
	for _, portSpec := range svc.Ports {
		if err := ValidatePortString(portSpec); err != nil {
			validator.AddError(fmt.Errorf("service '%s' port '%s': %v", svc.Name, portSpec, err))
		}
	}
	
	// Validate folder mapping
	if svc.Folder != "" {
		if err := ValidatePath(svc.Folder); err != nil {
			validator.AddError(fmt.Errorf("service '%s' folder: %v", svc.Name, err))
		}
		
		// Warn if folder is an absolute path
		if filepath.IsAbs(svc.Folder) {
			validator.AddWarning(fmt.Sprintf("service '%s' uses absolute path '%s', relative paths are recommended", 
				svc.Name, svc.Folder))
		}
	}
	
	// Validate volumes
	for _, volume := range svc.Volumes {
		if err := ValidateVolume(volume); err != nil {
			validator.AddError(fmt.Errorf("service '%s' volume '%s': %v", svc.Name, volume, err))
		}
	}
	
	// Validate environment variables
	for key, value := range svc.Environment {
		if err := ValidateEnvironmentVariable(key, value); err != nil {
			validator.AddError(fmt.Errorf("service '%s' environment: %v", svc.Name, err))
		}
	}
	
	// Validate runtime and framework combinations
	if svc.Runtime != "" && svc.Framework != "" {
		if err := cv.validateRuntimeFramework(svc.Runtime, svc.Framework); err != nil {
			validator.AddError(fmt.Errorf("service '%s': %v", svc.Name, err))
		}
	}
	
	// Validate database configuration
	if svc.Database != "" {
		if err := cv.validateDatabaseConfig(svc); err != nil {
			validator.AddError(err)
		}
	}
	
	// Check for common misconfigurations
	cv.checkCommonIssues(svc, validator)
	
	return validator
}

// validateRuntimeFramework validates runtime and framework combinations
func (cv *ConfigValidator) validateRuntimeFramework(runtime, framework string) error {
	// PHP frameworks require PHP runtime
	phpFrameworks := []string{"laravel", "symfony", "wordpress", "drupal", "codeigniter"}
	for _, phpFramework := range phpFrameworks {
		if framework == phpFramework && !strings.HasPrefix(runtime, "php") {
			return fmt.Errorf("framework '%s' requires PHP runtime", framework)
		}
	}
	
	// Node.js frameworks
	nodeFrameworks := []string{"express", "nextjs", "nuxtjs", "nestjs"}
	for _, nodeFramework := range nodeFrameworks {
		if framework == nodeFramework && !strings.HasPrefix(runtime, "node") {
			return fmt.Errorf("framework '%s' requires Node.js runtime", framework)
		}
	}
	
	return nil
}

// validateDatabaseConfig validates database-specific configuration
func (cv *ConfigValidator) validateDatabaseConfig(svc *ServiceConfig) error {
	// Parse database type and version
	parts := strings.Split(svc.Database, ":")
	if len(parts) == 0 {
		return fmt.Errorf("service '%s': invalid database specification", svc.Name)
	}
	
	dbType := strings.ToLower(parts[0])
	
	// Check supported database types
	supportedDbs := []string{"mysql", "postgres", "postgresql", "mongodb", "mariadb"}
	supported := false
	for _, db := range supportedDbs {
		if dbType == db {
			supported = true
			break
		}
	}
	
	if !supported {
		return fmt.Errorf("service '%s': unsupported database type '%s'", svc.Name, dbType)
	}
	
	return nil
}

// checkCommonIssues checks for common configuration issues
func (cv *ConfigValidator) checkCommonIssues(svc *ServiceConfig, validator *Validator) {
	// Warn if using latest tag
	if strings.HasSuffix(svc.Image, ":latest") {
		validator.AddWarning(fmt.Sprintf("service '%s' uses ':latest' tag, consider using a specific version for reproducibility", svc.Name))
	}
	
	// Warn if no restart policy would be set
	// (Default is unless-stopped, so this is just informational)
	
	// Check for missing dependencies
	if svc.Database != "" && !cv.hasDependency(svc.Needs, "database") {
		validator.AddWarning(fmt.Sprintf("service '%s' uses database but doesn't declare dependency", svc.Name))
	}
	
	if svc.Cache != "" && !cv.hasDependency(svc.Needs, "cache") {
		validator.AddWarning(fmt.Sprintf("service '%s' uses cache but doesn't declare dependency", svc.Name))
	}
	
	// Check for port conflicts with well-known services
	if svc.Port == 3306 && !strings.Contains(svc.Image, "mysql") && !strings.Contains(svc.Image, "mariadb") {
		validator.AddWarning(fmt.Sprintf("service '%s' uses port 3306 (MySQL default) but is not a MySQL service", svc.Name))
	}
	
	if svc.Port == 5432 && !strings.Contains(svc.Image, "postgres") {
		validator.AddWarning(fmt.Sprintf("service '%s' uses port 5432 (PostgreSQL default) but is not a PostgreSQL service", svc.Name))
	}
}

// hasDependency checks if a dependency exists
func (cv *ConfigValidator) hasDependency(needs []string, depType string) bool {
	for _, need := range needs {
		if strings.Contains(need, depType) {
			return true
		}
	}
	return false
}

// ProjectConfig represents the overall project configuration
type ProjectConfig struct {
	Project  string
	Services []ServiceConfig
}

// ValidateProject validates the entire project configuration
func (cv *ConfigValidator) ValidateProject(config *ProjectConfig) *Validator {
	validator := NewValidator()
	
	// Validate project name
	if config.Project == "" {
		validator.AddError(fmt.Errorf("project name is required"))
	} else if err := ValidateServiceName(config.Project); err != nil {
		validator.AddError(fmt.Errorf("invalid project name: %v", err))
	}
	
	// Must have at least one service
	if len(config.Services) == 0 {
		validator.AddError(fmt.Errorf("at least one service must be defined"))
	}
	
	// Check for duplicate service names
	serviceNames := make(map[string]bool)
	for _, svc := range config.Services {
		if serviceNames[svc.Name] {
			validator.AddError(fmt.Errorf("duplicate service name: %s", svc.Name))
		}
		serviceNames[svc.Name] = true
	}
	
	// Validate each service
	for _, svc := range config.Services {
		svcValidator := cv.ValidateService(&svc)
		
		// Merge errors and warnings
		for _, err := range svcValidator.GetErrors() {
			validator.AddError(err)
		}
		for _, warning := range svcValidator.GetWarnings() {
			validator.AddWarning(warning)
		}
	}
	
	// Check cross-service validations
	cv.validateCrossService(config, validator)
	
	// In strict mode, treat warnings as errors
	if cv.strict && validator.HasWarnings() {
		for _, warning := range validator.GetWarnings() {
			validator.AddError(fmt.Errorf("strict mode: %s", warning))
		}
	}
	
	return validator
}

// validateCrossService performs validation across services
func (cv *ConfigValidator) validateCrossService(config *ProjectConfig, validator *Validator) {
	// Check port conflicts
	portValidator := NewPortValidator()
	for _, svc := range config.Services {
		if svc.Port > 0 {
			if err := portValidator.RegisterPort(svc.Port, svc.Name); err != nil {
				validator.AddError(err)
			}
		}
		
		for _, portSpec := range svc.Ports {
			if err := portValidator.RegisterPortRange(portSpec, svc.Name); err != nil {
				validator.AddError(err)
			}
		}
	}
	
	// Check for port conflicts
	portResult := portValidator.Validate()
	for _, err := range portResult.GetErrors() {
		validator.AddError(err)
	}
	for _, warning := range portResult.GetWarnings() {
		validator.AddWarning(warning)
	}
	
	// Check service compatibility
	serviceList := []string{}
	for _, svc := range config.Services {
		serviceList = append(serviceList, svc.Name)
		if svc.Database != "" {
			serviceList = append(serviceList, strings.Split(svc.Database, ":")[0])
		}
		if svc.Cache != "" {
			serviceList = append(serviceList, strings.Split(svc.Cache, ":")[0])
		}
	}
	
	// Check compatibility rules
	rules := NewServiceCompatibilityRules()
	compatResult := rules.Validate(serviceList)
	for _, warning := range compatResult.GetWarnings() {
		validator.AddWarning(warning)
	}
	
	// Check dependency cycles
	depValidator := NewDependencyValidator()
	for _, svc := range config.Services {
		for _, dep := range svc.Needs {
			depValidator.AddDependency(svc.Name, dep)
		}
	}
	
	if err := depValidator.CheckCycles(); err != nil {
		validator.AddError(err)
	}
}