# Contributing to Auth Platform

Thank you for your interest in contributing to the Auth Platform!

## Development Setup

### Prerequisites

- Rust 1.75+ (for auth-edge, token services)
- Go 1.23+ (for iam-policy service)
- Elixir 1.16+ / OTP 26+ (for session-identity, mfa services)
- Docker & Docker Compose
- protoc (Protocol Buffers compiler)

### Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd auth-platform

# Start infrastructure
docker-compose -f deploy/docker/docker-compose.yml up -d postgres redis

# Build all services
make build

# Run tests
make test
```

## Project Structure

```
services/       # Domain microservices
libs/           # Shared libraries (by language)
platform/       # Platform/infrastructure services
api/proto/      # Protocol Buffer definitions
deploy/         # Docker and Kubernetes configs
docs/           # Documentation
sdk/            # Client SDKs
```

## Coding Standards

### General
- Follow existing patterns in the codebase
- Write tests for new functionality
- Update documentation for API changes

### Language-Specific

**Rust:**
- Run `cargo fmt` and `cargo clippy` before committing
- Property-based tests with `proptest` (100+ cases)

**Go:**
- Run `go fmt` and `golangci-lint`
- Follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout)

**Elixir:**
- Run `mix format` and `mix credo`
- Use Dialyzer for type checking

## Pull Request Process

1. Create a feature branch from `develop`
2. Make your changes with tests
3. Ensure CI passes (lint, test, security scan)
4. Request review from maintainers
5. Squash and merge after approval

## Security

- Never commit secrets or credentials
- Report security issues privately to security@example.com
- Follow OWASP guidelines for authentication code

## Questions?

Open an issue or reach out to the platform team.
