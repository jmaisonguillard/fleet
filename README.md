# Fleet - Simple Docker Service Orchestration

Fleet is a dead-simple Docker orchestration tool that makes it easy for developers (even beginners) to manage multiple Docker services with a single configuration file.

## Why Fleet?

- **Simple Configuration**: Just TOML/YAML/JSON - no complex syntax
- **Single Binary**: No dependencies, just download and run
- **Beginner Friendly**: Designed for developers new to Docker
- **Smart Defaults**: Sensible defaults that just work
- **Docker Compose Compatible**: Generates standard docker-compose.yml

## Quick Start

### 1. Build Fleet

```bash
# Using the build script (handles Go installation)
./build.sh

# Or with Make (if you have Go installed)
make build
```

### 2. Create Your First Project

```bash
# Create a sample configuration
./build/fleet init

# This creates:
# - fleet.toml (configuration)
# - website/index.html (sample website)
```

### 3. Start Your Services

```bash
# Start all services
./build/fleet up

# Or run in background
./build/fleet up -d
```

Your website is now running at http://localhost:8080 🚀

## Configuration

Fleet uses simple TOML configuration (also supports YAML/JSON):

```toml
project = "my-app"

# Simple web server
[[services]]
name = "website"
image = "nginx"
port = 80
folder = "./my-website"  # Maps to container's /app

# Database with password
[[services]]
name = "database"
image = "postgres"
port = 5432
password = "secret"  # Auto-configures based on image
```

## Commands

```bash
fleet init          # Create sample configuration
fleet up            # Start all services
fleet up -d         # Start in background
fleet down          # Stop all services
fleet restart       # Restart services
fleet status        # Show service status
fleet logs          # View all logs
fleet logs web      # View specific service logs
```

## Examples

### WordPress + MySQL

```toml
project = "my-blog"

[[services]]
name = "wordpress"
image = "wordpress"
port = 8080
[services.env]
WORDPRESS_DB_HOST = "mysql"
WORDPRESS_DB_PASSWORD = "secret"

[[services]]
name = "mysql"
image = "mysql:5.7"
password = "secret"
volumes = ["db-data:/var/lib/mysql"]
```

### Node.js + Redis + PostgreSQL

```toml
project = "node-app"

[[services]]
name = "api"
build = "./api"  # Build from Dockerfile
port = 3000
needs = ["postgres", "redis"]
[services.env]
DATABASE_URL = "postgresql://postgres:secret@postgres:5432/myapp"
REDIS_URL = "redis://redis:6379"

[[services]]
name = "postgres"
image = "postgres:15"
password = "secret"
volumes = ["pg-data:/var/lib/postgresql/data"]

[[services]]
name = "redis"
image = "redis:7"
```

## Building from Source

### Prerequisites
- Go 1.21+ (optional - build script can install it)
- Docker

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install to /usr/local/bin
make install

# Clean build artifacts
make clean
```

## How It Works

1. **Read Configuration**: Fleet reads your fleet.toml file
2. **Generate Docker Compose**: Converts to docker-compose.yml
3. **Run Docker**: Executes docker compose commands
4. **Manage Services**: Start, stop, restart with simple commands

## Features

- ✅ Multiple service orchestration
- ✅ Auto-detects database passwords
- ✅ Volume management
- ✅ Network isolation
- ✅ Service dependencies
- ✅ Health checks
- ✅ Build from Dockerfile
- ✅ Environment variables
- ✅ Cross-platform (Linux, macOS, Windows)

## License

MIT

## Contributing

Pull requests welcome! Keep it simple - that's the goal.

---

**Fleet**: Docker made simple for everyone 🚀