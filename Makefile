.PHONY: build build-all build-fleet-php clean deps install uninstall test dev

# Binary name
BINARY_NAME=fleet
PHP_BINARY_NAME=fleet-php
NODE_BINARY_NAME=fleet-node
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
build:
	@echo "🔨 Building fleet binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"
	@$(MAKE) build-fleet-php
	@$(MAKE) build-fleet-node

# Build fleet-php binary
build-fleet-php:
	@echo "🔨 Building fleet-php binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME) ./cmd/fleet-php
	@echo "✅ PHP binary built: $(BUILD_DIR)/$(PHP_BINARY_NAME)"

# Build fleet-node binary
build-fleet-node:
	@echo "🔨 Building fleet-node binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME) ./cmd/fleet-node
	@echo "✅ Node binary built: $(BUILD_DIR)/$(NODE_BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME)-linux-amd64 ./cmd/fleet-php
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME)-linux-amd64 ./cmd/fleet-node
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME)-linux-arm64 ./cmd/fleet-php
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME)-linux-arm64 ./cmd/fleet-node
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME)-darwin-amd64 ./cmd/fleet-php
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME)-darwin-amd64 ./cmd/fleet-node
	# macOS ARM64 (M1/M2)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME)-darwin-arm64 ./cmd/fleet-php
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME)-darwin-arm64 ./cmd/fleet-node
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(PHP_BINARY_NAME)-windows-amd64.exe ./cmd/fleet-php
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(NODE_BINARY_NAME)-windows-amd64.exe ./cmd/fleet-node
	@echo "✅ All binaries built in $(BUILD_DIR)/"

clean:
	@echo "🧹 Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf .fleet
	@echo "✅ Cleaned"

deps:
	@echo "📦 Getting dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "✅ Dependencies installed"

install: build
	@echo "📦 Installing fleet to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Fleet installed successfully"

uninstall:
	@echo "🗑️ Uninstalling fleet..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Fleet uninstalled"

test:
	@echo "🧪 Running tests..."
	@$(GOTEST) -v ./...

# Development helper - runs the application without building
dev:
	@$(GOCMD) run . $(ARGS)