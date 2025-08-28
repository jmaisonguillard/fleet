# Fleet DNS Setup Script for Windows PowerShell
# This script backs up the system hosts file and configures it to use the Fleet dnsmasq server

param(
    [Parameter(Position=0)]
    [string]$Action = "setup"
)

$ErrorActionPreference = "Stop"

# Configuration
$DNSMASQ_IP = "127.0.0.1"
$HOSTS_FILE = "$env:SystemRoot\System32\drivers\etc\hosts"
$HOSTS_DIR = "$env:SystemRoot\System32\drivers\etc"
$BACKUP_DIR = "$env:USERPROFILE\.fleet\backups"

# Colors for output
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

# Check if running as Administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Backup hosts file
function Backup-HostsFile {
    Write-Host "Creating backup of hosts file..." -ForegroundColor Yellow
    
    # Create backup directory if it doesn't exist
    if (!(Test-Path $BACKUP_DIR)) {
        New-Item -ItemType Directory -Path $BACKUP_DIR -Force | Out-Null
    }
    
    # Check if hosts file exists
    if (!(Test-Path $HOSTS_FILE)) {
        Write-Host "Error: Hosts file not found at $HOSTS_FILE" -ForegroundColor Red
        exit 1
    }
    
    # Create backups with timestamp
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupFile = "$BACKUP_DIR\hosts_backup_$timestamp"
    $localBackup = "$HOSTS_DIR\hosts.backup"
    
    try {
        # Create timestamped backup in user directory
        Copy-Item -Path $HOSTS_FILE -Destination $backupFile -Force
        Write-Host "Backup created at: $backupFile" -ForegroundColor Green
        
        # Create local backup in the same directory as hosts file
        Copy-Item -Path $HOSTS_FILE -Destination $localBackup -Force
        Write-Host "Local backup created at: $localBackup" -ForegroundColor Green
        
        return $backupFile
    }
    catch {
        Write-Host "Error: Failed to backup hosts file. $_" -ForegroundColor Red
        exit 1
    }
}

# Check if Fleet DNS entry already exists
function Test-ExistingEntry {
    $content = Get-Content $HOSTS_FILE -Raw
    return $content -match "# Fleet DNS Configuration"
}

# Add Fleet DNS configuration
function Add-FleetDNS {
    Write-Host "Checking for existing Fleet DNS configuration..." -ForegroundColor Yellow
    
    $content = Get-Content $HOSTS_FILE -Raw
    
    if (Test-ExistingEntry) {
        Write-Host "Fleet DNS configuration already exists in hosts file." -ForegroundColor Yellow
        $response = Read-Host "Do you want to update it? (y/n)"
        if ($response -ne 'y' -and $response -ne 'Y') {
            Write-Host "Skipping hosts file modification." -ForegroundColor Yellow
            return
        }
        # Remove existing Fleet DNS configuration
        $content = $content -replace '(?ms)# Fleet DNS Configuration - Start.*?# Fleet DNS Configuration - End\r?\n?', ''
    }
    
    # Add Fleet DNS configuration
    $fleetConfig = @"

# Fleet DNS Configuration - Start
# This configuration routes .test domains to the Fleet dnsmasq server
# To remove, delete everything between the Start and End markers
$DNSMASQ_IP dnsmasq.test
# Fleet DNS Configuration - End
"@
    
    $newContent = $content.TrimEnd() + $fleetConfig
    
    try {
        Write-Host "Adding Fleet DNS configuration to hosts file..." -ForegroundColor Yellow
        Set-Content -Path $HOSTS_FILE -Value $newContent -Force
        Write-Host "Fleet DNS configuration added successfully!" -ForegroundColor Green
    }
    catch {
        Write-Host "Error: Failed to update hosts file. $_" -ForegroundColor Red
        exit 1
    }
}

# Remove Fleet DNS configuration
function Remove-FleetDNS {
    if (!(Test-ExistingEntry)) {
        Write-Host "No Fleet DNS configuration found in hosts file." -ForegroundColor Yellow
        return
    }
    
    Write-Host "Removing Fleet DNS configuration from hosts file..." -ForegroundColor Yellow
    
    $content = Get-Content $HOSTS_FILE -Raw
    $content = $content -replace '(?ms)# Fleet DNS Configuration - Start.*?# Fleet DNS Configuration - End\r?\n?', ''
    
    try {
        Set-Content -Path $HOSTS_FILE -Value $content.TrimEnd() -Force
        Write-Host "Fleet DNS configuration removed successfully!" -ForegroundColor Green
    }
    catch {
        Write-Host "Error: Failed to update hosts file. $_" -ForegroundColor Red
        exit 1
    }
}

# Main script
Write-Host "Fleet DNS Setup Script" -ForegroundColor Green
Write-Host "======================" -ForegroundColor Green

# Check if running as Administrator
if (!(Test-Administrator)) {
    Write-Host "Error: This script must be run as Administrator." -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

Write-Host "Running as Administrator" -ForegroundColor Green
Write-Host "Hosts file location: $HOSTS_FILE" -ForegroundColor Yellow

switch ($Action) {
    "remove" {
        Remove-FleetDNS
    }
    "setup" {
        # Backup hosts file
        $backupFile = Backup-HostsFile
        
        # Add Fleet DNS configuration
        Add-FleetDNS
        
        Write-Host ""
        Write-Host "Setup complete!" -ForegroundColor Green
        Write-Host "Your original hosts file has been backed up to:" -ForegroundColor Yellow
        Write-Host "  - $backupFile (timestamped)" -ForegroundColor Yellow
        Write-Host "  - $HOSTS_DIR\hosts.backup (local)" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "To test the DNS configuration:"
        Write-Host "  1. Start the dnsmasq container: docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d"
        Write-Host "  2. Test DNS resolution: nslookup test.test 127.0.0.1"
        Write-Host ""
        Write-Host "To remove Fleet DNS configuration:"
        Write-Host "  Run: .\setup-dns.ps1 remove"
    }
    default {
        Write-Host "Invalid action. Use 'setup' or 'remove'" -ForegroundColor Red
        exit 1
    }
}