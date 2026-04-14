# Build configuration
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Directories and files
BUILD_DIR := ./build
BINARY_NAME := memenow-resource-manager
CMD_DIR := ./cmd

# Docker configuration
DOCKER_IMAGE_NAME := ghcr.io/memenow/memenow-resource-manager

# Go build configuration
GO := go
GOFLAGS := -v
CGO_ENABLED := 0
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Linker flags to inject version information
LDFLAGS := -w -s \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT)

# Build command
GO_BUILD_CMD := CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

.PHONY: all build clean test lint fmt vet container container-push help

# Default target
all: clean fmt vet test build

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  lint           - Run golangci-lint"
	@echo "  fmt            - Format code with gofmt"
	@echo "  vet            - Run go vet"
	@echo "  container      - Build Docker container"
	@echo "  container-push - Build and push Docker container"
	@echo "  all            - Run clean, fmt, vet, test, and build"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD_CMD)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install from https://golangci-lint.run/"; \
		exit 1; \
	fi

## fmt: Format all Go code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "Formatting complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "Vet complete"

## mod-download: Download dependencies
mod-download:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@echo "Dependencies downloaded"

## mod-tidy: Tidy dependencies
mod-tidy:
	@echo "Tidying dependencies..."
	@$(GO) mod tidy
	@echo "Dependencies tidied"

## mod-verify: Verify dependencies
mod-verify:
	@echo "Verifying dependencies..."
	@$(GO) mod verify
	@echo "Dependencies verified"

## container: Build Docker container
container: build
	@echo "Building Docker image $(DOCKER_IMAGE_NAME):$(VERSION)..."
	@docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) $(BUILD_DIR)
	@docker tag $(DOCKER_IMAGE_NAME):$(VERSION) $(DOCKER_IMAGE_NAME):latest
	@echo "Docker image built: $(DOCKER_IMAGE_NAME):$(VERSION)"

## container-push: Build and push Docker container
container-push: container
	@echo "Pushing Docker image $(DOCKER_IMAGE_NAME):$(VERSION)..."
	@docker push $(DOCKER_IMAGE_NAME):$(VERSION)
	@docker push $(DOCKER_IMAGE_NAME):latest
	@echo "Docker image pushed"

## run: Build and run the binary
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"
