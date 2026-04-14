# memenow-resource-manager

A lightweight Kubernetes resource management service that provides HTTP API endpoints for managing Helm chart installations.

## Overview

This service provides RESTful API endpoints to manage Helm chart deployments on Kubernetes clusters. It's built with Go and uses the Gin web framework for HTTP routing and Helm v3 SDK for chart management.

## Features

- RESTful API for Helm chart installation
- Graceful shutdown handling
- Request timeout and context cancellation support
- Input validation and comprehensive error handling
- Health check endpoints
- Structured logging
- Static binary compilation for containerized deployments

## Requirements

- Go 1.26 or later
- Kubernetes cluster access
- Helm v3

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/memenow/memenow-resource-manager.git
cd memenow-resource-manager

# Build the binary
make build

# Run the service
./build/memenow-resource-manager
```

### Using Docker

```bash
# Build Docker image
make container

# Run the container
docker run -p 8080:8080 ghcr.io/memenow/memenow-resource-manager:latest
```

## Configuration

The service can be configured using environment variables:

- `PORT`: HTTP server port (default: `8080`)
- `GIN_MODE`: Gin framework mode (`debug`, `release`, `test`)
- `HELM_DRIVER`: Helm storage driver (e.g., `secret`, `configmap`)

## API Endpoints

### Health Check

Check if the service is running:

```bash
GET /ok
GET /health
```

Response:
```json
{
  "status": "healthy",
  "message": "Service is running"
}
```

### Version Information

Get build version and metadata:

```bash
GET /version
```

Response:
```json
{
  "version": "8aa2e00-dirty",
  "gitCommit": "8aa2e00",
  "buildTime": "2025-11-06_17:41:37"
}
```

### Create Helm Release

Install a Helm chart to a Kubernetes namespace:

```bash
POST /v1/create?chart=<chart-path>&namespace=<namespace>&release=<release-name>
```

Parameters:
- `chart`: Path to the Helm chart
- `namespace`: Kubernetes namespace for the release
- `release`: Name for the Helm release

Success Response (200 OK):
```json
{
  "status": "success",
  "message": "Helm chart installed successfully",
  "release": "my-release"
}
```

Error Response (400 Bad Request):
```json
{
  "error": "Missing required parameter: chart"
}
```

Error Response (500 Internal Server Error):
```json
{
  "error": "Failed to install Helm chart",
  "message": "detailed error message"
}
```

## Development

### Building

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Build binary
make build

# Run all quality checks and build
make all
```

### Running Tests

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

The project uses several tools to maintain code quality:

- `gofmt`: Code formatting
- `go vet`: Static analysis
- `golangci-lint`: Comprehensive linting

Install development tools:
```bash
make install-tools
```

## Makefile Targets

- `make build` - Build the binary
- `make clean` - Remove build artifacts
- `make test` - Run tests
- `make lint` - Run linter
- `make fmt` - Format code
- `make vet` - Run go vet
- `make container` - Build Docker image
- `make container-push` - Build and push Docker image
- `make all` - Run all quality checks and build
- `make help` - Show available targets

## Architecture

### Main Components

1. **HTTP Server** (`cmd/main.go`)
   - Gin-based HTTP server
   - Request routing and validation
   - Graceful shutdown handling
   - Context management for timeouts

2. **Helm Operator** (`operator/operator.go`)
   - Helm chart installation logic
   - Context-aware operations
   - Error handling and validation
   - Release verification

### Code Structure

```
.
├── cmd/
│   └── main.go          # Application entry point
├── operator/
│   └── operator.go      # Helm operations
├── build/
│   └── Dockerfile       # Container image definition
├── go.mod              # Go module dependencies
├── Makefile            # Build automation
└── README.md           # This file
```

