package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmailServiceTestSuite struct {
	suite.Suite
}

func (suite *EmailServiceTestSuite) TestParseEmailType() {
	testCases := []struct {
		name          string
		input         string
		expectType    string
		expectVersion string
	}{
		{"Mailpit with version", "mailpit:1.20", "mailpit", "1.20"},
		{"Mailpit 1.19", "mailpit:1.19", "mailpit", "1.19"},
		{"Mailpit without version", "mailpit", "mailpit", "1.20"}, // Should use default
		{"Case insensitive", "MAILPIT:1.18", "mailpit", "1.18"},
		{"Empty string", "", "", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			emailType, version := parseEmailType(tc.input)
			suite.Equal(tc.expectType, emailType)
			suite.Equal(tc.expectVersion, version)
		})
	}
}

func (suite *EmailServiceTestSuite) TestGetEmailImage() {
	testCases := []struct {
		name      string
		emailType string
		version   string
		expected  string
	}{
		{"Mailpit 1.20", "mailpit", "1.20", "axllent/mailpit:v1.20"},
		{"Mailpit v1.19", "mailpit", "v1.19", "axllent/mailpit:v1.19"},
		{"Mailpit 1.18", "mailpit", "1.18", "axllent/mailpit:v1.18"},
		{"Mailpit latest", "mailpit", "latest", "axllent/mailpit:latest"},
		{"Mailpit unknown version", "mailpit", "999", "axllent/mailpit:v1.20"}, // Falls back to default
		{"Unknown email type", "unknown", "1.0", ""},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			image := getEmailImage(tc.emailType, tc.version)
			suite.Equal(tc.expected, image)
		})
	}
}

func (suite *EmailServiceTestSuite) TestGetEmailServiceName() {
	// Test that email service always returns the same name (singleton)
	testCases := []struct {
		name      string
		emailType string
		expected  string
	}{
		{"Mailpit service name", "mailpit", "mailpit"},
		{"Any type returns mailpit", "anytype", "mailpit"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			serviceName := getEmailServiceName(tc.emailType)
			suite.Equal(tc.expected, serviceName)
		})
	}
}

func (suite *EmailServiceTestSuite) TestEmailServiceSingleton() {
	// Test that only one email service can be created
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "api1",
				Image: "node:18",
				Email: "mailpit:1.20",
			},
			{
				Name:  "api2",
				Image: "python:3.9",
				Email: "mailpit:1.19", // Different version, but should share the same service
			},
		},
	}

	compose := &DockerCompose{
		Services: make(map[string]DockerService),
		Volumes:  make(map[string]DockerVolume),
		Networks: make(map[string]DockerNetwork),
	}

	// Add first service
	compose.Services["api1"] = DockerService{Image: "node:18"}
	addEmailService(compose, &config.Services[0], config)

	// Add second service
	compose.Services["api2"] = DockerService{Image: "python:3.9"}
	addEmailService(compose, &config.Services[1], config)

	// Should only have one email service (mailpit)
	_, exists := compose.Services["mailpit"]
	suite.True(exists, "Mailpit service should exist")

	// Count email services
	emailCount := 0
	for name := range compose.Services {
		if name == "mailpit" {
			emailCount++
		}
	}
	suite.Equal(1, emailCount, "Should only have one email service (singleton)")

	// Both app services should depend on the same email service
	api1Service := compose.Services["api1"]
	api2Service := compose.Services["api2"]
	suite.Contains(api1Service.DependsOn, "mailpit")
	suite.Contains(api2Service.DependsOn, "mailpit")
}

func (suite *EmailServiceTestSuite) TestConfigureMailpitService() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:          "myapp",
		Email:         "mailpit:1.20",
		EmailUsername: "testuser",
		EmailPassword: "testpass",
	}

	configureMailpitService(service, svc, "mailpit")

	// Check volume
	suite.Contains(service.Volumes, "mailpit-data:/data")

	// Check environment variables
	suite.Equal("/data/mailpit.db", service.Environment["MP_DATA_FILE"])
	suite.Equal("1", service.Environment["MP_SMTP_AUTH_ACCEPT_ANY"])
	suite.Equal("1", service.Environment["MP_SMTP_AUTH_ALLOW_INSECURE"])
	suite.Equal("testuser:testpass", service.Environment["MP_SMTP_AUTH"])
	suite.Equal("5000", service.Environment["MP_MAX_MESSAGES"])
	suite.Equal("/data/mailpit.db", service.Environment["MP_DATABASE"])

	// Check health check
	suite.NotNil(service.HealthCheck)
	suite.Contains(service.HealthCheck.Test, "/mailpit")
	suite.Contains(service.HealthCheck.Test, "healthcheck")
}

func (suite *EmailServiceTestSuite) TestConfigureMailpitServiceNoAuth() {
	service := &DockerService{
		Environment: make(map[string]string),
		Volumes:     []string{},
	}
	svc := &Service{
		Name:  "myapp",
		Email: "mailpit:1.20",
	}

	configureMailpitService(service, svc, "mailpit")

	// Check no auth is set
	suite.Empty(service.Environment["MP_SMTP_AUTH"])
}

func (suite *EmailServiceTestSuite) TestAddEmailEnvVars() {
	service := &DockerService{
		Environment: make(map[string]string),
	}
	svc := &Service{
		Name:          "testapp",
		EmailUsername: "smtp_user",
		EmailPassword: "smtp_pass",
	}

	addEmailEnvVars(service, "mailpit", "mailpit", svc)

	// Check SMTP configuration
	suite.Equal("mailpit", service.Environment["SMTP_HOST"])
	suite.Equal("1025", service.Environment["SMTP_PORT"])
	suite.Equal("mailpit", service.Environment["MAIL_HOST"])
	suite.Equal("1025", service.Environment["MAIL_PORT"])
	
	// Check auth
	suite.Equal("smtp_user", service.Environment["SMTP_USERNAME"])
	suite.Equal("smtp_pass", service.Environment["SMTP_PASSWORD"])
	suite.Equal("smtp_user", service.Environment["MAIL_USERNAME"])
	suite.Equal("smtp_pass", service.Environment["MAIL_PASSWORD"])
	
	// Check mail settings
	suite.Equal("smtp", service.Environment["MAIL_DRIVER"])
	suite.Equal("smtp", service.Environment["MAIL_MAILER"])
	suite.Equal("", service.Environment["MAIL_ENCRYPTION"])
	suite.Equal("false", service.Environment["SMTP_SECURE"])
	
	// Check default from address
	suite.Equal("noreply@example.com", service.Environment["MAIL_FROM_ADDRESS"])
	suite.Equal("Fleet App", service.Environment["MAIL_FROM_NAME"])
	
	// Check UI URL
	suite.Equal("http://mailpit:8025", service.Environment["MAILPIT_UI_URL"])
}

func (suite *EmailServiceTestSuite) TestAddEmailEnvVarsNoAuth() {
	service := &DockerService{
		Environment: make(map[string]string),
	}
	svc := &Service{
		Name: "testapp",
	}

	addEmailEnvVars(service, "mailpit", "mailpit", svc)

	// Check no auth is set
	suite.Empty(service.Environment["SMTP_USERNAME"])
	suite.Empty(service.Environment["SMTP_PASSWORD"])
	suite.Empty(service.Environment["MAIL_USERNAME"])
	suite.Empty(service.Environment["MAIL_PASSWORD"])
}

func (suite *EmailServiceTestSuite) TestEmailServiceExists() {
	compose := &DockerCompose{
		Services: map[string]DockerService{
			"api": {Image: "node:18"},
		},
	}

	// Initially no email service
	suite.False(emailServiceExists(compose))

	// Add email service
	compose.Services["mailpit"] = DockerService{Image: "axllent/mailpit:v1.20"}

	// Now email service exists
	suite.True(emailServiceExists(compose))
}

func (suite *EmailServiceTestSuite) TestIntegrationComposeGenerationWithEmail() {
	config := &Config{
		Project: "testproject",
		Services: []Service{
			{
				Name:          "webapp",
				Image:         "node:18",
				Port:          3000,
				Email:         "mailpit:1.20",
				EmailUsername: "webapp_user",
				EmailPassword: "webapp_pass",
			},
			{
				Name:  "api",
				Image: "python:3.9",
				Port:  8000,
				Email: "mailpit:1.19", // Different version but still shares the same service
			},
			{
				Name:  "worker",
				Image: "golang:1.21",
				Email: "mailpit", // No version specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check that email service was created
	_, mailpitExists := compose.Services["mailpit"]
	suite.True(mailpitExists, "Mailpit service should be created")

	// Check dependencies
	webappService := compose.Services["webapp"]
	apiService := compose.Services["api"]
	workerService := compose.Services["worker"]

	suite.Contains(webappService.DependsOn, "mailpit", "Webapp should depend on Mailpit")
	suite.Contains(apiService.DependsOn, "mailpit", "API should depend on Mailpit")
	suite.Contains(workerService.DependsOn, "mailpit", "Worker should depend on Mailpit")

	// Check that volume is created
	_, mailpitVolExists := compose.Volumes["mailpit-data"]
	suite.True(mailpitVolExists, "Mailpit volume should be created")

	// Check environment variables are set for services
	suite.Equal("mailpit", webappService.Environment["SMTP_HOST"])
	suite.Equal("1025", webappService.Environment["SMTP_PORT"])
	suite.Equal("webapp_user", webappService.Environment["SMTP_USERNAME"])
	
	// Second service should also have email config but use first service's auth
	suite.Equal("mailpit", apiService.Environment["SMTP_HOST"])
}

func (suite *EmailServiceTestSuite) TestServiceWithoutEmail() {
	// Test that services without email configuration work correctly
	config := &Config{
		Project: "test",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				// No email specified
			},
		},
	}

	compose := generateDockerCompose(config)

	// Should only have the web service
	suite.NotNil(compose.Services["web"])

	// Should not have any email services
	_, hasMailpit := compose.Services["mailpit"]
	suite.False(hasMailpit, "Should not have mailpit service when no email is configured")
}

func (suite *EmailServiceTestSuite) TestEmailWithFullStack() {
	// Test complete setup with email, database, cache, search, and compat
	config := &Config{
		Project: "fullstack",
		Services: []Service{
			{
				Name:            "app",
				Image:           "node:18",
				Port:            3000,
				Database:        "postgres:15",
				Cache:           "redis:7.2",
				Search:          "meilisearch:1.6",
				Compat:          "minio:2024",
				Email:           "mailpit:1.20",
				EmailUsername:   "app_smtp",
				EmailPassword:   "app_pass",
			},
		},
	}

	compose := generateDockerCompose(config)

	// Check all supporting services exist
	_, dbExists := compose.Services["postgres-15"]
	_, cacheExists := compose.Services["redis-72"]
	_, searchExists := compose.Services["meilisearch-16"]
	_, minioExists := compose.Services["minio-2024"]
	_, emailExists := compose.Services["mailpit"]

	suite.True(dbExists, "Database service should exist")
	suite.True(cacheExists, "Cache service should exist")
	suite.True(searchExists, "Search service should exist")
	suite.True(minioExists, "MinIO service should exist")
	suite.True(emailExists, "Email service should exist")

	// Check app dependencies
	appService := compose.Services["app"]
	suite.Contains(appService.DependsOn, "postgres-15")
	suite.Contains(appService.DependsOn, "redis-72")
	suite.Contains(appService.DependsOn, "meilisearch-16")
	suite.Contains(appService.DependsOn, "minio-2024")
	suite.Contains(appService.DependsOn, "mailpit")
}

func TestEmailServiceSuite(t *testing.T) {
	suite.Run(t, new(EmailServiceTestSuite))
}