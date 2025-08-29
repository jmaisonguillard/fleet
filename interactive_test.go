package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type InteractiveSuite struct {
	suite.Suite
	tempDir string
}

func (s *InteractiveSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "fleet-interactive-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Change to temp directory
	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)
}

func (s *InteractiveSuite) TearDownTest() {
	// Clean up temp directory
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *InteractiveSuite) TestNewInteractiveBuilder() {
	builder := NewInteractiveBuilder()
	s.NotNil(builder)
	s.Equal("", builder.config.Project)
	s.Empty(builder.config.Services)
}

func (s *InteractiveSuite) TestSaveConfig() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "test-project"
	builder.config.Services = []Service{
		{
			Name:  "web",
			Image: "nginx:alpine",
			Port:  8080,
		},
		{
			Name:     "db",
			Database: "postgres:15",
			Port:     5432,
		},
	}

	// Save config
	err := builder.SaveConfig("test.toml")
	s.NoError(err)

	// Verify file exists
	_, err = os.Stat("test.toml")
	s.NoError(err)

	// Load and verify config
	loaded, err := loadConfig("test.toml")
	s.NoError(err)
	s.Equal("test-project", loaded.Project)
	s.Len(loaded.Services, 2)
	s.Equal("web", loaded.Services[0].Name)
	s.Equal("nginx:alpine", loaded.Services[0].Image)
	s.Equal(8080, loaded.Services[0].Port)
	s.Equal("db", loaded.Services[1].Name)
	s.Equal("postgres:15", loaded.Services[1].Database)
	s.Equal(5432, loaded.Services[1].Port)
}

func (s *InteractiveSuite) TestConfigWithPHPService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "php-app"
	builder.config.Services = []Service{
		{
			Name:      "web",
			Runtime:   "php:8.4",
			Framework: "laravel",
			Folder:    "./web",
			Port:      8080,
			Domain:    "myapp.test",
			SSL:       true,
			Debug:     true,
			Reverb:    true,
		},
	}

	// Save config
	err := builder.SaveConfig("php.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("php.toml")
	s.NoError(err)
	s.Equal("php-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("web", svc.Name)
	s.Equal("php:8.4", svc.Runtime)
	s.Equal("laravel", svc.Framework)
	s.Equal("./web", svc.Folder)
	s.Equal(8080, svc.Port)
	s.Equal("myapp.test", svc.Domain)
	s.True(svc.SSL)
	s.True(svc.Debug)
	s.True(svc.Reverb)
}

func (s *InteractiveSuite) TestConfigWithDatabaseService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "db-app"
	builder.config.Services = []Service{
		{
			Name:                 "database",
			Database:             "postgres:15",
			DatabaseName:         "mydb",
			DatabaseUser:         "dbuser",
			DatabasePassword:     "secret",
			DatabaseRootPassword: "rootsecret",
			DatabaseExtensions:   []string{"postgis", "pgvector"},
			Port:                 5432,
		},
	}

	// Save config
	err := builder.SaveConfig("db.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("db.toml")
	s.NoError(err)
	s.Equal("db-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("database", svc.Name)
	s.Equal("postgres:15", svc.Database)
	s.Equal("mydb", svc.DatabaseName)
	s.Equal("dbuser", svc.DatabaseUser)
	s.Equal("secret", svc.DatabasePassword)
	s.Equal("rootsecret", svc.DatabaseRootPassword)
	s.Contains(svc.DatabaseExtensions, "postgis")
	s.Contains(svc.DatabaseExtensions, "pgvector")
	s.Equal(5432, svc.Port)
}

func (s *InteractiveSuite) TestConfigWithCacheService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "cache-app"
	builder.config.Services = []Service{
		{
			Name:          "cache",
			Cache:         "redis:7.2",
			CachePassword: "redispass",
			Port:          6379,
		},
	}

	// Save config
	err := builder.SaveConfig("cache.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("cache.toml")
	s.NoError(err)
	s.Equal("cache-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("cache", svc.Name)
	s.Equal("redis:7.2", svc.Cache)
	s.Equal("redispass", svc.CachePassword)
	s.Equal(6379, svc.Port)
}

func (s *InteractiveSuite) TestConfigWithSearchService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "search-app"
	builder.config.Services = []Service{
		{
			Name:            "search",
			Search:          "meilisearch:1.6",
			SearchMasterKey: "masterkey",
			Port:            7700,
		},
	}

	// Save config
	err := builder.SaveConfig("search.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("search.toml")
	s.NoError(err)
	s.Equal("search-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("search", svc.Name)
	s.Equal("meilisearch:1.6", svc.Search)
	s.Equal("masterkey", svc.SearchMasterKey)
	s.Equal(7700, svc.Port)
}

func (s *InteractiveSuite) TestConfigWithEmailService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "email-app"
	builder.config.Services = []Service{
		{
			Name:          "mail",
			Email:         "mailpit:1.20",
			EmailUsername: "mailuser",
			EmailPassword: "mailpass",
			Port:          8025,
		},
	}

	// Save config
	err := builder.SaveConfig("email.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("email.toml")
	s.NoError(err)
	s.Equal("email-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("mail", svc.Name)
	s.Equal("mailpit:1.20", svc.Email)
	s.Equal("mailuser", svc.EmailUsername)
	s.Equal("mailpass", svc.EmailPassword)
	s.Equal(8025, svc.Port)
}

func (s *InteractiveSuite) TestConfigWithCustomService() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "custom-app"
	builder.config.Services = []Service{
		{
			Name:    "api",
			Build:   "./api",
			Port:    3000,
			Folder:  "./api",
			Command: "npm start",
			Environment: map[string]string{
				"NODE_ENV": "production",
				"PORT":     "3000",
			},
			Needs: []string{"database", "cache"},
		},
	}

	// Save config
	err := builder.SaveConfig("custom.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("custom.toml")
	s.NoError(err)
	s.Equal("custom-app", loaded.Project)
	s.Len(loaded.Services, 1)
	
	svc := loaded.Services[0]
	s.Equal("api", svc.Name)
	s.Equal("./api", svc.Build)
	s.Equal(3000, svc.Port)
	s.Equal("./api", svc.Folder)
	s.Equal("npm start", svc.Command)
	s.Equal("production", svc.Environment["NODE_ENV"])
	s.Equal("3000", svc.Environment["PORT"])
	s.Contains(svc.Needs, "database")
	s.Contains(svc.Needs, "cache")
}

func (s *InteractiveSuite) TestComplexMultiServiceConfig() {
	builder := NewInteractiveBuilder()
	builder.config.Project = "full-stack-app"
	builder.config.Services = []Service{
		{
			Name:      "frontend",
			Runtime:   "php:8.4",
			Framework: "laravel",
			Folder:    "./frontend",
			Port:      8080,
			Domain:    "app.test",
			SSL:       true,
			Reverb:    true,
		},
		{
			Name:                 "database",
			Database:             "postgres:15",
			DatabaseName:         "appdb",
			DatabaseUser:         "appuser",
			DatabasePassword:     "dbpass",
			DatabaseRootPassword: "rootpass",
			DatabaseExtensions:   []string{"uuid-ossp"},
			Port:                 5432,
		},
		{
			Name:          "cache",
			Cache:         "redis:7.2",
			CachePassword: "cachepass",
			Port:          6379,
		},
		{
			Name:            "search",
			Search:          "meilisearch:1.6",
			SearchMasterKey: "searchkey",
			Port:            7700,
		},
		{
			Name:  "mail",
			Email: "mailpit:1.20",
			Port:  8025,
		},
	}

	// Save config
	err := builder.SaveConfig("fullstack.toml")
	s.NoError(err)

	// Load and verify
	loaded, err := loadConfig("fullstack.toml")
	s.NoError(err)
	s.Equal("full-stack-app", loaded.Project)
	s.Len(loaded.Services, 5)
	
	// Verify each service
	for _, svc := range loaded.Services {
		switch svc.Name {
		case "frontend":
			s.Equal("php:8.4", svc.Runtime)
			s.Equal("laravel", svc.Framework)
			s.True(svc.SSL)
			s.True(svc.Reverb)
		case "database":
			s.Equal("postgres:15", svc.Database)
			s.Contains(svc.DatabaseExtensions, "uuid-ossp")
		case "cache":
			s.Equal("redis:7.2", svc.Cache)
		case "search":
			s.Equal("meilisearch:1.6", svc.Search)
		case "mail":
			s.Equal("mailpit:1.20", svc.Email)
		}
	}
}

func TestInteractiveSuite(t *testing.T) {
	suite.Run(t, new(InteractiveSuite))
}