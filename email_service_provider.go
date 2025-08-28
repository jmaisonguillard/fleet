package main

import (
	"fmt"
)

// EmailServiceProvider implements ServiceProvider for email services
type EmailServiceProvider struct {
	defaultVersions map[string]string
	supportedVersions map[string][]string
}

// NewEmailServiceProvider creates a new email service provider
func NewEmailServiceProvider() *EmailServiceProvider {
	return &EmailServiceProvider{
		defaultVersions: map[string]string{
			"mailpit": "1.20",
		},
		supportedVersions: map[string][]string{
			"mailpit": {"1.13", "1.14", "1.15", "1.16", "1.17", "1.18", "1.19", "1.20", "latest"},
		},
	}
}

// GetServiceName returns the container name for the email service
func (p *EmailServiceProvider) GetServiceName(serviceType, version string) string {
	return getEmailServiceName(serviceType)
}

// AddService adds the email service to the Docker Compose configuration
func (p *EmailServiceProvider) AddService(compose *DockerCompose, svc *Service, config *Config) {
	addEmailService(compose, svc, config)
}

// ValidateConfig validates the email service configuration
func (p *EmailServiceProvider) ValidateConfig(svc *Service) error {
	if svc.Email == "" {
		return nil // No email service configured, nothing to validate
	}
	
	emailType, version := parseEmailType(svc.Email)
	if emailType == "" {
		return fmt.Errorf("invalid email service type: %s", svc.Email)
	}
	
	// Check if version is supported
	if version != "" {
		if supportedVersions, ok := p.supportedVersions[emailType]; ok {
			found := false
			for _, v := range supportedVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported %s version: %s", emailType, version)
			}
		}
	}
	
	return nil
}

// GetDefaultVersion returns the default version for the email type
func (p *EmailServiceProvider) GetDefaultVersion() string {
	return "1.20" // Default Mailpit version
}

// GetSupportedVersions returns all supported versions
func (p *EmailServiceProvider) GetSupportedVersions() []string {
	var versions []string
	for emailType, emailVersions := range p.supportedVersions {
		for _, v := range emailVersions {
			versions = append(versions, fmt.Sprintf("%s:%s", emailType, v))
		}
	}
	return versions
}

// IsShared indicates if email services use shared containers
func (p *EmailServiceProvider) IsShared() bool {
	return true // Email service is singleton/shared
}

// GetEnvironmentVariables returns environment variables for dependent services
func (p *EmailServiceProvider) GetEnvironmentVariables(svc *Service, config *Config) map[string]string {
	if svc.Email == "" {
		return nil
	}
	
	emailType, _ := parseEmailType(svc.Email)
	serviceName := p.GetServiceName(emailType, "")
	
	env := make(map[string]string)
	
	// Set email service specific environment variables
	switch emailType {
	case "mailpit":
		// SMTP settings
		smtpPort := "1025"
		
		env["MAIL_MAILER"] = "smtp"
		env["MAIL_HOST"] = serviceName
		env["MAIL_PORT"] = smtpPort
		env["MAIL_ENCRYPTION"] = "null"
		
		// Alternative naming conventions
		env["SMTP_HOST"] = serviceName
		env["SMTP_PORT"] = smtpPort
		
		// Authentication if configured
		if svc.EmailUsername != "" {
			env["MAIL_USERNAME"] = svc.EmailUsername
			env["SMTP_USERNAME"] = svc.EmailUsername
		}
		if svc.EmailPassword != "" {
			env["MAIL_PASSWORD"] = svc.EmailPassword
			env["SMTP_PASSWORD"] = svc.EmailPassword
		}
		
		// Mailpit UI URL
		env["MAILPIT_URL"] = fmt.Sprintf("http://%s:8025", serviceName)
		env["MAILPIT_UI_URL"] = env["MAILPIT_URL"]
		
		// From address defaults
		env["MAIL_FROM_ADDRESS"] = fmt.Sprintf("noreply@%s.local", config.Project)
		env["MAIL_FROM_NAME"] = config.Project
	}
	
	return env
}