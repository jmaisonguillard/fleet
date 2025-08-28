package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SSLServiceSuite struct {
	suite.Suite
	helper *TestHelper
}

func (suite *SSLServiceSuite) SetupTest() {
	suite.helper = NewTestHelper(suite.T())
}

func (suite *SSLServiceSuite) TearDownTest() {
	suite.helper.Cleanup()
	// Clean up any .fleet directory created during tests
	os.RemoveAll(".fleet")
}

func (suite *SSLServiceSuite) TestSanitizeDomainForFilename() {
	testCases := []struct {
		domain   string
		expected string
	}{
		{"example.test", "example_test"},
		{"*.example.test", "wildcard_example_test"},
		{"sub.domain.test", "sub_domain_test"},
		{"localhost", "localhost"},
	}

	for _, tc := range testCases {
		result := sanitizeDomainForFilename(tc.domain)
		assert.Equal(suite.T(), tc.expected, result)
	}
}

func (suite *SSLServiceSuite) TestGenerateSSLCertificatesNoSSL() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Domain: "web.test",
				Port:   80,
				SSL:    false, // SSL disabled
			},
		},
	}

	err := generateSSLCertificates(config)
	assert.NoError(suite.T(), err)

	// Should still create default certificate
	defaultCertPath := filepath.Join(".fleet", "ssl", "default.crt")
	defaultKeyPath := filepath.Join(".fleet", "ssl", "default.key")
	assert.FileExists(suite.T(), defaultCertPath)
	assert.FileExists(suite.T(), defaultKeyPath)

	// Should not create service-specific certificate
	webCertPath := filepath.Join(".fleet", "ssl", "web_test.crt")
	assert.NoFileExists(suite.T(), webCertPath)
}

func (suite *SSLServiceSuite) TestGenerateSSLCertificatesWithSSL() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Domain: "web.test",
				Port:   80,
				SSL:    true, // SSL enabled
			},
			{
				Name:   "api",
				Domain: "api.test",
				Port:   8080,
				SSL:    true, // SSL enabled
			},
		},
	}

	err := generateSSLCertificates(config)
	assert.NoError(suite.T(), err)

	// Check default certificate
	defaultCertPath := filepath.Join(".fleet", "ssl", "default.crt")
	defaultKeyPath := filepath.Join(".fleet", "ssl", "default.key")
	assert.FileExists(suite.T(), defaultCertPath)
	assert.FileExists(suite.T(), defaultKeyPath)

	// Check web service certificate
	webCertPath := filepath.Join(".fleet", "ssl", "web_test.crt")
	webKeyPath := filepath.Join(".fleet", "ssl", "web_test.key")
	assert.FileExists(suite.T(), webCertPath)
	assert.FileExists(suite.T(), webKeyPath)

	// Check api service certificate
	apiCertPath := filepath.Join(".fleet", "ssl", "api_test.crt")
	apiKeyPath := filepath.Join(".fleet", "ssl", "api_test.key")
	assert.FileExists(suite.T(), apiCertPath)
	assert.FileExists(suite.T(), apiKeyPath)
}

func (suite *SSLServiceSuite) TestGenerateSSLCertificatesMultipleDomains() {
	config := &Config{
		Project: "test-project",
		Services: []Service{
			{
				Name:   "web",
				Domain: "web.test, www.web.test",
				Port:   80,
				SSL:    true,
			},
		},
	}

	err := generateSSLCertificates(config)
	assert.NoError(suite.T(), err)

	// Should create certificates for both domains
	cert1Path := filepath.Join(".fleet", "ssl", "web_test.crt")
	cert2Path := filepath.Join(".fleet", "ssl", "www_web_test.crt")
	assert.FileExists(suite.T(), cert1Path)
	assert.FileExists(suite.T(), cert2Path)
}

func (suite *SSLServiceSuite) TestNeedsNewCertificate() {
	// Test non-existent files
	assert.True(suite.T(), needsNewCertificate("/nonexistent/cert.crt", "/nonexistent/key.key"))

	// Create temporary certificate files
	tempDir := suite.helper.TempDir()
	certPath := filepath.Join(tempDir, "test.crt")
	keyPath := filepath.Join(tempDir, "test.key")

	// Generate a test certificate
	cert := SSLCertificate{
		Domain:     "test.local",
		CertPath:   certPath,
		KeyPath:    keyPath,
		CommonName: "test.local",
	}
	err := generateSelfSignedCertificate(cert)
	assert.NoError(suite.T(), err)

	// Should not need new certificate for valid files
	assert.False(suite.T(), needsNewCertificate(certPath, keyPath))

	// Test with missing key file
	os.Remove(keyPath)
	assert.True(suite.T(), needsNewCertificate(certPath, keyPath))
}

func (suite *SSLServiceSuite) TestHasSSLServices() {
	// No SSL services
	config1 := &Config{
		Services: []Service{
			{Name: "web", Domain: "web.test", SSL: false},
		},
	}
	assert.False(suite.T(), hasSSLServices(config1))

	// Has SSL service with domain
	config2 := &Config{
		Services: []Service{
			{Name: "web", Domain: "web.test", SSL: true},
		},
	}
	assert.True(suite.T(), hasSSLServices(config2))

	// Has SSL but no domain (should return false)
	config3 := &Config{
		Services: []Service{
			{Name: "web", SSL: true},
		},
	}
	assert.False(suite.T(), hasSSLServices(config3))
}

func (suite *SSLServiceSuite) TestGetServiceSSLPorts() {
	// Default ports
	service1 := &Service{Name: "web"}
	httpPort, httpsPort := getServiceSSLPorts(service1)
	assert.Equal(suite.T(), 80, httpPort)
	assert.Equal(suite.T(), 443, httpsPort)

	// Custom HTTP port
	service2 := &Service{Name: "web", Port: 8080}
	httpPort, httpsPort = getServiceSSLPorts(service2)
	assert.Equal(suite.T(), 8080, httpPort)
	assert.Equal(suite.T(), 443, httpsPort)

	// Custom HTTPS port
	service3 := &Service{Name: "web", SSLPort: 8443}
	httpPort, httpsPort = getServiceSSLPorts(service3)
	assert.Equal(suite.T(), 80, httpPort)
	assert.Equal(suite.T(), 8443, httpsPort)

	// Both custom ports
	service4 := &Service{Name: "web", Port: 8080, SSLPort: 8443}
	httpPort, httpsPort = getServiceSSLPorts(service4)
	assert.Equal(suite.T(), 8080, httpPort)
	assert.Equal(suite.T(), 8443, httpsPort)
}

func (suite *SSLServiceSuite) TestGenerateSelfSignedCertificate() {
	tempDir := suite.helper.TempDir()
	cert := SSLCertificate{
		Domain:     "test.local",
		CertPath:   filepath.Join(tempDir, "test.crt"),
		KeyPath:    filepath.Join(tempDir, "test.key"),
		CommonName: "test.local",
	}

	err := generateSelfSignedCertificate(cert)
	assert.NoError(suite.T(), err)

	// Check files exist
	assert.FileExists(suite.T(), cert.CertPath)
	assert.FileExists(suite.T(), cert.KeyPath)

	// Check file permissions (key should be 0600)
	keyInfo, err := os.Stat(cert.KeyPath)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), os.FileMode(0600), keyInfo.Mode().Perm())
}

func TestSSLServiceSuite(t *testing.T) {
	suite.Run(t, new(SSLServiceSuite))
}