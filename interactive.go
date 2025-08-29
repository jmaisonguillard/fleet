package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pelletier/go-toml/v2"
)

// InteractiveBuilder handles interactive configuration building
type InteractiveBuilder struct {
	config Config
}

// NewInteractiveBuilder creates a new interactive builder
func NewInteractiveBuilder() *InteractiveBuilder {
	return &InteractiveBuilder{
		config: Config{
			Services: []Service{},
		},
	}
}

// Build starts the interactive configuration building process
func (ib *InteractiveBuilder) Build() (*Config, error) {
	fmt.Println("🚀 Fleet Interactive Configuration Builder")
	fmt.Println("==========================================")
	fmt.Println()

	// Get project name
	if err := ib.promptProjectName(); err != nil {
		return nil, err
	}

	// Main menu loop
	for {
		action := ""
		prompt := &survey.Select{
			Message: "What would you like to do?",
			Options: []string{
				"Add a service",
				"View current configuration",
				"Save and exit",
				"Cancel",
			},
		}

		if err := survey.AskOne(prompt, &action); err != nil {
			if err == terminal.InterruptErr {
				fmt.Println("\n❌ Configuration cancelled")
				return nil, err
			}
			return nil, err
		}

		switch action {
		case "Add a service":
			if err := ib.addService(); err != nil {
				if err == terminal.InterruptErr {
					continue
				}
				return nil, err
			}
		case "View current configuration":
			ib.displayConfig()
		case "Save and exit":
			if len(ib.config.Services) == 0 {
				fmt.Println("⚠️  No services configured. Please add at least one service.")
				continue
			}
			return &ib.config, nil
		case "Cancel":
			fmt.Println("❌ Configuration cancelled")
			return nil, fmt.Errorf("cancelled by user")
		}
	}
}

func (ib *InteractiveBuilder) promptProjectName() error {
	projectName := ""
	prompt := &survey.Input{
		Message: "Enter project name:",
		Default: "my-app",
	}

	if err := survey.AskOne(prompt, &projectName, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	ib.config.Project = projectName
	return nil
}

func (ib *InteractiveBuilder) addService() error {
	serviceType := ""
	prompt := &survey.Select{
		Message: "Select service type:",
		Options: []string{
			"Web Application",
			"Database",
			"Cache",
			"Search Engine",
			"Email Service",
			"Custom Service",
		},
	}

	if err := survey.AskOne(prompt, &serviceType); err != nil {
		return err
	}

	switch serviceType {
	case "Web Application":
		return ib.addWebService()
	case "Database":
		return ib.addDatabaseService()
	case "Cache":
		return ib.addCacheService()
	case "Search Engine":
		return ib.addSearchService()
	case "Email Service":
		return ib.addEmailService()
	case "Custom Service":
		return ib.addCustomService()
	}

	return nil
}

func (ib *InteractiveBuilder) addWebService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
		Default: "web",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Framework selection
	framework := ""
	if err := survey.AskOne(&survey.Select{
		Message: "Select framework/runtime:",
		Options: []string{
			"PHP (Laravel)",
			"PHP (Symfony)",
			"PHP (WordPress)",
			"PHP (Custom)",
			"Node.js",
			"Python",
			"Ruby",
			"Static Files (Nginx)",
			"Custom Docker Image",
		},
	}, &framework); err != nil {
		return err
	}

	switch framework {
	case "PHP (Laravel)":
		service.Runtime = "php:8.4"
		service.Framework = "laravel"
		service.Folder = "./" + service.Name
	case "PHP (Symfony)":
		service.Runtime = "php:8.4"
		service.Framework = "symfony"
		service.Folder = "./" + service.Name
	case "PHP (WordPress)":
		service.Runtime = "php:8.1"
		service.Framework = "wordpress"
		service.Folder = "./" + service.Name
	case "PHP (Custom)":
		phpVersion := ""
		if err := survey.AskOne(&survey.Select{
			Message: "PHP version:",
			Options: []string{"8.4", "8.3", "8.2", "8.1", "8.0", "7.4"},
			Default: "8.4",
		}, &phpVersion); err != nil {
			return err
		}
		service.Runtime = "php:" + phpVersion
		service.Folder = "./" + service.Name
	case "Node.js":
		service.Image = "node:20-alpine"
		service.Folder = "./" + service.Name
		service.Command = "npm start"
	case "Python":
		service.Image = "python:3.11-slim"
		service.Folder = "./" + service.Name
		service.Command = "python app.py"
	case "Ruby":
		service.Image = "ruby:3.2-slim"
		service.Folder = "./" + service.Name
		service.Command = "bundle exec rails server"
	case "Static Files (Nginx)":
		service.Image = "nginx:alpine"
		service.Folder = "./" + service.Name
	case "Custom Docker Image":
		if err := survey.AskOne(&survey.Input{
			Message: "Docker image:",
		}, &service.Image, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Port configuration
	var port int
	if err := survey.AskOne(&survey.Input{
		Message: "Port number:",
		Default: "8080",
	}, &port); err != nil {
		return err
	}
	service.Port = port

	// Domain configuration
	useDomain := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Configure a domain?",
		Default: true,
	}, &useDomain); err != nil {
		return err
	}

	if useDomain {
		domain := ""
		if err := survey.AskOne(&survey.Input{
			Message: "Domain (e.g., myapp.test):",
			Default: service.Name + ".test",
		}, &domain); err != nil {
			return err
		}
		service.Domain = domain

		// SSL configuration
		useSSL := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Enable SSL?",
			Default: false,
		}, &useSSL); err != nil {
			return err
		}
		service.SSL = useSSL
	}

	// Laravel specific features
	if service.Framework == "laravel" {
		useReverb := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Enable Laravel Reverb (WebSockets)?",
			Default: false,
		}, &useReverb); err != nil {
			return err
		}
		service.Reverb = useReverb
	}

	// Debug mode
	if service.Runtime != "" && strings.HasPrefix(service.Runtime, "php") {
		useDebug := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Enable Xdebug?",
			Default: false,
		}, &useDebug); err != nil {
			return err
		}
		service.Debug = useDebug
	}

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added %s service: %s\n\n", framework, service.Name)
	return nil
}

func (ib *InteractiveBuilder) addDatabaseService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
		Default: "database",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Database type
	dbType := ""
	if err := survey.AskOne(&survey.Select{
		Message: "Database type:",
		Options: []string{
			"MySQL 8.0",
			"MySQL 5.7",
			"PostgreSQL 15",
			"PostgreSQL 14",
			"PostgreSQL 13",
			"MariaDB 10.11",
			"MariaDB 10.6",
			"MongoDB 7",
			"MongoDB 6",
		},
	}, &dbType); err != nil {
		return err
	}

	// Parse database type and version
	parts := strings.Split(dbType, " ")
	dbEngine := strings.ToLower(parts[0])
	dbVersion := parts[1]

	// Handle PostgreSQL extensions for specific versions
	if dbEngine == "postgresql" {
		service.Database = "postgres:" + dbVersion

		// Ask about extensions
		useExtensions := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Add PostgreSQL extensions?",
			Default: false,
		}, &useExtensions); err != nil {
			return err
		}

		if useExtensions {
			extensions := []string{}
			extensionPrompt := &survey.MultiSelect{
				Message: "Select extensions:",
				Options: []string{
					"postgis (Spatial database)",
					"pgvector (Vector similarity)",
					"uuid-ossp (UUID generation)",
					"hstore (Key-value store)",
					"pg_trgm (Trigram matching)",
				},
			}
			if err := survey.AskOne(extensionPrompt, &extensions); err != nil {
				return err
			}

			// Convert display names to actual extension names
			var actualExtensions []string
			for _, ext := range extensions {
				switch {
				case strings.HasPrefix(ext, "postgis"):
					actualExtensions = append(actualExtensions, "postgis")
				case strings.HasPrefix(ext, "pgvector"):
					actualExtensions = append(actualExtensions, "pgvector")
				case strings.HasPrefix(ext, "uuid-ossp"):
					actualExtensions = append(actualExtensions, "uuid-ossp")
				case strings.HasPrefix(ext, "hstore"):
					actualExtensions = append(actualExtensions, "hstore")
				case strings.HasPrefix(ext, "pg_trgm"):
					actualExtensions = append(actualExtensions, "pg_trgm")
				}
			}
			service.DatabaseExtensions = actualExtensions
		}
	} else {
		service.Database = dbEngine + ":" + dbVersion
	}

	// Database configuration
	if err := survey.AskOne(&survey.Input{
		Message: "Database name:",
		Default: ib.config.Project,
	}, &service.DatabaseName); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{
		Message: "Database user:",
		Default: "dbuser",
	}, &service.DatabaseUser); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{
		Message: "Database password:",
		Default: "changeme",
	}, &service.DatabasePassword); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{
		Message: "Root password:",
		Default: "rootpass",
	}, &service.DatabaseRootPassword); err != nil {
		return err
	}

	// Port
	defaultPort := 3306
	if dbEngine == "postgresql" || dbEngine == "postgres" {
		defaultPort = 5432
	} else if dbEngine == "mongodb" {
		defaultPort = 27017
	}

	var port int
	if err := survey.AskOne(&survey.Input{
		Message: "Port:",
		Default: fmt.Sprintf("%d", defaultPort),
	}, &port); err != nil {
		return err
	}
	service.Port = port

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added database service: %s (%s)\n\n", service.Name, dbType)
	return nil
}

func (ib *InteractiveBuilder) addCacheService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
		Default: "cache",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Cache type
	cacheType := ""
	if err := survey.AskOne(&survey.Select{
		Message: "Cache type:",
		Options: []string{
			"Redis 7.4",
			"Redis 7.2",
			"Redis 7.0",
			"Redis 6.2",
			"Memcached 1.6",
		},
	}, &cacheType); err != nil {
		return err
	}

	// Parse cache type
	parts := strings.Split(cacheType, " ")
	cacheEngine := strings.ToLower(parts[0])
	cacheVersion := parts[1]
	service.Cache = cacheEngine + ":" + cacheVersion

	// Port
	defaultPort := 6379
	if cacheEngine == "memcached" {
		defaultPort = 11211
	}

	var port int
	if err := survey.AskOne(&survey.Input{
		Message: "Port:",
		Default: fmt.Sprintf("%d", defaultPort),
	}, &port); err != nil {
		return err
	}
	service.Port = port

	// Redis password (not for memcached)
	if cacheEngine == "redis" {
		usePassword := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Set Redis password?",
			Default: true,
		}, &usePassword); err != nil {
			return err
		}

		if usePassword {
			if err := survey.AskOne(&survey.Input{
				Message: "Redis password:",
				Default: "changeme",
			}, &service.CachePassword); err != nil {
				return err
			}
		}
	}

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added cache service: %s (%s)\n\n", service.Name, cacheType)
	return nil
}

func (ib *InteractiveBuilder) addSearchService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
		Default: "search",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Search engine type
	searchType := ""
	if err := survey.AskOne(&survey.Select{
		Message: "Search engine:",
		Options: []string{
			"Meilisearch 1.6",
			"Meilisearch 1.5",
			"Meilisearch 1.4",
			"Typesense 27.1",
			"Typesense 26.0",
		},
	}, &searchType); err != nil {
		return err
	}

	// Parse search type
	parts := strings.Split(searchType, " ")
	searchEngine := strings.ToLower(parts[0])
	searchVersion := parts[1]
	service.Search = searchEngine + ":" + searchVersion

	// Port
	defaultPort := 7700
	if searchEngine == "typesense" {
		defaultPort = 8108
	}

	var port int
	if err := survey.AskOne(&survey.Input{
		Message: "Port:",
		Default: fmt.Sprintf("%d", defaultPort),
	}, &port); err != nil {
		return err
	}
	service.Port = port

	// API Keys
	if searchEngine == "meilisearch" {
		if err := survey.AskOne(&survey.Input{
			Message: "Master key (leave empty for dev mode):",
		}, &service.SearchMasterKey); err != nil {
			return err
		}
	} else if searchEngine == "typesense" {
		if err := survey.AskOne(&survey.Input{
			Message: "API key:",
			Default: "changeme",
		}, &service.SearchApiKey, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added search service: %s (%s)\n\n", service.Name, searchType)
	return nil
}

func (ib *InteractiveBuilder) addEmailService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
		Default: "mail",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Email service type (currently only Mailpit)
	service.Email = "mailpit:1.20"

	// SMTP Port (we'll store this differently since there's no specific field)
	var smtpPort int
	if err := survey.AskOne(&survey.Input{
		Message: "SMTP port:",
		Default: "1025",
	}, &smtpPort); err != nil {
		return err
	}
	// Note: SMTP port will be handled in compose generation

	// Web UI Port
	var webPort int
	if err := survey.AskOne(&survey.Input{
		Message: "Web UI port:",
		Default: "8025",
	}, &webPort); err != nil {
		return err
	}
	service.Port = webPort

	// Authentication
	useAuth := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable SMTP authentication?",
		Default: false,
	}, &useAuth); err != nil {
		return err
	}

	if useAuth {
		if err := survey.AskOne(&survey.Input{
			Message: "SMTP username:",
			Default: "mailpit",
		}, &service.EmailUsername); err != nil {
			return err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "SMTP password:",
			Default: "secret",
		}, &service.EmailPassword); err != nil {
			return err
		}
	}

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added email service: %s (Mailpit)\n\n", service.Name)
	return nil
}

func (ib *InteractiveBuilder) addCustomService() error {
	service := Service{}

	// Service name
	if err := survey.AskOne(&survey.Input{
		Message: "Service name:",
	}, &service.Name, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Build or image
	buildType := ""
	if err := survey.AskOne(&survey.Select{
		Message: "How to build the service?",
		Options: []string{
			"Use Docker image",
			"Build from Dockerfile",
		},
	}, &buildType); err != nil {
		return err
	}

	if buildType == "Use Docker image" {
		if err := survey.AskOne(&survey.Input{
			Message: "Docker image (e.g., nginx:alpine):",
		}, &service.Image, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	} else {
		if err := survey.AskOne(&survey.Input{
			Message: "Build directory (with Dockerfile):",
			Default: "./" + service.Name,
		}, &service.Build, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Port
	usePort := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Expose a port?",
		Default: true,
	}, &usePort); err != nil {
		return err
	}

	if usePort {
		var port int
		if err := survey.AskOne(&survey.Input{
			Message: "Port number:",
			Default: "8080",
		}, &port); err != nil {
			return err
		}
		service.Port = port
	}

	// Mount folder
	mountFolder := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Mount a local folder?",
		Default: false,
	}, &mountFolder); err != nil {
		return err
	}

	if mountFolder {
		if err := survey.AskOne(&survey.Input{
			Message: "Local folder path:",
			Default: "./" + service.Name,
		}, &service.Folder); err != nil {
			return err
		}
	}

	// Environment variables
	addEnvVars := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Add environment variables?",
		Default: false,
	}, &addEnvVars); err != nil {
		return err
	}

	if addEnvVars {
		service.Environment = make(map[string]string)
		for {
			addMore := false
			key := ""
			value := ""

			if err := survey.AskOne(&survey.Input{
				Message: "Environment variable name (e.g., NODE_ENV):",
			}, &key); err != nil {
				return err
			}

			if err := survey.AskOne(&survey.Input{
				Message: fmt.Sprintf("Value for %s:", key),
			}, &value); err != nil {
				return err
			}

			service.Environment[key] = value

			if err := survey.AskOne(&survey.Confirm{
				Message: "Add another environment variable?",
				Default: false,
			}, &addMore); err != nil {
				return err
			}

			if !addMore {
				break
			}
		}
	}

	// Dependencies
	if len(ib.config.Services) > 0 {
		addDeps := false
		if err := survey.AskOne(&survey.Confirm{
			Message: "Does this service depend on other services?",
			Default: false,
		}, &addDeps); err != nil {
			return err
		}

		if addDeps {
			var serviceNames []string
			for _, svc := range ib.config.Services {
				serviceNames = append(serviceNames, svc.Name)
			}

			var deps []string
			depsPrompt := &survey.MultiSelect{
				Message: "Select dependencies:",
				Options: serviceNames,
			}
			if err := survey.AskOne(depsPrompt, &deps); err != nil {
				return err
			}
			service.Needs = deps
		}
	}

	ib.config.Services = append(ib.config.Services, service)
	fmt.Printf("✅ Added custom service: %s\n\n", service.Name)
	return nil
}

func (ib *InteractiveBuilder) displayConfig() {
	fmt.Println("\n📋 Current Configuration")
	fmt.Println("========================")
	fmt.Printf("Project: %s\n", ib.config.Project)
	fmt.Printf("Services: %d\n\n", len(ib.config.Services))

	for i, svc := range ib.config.Services {
		fmt.Printf("%d. %s\n", i+1, svc.Name)
		
		if svc.Image != "" {
			fmt.Printf("   Image: %s\n", svc.Image)
		}
		if svc.Build != "" {
			fmt.Printf("   Build: %s\n", svc.Build)
		}
		if svc.Runtime != "" {
			fmt.Printf("   Runtime: %s\n", svc.Runtime)
			if svc.Framework != "" {
				fmt.Printf("   Framework: %s\n", svc.Framework)
			}
		}
		if svc.Database != "" {
			fmt.Printf("   Database: %s\n", svc.Database)
		}
		if svc.Cache != "" {
			fmt.Printf("   Cache: %s\n", svc.Cache)
		}
		if svc.Search != "" {
			fmt.Printf("   Search: %s\n", svc.Search)
		}
		if svc.Email != "" {
			fmt.Printf("   Email: %s\n", svc.Email)
		}
		if svc.Port > 0 {
			fmt.Printf("   Port: %d\n", svc.Port)
		}
		if svc.Domain != "" {
			fmt.Printf("   Domain: %s\n", svc.Domain)
			if svc.SSL {
				fmt.Printf("   SSL: enabled\n")
			}
		}
		if len(svc.Needs) > 0 {
			fmt.Printf("   Dependencies: %s\n", strings.Join(svc.Needs, ", "))
		}
		fmt.Println()
	}
}

// SaveConfig saves the configuration to a TOML file
func (ib *InteractiveBuilder) SaveConfig(filename string) error {
	data, err := toml.Marshal(&ib.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}