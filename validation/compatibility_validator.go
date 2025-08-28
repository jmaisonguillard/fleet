package validation

import (
	"fmt"
	"strings"
)

// CompatibilityValidator checks service compatibility
type CompatibilityValidator struct {
	incompatibilities map[string][]string // service -> incompatible services
	requirements      map[string][]string // service -> required services
	warnings          []string
}

// NewCompatibilityValidator creates a new compatibility validator
func NewCompatibilityValidator() *CompatibilityValidator {
	cv := &CompatibilityValidator{
		incompatibilities: make(map[string][]string),
		requirements:      make(map[string][]string),
		warnings:          []string{},
	}
	
	// Define known incompatibilities
	cv.AddIncompatibility("mysql", "mariadb") // Can't run both MySQL and MariaDB
	cv.AddIncompatibility("meilisearch", "typesense") // Usually only need one search engine
	
	// Define known requirements
	cv.AddRequirement("reverb", "laravel") // Reverb requires Laravel
	cv.AddRequirement("xdebug", "php") // Xdebug requires PHP
	
	return cv
}

// AddIncompatibility marks two services as incompatible
func (cv *CompatibilityValidator) AddIncompatibility(service1, service2 string) {
	cv.incompatibilities[service1] = append(cv.incompatibilities[service1], service2)
	cv.incompatibilities[service2] = append(cv.incompatibilities[service2], service1)
}

// AddRequirement marks that service1 requires service2
func (cv *CompatibilityValidator) AddRequirement(service1, service2 string) {
	cv.requirements[service1] = append(cv.requirements[service1], service2)
}

// CheckService checks compatibility for a single service
func (cv *CompatibilityValidator) CheckService(serviceName string, otherServices []string) error {
	// Check for incompatible services
	if incompatible, exists := cv.incompatibilities[serviceName]; exists {
		for _, other := range otherServices {
			for _, incompat := range incompatible {
				if strings.Contains(other, incompat) {
					return fmt.Errorf("service '%s' is incompatible with '%s'", 
						serviceName, other)
				}
			}
		}
	}
	
	// Check for required services
	if required, exists := cv.requirements[serviceName]; exists {
		for _, req := range required {
			found := false
			for _, other := range otherServices {
				if strings.Contains(other, req) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("service '%s' requires '%s' to be present", 
					serviceName, req)
			}
		}
	}
	
	return nil
}

// CheckVersionCompatibility checks if service versions are compatible
func (cv *CompatibilityValidator) CheckVersionCompatibility(services map[string]string) *Validator {
	validator := NewValidator()
	
	// Check PHP version compatibility with extensions
	phpVersion := ""
	hasXdebug := false
	
	for service, version := range services {
		if strings.HasPrefix(service, "php") {
			phpVersion = version
		}
		if service == "xdebug" {
			hasXdebug = true
		}
	}
	
	if hasXdebug && phpVersion != "" {
		// Check if Xdebug is compatible with PHP version
		if strings.HasPrefix(phpVersion, "7.") {
			validator.AddWarning("Xdebug 3.x recommended for PHP 7.x")
		}
	}
	
	// Check database version compatibility
	mysqlVersion := ""
	wordpressPresent := false
	
	for service, version := range services {
		if service == "mysql" {
			mysqlVersion = version
		}
		if service == "wordpress" {
			wordpressPresent = true
		}
	}
	
	if wordpressPresent && mysqlVersion != "" {
		if strings.HasPrefix(mysqlVersion, "8.") {
			validator.AddWarning("WordPress may have compatibility issues with MySQL 8.x default authentication")
		}
	}
	
	return validator
}

// ServiceCompatibilityRules defines compatibility rules for services
type ServiceCompatibilityRules struct {
	rules []CompatibilityRule
}

// CompatibilityRule represents a single compatibility rule
type CompatibilityRule struct {
	Name        string
	Description string
	Check       func(services []string) error
}

// NewServiceCompatibilityRules creates standard compatibility rules
func NewServiceCompatibilityRules() *ServiceCompatibilityRules {
	rules := &ServiceCompatibilityRules{
		rules: []CompatibilityRule{},
	}
	
	// Add standard rules
	rules.AddRule(CompatibilityRule{
		Name:        "SingleDatabase",
		Description: "Only one database type should be used",
		Check: func(services []string) error {
			databases := []string{}
			dbTypes := []string{"mysql", "postgres", "mongodb", "mariadb"}
			
			for _, service := range services {
				for _, dbType := range dbTypes {
					if strings.Contains(service, dbType) {
						databases = append(databases, service)
						break
					}
				}
			}
			
			if len(databases) > 1 {
				return fmt.Errorf("multiple database types detected: %s. Consider using only one",
					strings.Join(databases, ", "))
			}
			
			return nil
		},
	})
	
	rules.AddRule(CompatibilityRule{
		Name:        "SingleCache",
		Description: "Only one cache type should be used",
		Check: func(services []string) error {
			caches := []string{}
			cacheTypes := []string{"redis", "memcached"}
			
			for _, service := range services {
				for _, cacheType := range cacheTypes {
					if strings.Contains(service, cacheType) {
						caches = append(caches, service)
						break
					}
				}
			}
			
			if len(caches) > 1 {
				return fmt.Errorf("multiple cache types detected: %s. Consider using only one",
					strings.Join(caches, ", "))
			}
			
			return nil
		},
	})
	
	rules.AddRule(CompatibilityRule{
		Name:        "PHPFramework",
		Description: "PHP services should have consistent framework",
		Check: func(services []string) error {
			frameworks := []string{}
			frameworkTypes := []string{"laravel", "symfony", "wordpress", "drupal"}
			
			for _, service := range services {
				for _, framework := range frameworkTypes {
					if strings.Contains(service, framework) {
						frameworks = append(frameworks, framework)
						break
					}
				}
			}
			
			if len(frameworks) > 1 {
				return fmt.Errorf("multiple PHP frameworks detected: %s. This may cause conflicts",
					strings.Join(frameworks, ", "))
			}
			
			return nil
		},
	})
	
	return rules
}

// AddRule adds a compatibility rule
func (r *ServiceCompatibilityRules) AddRule(rule CompatibilityRule) {
	r.rules = append(r.rules, rule)
}

// Validate checks all compatibility rules
func (r *ServiceCompatibilityRules) Validate(services []string) *Validator {
	validator := NewValidator()
	
	for _, rule := range r.rules {
		if err := rule.Check(services); err != nil {
			validator.AddWarning(fmt.Sprintf("%s: %v", rule.Name, err))
		}
	}
	
	return validator
}

// CheckDependencyTree checks service dependency tree for issues
type DependencyValidator struct {
	dependencies map[string][]string
	visited      map[string]bool
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator() *DependencyValidator {
	return &DependencyValidator{
		dependencies: make(map[string][]string),
		visited:      make(map[string]bool),
	}
}

// AddDependency adds a service dependency
func (dv *DependencyValidator) AddDependency(service, dependsOn string) {
	dv.dependencies[service] = append(dv.dependencies[service], dependsOn)
}

// CheckCycles checks for circular dependencies
func (dv *DependencyValidator) CheckCycles() error {
	for service := range dv.dependencies {
		dv.visited = make(map[string]bool)
		if err := dv.checkCyclesFrom(service, []string{}); err != nil {
			return err
		}
	}
	return nil
}

// checkCyclesFrom performs DFS to detect cycles
func (dv *DependencyValidator) checkCyclesFrom(service string, path []string) error {
	// Check if we've seen this service in the current path (cycle)
	for _, p := range path {
		if p == service {
			return fmt.Errorf("circular dependency detected: %s -> %s",
				strings.Join(append(path, service), " -> "), service)
		}
	}
	
	// Mark as visited
	dv.visited[service] = true
	
	// Add to path
	newPath := append(path, service)
	
	// Check dependencies
	if deps, exists := dv.dependencies[service]; exists {
		for _, dep := range deps {
			if err := dv.checkCyclesFrom(dep, newPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// GetStartOrder returns the order in which services should be started
func (dv *DependencyValidator) GetStartOrder() ([]string, error) {
	// Check for cycles first
	if err := dv.CheckCycles(); err != nil {
		return nil, err
	}
	
	order := []string{}
	visited := make(map[string]bool)
	
	// Topological sort
	var visit func(string) error
	visit = func(service string) error {
		if visited[service] {
			return nil
		}
		
		visited[service] = true
		
		// Visit dependencies first
		if deps, exists := dv.dependencies[service]; exists {
			for _, dep := range deps {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		
		order = append(order, service)
		return nil
	}
	
	// Visit all services
	for service := range dv.dependencies {
		if err := visit(service); err != nil {
			return nil, err
		}
	}
	
	return order, nil
}