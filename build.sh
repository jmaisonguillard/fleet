#!/bin/bash

# Fleet CLI Build Script
# This script builds the Fleet binary without requiring Go installed

set -e

echo "🚀 Fleet Build Script"
echo "===================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}⚠️  Go is not installed${NC}"
    echo "Would you like to:"
    echo "1) Install Go automatically (requires sudo)"
    echo "2) Get instructions for manual installation"
    echo "3) Use Docker to build (requires Docker)"
    echo "4) Exit"
    read -p "Choose option (1-4): " choice
    
    case $choice in
        1)
            echo "Installing Go..."
            # Download and install Go
            GO_VERSION="1.21.5"
            wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
            rm go${GO_VERSION}.linux-amd64.tar.gz
            
            # Detect user's shell and update appropriate config file
            SHELL_NAME=$(basename "$SHELL")
            GO_PATH_LINE='export PATH=$PATH:/usr/local/go/bin'
            
            # Function to add PATH to a config file if not already present
            add_path_to_file() {
                local file=$1
                if [ -f "$file" ]; then
                    if ! grep -q "/usr/local/go/bin" "$file"; then
                        echo "$GO_PATH_LINE" >> "$file"
                        echo "   Added Go to PATH in $file"
                    else
                        echo "   Go PATH already in $file"
                    fi
                fi
            }
            
            # Update shell config based on detected shell
            case "$SHELL_NAME" in
                zsh)
                    # For zsh, update .zshrc
                    add_path_to_file "$HOME/.zshrc"
                    # Also update .zprofile for login shells
                    add_path_to_file "$HOME/.zprofile"
                    ;;
                bash)
                    # For bash, update .bashrc for interactive shells
                    add_path_to_file "$HOME/.bashrc"
                    # Also update .bash_profile or .profile for login shells
                    if [ -f "$HOME/.bash_profile" ]; then
                        add_path_to_file "$HOME/.bash_profile"
                    elif [ -f "$HOME/.profile" ]; then
                        add_path_to_file "$HOME/.profile"
                    fi
                    ;;
                fish)
                    # For fish shell
                    FISH_CONFIG="$HOME/.config/fish/config.fish"
                    mkdir -p "$HOME/.config/fish"
                    if [ -f "$FISH_CONFIG" ]; then
                        if ! grep -q "/usr/local/go/bin" "$FISH_CONFIG"; then
                            echo "set -gx PATH \$PATH /usr/local/go/bin" >> "$FISH_CONFIG"
                            echo "   Added Go to PATH in $FISH_CONFIG"
                        fi
                    fi
                    ;;
                *)
                    # Default: update .profile as fallback
                    add_path_to_file "$HOME/.profile"
                    add_path_to_file "$HOME/.bashrc"
                    echo "   Unknown shell ($SHELL_NAME), updated .profile and .bashrc"
                    ;;
            esac
            
            # Export for current session
            export PATH=$PATH:/usr/local/go/bin
            
            echo -e "${GREEN}✅ Go installed successfully${NC}"
            echo "   Go version: $(go version 2>/dev/null || echo 'Please restart your shell')"
            echo ""
            echo "   ${YELLOW}Note: You may need to restart your shell or run:${NC}"
            echo "   ${GREEN}source ~/.$SHELL_NAME*rc${NC}"
            ;;
        2)
            echo -e "${YELLOW}To install Go manually:${NC}"
            echo "1. Visit https://golang.org/dl/"
            echo "2. Download the installer for your OS"
            echo "3. Follow the installation instructions"
            echo "4. Run this script again"
            exit 0
            ;;
        3)
            echo "Building with Docker..."
            # Use Docker to build
            docker run --rm \
                -v "$PWD/cli":/app \
                -v "$PWD/build":/build \
                -w /app \
                golang:1.21-alpine \
                sh -c "apk add --no-cache git && go mod download && go build -ldflags '-s -w' -o /build/fleet ."
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✅ Fleet built successfully with Docker${NC}"
                echo "Binary location: ./build/fleet"
                chmod +x ./build/fleet
                echo ""
                echo "To install system-wide: sudo cp ./build/fleet /usr/local/bin/"
            else
                echo -e "${RED}❌ Build failed${NC}"
                exit 1
            fi
            exit 0
            ;;
        4)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            exit 1
            ;;
    esac
fi

# Go is installed, proceed with build
echo "Building Fleet CLI..."

cd cli

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download
go mod tidy

# Build binary
echo "🔨 Building binary..."
mkdir -p ../build
go build -ldflags "-s -w" -o ../build/fleet .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Build successful!${NC}"
    echo "Binary location: ./build/fleet"
    chmod +x ../build/fleet
    
    echo ""
    echo "Next steps:"
    echo "  1. Test: ./build/fleet help"
    echo "  2. Install system-wide: sudo cp ./build/fleet /usr/local/bin/"
    echo "  3. Create config: fleet init"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi