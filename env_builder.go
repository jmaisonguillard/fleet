package main

import (
	"fmt"
	"strings"
)

// EnvBuilder provides a fluent API for building environment variables
type EnvBuilder struct {
	env map[string]string
}

// NewEnvBuilder creates a new environment variable builder
func NewEnvBuilder() *EnvBuilder {
	return &EnvBuilder{
		env: make(map[string]string),
	}
}

// Set adds a single environment variable
func (b *EnvBuilder) Set(key, value string) *EnvBuilder {
	b.env[key] = value
	return b
}

// SetIf conditionally sets an environment variable
func (b *EnvBuilder) SetIf(condition bool, key, value string) *EnvBuilder {
	if condition {
		b.env[key] = value
	}
	return b
}

// SetIfNotEmpty sets an environment variable only if the value is not empty
func (b *EnvBuilder) SetIfNotEmpty(key, value string) *EnvBuilder {
	if value != "" {
		b.env[key] = value
	}
	return b
}

// SetDefault sets a value only if the key doesn't exist
func (b *EnvBuilder) SetDefault(key, value string) *EnvBuilder {
	if _, exists := b.env[key]; !exists {
		b.env[key] = value
	}
	return b
}

// SetMultiple sets multiple environment variables at once
func (b *EnvBuilder) SetMultiple(vars map[string]string) *EnvBuilder {
	for k, v := range vars {
		b.env[k] = v
	}
	return b
}

// Build returns the built environment variables map
func (b *EnvBuilder) Build() map[string]string {
	// Return a copy to prevent external modifications
	result := make(map[string]string, len(b.env))
	for k, v := range b.env {
		result[k] = v
	}
	return result
}

// Merge combines another environment map into this builder
func (b *EnvBuilder) Merge(other map[string]string) *EnvBuilder {
	for k, v := range other {
		b.env[k] = v
	}
	return b
}

// Has checks if a key exists
func (b *EnvBuilder) Has(key string) bool {
	_, exists := b.env[key]
	return exists
}

// Get retrieves a value by key
func (b *EnvBuilder) Get(key string) string {
	return b.env[key]
}

// ConnectionStringBuilder builds database connection strings
type ConnectionStringBuilder struct {
	*EnvBuilder
}

// NewConnectionStringBuilder creates a builder for connection strings
func NewConnectionStringBuilder() *ConnectionStringBuilder {
	return &ConnectionStringBuilder{
		EnvBuilder: NewEnvBuilder(),
	}
}

// SetDatabaseConnection sets standard database connection variables
func (b *ConnectionStringBuilder) SetDatabaseConnection(dbType, host, port, database, username, password string) *ConnectionStringBuilder {
	b.Set("DB_CONNECTION", dbType)
	b.Set("DB_HOST", host)
	b.Set("DB_PORT", port)
	b.Set("DB_DATABASE", database)
	b.Set("DB_USERNAME", username)
	b.SetIfNotEmpty("DB_PASSWORD", password)
	
	// Build DATABASE_URL based on type
	var databaseURL string
	switch dbType {
	case "mysql", "mariadb":
		if password != "" {
			databaseURL = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", username, password, host, port, database)
		} else {
			databaseURL = fmt.Sprintf("mysql://%s@%s:%s/%s", username, host, port, database)
		}
		
	case "pgsql", "postgres", "postgresql":
		if password != "" {
			databaseURL = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", username, password, host, port, database)
		} else {
			databaseURL = fmt.Sprintf("postgresql://%s@%s:%s/%s", username, host, port, database)
		}
		
	case "mongodb", "mongo":
		if password != "" && username != "" {
			databaseURL = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", username, password, host, port, database)
		} else {
			databaseURL = fmt.Sprintf("mongodb://%s:%s/%s", host, port, database)
		}
		b.Set("MONGODB_URL", databaseURL)
	}
	
	if databaseURL != "" {
		b.Set("DATABASE_URL", databaseURL)
	}
	
	return b
}

// SetCacheConnection sets standard cache connection variables
func (b *ConnectionStringBuilder) SetCacheConnection(cacheType, host, port, password string) *ConnectionStringBuilder {
	switch cacheType {
	case "redis":
		b.Set("REDIS_HOST", host)
		b.Set("REDIS_PORT", port)
		b.Set("CACHE_DRIVER", "redis")
		
		var redisURL string
		if password != "" {
			b.Set("REDIS_PASSWORD", password)
			redisURL = fmt.Sprintf("redis://:%s@%s:%s/0", password, host, port)
		} else {
			redisURL = fmt.Sprintf("redis://%s:%s/0", host, port)
		}
		b.Set("REDIS_URL", redisURL)
		
	case "memcached":
		b.Set("MEMCACHED_HOST", host)
		b.Set("MEMCACHED_PORT", port)
		b.Set("CACHE_DRIVER", "memcached")
		b.Set("MEMCACHED_URL", fmt.Sprintf("%s:%s", host, port))
	}
	
	return b
}

// SetMailConnection sets standard email/SMTP connection variables
func (b *ConnectionStringBuilder) SetMailConnection(host, port, username, password string, useTLS bool) *ConnectionStringBuilder {
	b.Set("MAIL_MAILER", "smtp")
	b.Set("MAIL_HOST", host)
	b.Set("MAIL_PORT", port)
	
	// Alternative naming
	b.Set("SMTP_HOST", host)
	b.Set("SMTP_PORT", port)
	
	if username != "" {
		b.Set("MAIL_USERNAME", username)
		b.Set("SMTP_USERNAME", username)
	}
	
	if password != "" {
		b.Set("MAIL_PASSWORD", password)
		b.Set("SMTP_PASSWORD", password)
	}
	
	if useTLS {
		b.Set("MAIL_ENCRYPTION", "tls")
		b.Set("SMTP_ENCRYPTION", "tls")
	} else {
		b.Set("MAIL_ENCRYPTION", "null")
		b.Set("SMTP_ENCRYPTION", "null")
	}
	
	return b
}

// SetS3Connection sets S3-compatible storage connection variables
func (b *ConnectionStringBuilder) SetS3Connection(endpoint, region, bucket, accessKey, secretKey string) *ConnectionStringBuilder {
	b.Set("S3_ENDPOINT", endpoint)
	b.Set("S3_REGION", region)
	b.Set("S3_BUCKET", bucket)
	
	// AWS SDK compatibility
	b.Set("AWS_ENDPOINT", endpoint)
	b.Set("AWS_REGION", region)
	b.Set("AWS_DEFAULT_REGION", region)
	b.Set("AWS_BUCKET", bucket)
	
	if accessKey != "" {
		b.Set("AWS_ACCESS_KEY_ID", accessKey)
		b.Set("S3_ACCESS_KEY", accessKey)
	}
	
	if secretKey != "" {
		b.Set("AWS_SECRET_ACCESS_KEY", secretKey)
		b.Set("S3_SECRET_KEY", secretKey)
	}
	
	b.Set("STORAGE_URL", endpoint)
	
	return b
}

// SetSearchConnection sets search engine connection variables
func (b *ConnectionStringBuilder) SetSearchConnection(searchType, host, port, apiKey string) *ConnectionStringBuilder {
	b.Set("SEARCH_ENGINE", searchType)
	
	switch searchType {
	case "meilisearch":
		url := fmt.Sprintf("http://%s:%s", host, port)
		b.Set("MEILISEARCH_HOST", url)
		b.Set("MEILISEARCH_URL", url)
		if apiKey != "" {
			b.Set("MEILISEARCH_KEY", apiKey)
			b.Set("MEILISEARCH_MASTER_KEY", apiKey)
		}
		
	case "typesense":
		b.Set("TYPESENSE_HOST", host)
		b.Set("TYPESENSE_PORT", port)
		b.Set("TYPESENSE_URL", fmt.Sprintf("http://%s:%s", host, port))
		if apiKey != "" {
			b.Set("TYPESENSE_API_KEY", apiKey)
		}
	}
	
	return b
}

// StandardEnvPatterns provides common environment variable naming patterns
type StandardEnvPatterns struct{}

// GetDatabaseEnvKeys returns standard database environment variable keys
func (p *StandardEnvPatterns) GetDatabaseEnvKeys() []string {
	return []string{
		"DB_CONNECTION",
		"DB_HOST",
		"DB_PORT",
		"DB_DATABASE",
		"DB_USERNAME",
		"DB_PASSWORD",
		"DATABASE_URL",
	}
}

// GetCacheEnvKeys returns standard cache environment variable keys
func (p *StandardEnvPatterns) GetCacheEnvKeys() []string {
	return []string{
		"CACHE_DRIVER",
		"REDIS_HOST",
		"REDIS_PORT",
		"REDIS_PASSWORD",
		"REDIS_URL",
		"MEMCACHED_HOST",
		"MEMCACHED_PORT",
		"MEMCACHED_URL",
	}
}

// GetMailEnvKeys returns standard mail environment variable keys
func (p *StandardEnvPatterns) GetMailEnvKeys() []string {
	return []string{
		"MAIL_MAILER",
		"MAIL_HOST",
		"MAIL_PORT",
		"MAIL_USERNAME",
		"MAIL_PASSWORD",
		"MAIL_ENCRYPTION",
		"MAIL_FROM_ADDRESS",
		"MAIL_FROM_NAME",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
		"SMTP_ENCRYPTION",
	}
}

// NormalizeEnvKey converts environment variable names to standard format
func NormalizeEnvKey(key string) string {
	// Convert to uppercase
	key = strings.ToUpper(key)
	
	// Replace common separators with underscore
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, " ", "_")
	
	return key
}