package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SSLCertificate represents an SSL certificate configuration
type SSLCertificate struct {
	Domain     string
	CertPath   string
	KeyPath    string
	CommonName string
}

// generateSSLCertificates generates self-signed SSL certificates for services with domains
func generateSSLCertificates(config *Config) error {
	// Create SSL directory in .fleet
	sslDir := filepath.Join(".fleet", "ssl")
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return fmt.Errorf("failed to create SSL directory: %v", err)
	}

	certificates := []SSLCertificate{}

	// Always generate a default certificate for the catch-all server
	defaultCert := SSLCertificate{
		Domain:     "default",
		CertPath:   filepath.Join(sslDir, "default.crt"),
		KeyPath:    filepath.Join(sslDir, "default.key"),
		CommonName: "localhost",
	}
	
	if !needsNewCertificate(defaultCert.CertPath, defaultCert.KeyPath) {
		fmt.Println("Default SSL certificate already exists and is valid")
	} else {
		if err := generateSelfSignedCertificate(defaultCert); err != nil {
			return fmt.Errorf("failed to generate default certificate: %v", err)
		}
		fmt.Println("Generated default SSL certificate")
	}
	certificates = append(certificates, defaultCert)

	// Generate certificates for each service with SSL enabled and a domain
	for _, service := range config.Services {
		if service.SSL && service.Domain != "" {
			domains := strings.Split(service.Domain, ",")
			for _, domain := range domains {
				domain = strings.TrimSpace(domain)
				cert := SSLCertificate{
					Domain:     domain,
					CertPath:   filepath.Join(sslDir, fmt.Sprintf("%s.crt", sanitizeDomainForFilename(domain))),
					KeyPath:    filepath.Join(sslDir, fmt.Sprintf("%s.key", sanitizeDomainForFilename(domain))),
					CommonName: domain,
				}

				// Check if certificate already exists and is valid
				if !needsNewCertificate(cert.CertPath, cert.KeyPath) {
					fmt.Printf("SSL certificate for %s already exists and is valid\n", domain)
					certificates = append(certificates, cert)
					continue
				}

				// Generate new certificate
				if err := generateSelfSignedCertificate(cert); err != nil {
					return fmt.Errorf("failed to generate certificate for %s: %v", domain, err)
				}

				fmt.Printf("Generated SSL certificate for %s\n", domain)
				certificates = append(certificates, cert)
			}
		}
	}

	// Generate nginx SSL configuration
	if len(certificates) > 0 {
		if err := generateNginxSSLConfig(certificates, sslDir); err != nil {
			return fmt.Errorf("failed to generate nginx SSL config: %v", err)
		}
	}

	return nil
}

// sanitizeDomainForFilename converts a domain to a safe filename
func sanitizeDomainForFilename(domain string) string {
	// Replace dots and wildcards with underscores
	safe := strings.ReplaceAll(domain, ".", "_")
	safe = strings.ReplaceAll(safe, "*", "wildcard")
	return safe
}

// needsNewCertificate checks if we need to generate a new certificate
func needsNewCertificate(certPath, keyPath string) bool {
	// Check if both files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return true
	}

	// Check if certificate is still valid (not expired)
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return true
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}

	// Check if certificate expires within 30 days
	if time.Until(cert.NotAfter) < 30*24*time.Hour {
		return true
	}

	return false
}

// generateSelfSignedCertificate generates a self-signed SSL certificate
func generateSelfSignedCertificate(cert SSLCertificate) error {
	// Generate RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Fleet Local Development"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    cert.CommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add Subject Alternative Names
	hosts := []string{cert.Domain}
	
	// Add wildcard support
	if !strings.HasPrefix(cert.Domain, "*.") {
		// Add www subdomain if not a wildcard
		if !strings.HasPrefix(cert.Domain, "www.") {
			hosts = append(hosts, "www."+cert.Domain)
		}
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// Also add localhost for local development
	template.DNSNames = append(template.DNSNames, "localhost")
	template.IPAddresses = append(template.IPAddresses, net.IPv4(127, 0, 0, 1))

	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// Write certificate to file
	certFile, err := os.Create(cert.CertPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Write private key to file
	keyFile, err := os.Create(cert.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer keyFile.Close()

	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	if err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(cert.KeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set key file permissions: %v", err)
	}

	return nil
}

// generateNginxSSLConfig generates nginx SSL configuration
func generateNginxSSLConfig(certificates []SSLCertificate, sslDir string) error {
	// Create SSL params file with modern SSL configuration
	sslParamsPath := filepath.Join(sslDir, "ssl-params.conf")
	sslParams := `# Modern SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-DSS-AES128-GCM-SHA256:kEDH+AESGCM:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:ECDHE-ECDSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-DSS-AES128-SHA256:DHE-RSA-AES256-SHA256:DHE-DSS-AES256-SHA:DHE-RSA-AES256-SHA:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:AES:CAMELLIA:DES-CBC3-SHA:!aNULL:!eNULL:!EXPORT:!DES:!RC4:!MD5:!PSK:!aECDH:!EDH-DSS-DES-CBC3-SHA:!EDH-RSA-DES-CBC3-SHA:!KRB5-DES-CBC3-SHA;
ssl_prefer_server_ciphers off;

# HSTS (optional - uncomment if needed)
# add_header Strict-Transport-Security "max-age=63072000" always;

# SSL session caching
ssl_session_timeout 1d;
ssl_session_cache shared:SSL:10m;
ssl_session_tickets off;

# OCSP stapling (disabled for self-signed certs)
ssl_stapling off;
ssl_stapling_verify off;

# Disable SSL session tickets
ssl_session_tickets off;
`

	if err := os.WriteFile(sslParamsPath, []byte(sslParams), 0644); err != nil {
		return fmt.Errorf("failed to write SSL params: %v", err)
	}

	// Generate dhparam file (use a pre-generated one for speed in development)
	dhparamPath := filepath.Join(sslDir, "dhparam.pem")
	if _, err := os.Stat(dhparamPath); os.IsNotExist(err) {
		// Use a pre-generated 2048-bit dhparam for development
		// In production, you'd want to generate this with: openssl dhparam -out dhparam.pem 2048
		dhparam := `-----BEGIN DH PARAMETERS-----
MIIBCAKCAQEAz+Z6hg1J5XxU+fMvJV8zPvU3ajPPNN6qpYvTNaM9e4PJzIqJq7sM
t6XjPWLKhqwH2J+9VRNBZxn9C8BgzxMnG7bRDw2yUYY1gm3XJYzswvQ7pJQ6CH8h
vNn5DVWqzKvT+KPlYqGr6TmKCVvIBGPkPyPDH8/OQx3xBe3f4YMmLxK3v4BTzKXX
FUu6g6KKCduNXej0xSjZHWkOxPwJvvLH7T7QkkLPTJCw9yCw0hovqmmJZKKBPsJF
kxnsmFzV1FQsJYfDzJqKUmgeL9TZ7iBpiTqC3nVkZCPLxAUi8EyQFHCTbCXev1Fh
E2KSA6pDYLKqV9neLFPx5fwKMgbcCzFjIwIBAg==
-----END DH PARAMETERS-----
`
		if err := os.WriteFile(dhparamPath, []byte(dhparam), 0644); err != nil {
			return fmt.Errorf("failed to write dhparam: %v", err)
		}
	}

	return nil
}

// hasSSLServices checks if any service has SSL enabled
func hasSSLServices(config *Config) bool {
	for _, service := range config.Services {
		if service.SSL && service.Domain != "" {
			return true
		}
	}
	return false
}

// getServiceSSLPorts returns the HTTP and HTTPS ports for a service
func getServiceSSLPorts(service *Service) (httpPort int, httpsPort int) {
	httpPort = 80
	httpsPort = 443

	// Override with service-specific ports if defined
	if service.Port != 0 {
		httpPort = service.Port
	}
	if service.SSLPort != 0 {
		httpsPort = service.SSLPort
	}

	return httpPort, httpsPort
}