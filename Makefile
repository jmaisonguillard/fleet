.PHONY: build build-all clean deps install uninstall test dev

# Binary name
BINARY_NAME=fleet
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