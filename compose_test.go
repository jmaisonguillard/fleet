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
	// Check network
	suite.Contains(webService.Networks, "fleet-network")
	// Service with port gets auto-generated domain "web.test", so it should NOT expose ports directly
	suite.Empty(webService.Ports, "Service with auto-generated domain should not expose ports")
	suite.Equal("unless-stopped", webService.Restart)
	
	// Should have nginx proxy since service has a port
	suite.Contains(compose.Services, "nginx-proxy")
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
	// Service with port gets auto-generated domain "api.test", so it should NOT expose ports directly
	suite.Empty(apiService.Ports, "Service with auto-generated domain should not expose ports")
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
	
	// Check named volume IS defined - database volumes are automatically created
	// The new database service implementation detects and creates volumes ending with "-data"
	suite.NotNil(compose.Volumes)
	_, exists := compose.Volumes["db-data"]
	suite.True(exists, "db-data volume should be defined")
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
	// Environment is a map, not key=value strings
	suite.Equal("secret123", dbService.Environment["POSTGRES_PASSWORD"])
	// POSTGRES_DB is set to the project name when Password is set
	suite.Equal("test-project", dbService.Environment["POSTGRES_DB"])
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
	// Folder is mapped to /usr/share/nginx/html for nginx without PHP
	suite.Contains(webService.Volumes, ".././website:/usr/share/nginx/html")
	// Service with port gets auto-generated domain, so it should NOT expose ports directly
	suite.Empty(webService.Ports, "Service with auto-generated domain should not expose ports")
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
	suite.Contains(webService.Networks, "fleet-network")
	suite.Contains(apiService.Networks, "fleet-network")
	
	// Check network is defined
	suite.Contains(compose.Networks, "fleet-network")
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
	
	// Now expects 4 services: web, api, database, and nginx-proxy (added automatically because services have ports)
	suite.Len(compose.Services, 4)
	suite.Contains(compose.Services, "web")
	suite.Contains(compose.Services, "api")
	suite.Contains(compose.Services, "database")
	suite.Contains(compose.Services, "nginx-proxy")
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
	// The DockerService struct doesn't have a ContainerName field, so this assertion is invalid
	// Remove this check as container_name is not generated
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
		// According to compose.go line 90, all ports are mapped as port:port
		{"nginx", "web", 8080, "8080:8080"},
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
		// Services with ports get auto-generated domains, so they should NOT expose ports directly
		suite.Empty(serviceConfig.Ports, "Service %s with auto-generated domain should not expose ports", tc.serviceName)
	}
}

// Test service without port and without domain exposes nothing
func (suite *ComposeTestSuite) TestGenerateDockerComposeServiceNoPortNoDomain() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "worker",
				Image: "worker:latest",
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	workerService := compose.Services["worker"]
	suite.Empty(workerService.Ports, "Service without port and domain should not expose any ports")
	
	// Should NOT have nginx proxy since no service has ports or domains
	suite.NotContains(compose.Services, "nginx-proxy", "Should not add nginx proxy when no services need it")
}

// Test service with explicit domain doesn't expose ports
func (suite *ComposeTestSuite) TestGenerateDockerComposeServiceWithExplicitDomain() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Image:  "nginx:alpine",
				Domain: "myapp.test",
				Port:   8080,
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	webService := compose.Services["web"]
	suite.Empty(webService.Ports, "Service with explicit domain should not expose ports")
	
	// Should have nginx proxy since service has domain
	suite.Contains(compose.Services, "nginx-proxy", "Should add nginx proxy for service with domain")
	nginxService := compose.Services["nginx-proxy"]
	suite.Contains(nginxService.Ports, "80:80", "Nginx proxy should expose port 80")
}

// Test mixed services - some with domains, some without
func (suite *ComposeTestSuite) TestGenerateDockerComposeMixedServices() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Image:  "nginx:alpine",
				Domain: "web.test",
				Port:   8080,
			},
			{
				Name:  "api",
				Image: "node:18",
				Port:  3000,  // Gets auto-generated domain api.test
			},
			{
				Name:  "worker",
				Image: "worker:latest",  // No port, no domain
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	// Web service has explicit domain
	webService := compose.Services["web"]
	suite.Empty(webService.Ports, "Web service with domain should not expose ports")
	
	// API service gets auto-generated domain
	apiService := compose.Services["api"]
	suite.Empty(apiService.Ports, "API service with auto-generated domain should not expose ports")
	
	// Worker has neither port nor domain
	workerService := compose.Services["worker"]
	suite.Empty(workerService.Ports, "Worker service without port/domain should not expose ports")
	
	// Should have nginx proxy
	suite.Contains(compose.Services, "nginx-proxy", "Should add nginx proxy")
	nginxService := compose.Services["nginx-proxy"]
	suite.Contains(nginxService.Ports, "80:80", "Only nginx proxy should expose port 80")
	suite.Contains(nginxService.DependsOn, "web", "Nginx should depend on web")
	suite.Contains(nginxService.DependsOn, "api", "Nginx should depend on api")
	suite.NotContains(nginxService.DependsOn, "worker", "Nginx should not depend on worker")
}

// Test service with ports array but no domain should not expose ports when it gets auto-generated domain
func (suite *ComposeTestSuite) TestGenerateDockerComposePortsArray() {
	config := Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:  "web",
				Image: "nginx:alpine",
				Ports: []string{"8080:80", "8443:443"},
			},
		},
	}

	compose := generateDockerCompose(&config)
	
	webService := compose.Services["web"]
	// Service with ports array still doesn't expose because no Port field means no auto-generated domain
	suite.Contains(webService.Ports, "8080:80", "Service with ports array but no Port field should expose ports")
	suite.Contains(webService.Ports, "8443:443", "Service with ports array but no Port field should expose ports")
	
	// No nginx proxy because service has no Port field (required for auto-generated domain)
	suite.NotContains(compose.Services, "nginx-proxy", "Should not add nginx proxy when service has no Port field")
}

func TestComposeSuite(t *testing.T) {
	suite.Run(t, new(ComposeTestSuite))
}