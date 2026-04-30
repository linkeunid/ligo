# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Ligo** is a modular Go framework with lightweight dependency injection, inspired by NestJS.

- **Module**: `github.com/linkeunid/ligo`
- **Go version**: 1.25.9
- **License**: MIT

## Commands

This project uses standard Go tooling:

```bash
# Build
go build ./...

# Run tests
go test ./...
go test -v ./...

# Run a single test
go test -run TestName ./...

# Lint (install golangci-lint if not present)
golangci-lint run

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy

# Verify module
go mod verify
```

## Architecture

The project is in early development. The intended architecture follows a NestJS-inspired modular design with:

- **Modules**: Self-contained units that encapsulate related providers and controllers
- **Dependency Injection**: Lightweight container for resolving and injecting dependencies
- **Providers**: Injectable services and factories
- **Controllers**: HTTP/gRPC request handlers (if applicable)

As the codebase grows, update this section with the concrete package structure and DI container implementation details.
