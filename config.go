package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Project  string    `toml:"project" yaml:"project" json:"project"`
	Services []Service `toml:"services" yaml:"services" json:"services"`
}

type Service struct {
	Name        string            `toml:"name" yaml:"name" json:"name"`
	Image       string            `toml:"image" yaml:"image" json:"image"`
	Build       string            `toml:"build,omitempty" yaml:"build,omitempty" json:"build,omitempty"`
	Port        int               `toml:"port,omitempty" yaml:"port,omitempty" json:"port,omitempty"`
	Ports       []string          `toml:"ports,omitempty" yaml:"ports,omitempty" json:"ports,omitempty"`
	Domain      string            `toml:"domain,omitempty" yaml:"domain,omitempty" json:"domain,omitempty"`
	Runtime     string            `toml:"runtime,omitempty" yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Framework   string            `toml:"framework,omitempty" yaml:"framework,omitempty" json:"framework,omitempty"`
	Folder      string            `toml:"folder,omitempty" yaml:"folder,omitempty" json:"folder,omitempty"`
	Password    string            `toml:"password,omitempty" yaml:"password,omitempty" json:"password,omitempty"`
	Database    string            `toml:"database,omitempty" yaml:"database,omitempty" json:"database,omitempty"`
	DatabaseName string           `toml:"database_name,omitempty" yaml:"database_name,omitempty" json:"database_name,omitempty"`
	DatabaseUser string           `toml:"database_user,omitempty" yaml:"database_user,omitempty" json:"database_user,omitempty"`
	DatabasePassword string       `toml:"database_password,omitempty" yaml:"database_password,omitempty" json:"database_password,omitempty"`
	DatabaseRootPassword string   `toml:"database_root_password,omitempty" yaml:"database_root_password,omitempty" json:"database_root_password,omitempty"`
	Cache           string        `toml:"cache,omitempty" yaml:"cache,omitempty" json:"cache,omitempty"`
	CachePassword   string        `toml:"cache_password,omitempty" yaml:"cache_password,omitempty" json:"cache_password,omitempty"`
	CacheMaxMemory  string        `toml:"cache_max_memory,omitempty" yaml:"cache_max_memory,omitempty" json:"cache_max_memory,omitempty"`
	Search          string        `toml:"search,omitempty" yaml:"search,omitempty" json:"search,omitempty"`
	SearchApiKey    string        `toml:"search_api_key,omitempty" yaml:"search_api_key,omitempty" json:"search_api_key,omitempty"`
	SearchMasterKey string        `toml:"search_master_key,omitempty" yaml:"search_master_key,omitempty" json:"search_master_key,omitempty"`
	Compat          string        `toml:"compat,omitempty" yaml:"compat,omitempty" json:"compat,omitempty"`
	CompatAccessKey string        `toml:"compat_access_key,omitempty" yaml:"compat_access_key,omitempty" json:"compat_access_key,omitempty"`
	CompatSecretKey string        `toml:"compat_secret_key,omitempty" yaml:"compat_secret_key,omitempty" json:"compat_secret_key,omitempty"`
	CompatRegion    string        `toml:"compat_region,omitempty" yaml:"compat_region,omitempty" json:"compat_region,omitempty"`
	Email           string        `toml:"email,omitempty" yaml:"email,omitempty" json:"email,omitempty"`
	EmailUsername   string        `toml:"email_username,omitempty" yaml:"email_username,omitempty" json:"email_username,omitempty"`
	EmailPassword   string        `toml:"email_password,omitempty" yaml:"email_password,omitempty" json:"email_password,omitempty"`
	Reverb          bool          `toml:"reverb,omitempty" yaml:"reverb,omitempty" json:"reverb,omitempty"`
	ReverbHost      string        `toml:"reverb_host,omitempty" yaml:"reverb_host,omitempty" json:"reverb_host,omitempty"`
	ReverbPort      int           `toml:"reverb_port,omitempty" yaml:"reverb_port,omitempty" json:"reverb_port,omitempty"`
	ReverbAppId     string        `toml:"reverb_app_id,omitempty" yaml:"reverb_app_id,omitempty" json:"reverb_app_id,omitempty"`
	ReverbAppKey    string        `toml:"reverb_app_key,omitempty" yaml:"reverb_app_key,omitempty" json:"reverb_app_key,omitempty"`
	ReverbAppSecret string        `toml:"reverb_app_secret,omitempty" yaml:"reverb_app_secret,omitempty" json:"reverb_app_secret,omitempty"`
	SSL             bool          `toml:"ssl,omitempty" yaml:"ssl,omitempty" json:"ssl,omitempty"`
	SSLPort         int           `toml:"ssl_port,omitempty" yaml:"ssl_port,omitempty" json:"ssl_port,omitempty"`
	Debug           bool          `toml:"debug,omitempty" yaml:"debug,omitempty" json:"debug,omitempty"`
	DebugPort       int           `toml:"debug_port,omitempty" yaml:"debug_port,omitempty" json:"debug_port,omitempty"`
	DatabaseExtensions []string   `toml:"database_extensions,omitempty" yaml:"database_extensions,omitempty" json:"database_extensions,omitempty"`
	Environment map[string]string `toml:"env,omitempty" yaml:"env,omitempty" json:"env,omitempty"`
	Volumes     []string          `toml:"volumes,omitempty" yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Needs       []string          `toml:"needs,omitempty" yaml:"needs,omitempty" json:"needs,omitempty"`
	Command     string            `toml:"command,omitempty" yaml:"command,omitempty" json:"command,omitempty"`
	HealthCheck HealthCheck       `toml:"health,omitempty" yaml:"health,omitempty" json:"health,omitempty"`
}

type HealthCheck struct {
	Test     string `toml:"test,omitempty" yaml:"test,omitempty" json:"test,omitempty"`
	Interval string `toml:"interval,omitempty" yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout  string `toml:"timeout,omitempty" yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries  int    `toml:"retries,omitempty" yaml:"retries,omitempty" json:"retries,omitempty"`
}

func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	ext := filepath.Ext(filename)

	switch ext {
	case ".toml":
		err = toml.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = json.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config format: %s (use .toml, .yaml, .yml, or .json)", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Project == "" {
		config.Project = "fleet-project"
	}

	if len(config.Services) == 0 {
		return fmt.Errorf("no services defined")
	}

	for i, svc := range config.Services {
		if svc.Name == "" {
			return fmt.Errorf("service #%d: name is required", i+1)
		}
		if svc.Image == "" && svc.Build == "" {
			return fmt.Errorf("service %s: either 'image' or 'build' is required", svc.Name)
		}
	}

	return nil
}