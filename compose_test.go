package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

// ComposeTestSuite tests the Docker Compose generation functionality
type ComposeTestSuite struct {
	suite.Suite
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeBasic() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				Port:  8080,
			},
		},
	}

	compose := generateDockerCompose(&config)
	suite.Require().NotNil(compose)

	// Check version
	suite.Equal("3.8", compose.Version)

	// Check services
	suite.Contains(compose.Services, "web")
	webService := compose.Services["web"]
	suite.Equal("nginx:alpine", webService.Image)
	// Container name is generated, check it exists
	suite.Contains(webService.Networks, "test-project-network")
	suite.Contains(webService.Ports, "8080:80")
	suite.Equal("unless-stopped", webService.Restart)
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeWithBuild() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "api",
				Build: "./api",
				Port:  3000,
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	apiService := compose.Services["api"]
	suite.Equal("./api", apiService.Build)
	suite.Empty(apiService.Image)
	suite.Contains(apiService.Ports, "3000:3000")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeWithVolumes() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "database",
				Image: "postgres:15",
				Port:  5432,
				Volumes: []string{
					"db-data:/var/lib/postgresql/data",
					"./init.sql:/docker-entrypoint-initdb.d/init.sql",
				},
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	dbService := compose.Services["database"]
	suite.Len(dbService.Volumes, 2)
	suite.Contains(dbService.Volumes, "db-data:/var/lib/postgresql/data")
	suite.Contains(dbService.Volumes, "./init.sql:/docker-entrypoint-initdb.d/init.sql")
	
	// Check named volume is defined
	suite.Contains(compose.Volumes, "db-data")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeWithEnvironment() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:     "database",
				Image:    "postgres:15",
				Port:     5432,
				Password: "secret123",
				Environment: map[string]string{
					"POSTGRES_DB": "testdb",
				},
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	dbService := compose.Services["database"]
	suite.Contains(dbService.Environment, "POSTGRES_PASSWORD=secret123")
	suite.Contains(dbService.Environment, "POSTGRES_DB=testdb")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeWithDependencies() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "database",
				Image: "postgres:15",
				Port:  5432,
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,
				Needs: []string{"database"},
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	apiService := compose.Services["api"]
	suite.Len(apiService.DependsOn, 1)
	suite.Contains(apiService.DependsOn, "database")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeWithFolder() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Image:  "nginx:alpine",
				Port:   8080,
				Folder: "./website",
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	webService := compose.Services["web"]
	suite.Contains(webService.Volumes, "./website:/usr/share/nginx/html")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeNetworks() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				Port:  8080,
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	// Check that services are on the same network
	webService := compose.Services["web"]
	apiService := compose.Services["api"]
	suite.Contains(webService.Networks, "test-project-network")
	suite.Contains(apiService.Networks, "test-project-network")
	
	// Check network is defined
	suite.Contains(compose.Networks, "test-project-network")
}

func (suite *ComposeTestSuite) TestGenerateDockerComposeMultipleServices() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				Port:  8080,
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,
			},
			{
				Name:  "database",
				Image: "postgres:15",
				Port:  5432,
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	suite.Len(compose.Services, 3)
	suite.Contains(compose.Services, "web")
	suite.Contains(compose.Services, "api")
	suite.Contains(compose.Services, "database")
}

func (suite *ComposeTestSuite) TestWriteDockerComposeYAML() {
	compose := DockerCompose{
		Version: "3.8",
		Services: map[string]DockerService{
			"web": {
				Image:    "nginx:alpine",
				Ports:    []string{"8080:80"},
				Networks: []string{"test-network"},
				Restart:  "unless-stopped",
			},
		},
		Networks: map[string]DockerNetwork{
			"test-network": {Driver: "bridge"},
		},
		Volumes: map[string]DockerVolume{},
	}

	// Convert to YAML and check structure
	yamlBytes, err := yaml.Marshal(compose)
	suite.Require().NoError(err)

	yamlStr := string(yamlBytes)
	
	// Check key elements are present
	suite.Contains(yamlStr, "version: \"3.8\"")
	suite.Contains(yamlStr, "services:")
	suite.Contains(yamlStr, "web:")
	suite.Contains(yamlStr, "image: nginx:alpine")
	suite.Contains(yamlStr, "container_name: test-web")
	suite.Contains(yamlStr, "networks:")
	suite.Contains(yamlStr, "test-network:")
}

func (suite *ComposeTestSuite) TestDetectPortForService() {
	testCases := []struct {
		image        string
		serviceName  string
		configPort   int
		expectedPort string
	}{
		{"nginx", "web", 8080, "8080:80"},
		{"postgres:15", "db", 5432, "5432:5432"},
		{"redis:7", "cache", 6379, "6379:6379"},
		{"mysql:8", "database", 3306, "3306:3306"},
		{"node:18", "api", 3000, "3000:3000"},
		{"custom-image", "service", 9000, "9000:9000"},
	}

	for _, tc := range testCases {
		service := Service{
			Name:  tc.serviceName,
			Image: tc.image,
			Port:  tc.configPort,
		}

		config := Config{
			Project:  "test",
			Services: []Service{service},
		}
		compose := generateDockerCompose(&config)

		serviceConfig := compose.Services[tc.serviceName]
		suite.Contains(serviceConfig.Ports[0], tc.expectedPort)
	}
}

func TestComposeSuite(t *testing.T) {
	suite.Run(t, new(ComposeTestSuite))
}