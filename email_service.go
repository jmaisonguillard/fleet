package main

import (
	"fmt"
	"strings"
)

// Email service configuration
type EmailConfig struct {
	Type     string // mailpit
	Version  string
	SMTPPort int    // Custom SMTP port (default 1025)
	UIPort   int    // Custom UI port (default 8025)
	Username string // Optional SMTP username
	Password string // Optional SMTP password
}

// Supported email testing service versions
var supportedEmailVersions = map[string]map[string]string{
	"mailpit": {
		"1.13":    "axllent/mailpit:v1.13",
		"1.14":    "axllent/mailpit:v1.14",
		"1.15":    "axllent/mailpit:v1.15",
		"1.16":    "axllent/mailpit:v1.16",
		"1.17":    "axllent/mailpit:v1.17",
		"1.18":    "axllent/mailpit:v1.18",
		"1.19":    "axllent/mailpit:v1.19",
		"1.20":    "axllent/mailpit:v1.20",
		"latest":  "axllent/mailpit:latest",
		"default": "axllent/mailpit:v1.20",
	},
}

// parseEmailType parses email service type and version from a string like "mailpit:1.20"
func parseEmailType(emailString string) (emailType string, version string) {
	if emailString == "" {
		return "", ""
	}
	
	parts := strings.Split(emailString, ":")
	emailType = strings.ToLower(parts[0])
	
	if len(parts) > 1 {
		version = parts[1]
	} else {
		// Use default version if not specified
		if versions, ok := supportedEmailVersions[emailType]; ok {
			version = versions["default"]
			// Extract just the version number
			if idx := strings.LastIndex(version, ":"); idx >= 0 {
				version = version[idx+1:]
			}
			// Remove v prefix for consistency
			version = strings.TrimPrefix(version, "v")
		}
	}
	
	return emailType, version
}

// getEmailImage returns the appropriate Docker image for an email service
func getEmailImage(emailType, version string) string {
	emailType = strings.ToLower(emailType)
	
	// Add v prefix for mailpit versions if not present and not "latest"
	if emailType == "mailpit" && version != "" && version != "latest" && !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	
	if versions, ok := supportedEmailVersions[emailType]; ok {
		// Try exact match first
		if image, ok := versions[version]; ok {
			return image
		}
		// Try without v prefix for mailpit
		if emailType == "mailpit" {
			versionNoV := strings.TrimPrefix(version, "v")
			if image, ok := versions[versionNoV]; ok {
				return image
			}
		}
		// Fallback to default
		return versions["default"]
	}
	
	// Unknown email type
	return ""
}

// getEmailServiceName returns the service name for the email testing service
// Since only one email service is allowed per project, we use a fixed name
func getEmailServiceName(emailType string) string {
	// Always return the same name to ensure singleton behavior
	return "mailpit"
}

// emailServiceExists checks if an email service already exists in the compose file
func emailServiceExists(compose *DockerCompose) bool {
	// Check for the fixed email service name
	_, exists := compose.Services["mailpit"]
	return exists
}

// addEmailService adds an email testing service to the compose file
func addEmailService(compose *DockerCompose, svc *Service, config *Config) {
	if svc.Email == "" {
		return
	}
	
	// Check if email service already exists (singleton pattern)
	if emailServiceExists(compose) {
		// Email service already exists, just ensure the app service depends on it
		if appService, ok := compose.Services[svc.Name]; ok {
			if !containsString(appService.DependsOn, "mailpit") {
				appService.DependsOn = append(appService.DependsOn, "mailpit")
				compose.Services[svc.Name] = appService
			}
		}
		// Also add email environment variables
		if appService, ok := compose.Services[svc.Name]; ok {
			addEmailEnvVars(&appService, "mailpit", "mailpit", svc)
			compose.Services[svc.Name] = appService
		}
		return
	}
	
	emailType, version := parseEmailType(svc.Email)
	if emailType == "" {
		return
	}
	
	// Get the email service name (always "mailpit" for singleton)
	emailServiceName := getEmailServiceName(emailType)
	
	// Create the email service
	emailImage := getEmailImage(emailType, version)
	if emailImage == "" {
		return
	}
	
	emailService := DockerService{
		Image:    emailImage,
		Networks: []string{"fleet-network"},
		Restart:  "unless-stopped",
		Volumes:  []string{},
		Environment: make(map[string]string),
	}
	
	// Configure based on email type
	switch emailType {
	case "mailpit":
		configureMailpitService(&emailService, svc, emailServiceName)
	}
	
	// Add the service to compose
	compose.Services[emailServiceName] = emailService
	
	// Update app service to depend on email service
	if appService, ok := compose.Services[svc.Name]; ok {
		if !containsString(appService.DependsOn, emailServiceName) {
			appService.DependsOn = append(appService.DependsOn, emailServiceName)
		}
		
		// Add email connection environment variables to the app
		addEmailEnvVars(&appService, emailType, emailServiceName, svc)
		compose.Services[svc.Name] = appService
	}
}

// configureMailpitService configures a Mailpit email testing service
func configureMailpitService(service *DockerService, svc *Service, emailServiceName string) {
	// Data volume for persistence (optional, stores emails)
	service.Volumes = append(service.Volumes, fmt.Sprintf("%s-data:/data", emailServiceName))
	
	// Environment variables for Mailpit configuration
	service.Environment["MP_DATA_FILE"] = "/data/mailpit.db"
	service.Environment["MP_SMTP_AUTH_ACCEPT_ANY"] = "1"
	service.Environment["MP_SMTP_AUTH_ALLOW_INSECURE"] = "1"
	
	// Set authentication if provided
	if svc.EmailUsername != "" && svc.EmailPassword != "" {
		service.Environment["MP_SMTP_AUTH"] = fmt.Sprintf("%s:%s", svc.EmailUsername, svc.EmailPassword)
	}
	
	// Configure max message size (10MB default)
	service.Environment["MP_MAX_MESSAGES"] = "5000"
	service.Environment["MP_MESSAGE_LIMIT"] = "10"  // 10MB
	
	// Database settings
	service.Environment["MP_DATABASE"] = "/data/mailpit.db"
	
	// UI settings
	service.Environment["MP_WEBROOT"] = "/"
	
	// Health check
	service.HealthCheck = &HealthCheckYAML{
		Test:     []string{"CMD", "/mailpit", "healthcheck"},
		Interval: "30s",
		Timeout:  "3s",
		Retries:  3,
	}
}

// addEmailEnvVars adds email service environment variables to the app service
func addEmailEnvVars(service *DockerService, emailType, emailServiceName string, svc *Service) {
	if service.Environment == nil {
		service.Environment = make(map[string]string)
	}
	
	switch emailType {
	case "mailpit":
		// SMTP configuration
		service.Environment["SMTP_HOST"] = emailServiceName
		service.Environment["SMTP_PORT"] = "1025"
		service.Environment["MAIL_HOST"] = emailServiceName
		service.Environment["MAIL_PORT"] = "1025"
		
		// Common mail environment variables
		service.Environment["MAIL_DRIVER"] = "smtp"
		service.Environment["MAIL_MAILER"] = "smtp"
		
		// Authentication if provided
		if svc.EmailUsername != "" && svc.EmailPassword != "" {
			service.Environment["SMTP_USERNAME"] = svc.EmailUsername
			service.Environment["SMTP_PASSWORD"] = svc.EmailPassword
			service.Environment["MAIL_USERNAME"] = svc.EmailUsername
			service.Environment["MAIL_PASSWORD"] = svc.EmailPassword
		}
		
		// Disable encryption for local testing
		service.Environment["MAIL_ENCRYPTION"] = ""
		service.Environment["SMTP_SECURE"] = "false"
		
		// Default from address
		service.Environment["MAIL_FROM_ADDRESS"] = "noreply@example.com"
		service.Environment["MAIL_FROM_NAME"] = "Fleet App"
		
		// Mailpit Web UI URL
		service.Environment["MAILPIT_UI_URL"] = fmt.Sprintf("http://%s:8025", emailServiceName)
	}
}