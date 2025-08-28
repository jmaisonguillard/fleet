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
	Folder      string            `toml:"folder,omitempty" yaml:"folder,omitempty" json:"folder,omitempty"`
	Password    string            `toml:"password,omitempty" yaml:"password,omitempty" json:"password,omitempty"`
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