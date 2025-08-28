package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NginxSSLSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *NginxSSLSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *NginxSSLSuite) TearDownTest() {
	suite.helper.Cleanup()
	// Clean up any .fleet directory created during tests
	os.RemoveAll(".fleet")
}

func (suite *NginxSSLSuite) TestNginxConfigWithSSL() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:    "web",
				Domain:  "web.test",
				Port:    80,
				SSL:     true,
				SSLPort: 443,
			},
			{
				Name:   "api",
				Domain: "api.test",
				Port:   8080,
				SSL:    false, // No SSL for this one
			},
		},
	}

	nginxConf, err := generateNginxConfig(config)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), nginxConf)

	// Check for SSL configuration in web service
	assert.Contains(suite.T(), nginxConf, "listen 443 ssl")
	assert.Contains(suite.T(), nginxConf, "ssl_certificate /etc/nginx/ssl/web_test.crt")
	assert.Contains(suite.T(), nginxConf, "ssl_certificate_key /etc/nginx/ssl/web_test.key")
	assert.Contains(suite.T(), nginxConf, "ssl_protocols TLSv1.2 TLSv1.3")
	assert.Contains(suite.T(), nginxConf, "return 301 https://$server_name$request_uri")

	// Check API service doesn't have SSL
	apiSection := strings.Split(nginxConf, "server_name api.test")[1]
	if idx := strings.Index(apiSection, "server {"); idx > 0 {
		apiSection = apiSection[:idx]
	}
	assert.NotContains(suite.T(), apiSection, "ssl_certificate")
}

func (suite *NginxSSLSuite) TestNginxConfigCustomSSLPort() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:    "web",
				Domain:  "web.test",
				Port:    80,
				SSL:     true,
				SSLPort: 8443, // Custom SSL port
			},
		},
	}

	nginxConf, err := generateNginxConfig(config)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), nginxConf, "listen 8443 ssl")
}

func (suite *NginxSSLSuite) TestNginxProxyWithSSLVolumes() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Domain: "web.test",
				Port:   80,
				SSL:    true,
			},
		},
	}

	// Create a mock docker-compose structure
	compose := &DockerCompose{
		Version:  "3.8",
		Services: make(map[string]DockerService),
		Networks: make(map[string]DockerNetwork),
	}

	// Add nginx proxy
	addNginxProxyToCompose(compose, config)

	// Check nginx service exists
	nginxService, exists := compose.Services["nginx-proxy"]
	assert.True(suite.T(), exists)

	// Check ports include both 80 and 443
	assert.Contains(suite.T(), nginxService.Ports, "80:80")
	assert.Contains(suite.T(), nginxService.Ports, "443:443")

	// Check SSL volume mount
	sslVolumeFound := false
	for _, volume := range nginxService.Volumes {
		if strings.Contains(volume, "/etc/nginx/ssl:ro") {
			sslVolumeFound = true
			break
		}
	}
	assert.True(suite.T(), sslVolumeFound, "SSL volume mount should be present")
}

func (suite *NginxSSLSuite) TestServiceWithDomainStructure() {
	service := Service{
		Name:    "test",
		Domain:  "test.local",
		Port:    8080,
		SSL:     true,
		SSLPort: 8443,
	}

	svcWithDomain := ServiceWithDomain{
		Name:            service.Name,
		Domain:          service.Domain,
		Port:            service.Port,
		SSL:             service.SSL,
		SSLPort:         service.SSLPort,
		SanitizedDomain: sanitizeDomainForFilename(service.Domain),
	}

	assert.Equal(suite.T(), "test", svcWithDomain.Name)
	assert.Equal(suite.T(), "test.local", svcWithDomain.Domain)
	assert.Equal(suite.T(), 8080, svcWithDomain.Port)
	assert.True(suite.T(), svcWithDomain.SSL)
	assert.Equal(suite.T(), 8443, svcWithDomain.SSLPort)
	assert.Equal(suite.T(), "test_local", svcWithDomain.SanitizedDomain)
}

func (suite *NginxSSLSuite) TestGenerateNginxSSLConfig() {
	// Create test certificates
	certs := []SSLCertificate{
		{
			Domain:   "test.local",
			CertPath: "/tmp/test.crt",
			KeyPath:  "/tmp/test.key",
		},
	}

	sslDir := suite.helper.TempDir()
	err := generateNginxSSLConfig(certs, sslDir)
	assert.NoError(suite.T(), err)

	// Check SSL params file was created
	sslParamsPath := filepath.Join(sslDir, "ssl-params.conf")
	assert.FileExists(suite.T(), sslParamsPath)

	// Check dhparam file was created
	dhparamPath := filepath.Join(sslDir, "dhparam.pem")
	assert.FileExists(suite.T(), dhparamPath)

	// Check SSL params content
	content, err := os.ReadFile(sslParamsPath)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(content), "ssl_protocols TLSv1.2 TLSv1.3")
	assert.Contains(suite.T(), string(content), "ssl_session_cache shared:SSL:10m")
}

func TestNginxSSLSuite(t *testing.T) {
	suite.Run(t, new(NginxSSLSuite))
}