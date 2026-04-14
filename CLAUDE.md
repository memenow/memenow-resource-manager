# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-based REST API service that acts as a Helm operator — installs and manages Helm chart deployments in Kubernetes clusters via HTTP endpoints. Uses Gin for routing and the Helm v3 SDK for chart operations.

## Build and Verify

- Build binary: `make build` (outputs to `./build/`)
- Build container: `make container` (builds binary first, then Docker image)
- Lint: `golangci-lint run ./...` (binary at `~/go/bin/golangci-lint` if not in PATH)
- Vet: `go vet ./...`
- Run tests: `make test` (runs `go test -v -race -coverprofile=coverage.out ./...`)
- Coverage report: `make test-coverage` (generates coverage.html)

## Project Structure

- `cmd/main.go` — Gin server entry point, defines `/v1/create` endpoint
- `operator/operator.go` — Helm operator logic (load chart, install release, verify)
- `build/Dockerfile` — Container image definition (debian:bookworm-slim)
- `helm/memenow-resource-manager/` — Helm chart for deploying this service
- `stable-diffusion-webui-on-k8s/` — Helm chart for Stable Diffusion WebUI on K8s

## Git Conventions

- Commit messages: Conventional Commits format (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `ci:`)
- CI triggers on release creation — builds container, pushes to GHCR, signs with Cosign

## Code Style

- Standard `gofmt` formatting
- Follow Go conventions and idioms
