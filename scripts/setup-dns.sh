#!/bin/bash

# Fleet DNS Setup Script
# This script backs up the system hosts file and configures it to use the Fleet dnsmasq server

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# DNS server IP (localhost where dnsmasq container runs)
DNSMASQ_IP="127.0.0.1"

# Function to detect OS
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "win32" ]]; then
        echo "windows"
    else
        echo "unknown"
    fi
}

# Function to get hosts file path based on OS
get_hosts_path() {
    local os=$1
    case $os in
        linux|macos)
            echo "/etc/hosts"
            ;;
        windows)
            echo "/c/Windows/System32/drivers/etc/hosts"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Function to backup hosts file
backup_hosts() {
    local hosts_file=$1
    local backup_dir="$HOME/.fleet/backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$backup_dir/hosts_backup_$timestamp"
    local hosts_dir=$(dirname "$hosts_file")
    local local_backup="${hosts_dir}/hosts.backup"
    
    # Create backup directory if it doesn't exist
    mkdir -p "$backup_dir"
    
    # Check if hosts file exists
    if [ ! -f "$hosts_file" ]; then
        echo -e "${RED}Error: Hosts file not found at $hosts_file${NC}"
        exit 1
    fi
    
    # Create backups
    echo -e "${YELLOW}Creating backup of hosts file...${NC}"
    if [ "$OS" == "windows" ]; then
        # For Windows, we need to handle permissions differently
        cp "$hosts_file" "$backup_file" 2>/dev/null || {
            echo -e "${RED}Error: Failed to backup hosts file. Please run as Administrator.${NC}"
            exit 1
        }
        cp "$hosts_file" "$local_backup" 2>/dev/null
    else
        # For Unix-like systems, use sudo if needed
        if [ -w "$hosts_file" ]; then
            cp "$hosts_file" "$backup_file"
            cp "$hosts_file" "$local_backup"
        else
            sudo cp "$hosts_file" "$backup_file"
            sudo chown $(whoami) "$backup_file"
            sudo cp "$hosts_file" "$local_backup"
        fi
    fi
    
    echo -e "${GREEN}Backup created at: $backup_file${NC}"
    echo -e "${GREEN}Local backup created at: $local_backup${NC}"
    echo "$backup_file"
}

# Function to check if Fleet DNS entry already exists
check_existing_entry() {
    local hosts_file=$1
    if grep -q "# Fleet DNS Configuration" "$hosts_file" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to add Fleet DNS configuration to hosts file
add_fleet_dns() {
    local hosts_file=$1
    local temp_file=$(mktemp)
    
    # Check if entry already exists
    if check_existing_entry "$hosts_file"; then
        echo -e "${YELLOW}Fleet DNS configuration already exists in hosts file.${NC}"
        read -p "Do you want to update it? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}Skipping hosts file modification.${NC}"
            return 0
        fi
        # Remove existing Fleet DNS configuration
        sed '/# Fleet DNS Configuration - Start/,/# Fleet DNS Configuration - End/d' "$hosts_file" > "$temp_file"
    else
        cat "$hosts_file" > "$temp_file"
    fi
    
    # Add Fleet DNS configuration
    cat >> "$temp_file" << EOF

# Fleet DNS Configuration - Start
# This configuration routes .test domains to the Fleet dnsmasq server
# To remove, delete everything between the Start and End markers
${DNSMASQ_IP} dnsmasq.test
# Fleet DNS Configuration - End
EOF
    
    # Update hosts file
    echo -e "${YELLOW}Adding Fleet DNS configuration to hosts file...${NC}"
    if [ "$OS" == "windows" ]; then
        # For Windows
        cp "$temp_file" "$hosts_file" 2>/dev/null || {
            echo -e "${RED}Error: Failed to update hosts file. Please run as Administrator.${NC}"
            rm "$temp_file"
            exit 1
        }
    else
        # For Unix-like systems
        if [ -w "$hosts_file" ]; then
            cp "$temp_file" "$hosts_file"
        else
            sudo cp "$temp_file" "$hosts_file"
        fi
    fi
    
    rm "$temp_file"
    echo -e "${GREEN}Fleet DNS configuration added successfully!${NC}"
}

# Function to remove Fleet DNS configuration
remove_fleet_dns() {
    local hosts_file=$1
    local temp_file=$(mktemp)
    
    if ! check_existing_entry "$hosts_file"; then
        echo -e "${YELLOW}No Fleet DNS configuration found in hosts file.${NC}"
        return 0
    fi
    
    # Remove Fleet DNS configuration
    sed '/# Fleet DNS Configuration - Start/,/# Fleet DNS Configuration - End/d' "$hosts_file" > "$temp_file"
    
    # Update hosts file
    echo -e "${YELLOW}Removing Fleet DNS configuration from hosts file...${NC}"
    if [ "$OS" == "windows" ]; then
        cp "$temp_file" "$hosts_file" 2>/dev/null || {
            echo -e "${RED}Error: Failed to update hosts file. Please run as Administrator.${NC}"
            rm "$temp_file"
            exit 1
        }
    else
        if [ -w "$hosts_file" ]; then
            cp "$temp_file" "$hosts_file"
        else
            sudo cp "$temp_file" "$hosts_file"
        fi
    fi
    
    rm "$temp_file"
    echo -e "${GREEN}Fleet DNS configuration removed successfully!${NC}"
}

# Main script
main() {
    echo -e "${GREEN}Fleet DNS Setup Script${NC}"
    echo "========================"
    
    # Detect OS
    OS=$(detect_os)
    echo -e "Detected OS: ${YELLOW}$OS${NC}"
    
    if [ "$OS" == "unknown" ]; then
        echo -e "${RED}Error: Unsupported operating system${NC}"
        exit 1
    fi
    
    # Get hosts file path
    HOSTS_FILE=$(get_hosts_path "$OS")
    echo -e "Hosts file location: ${YELLOW}$HOSTS_FILE${NC}"
    
    # Check for command argument
    if [ "$1" == "remove" ]; then
        remove_fleet_dns "$HOSTS_FILE"
        exit 0
    fi
    
    # Backup hosts file
    BACKUP_FILE=$(backup_hosts "$HOSTS_FILE")
    
    # Add Fleet DNS configuration
    add_fleet_dns "$HOSTS_FILE"
    
    echo
    echo -e "${GREEN}Setup complete!${NC}"
    echo -e "Your original hosts file has been backed up to:"
    echo -e "  - ${YELLOW}$BACKUP_FILE${NC} (timestamped)"
    echo -e "  - ${YELLOW}$(dirname "$HOSTS_FILE")/hosts.backup${NC} (local)"
    echo
    echo "To test the DNS configuration:"
    echo "  1. Start the dnsmasq container: docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d"
    echo "  2. Test DNS resolution: nslookup test.test 127.0.0.1"
    echo
    echo "To remove Fleet DNS configuration:"
    echo "  Run: $0 remove"
}

# Run main function
main "$@"