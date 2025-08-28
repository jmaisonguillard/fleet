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

# Run all tests
make test
# Or with Go directly
go test -v ./...

# Run specific test suite
go test -v -run TestComposeSuite ./...
go test -v -run TestDNSSuite ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Install to /usr/local/bin
make install

# Development mode (run without building)
make dev ARGS="up -d"
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
  - Creates fake docker executable in PATH
  - Simulates compose, ps, logs commands
  - Enables testing without Docker installed
- **Test Helpers**: `test_helpers.go` provides utilities for temp files and sample configs

### Important Implementation Details

1. **Network Naming**: Always creates "fleet-network" (not project-based naming)

2. **Port Mapping**: Maps ports as `port:port` (e.g., 8080:8080, not 8080:80)

3. **Environment Variables**: Stored as maps, not "key=value" strings

4. **Cross-Platform Compatibility**:
   - Script detection: `getScriptPath()` checks OS and selects .ps1 or .sh
   - Path handling: Use `filepath` package for OS-agnostic paths
   - Embedded resources work across all platforms

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