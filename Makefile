.PHONY: build clean install run dev deps

# Binary name
BINARY_NAME=fleet
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

build:
	@echo "🔨 Building fleet binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	# macOS ARM64 (M1/M2)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
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
	@echo "📦 Installing fleet..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Fleet installed to /usr/local/bin/$(BINARY_NAME)"

uninstall:
	@echo "🗑️ Uninstalling fleet..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Fleet uninstalled"

# Development helpers
dev:
	@$(GOCMD) run . $(ARGS)

test:
	@$(GOTEST) -v ./...

init:
	@./$(BUILD_DIR)/$(BINARY_NAME) init

up: build
	@./$(BUILD_DIR)/$(BINARY_NAME) up

down:
	@./$(BUILD_DIR)/$(BINARY_NAME) down

status:
	@./$(BUILD_DIR)/$(BINARY_NAME) status

# DNS Setup
dns-setup:
	@echo "🌐 Setting up Fleet DNS for .test domain..."
	@./scripts/setup-dns.sh
	@echo "✅ DNS setup complete"

dns-remove:
	@echo "🗑️ Removing Fleet DNS configuration..."
	@./scripts/setup-dns.sh remove
	@echo "✅ DNS configuration removed"

dns-start:
	@echo "🚀 Starting dnsmasq container..."
	@docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d
	@echo "✅ Dnsmasq started"

dns-stop:
	@echo "🛑 Stopping dnsmasq container..."
	@docker-compose -f templates/compose/docker-compose.dnsmasq.yml down
	@echo "✅ Dnsmasq stopped"

dns-test:
	@echo "🧪 Testing DNS configuration..."
	@./scripts/test-dns.sh

dns-logs:
	@echo "📋 Showing dnsmasq logs..."
	@docker logs dnsmasq -f