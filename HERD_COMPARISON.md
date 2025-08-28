# Laravel Herd vs Fleet Comparison

## Laravel Herd Services Overview

Laravel Herd is a native PHP development environment for macOS and Windows that provides a comprehensive set of services for Laravel development. Below is a complete breakdown of what Herd offers and how Fleet currently compares.

## Herd Core Features

### PHP Support
- **Versions**: PHP 7.4, 8.0, 8.1, 8.2, 8.3, 8.4
- **Version switching**: Instant switching between versions
- **Site isolation**: Pin specific PHP versions per site
- **Performance**: 35% faster tests, 100% faster web requests

### Web Server
- **nginx**: Built-in with automatic configuration
- **SSL**: Automatic SSL certificates for local development
- **DNS**: dnsmasq for .test domains
- **Site management**: Easy site creation and management

### Command Line Tools (Free)
- php
- composer
- laravel installer
- expose (tunnel service)
- node, npm, nvm

## Herd Pro Services

### Databases
1. **MySQL**
   - Multiple versions available
   - Default port: 3307
   - TablePlus integration

2. **PostgreSQL**
   - Versions: 14, 15, 16
   - Extensions: PostGIS, pgrouting, pgvector
   - TablePlus integration

3. **MongoDB**
   - Full MongoDB support
   - GUI management tools

### Caching & Queues
1. **Redis**
   - Multiple versions (specific versions not documented)
   - Queue management
   - Cache storage
   - TablePlus integration

### Search Engines
1. **Meilisearch**
   - Full-text search
   - Multiple versions

2. **Typesense**
   - Lightning-fast search
   - Multiple versions

### Storage
1. **MinIO**
   - S3-compatible storage
   - Local S3 testing

### Real-time
1. **Laravel Reverb**
   - WebSocket server
   - Real-time broadcasting

### Development Tools (Herd Pro)
1. **Mail Service**
   - Local email testing
   - Separate inboxes per application
   - Email debugging

2. **Log Viewer**
   - Real-time log monitoring
   - IDE integration
   - Search capabilities

3. **Dumps**
   - Intercept dump() and dd() calls
   - Better debugging experience

4. **Xdebug**
   - Automatic detection
   - PHPStorm integration
   - Breakpoint scanning

5. **Profiler**
   - Performance analysis
   - Bottleneck identification

6. **Forge Integration**
   - Deploy from Herd UI
   - Sync with Laravel Forge

## Fleet Current Capabilities

### ✅ Already Implemented
1. **PHP Support** (`runtime_php.go`)
   - Versions: 7.4, 8.0, 8.1, 8.2, 8.3, 8.4
   - Framework detection: Laravel, Symfony, WordPress, Drupal, CodeIgniter, Slim, Lumen
   - PHP-FPM containers

2. **Web Server** (`nginx.go`)
   - nginx proxy with auto-configuration
   - Domain mapping (.test domains)
   - Hosts file management

3. **Databases** (`database_services.go`)
   - MySQL (multiple versions)
   - PostgreSQL (multiple versions)
   - MariaDB
   - MongoDB

4. **Caching** (`cache_services.go`)
   - Redis: 6.0, 6.2, 7.0, 7.2, 7.4
   - Memcached: 1.6.x series

5. **Search** (`search_services.go`)
   - Meilisearch: 1.0-1.6
   - Typesense: 0.24-27.1

6. **Storage** (`compat_services.go`)
   - MinIO: 2023, 2024 releases

7. **Email Testing** (`email_service.go`)
   - Mailpit: v1.13-1.20
   - SMTP testing
   - Web UI for viewing emails

8. **DNS** (`dns.go`)
   - dnsmasq container
   - .test domain resolution

## Gap Analysis - Services Fleet is Missing

### Critical Gaps
1. **Laravel Reverb** (WebSocket server)
   - Not implemented
   - Important for real-time features

2. **Development Tools**
   - No Xdebug support
   - No profiler
   - No dumps interceptor
   - No integrated log viewer

3. **Database Extensions**
   - PostgreSQL extensions (PostGIS, pgrouting, pgvector) not configured
   - No TablePlus integration

4. **Command Line Tools**
   - No built-in composer
   - No Laravel installer
   - No expose tunnel service
   - No node/npm/nvm management

### Minor Gaps
1. **SSL Certificates**
   - No automatic SSL for local development
   - Herd uses self-signed certificates

2. **Site Isolation**
   - No per-site PHP version pinning
   - All services share same PHP version

3. **Service UI**
   - No GUI for service management
   - Command-line only

4. **Forge Integration**
   - No deployment integration
   - No cloud provider connections

## Recommended Implementation Priority

### Phase 1: Critical Laravel Development Features
1. **Laravel Reverb Support** (High Priority)
   - Create `reverb_service.go`
   - WebSocket server configuration
   - Broadcasting integration

2. **Xdebug Support** (High Priority)
   - Add to PHP runtime configuration
   - Environment variable management
   - Port configuration (9003)

3. **PostgreSQL Extensions** (Medium Priority)
   - Add PostGIS, pgvector support
   - Configure in `database_services.go`

### Phase 2: Developer Experience
1. **SSL Support** (Medium Priority)
   - Self-signed certificates
   - nginx HTTPS configuration
   - Certificate generation

2. **Log Viewer** (Low Priority)
   - Could integrate with existing container logs
   - Web-based viewer optional

3. **Node.js Support** (Medium Priority)
   - Add Node runtime support
   - npm/yarn package management

### Phase 3: Nice-to-Have Features
1. **Profiler Integration**
   - Blackfire or XHProf support

2. **Database GUI Integration**
   - Connection string generation
   - TablePlus/DBeaver configs

## Implementation Notes

### For Laravel Reverb
```toml
[[services]]
name = "api"
runtime = "php:8.3"
framework = "laravel"
reverb = true  # Auto-configure Laravel Reverb
```

### For Xdebug
```toml
[[services]]
name = "api"
runtime = "php:8.3"
debug = true  # Enable Xdebug
debug_port = 9003
```

### For PostgreSQL Extensions
```toml
[[services]]
name = "api"
database = "postgres:16"
database_extensions = ["postgis", "pgvector"]
```

## Summary

Fleet already covers most of Herd's core infrastructure services (databases, cache, search, storage, email). The main gaps are in developer tools (Xdebug, profiling) and Laravel-specific features (Reverb). Implementing Laravel Reverb and Xdebug support would make Fleet a very competitive alternative to Herd for Laravel development.