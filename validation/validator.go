package validation

import (
	"fmt"
	"strconv"
	"strings"
)

// Validator provides centralized validation for Fleet configurations
type Validator struct {
	errors   []error
	warnings []string
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors:   []error{},
		warnings: []string{},
	}
}

// AddError adds a validation error
func (v *Validator) AddError(err error) {
	v.errors = append(v.errors, err)
}

// AddWarning adds a validation warning
func (v *Validator) AddWarning(msg string) {
	v.warnings = append(v.warnings, msg)
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// HasWarnings returns true if there are validation warnings
func (v *Validator) HasWarnings() bool {
	return len(v.warnings) > 0
}

// GetErrors returns all validation errors
func (v *Validator) GetErrors() []error {
	return v.errors
}

// GetWarnings returns all validation warnings
func (v *Validator) GetWarnings() []string {
	return v.warnings
}

// Clear resets the validator
func (v *Validator) Clear() {
	v.errors = []error{}
	v.warnings = []string{}
}

// Result returns a formatted validation result
func (v *Validator) Result() error {
	if !v.HasErrors() {
		return nil
	}
	
	var msgs []string
	for _, err := range v.errors {
		msgs = append(msgs, err.Error())
	}
	
	return fmt.Errorf("validation failed:\n%s", strings.Join(msgs, "\n"))
}

// ValidationContext holds context for validation
type ValidationContext struct {
	ServiceName string
	ConfigPath  string
	Strict      bool // If true, warnings become errors
}

// ServiceValidator validates individual services
type ServiceValidator interface {
	ValidateService(svc interface{}, ctx *ValidationContext) *Validator
}

// IConfigValidator validates entire configurations
type IConfigValidator interface {
	ValidateConfig(config interface{}, ctx *ValidationContext) *Validator
}

// ComposeValidator validates Docker Compose generation
type ComposeValidator interface {
	ValidateCompose(compose interface{}, ctx *ValidationContext) *Validator
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validate    func(value interface{}) error
}

// Apply applies the validation rule
func (r *ValidationRule) Apply(value interface{}) error {
	if r.Validate == nil {
		return nil
	}
	
	if err := r.Validate(value); err != nil {
		return fmt.Errorf("%s: %v", r.Name, err)
	}
	
	return nil
}

// RuleSet is a collection of validation rules
type RuleSet struct {
	rules []ValidationRule
}

// NewRuleSet creates a new rule set
func NewRuleSet() *RuleSet {
	return &RuleSet{
		rules: []ValidationRule{},
	}
}

// Add adds a rule to the set
func (rs *RuleSet) Add(rule ValidationRule) *RuleSet {
	rs.rules = append(rs.rules, rule)
	return rs
}

// Validate applies all rules and returns a validator with results
func (rs *RuleSet) Validate(value interface{}) *Validator {
	validator := NewValidator()
	
	for _, rule := range rs.rules {
		if err := rule.Apply(value); err != nil {
			validator.AddError(err)
		}
	}
	
	return validator
}

// Common validation functions

// ValidatePort validates a port number
func ValidatePort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port number %d: must be between 1 and 65535", port)
	}
	return nil
}

// ValidatePortString validates a port string (e.g., "8080:80")
func ValidatePortString(portStr string) error {
	parts := strings.Split(portStr, ":")
	
	for _, part := range parts {
		port, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("invalid port format '%s': %v", portStr, err)
		}
		
		if err := ValidatePort(port); err != nil {
			return err
		}
	}
	
	return nil
}

// ValidateImageName validates a Docker image name
func ValidateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}
	
	// Basic validation - could be extended
	if strings.Contains(image, " ") {
		return fmt.Errorf("image name cannot contain spaces: %s", image)
	}
	
	return nil
}

// ValidateServiceName validates a service name
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	
	// Docker service name restrictions
	if !isValidDockerName(name) {
		return fmt.Errorf("invalid service name '%s': must contain only letters, numbers, hyphens, and underscores", name)
	}
	
	return nil
}

// isValidDockerName checks if a name is valid for Docker
func isValidDockerName(name string) bool {
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || 
			(ch >= 'A' && ch <= 'Z') || 
			(ch >= '0' && ch <= '9') || 
			ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

// ValidateEnvironmentVariable validates an environment variable
func ValidateEnvironmentVariable(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}
	
	// Check for invalid characters in key
	if strings.Contains(key, "=") || strings.Contains(key, " ") {
		return fmt.Errorf("invalid environment variable key '%s': cannot contain '=' or spaces", key)
	}
	
	return nil
}

// ValidatePath validates a file or directory path
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	// Basic validation - could check if path exists
	return nil
}

// ValidateVolume validates a volume specification
func ValidateVolume(volume string) error {
	if volume == "" {
		return fmt.Errorf("volume specification cannot be empty")
	}
	
	// Check format: source:target[:mode]
	parts := strings.Split(volume, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid volume format '%s': expected source:target[:mode]", volume)
	}
	
	return nil
}