# PHP Composer Support Implementation Plan

## Overview
This document outlines the implementation plan for adding PHP Composer support to Fleet CLI, enabling developers to run Composer and PHP commands within their Fleet-managed Docker containers.

## Architecture Overview

### Core Components
1. **`fleet-php` binary** - A companion CLI tool that integrates with Fleet for PHP-specific commands
2. **Automatic deployment** - Fleet will copy `fleet-php` to `.fleet/bin/` when PHP projects are detected
3. **Context-aware execution** - Commands route to the correct PHP container based on service configuration

## Implementation Steps

### 1. Create PHP Runtime Manager (`php_runtime_manager.go`)
**Purpose**: Manage PHP service detection and routing logic

**Responsibilities**:
- Detect PHP services and their versions from fleet.toml
- Map services to their PHP-FPM containers (e.g., `web-php`, `api-php`)
- Handle multi-service routing logic
- Detect composer.json presence for auto-install trigger
- Provide service resolution for fleet-php CLI

**Key Functions**:
```go
- GetPHPServices() []PHPService
- GetDefaultPHPService() *PHPService
- GetPHPServiceByName(name string) *PHPService
- ShouldRunComposerInstall(service *PHPService) bool
- GetPHPContainerName(service *PHPService) string
```

### 2. Create fleet-php CLI (`cmd/fleet-php/main.go`)
**Purpose**: Standalone binary for PHP-specific commands

**Commands**:
- `fleet-php composer [args...]` - Run composer commands
- `fleet-php php [args...]` - Run PHP scripts
- `fleet-php artisan [args...]` - Laravel artisan commands (when Laravel detected)
- `fleet-php console [args...]` - Symfony console commands (when Symfony detected)

**Features**:
- Auto-detect framework from composer.json or framework files
- Execute commands via `docker exec` in appropriate PHP container
- Support `--service` flag for multi-service projects
- Version-specific routing based on service configuration
- Pass-through of all command arguments

### 3. Binary Deployment System (`binary_deployer.go`)
**Purpose**: Deploy fleet-php binary to project directory

**Functionality**:
- Embed fleet-php binary using `go:embed`
- Deploy to `.fleet/bin/fleet-php` when PHP runtime detected
- Set executable permissions
- Provide PATH setup instructions to user
- Clean up on `fleet down` with `--volumes` flag

**Implementation**:
```go
//go:embed embedded/fleet-php
var fleetPHPBinary []byte

func DeployPHPBinary() error
func RemovePHPBinary() error
func IsPHPBinaryDeployed() bool
```

### 4. Integrate with Main Fleet Commands

#### Modify `handleUp()` in `commands.go`:
- Check for PHP services in configuration
- Deploy fleet-php binary if PHP services exist
- Check for composer.json in service folders
- Auto-run `composer install` if:
  - composer.json exists
  - vendor/ directory doesn't exist
  - First time running the project
- Display available PHP commands to user

#### Modify `handleDown()`:
- Option to remove `.fleet/bin/` with `--volumes` flag
- Clean up deployed binaries

### 5. Framework-Specific Command Support

**Detection Logic**:
- Laravel: Check for `artisan` file and `laravel/framework` in composer.json
- Symfony: Check for `bin/console` or `symfony.lock`
- Lumen: Check for `artisan` and `laravel/lumen-framework` in composer.json
- WordPress: Check for `wp-config.php` or `wp-load.php`

**Command Availability**:
- All frameworks: `composer`, `php`
- Laravel/Lumen: Add `artisan`
- Symfony: Add `console`
- WordPress: Add `wp-cli` (future enhancement)

## File Structure

```
fleet/
├── cmd/
│   └── fleet-php/
│       ├── main.go              # fleet-php CLI entry point
│       ├── commands.go          # Command implementations
│       ├── detector.go          # Framework detection
│       └── executor.go          # Docker exec wrapper
├── php_runtime_manager.go       # PHP service detection/routing
├── binary_deployer.go           # Binary deployment to .fleet/bin/
├── embedded/
│   └── fleet-php               # Compiled fleet-php binary (embedded)
├── Makefile                     # Add fleet-php build targets
└── .fleet/                     # (generated at runtime)
    ├── bin/
    │   └── fleet-php           # Deployed binary
    └── docker-compose.yml      # Existing compose file
```

## User Experience

### Automatic Composer Install
```bash
# First run with composer.json present
$ fleet up
🚀 Starting Fleet project: my-app
📦 PHP project detected, deploying fleet-php CLI...
📦 Running composer install for service 'web'...
✅ Dependencies installed
✅ Services started

# Subsequent runs (vendor/ exists)
$ fleet up
🚀 Starting Fleet project: my-app
✅ Services started
💡 PHP commands available: fleet-php composer, fleet-php php, fleet-php artisan
```

### Manual Commands
```bash
# Composer commands
$ fleet-php composer require laravel/sanctum
$ fleet-php composer update
$ fleet-php composer dump-autoload

# Framework commands
$ fleet-php artisan migrate
$ fleet-php artisan make:controller UserController
$ fleet-php console cache:clear

# Generic PHP
$ fleet-php php -v
$ fleet-php php script.php
```

### Multi-Service Projects
```bash
# Default service (first PHP service)
$ fleet-php composer install

# Specific service
$ fleet-php --service=api composer update
$ fleet-php --service=worker artisan queue:work
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. Create `php_runtime_manager.go`
2. Create `binary_deployer.go`
3. Set up embedded binary structure

### Phase 2: fleet-php CLI
1. Create cmd/fleet-php structure
2. Implement basic composer and php commands
3. Add docker exec integration
4. Test with single service

### Phase 3: Fleet Integration
1. Modify handleUp() for auto-deployment
2. Add composer install auto-run
3. Update handleDown() for cleanup
4. Add user notifications

### Phase 4: Framework Support
1. Add framework detection
2. Enable framework-specific commands
3. Add multi-service support
4. Test with complex projects

### Phase 5: Polish & Testing
1. Add comprehensive tests
2. Update documentation
3. Add examples
4. Handle edge cases

## Technical Considerations

### Docker Execution
- Use `docker exec -w /app` to run in correct directory
- Pass environment variables from service configuration
- Handle interactive vs non-interactive commands
- Support TTY allocation for interactive commands

### Version Management
- Detect PHP version from runtime field
- Route to correct PHP-FPM container
- Support multiple PHP versions in same project

### Error Handling
- Graceful fallback if Docker not running
- Clear error messages for missing services
- Validate commands before execution
- Handle permission issues

### Performance
- Cache service detection results
- Minimize Docker API calls
- Quick binary deployment
- Efficient command pass-through

## Success Criteria

1. **Zero Configuration**: Works out of the box for single PHP service projects
2. **Auto-Install**: Automatically runs composer install on first run
3. **Framework Aware**: Enables relevant commands based on detected framework
4. **Multi-Service**: Handles projects with multiple PHP services gracefully
5. **User Friendly**: Clear messages and intuitive command structure
6. **Clean Separation**: Fleet handles orchestration, fleet-php handles PHP operations

## Future Enhancements

1. **Package Caching**: Mount composer cache volume for faster installs
2. **WP-CLI Support**: Add WordPress CLI commands
3. **REPL Support**: Interactive PHP shell with proper TTY
4. **Composer Scripts**: Run composer scripts directly
5. **Auto-Update**: Detect composer.json changes and prompt for update
6. **Global Packages**: Support for global composer packages
7. **PHP Extensions**: Auto-install required PHP extensions from composer.json

## Notes

- Keep fleet-php binary small and focused
- Ensure backward compatibility with existing Fleet projects
- Follow Go best practices for CLI development
- Maintain consistency with Fleet's existing command structure
- Prioritize developer experience and simplicity