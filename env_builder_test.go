package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EnvBuilderSuite struct {
	suite.Suite
	builder *EnvBuilder
}

func TestEnvBuilderSuite(t *testing.T) {
	suite.Run(t, new(EnvBuilderSuite))
}

func (suite *EnvBuilderSuite) SetupTest() {
	suite.builder = NewEnvBuilder()
}

func (suite *EnvBuilderSuite) TestSet() {
	suite.builder.Set("KEY1", "value1").Set("KEY2", "value2")
	
	env := suite.builder.Build()
	suite.Equal("value1", env["KEY1"])
	suite.Equal("value2", env["KEY2"])
}

func (suite *EnvBuilderSuite) TestSetIf() {
	suite.builder.
		SetIf(true, "KEY1", "value1").
		SetIf(false, "KEY2", "value2")
	
	env := suite.builder.Build()
	suite.Equal("value1", env["KEY1"])
	suite.NotContains(env, "KEY2")
}

func (suite *EnvBuilderSuite) TestSetIfNotEmpty() {
	suite.builder.
		SetIfNotEmpty("KEY1", "value1").
		SetIfNotEmpty("KEY2", "")
	
	env := suite.builder.Build()
	suite.Equal("value1", env["KEY1"])
	suite.NotContains(env, "KEY2")
}

func (suite *EnvBuilderSuite) TestSetDefault() {
	suite.builder.
		Set("KEY1", "original").
		SetDefault("KEY1", "default").
		SetDefault("KEY2", "default2")
	
	env := suite.builder.Build()
	suite.Equal("original", env["KEY1"])
	suite.Equal("default2", env["KEY2"])
}

func (suite *EnvBuilderSuite) TestMerge() {
	other := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	
	suite.builder.
		Set("KEY2", "original").
		Set("KEY3", "value3").
		Merge(other)
	
	env := suite.builder.Build()
	suite.Equal("value1", env["KEY1"])
	suite.Equal("value2", env["KEY2"]) // Overwritten by merge
	suite.Equal("value3", env["KEY3"])
}

func (suite *EnvBuilderSuite) TestHasAndGet() {
	suite.builder.Set("KEY1", "value1")
	
	suite.True(suite.builder.Has("KEY1"))
	suite.False(suite.builder.Has("KEY2"))
	suite.Equal("value1", suite.builder.Get("KEY1"))
	suite.Equal("", suite.builder.Get("KEY2"))
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderDatabase() {
	builder := NewConnectionStringBuilder()
	
	// Test MySQL connection
	builder.SetDatabaseConnection("mysql", "localhost", "3306", "mydb", "root", "password")
	env := builder.Build()
	
	suite.Equal("mysql", env["DB_CONNECTION"])
	suite.Equal("localhost", env["DB_HOST"])
	suite.Equal("3306", env["DB_PORT"])
	suite.Equal("mydb", env["DB_DATABASE"])
	suite.Equal("root", env["DB_USERNAME"])
	suite.Equal("password", env["DB_PASSWORD"])
	suite.Equal("mysql://root:password@localhost:3306/mydb", env["DATABASE_URL"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderPostgres() {
	builder := NewConnectionStringBuilder()
	
	// Test PostgreSQL connection
	builder.SetDatabaseConnection("postgres", "localhost", "5432", "mydb", "postgres", "secret")
	env := builder.Build()
	
	suite.Equal("postgres", env["DB_CONNECTION"])
	suite.Equal("localhost", env["DB_HOST"])
	suite.Equal("5432", env["DB_PORT"])
	suite.Equal("mydb", env["DB_DATABASE"])
	suite.Equal("postgres", env["DB_USERNAME"])
	suite.Equal("secret", env["DB_PASSWORD"])
	suite.Equal("postgresql://postgres:secret@localhost:5432/mydb", env["DATABASE_URL"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderMongoDB() {
	builder := NewConnectionStringBuilder()
	
	// Test MongoDB connection
	builder.SetDatabaseConnection("mongodb", "localhost", "27017", "mydb", "admin", "secret")
	env := builder.Build()
	
	suite.Equal("mongodb", env["DB_CONNECTION"])
	suite.Equal("localhost", env["DB_HOST"])
	suite.Equal("27017", env["DB_PORT"])
	suite.Equal("mydb", env["DB_DATABASE"])
	suite.Equal("admin", env["DB_USERNAME"])
	suite.Equal("secret", env["DB_PASSWORD"])
	suite.Equal("mongodb://admin:secret@localhost:27017/mydb", env["DATABASE_URL"])
	suite.Equal("mongodb://admin:secret@localhost:27017/mydb", env["MONGODB_URL"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderCache() {
	builder := NewConnectionStringBuilder()
	
	// Test Redis connection
	builder.SetCacheConnection("redis", "localhost", "6379", "secret")
	env := builder.Build()
	
	suite.Equal("localhost", env["REDIS_HOST"])
	suite.Equal("6379", env["REDIS_PORT"])
	suite.Equal("redis", env["CACHE_DRIVER"])
	suite.Equal("secret", env["REDIS_PASSWORD"])
	suite.Equal("redis://:secret@localhost:6379/0", env["REDIS_URL"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderMemcached() {
	builder := NewConnectionStringBuilder()
	
	// Test Memcached connection
	builder.SetCacheConnection("memcached", "localhost", "11211", "")
	env := builder.Build()
	
	suite.Equal("localhost", env["MEMCACHED_HOST"])
	suite.Equal("11211", env["MEMCACHED_PORT"])
	suite.Equal("memcached", env["CACHE_DRIVER"])
	suite.Equal("localhost:11211", env["MEMCACHED_URL"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderMail() {
	builder := NewConnectionStringBuilder()
	
	// Test SMTP connection
	builder.SetMailConnection("smtp.example.com", "587", "user@example.com", "password", true)
	env := builder.Build()
	
	suite.Equal("smtp", env["MAIL_MAILER"])
	suite.Equal("smtp.example.com", env["MAIL_HOST"])
	suite.Equal("587", env["MAIL_PORT"])
	suite.Equal("user@example.com", env["MAIL_USERNAME"])
	suite.Equal("password", env["MAIL_PASSWORD"])
	suite.Equal("tls", env["MAIL_ENCRYPTION"])
	
	// Also check alternative naming
	suite.Equal("smtp.example.com", env["SMTP_HOST"])
	suite.Equal("587", env["SMTP_PORT"])
	suite.Equal("user@example.com", env["SMTP_USERNAME"])
	suite.Equal("password", env["SMTP_PASSWORD"])
	suite.Equal("tls", env["SMTP_ENCRYPTION"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderS3() {
	builder := NewConnectionStringBuilder()
	
	// Test S3 connection
	builder.SetS3Connection("http://localhost:9000", "us-east-1", "mybucket", "accesskey", "secretkey")
	env := builder.Build()
	
	suite.Equal("http://localhost:9000", env["S3_ENDPOINT"])
	suite.Equal("us-east-1", env["S3_REGION"])
	suite.Equal("mybucket", env["S3_BUCKET"])
	suite.Equal("accesskey", env["AWS_ACCESS_KEY_ID"])
	suite.Equal("secretkey", env["AWS_SECRET_ACCESS_KEY"])
	suite.Equal("http://localhost:9000", env["STORAGE_URL"])
	
	// Check AWS compatibility
	suite.Equal("http://localhost:9000", env["AWS_ENDPOINT"])
	suite.Equal("us-east-1", env["AWS_REGION"])
	suite.Equal("mybucket", env["AWS_BUCKET"])
}

func (suite *EnvBuilderSuite) TestConnectionStringBuilderSearch() {
	builder := NewConnectionStringBuilder()
	
	// Test Meilisearch connection
	builder.SetSearchConnection("meilisearch", "localhost", "7700", "masterkey")
	env := builder.Build()
	
	suite.Equal("meilisearch", env["SEARCH_ENGINE"])
	suite.Equal("http://localhost:7700", env["MEILISEARCH_HOST"])
	suite.Equal("http://localhost:7700", env["MEILISEARCH_URL"])
	suite.Equal("masterkey", env["MEILISEARCH_KEY"])
	suite.Equal("masterkey", env["MEILISEARCH_MASTER_KEY"])
}

func (suite *EnvBuilderSuite) TestNormalizeEnvKey() {
	tests := []struct {
		input    string
		expected string
	}{
		{"key-with-dash", "KEY_WITH_DASH"},
		{"key.with.dot", "KEY_WITH_DOT"},
		{"key with space", "KEY_WITH_SPACE"},
		{"lowercase", "LOWERCASE"},
		{"MixedCase", "MIXEDCASE"},
		{"KEY_ALREADY_GOOD", "KEY_ALREADY_GOOD"},
	}

	for _, tt := range tests {
		result := NormalizeEnvKey(tt.input)
		suite.Equal(tt.expected, result, "Input: %s", tt.input)
	}
}

func (suite *EnvBuilderSuite) TestStandardEnvPatterns() {
	patterns := &StandardEnvPatterns{}
	
	dbKeys := patterns.GetDatabaseEnvKeys()
	suite.Contains(dbKeys, "DB_CONNECTION")
	suite.Contains(dbKeys, "DB_HOST")
	suite.Contains(dbKeys, "DATABASE_URL")
	
	cacheKeys := patterns.GetCacheEnvKeys()
	suite.Contains(cacheKeys, "CACHE_DRIVER")
	suite.Contains(cacheKeys, "REDIS_HOST")
	
	mailKeys := patterns.GetMailEnvKeys()
	suite.Contains(mailKeys, "MAIL_HOST")
	suite.Contains(mailKeys, "SMTP_HOST")
}