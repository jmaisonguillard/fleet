# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fleet is a Docker orchestration tool that simplifies multi-service Docker management for both developers and beginners. It generates standard docker-compose.yml files from simple TOML/YAML/JSON configuration.

## Build and Test Commands

```bash
# Build the binary for current platform
make build

# Build for all platforms (Linux, macOS, Windows - AMD64/ARM64)
make build-all

# Clean build artifacts and .fleet directory
make clean

# Download and tidy dependencies
make deps

# Install to /usr/local/bin
make install

# Uninstall from /usr/local/bin
make uninstall

# Development mode (run without building)
make dev ARGS="up -d"

# Alternative build using build.sh (handles Go installation)
./build.sh

# Run all tests
make test
# Or with Go directly
go test -v ./...

# Run specific test suite
go test -v -run TestComposeSuite ./...
go test -v -run TestDNSSuite ./...
go test -v -run TestPHPRuntimeSuite ./...
go test -v -run TestNodeRuntimeSuite ./...
go test -v -run TestNodeFrameworksSuite ./...
go test -v -run TestDatabaseServicesSuite ./...
go test -v -run TestNginxSuite ./...
go test -v -run TestCacheServicesSuite ./...
go test -v -run TestSearchServicesSuite ./...
go test -v -run TestCompatServicesSuite ./...
go test -v -run TestEmailServiceSuite ./...

# Run integration tests (disabled by default)
RUN_INTEGRATION=1 go test -v -run TestIntegration ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out  # View in browser
```

## Architecture and Code Structure

### Core Components

1. **Configuration System** (`config.go`)
   - Supports TOML, YAML, JSON formats
   - Main types: `Config`, `Service`, `HealthCheck`
   - Auto-validates configurations and sets defaults

2. **Docker Compose Generation** (`compose.go`)
   - Generates Docker Compose v3.8 files from Fleet config
   - Creates "fleet-network" with subnet 172.28.0.0/16
   - Auto-detects database types for password configuration
   - Maps `folder` to `/app` in containers
   - Volume detection has a known issue: checks for "/" or "." anywhere in string

3. **DNS Management** (`dns.go`)
   - Provides `.test` TLD resolution via dnsmasq container
   - Cross-platform support (Windows PowerShell, Unix bash)
   - Creates hosts file backups before modification
   - DNS container uses subnet 172.30.0.0/16

4. **Embedded Assets** (`assets.go`)
   - Uses Go embed to bundle scripts, templates, and configs
   - Scripts: setup-dns.sh, setup-dns.ps1, test-dns.sh
   - Templates: Docker Compose files and Dockerfiles
   - Configs: dnsmasq configuration files

5. **Command Handling** (`commands.go`)
   - All Docker operations go through `runDocker()`
   - Commands: init, up, down, restart, status, logs, dns
   - Generates docker-compose.yml before Docker operations

### Testing Strategy

- **Test Framework**: Testify suite-based testing
- **Docker Mocking**: Full Docker mock system in `docker_mock_test.go`
  - Creates fake docker executable in PATH during tests
  - Simulates compose (up/down/ps/restart), docker ps, docker logs commands
  - Enables testing without Docker installed
  - Mock supports various output patterns for different commands
- **Integration Tests**: 
  - Disabled by default, enable with `RUN_INTEGRATION=1`
  - Use `MockDockerForTest()` for full Docker simulation
  - Skip automatically in CI environments unless explicitly enabled
- **Benchmarking**: Performance tests for compose generation and config loading
- **Test Helpers**: `test_helpers.go` provides utilities for temp files and sample configs
- **CI Detection**: `IsTestEnvironment()` function detects CI/testing environments

### Important Implementation Details

1. **Network Naming**: Always creates "fleet-network" (not project-based naming)

2. **Port Mapping**: 
   - Supports both `port` (single) and `ports` (array) fields
   - Single port maps as `port:port` (e.g., 8080:8080)
   - Ports array uses Docker format ("8080:80")

3. **Environment Variables**: 
   - Stored as maps, not "key=value" strings
   - Automatic service-to-service dependency environment variables

4. **Cross-Platform Compatibility**:
   - Script detection: `getScriptPath()` checks OS and selects .ps1 or .sh
   - Path handling: Use `filepath` package for OS-agnostic paths
   - Embedded resources work across all platforms
   - Build system supports Linux/macOS/Windows (AMD64/ARM64)

5. **DNS Setup Flow**:
   ```
   1. Check if port 53 is available
   2. Create backup directories
   3. Backup hosts file (timestamp + local)
   4. Add Fleet DNS entries to hosts
   5. Start dnsmasq container
   6. Test resolution
   ```

## DNS and Nginx Integration Notes

When implementing nginx container for project domains:
- Domain naming: `{project-name}.test` (e.g., "fleet" → "fleet.test")
- Hosts file modification required for resolution
- Use existing DNS infrastructure in `dns.go`
- Consider nginx proxy configuration for multiple services
- Maintain cross-platform compatibility

## Testing Without Docker

The codebase includes comprehensive Docker mocking:
```bash
# Tests use MockDockerSetup() to create fake docker in PATH
# Mock handles: compose up/down/ps/restart, docker ps, docker logs
```

## Critical Constraints

1. **Test-Only Modifications**: When fixing tests, NEVER modify implementation files
2. **Cross-Platform**: Always ensure Windows/Mac/Linux compatibility
3. **Embedded Assets**: All scripts/templates must be embedded via `assets.go`
4. **Network Conflicts**: Avoid 172.20.0.0/16 (use 172.28.0.0/16 or 172.30.0.0/16)

## Recent Major Features

### PHP Runtime Support (`runtime_php.go`, `php_frameworks.go`, `php_configurator.go`)
- **Auto-detection**: Detects Laravel, Symfony, WordPress, Drupal, CodeIgniter, Slim, Lumen
- **PHP versions**: 7.4, 8.0, 8.1, 8.2, 8.3, 8.4 (default: 8.4)
- **Configuration**: `runtime = "php:8.4"` and optional `framework = "laravel"`
- **Container naming**: Each service gets own PHP-FPM container (e.g., `web-php`)
- **Nginx integration**: Auto-generates PHP-FPM nginx configs in `.fleet/`
- **Framework configs**: Each framework gets specific nginx routing rules
- **Composer support**: Automatically installed in all PHP containers
  - CLI tool: `fleet-php composer install`, `fleet-php composer require`
  - Framework commands: `fleet-php artisan` (Laravel), `fleet-php console` (Symfony)
- **Xdebug support**: Enable with `debug = true` and optionally `debug_port = 9003`
  - Automatic Xdebug installation and configuration
  - IDE integration (PHPStorm, VSCode)
  - Configurable debug port (default: 9003)
- **Profiler support**: Enable with `profile = true`
  - Uses Xdebug profiler to generate cachegrind files
  - Configuration options:
    - `profile_trigger`: "request" (default) or "always"
    - `profile_output`: Custom directory (default: `.fleet/profiles`)
  - Request-based profiling: Add `XDEBUG_TRIGGER=PROFILE` to GET/POST/COOKIE
  - View profiles with KCacheGrind, QCacheGrind, or WebGrind
  - Can be used together with debugging

### Node.js Runtime Support (`runtime_node.go`, `node_frameworks.go`, `node_configurator.go`)
- **Auto-detection**: Detects Express, Next.js, Nuxt, Angular, React, Vue, Svelte, NestJS, Fastify, Remix
- **Node versions**: 16, 18, 20, 22 (default: 20, LTS versions)
- **Configuration**: `runtime = "node:20"` and optional `framework = "nextjs"`
- **Two operational modes**:
  - **Service mode**: Long-running Node.js services (APIs, servers)
  - **Build mode**: One-time build containers for frontend compilation
- **Package manager detection**: Auto-detects npm, yarn, or pnpm from lock files
- **Container optimization**: 
  - Service mode: Named volume for node_modules to improve performance
  - Build mode: Builds assets and exits, served by nginx
- **Framework-specific ports**:
  - Next.js: 3000 (default)
  - Angular: 4200
  - Vue: 8080
  - Express/others: 3000
- **CLI tool**: `fleet-node` for running Node.js commands
  - Package management: `fleet-node npm install`, `fleet-node yarn add`
  - Framework commands: `fleet-node npm run dev`, `fleet-node npx`
  - Multi-service support: `fleet-node --service=api npm test`
- **Environment variables**:
  - `node_env`: Set NODE_ENV (development/production)
  - `build_command`: Custom build command for build mode
  - `package_manager`: Override detected package manager
- **Build mode configuration**: Use with nginx for static serving
  ```toml
  [[services]]
  name = "frontend"
  image = "nginx:alpine"      # Serve with nginx
  runtime = "node:20"         # Build with Node.js
  build_command = "npm run build"
  folder = "frontend"
  ```

### Database Services (`database_services.go`)
- **Supported**: MySQL, PostgreSQL, MongoDB, MariaDB (not Redis - handled separately)
- **Container sharing**: Services using same DB version share containers (e.g., all mysql:8.0 share one container)
- **Configuration fields**: `database`, `database_name`, `database_user`, `database_password`, `database_root_password`
- **Auto environment vars**: Sets DB_CONNECTION, DB_HOST, DATABASE_URL etc. for apps
- **Health checks**: Each database type has appropriate health check configured
- **Volumes**: Persistent data volumes auto-created (e.g., `mysql-80-data`)
- **PostgreSQL Extensions**: Configure with `database_extensions = ["postgis", "pgvector"]`
  - PostGIS: Full spatial database support (uses postgis/postgis image)
  - pgvector: Vector similarity search (uses pgvector/pgvector image)
  - Common extensions: uuid-ossp, hstore, pg_trgm, btree_gin, btree_gist, pgrouting
  - Auto-generates initialization scripts in `.fleet/`

### Nginx Proxy (`nginx.go`)
- **Auto-generation**: Creates nginx-proxy container when services have domains
- **Domain mapping**: Maps service domains to backend containers
- **Hosts file**: Updates /etc/hosts with Fleet-managed entries
- **Configuration**: Generates `.fleet/nginx.conf` with upstreams and virtual hosts
- **WebSocket support**: Includes upgrade headers for WebSocket connections

## Testing Patterns

All test suites use testify/suite pattern:
```go
type MyTestSuite struct {
    suite.Suite
    helper *TestHelper  // Optional test helper
}
```

Run specific test suites:
```bash
go test -v -run TestPHPRuntimeSuite ./...
go test -v -run TestNodeRuntimeSuite ./...
go test -v -run TestNodeFrameworksSuite ./...
go test -v -run TestDatabaseServicesSuite ./...
go test -v -run TestNginxSuite ./...
```

## Development Workflow

1. **Feature branches**: Create feature branches for new functionality
2. **Test-driven**: Write tests first, especially for complex logic
3. **Container sharing**: When adding services that could share containers, follow the database service pattern
4. **Configuration**: Add new fields to `Service` struct in `config.go`
5. **Compose generation**: Hook into `generateDockerCompose()` in `compose.go`
6. **Test coverage**: Use table-driven tests for multiple scenarios

## Common Patterns

### Adding a New Service Type
1. Create dedicated file (e.g., `redis_services.go`)
2. Implement detection/configuration functions
3. Add hook in `compose.go` after line 230
4. Create comprehensive test file with suite pattern
5. Add example configurations in `examples/`

### Service Detection Pattern
```go
// Check if service needs special handling
if svc.FieldName != "" {
    addSpecialService(compose, &svc, config)
}
```

### Shared Container Pattern
```go
serviceName := getSharedServiceName(type, version)
if _, exists := compose.Services[serviceName]; exists {
    // Service exists, just add dependency
    return
}
// Create new shared service
```

### Service Naming Conventions
- **Shared services**: `{type}-{version}` format (e.g., `mysql-80`, `redis-72`)
- **PHP containers**: `{service}-php` format (e.g., `web-php`)
- **Volume names**: `{service}-data` format (e.g., `postgres-15-data`)
- **Singleton services**: Fixed names (e.g., `mailpit` for email testing)

## Additional Service Types

### Cache Services (`cache_services.go`)
- **Supported**: Redis (6.0-7.4), Memcached (1.6.x)
- **Configuration**: `cache = "redis:7.2"` or `cache = "memcached:1.6"`
- **Shared containers**: Services using same cache version share containers
- **Environment vars**: Sets REDIS_HOST, REDIS_URL, MEMCACHED_HOST, CACHE_DRIVER
- **Redis features**: Optional password auth, AOF persistence, max memory limits
- **Memcached features**: Memory limits, connection limits

### Search Services (`search_services.go`)
- **Supported**: Meilisearch (1.0-1.6), Typesense (0.24-27.1)
- **Configuration**: `search = "meilisearch:1.6"` or `search = "typesense:27.1"`
- **Meilisearch**: Master key auth, production/dev modes, analytics disabled
- **Typesense**: API key auth (required), CORS enabled
- **Environment vars**: MEILISEARCH_HOST, TYPESENSE_URL, SEARCH_ENGINE, etc.
- **Health checks**: Each search service has appropriate health monitoring

### Compatibility Services (`compat_services.go`)
- **Supported**: MinIO (S3-compatible storage)
- **Configuration**: `compat = "minio:2024"`
- **MinIO features**: S3 API compatibility, console UI, access/secret keys
- **Environment vars**: S3_ENDPOINT, AWS_ACCESS_KEY_ID, MINIO_ENDPOINT
- **Ports**: API on 9000, Console on 9001
- **Region support**: Configurable AWS region emulation

### Email Services (`email_service.go`)
- **Supported**: Mailpit (email testing, v1.13-1.20)
- **Configuration**: `email = "mailpit:1.20"`
- **Singleton pattern**: Only one email service per project
- **SMTP port**: 1025 (configurable)
- **Web UI port**: 8025 for viewing captured emails
- **Authentication**: Optional SMTP username/password
- **Environment vars**: SMTP_HOST, MAIL_HOST, MAILPIT_UI_URL
- **Features**: Captures all outgoing email for testing, provides web UI

### Laravel Reverb (`reverb_service.go`)
- **WebSocket server**: Real-time broadcasting for Laravel applications
- **Configuration**: `reverb = true` (only for Laravel/Lumen apps)
- **Singleton pattern**: One Reverb service shared by all Laravel apps
- **Custom settings**: 
  - `reverb_port`: WebSocket port (default: 8080)
  - `reverb_app_id`, `reverb_app_key`, `reverb_app_secret`: App credentials
- **Auto-configuration**: Sets BROADCAST_DRIVER, VITE_REVERB_* variables
- **Health checks**: Monitors WebSocket server availability
- **Shared code**: Mounts application code for artisan commands

### SSL Support (`ssl_service.go`)
- **Self-signed certificates**: Auto-generates certificates for HTTPS
- **Per-service configuration**: Each service can enable SSL independently
- **Configuration fields**:
  - `ssl = true`: Enable SSL for a service
  - `ssl_port = 443`: Custom HTTPS port (default: 443)
- **Certificate management**:
  - Certificates stored in `.fleet/ssl/`
  - Auto-renewal when certificates expire within 30 days
  - Default certificate for catch-all server
- **Nginx integration**: Automatic HTTPS configuration with:
  - TLS 1.2 and 1.3 support
  - Modern cipher suites
  - HTTP to HTTPS redirection
  - SSL session caching
- **Example configuration**:
  ```toml
  [[services]]
  name = "secure-web"
  domain = "secure.test"
  ssl = true
  ssl_port = 443  # Optional, defaults to 443
  ```