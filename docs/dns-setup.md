# Fleet DNS Setup Guide

Fleet includes a built-in DNS service using dnsmasq to handle `.test` domain resolution for local development.

## Quick Start

```bash
# Setup DNS and modify hosts file
make dns-setup

# Start the dnsmasq container
make dns-start

# Test DNS resolution
make dns-test
```

## Features

- **Local .test TLD**: All `*.test` domains resolve to localhost by default
- **Custom domains**: Add custom .test domain mappings via configuration
- **Cross-platform**: Works on Linux, macOS, and Windows
- **Automatic backup**: Creates backups of your hosts file before modification
- **Easy removal**: Clean uninstall with restoration options

## Architecture

The DNS setup consists of:

1. **dnsmasq container**: Runs as a Docker container on port 53
2. **hosts file modification**: Points DNS queries to the local dnsmasq server
3. **Configuration files**: Manage domain mappings and DNS settings

## Installation

### Using Make (Recommended)

```bash
# Complete setup: backup hosts, modify hosts, start container
make dns-setup && make dns-start
```

### Manual Setup

#### Linux/macOS
```bash
# Run setup script (requires sudo)
./scripts/setup-dns.sh

# Start dnsmasq container
docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d
```

#### Windows (PowerShell as Administrator)
```powershell
# Run setup script
.\scripts\setup-dns.ps1

# Start dnsmasq container
docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d
```

## Configuration

### Default Configuration

- All `.test` domains resolve to `127.0.0.1`
- DNS server listens on `127.0.0.1:53`
- Upstream DNS: Google DNS (8.8.8.8, 8.8.4.4)

### Custom Domain Mappings

Edit `config/services/hosts.test` to add custom .test domains:

```
127.0.0.1 myapp.test
127.0.0.1 api.test
192.168.1.100 remote.test
```

After editing, restart the container:
```bash
make dns-stop && make dns-start
```

### Advanced Configuration

Edit `config/services/dnsmasq.conf` for advanced DNS settings:
- Change upstream DNS servers
- Modify cache settings
- Add additional domain rules
- Configure logging options

## Testing

Test DNS resolution using the included test script:

```bash
make dns-test
```

Or manually test specific domains:

```bash
# Using nslookup
nslookup test.test 127.0.0.1

# Using dig
dig @127.0.0.1 test.test

# Using host
host test.test 127.0.0.1
```

## Hosts File Backups

The setup script creates two backups of your hosts file:

1. **Timestamped backup**: `~/.fleet/backups/hosts_backup_YYYYMMDD_HHMMSS`
2. **Local backup**: `/etc/hosts.backup` (or equivalent on your OS)

## Troubleshooting

### Container not starting

Check if port 53 is already in use:
```bash
sudo lsof -i :53
```

Stop any conflicting services or change the port in docker-compose.yml.

### DNS not resolving

1. Check container status:
```bash
docker ps | grep fleet-dnsmasq
```

2. View container logs:
```bash
make dns-logs
```

3. Verify hosts file modification:
```bash
cat /etc/hosts | grep "Fleet DNS"
```

### Permission denied

- **Linux/macOS**: Run setup script with sudo or as root
- **Windows**: Run PowerShell as Administrator

## Uninstallation

Remove Fleet DNS configuration and restore original hosts file:

```bash
# Remove hosts file modifications
make dns-remove

# Stop and remove container
make dns-stop
```

Or manually:
```bash
./scripts/setup-dns.sh remove
docker-compose -f templates/compose/docker-compose.dnsmasq.yml down
```

## Security Considerations

- The dnsmasq container runs with `NET_ADMIN` capability for DNS operations
- DNS server binds only to localhost (127.0.0.1) by default
- Hosts file modifications are clearly marked for easy identification
- All changes are reversible with backup restoration

## Integration with Fleet Services

When running Fleet services, they can automatically use .test domains:

```yaml
# Example service configuration
services:
  web:
    domain: myapp.test
    port: 3000
```

The DNS service will resolve `myapp.test` to the appropriate container IP.