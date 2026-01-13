# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is an **enterprise-grade authentication and authorization platform** built with a polyglot microservices architecture. The platform implements OAuth 2.1, DPoP (RFC 9449), WebAuthn/FIDO2, Zero Trust security, and CAEP for continuous access evaluation.

**Architectural Score: ~95/100 (State of the Art 2025)**

## Key Technologies

- **Rust** (auth-edge, token services) - Tokio, Tonic
- **Go** (iam-policy, shared libraries) - OPA, gRPC, HTTP
- **Elixir** (session-identity, mfa) - Phoenix, OTP, Ecto
- **Infrastructure**: Envoy Gateway, Linkerd service mesh, HashiCorp Vault, Kubernetes

## Project Structure

```
microservices-base/
├── api/proto/           # Protocol Buffer definitions (auth, infra domains)
├── services/            # 5 microservices (auth-edge, token, session-identity, iam-policy, mfa)
├── libs/                # Shared libraries by language (go/, rust/)
├── sdk/                 # Client SDKs (go/, python/, typescript/)
├── platform/            # Platform services (resilience-service)
├── deploy/              # Docker Compose + Kubernetes manifests
└── docs/                # ADRs, API docs, runbooks, operational guides
```

## Build Commands

The repository uses a unified `Makefile` at the root level:

### Building

```bash
# Build everything
make build

# Build by language
make build-rust        # Rust services (auth-edge, token)
make build-go          # Go services (iam-policy)
make build-elixir      # Elixir services (session-identity, mfa)
```

### Testing

```bash
# Run all tests
make test

# Test by language
make test-rust         # Rust tests (cargo test)
make test-go           # Go tests (go test -v -race ./...)
make test-elixir       # Elixir tests (mix test)

# Advanced testing
make test-property     # Property-based tests (Rust proptest, Go rapid)
make test-contract     # Pact contract tests
```

### Linting

```bash
# Lint all code
make lint

# Lint by language
make lint-rust         # cargo fmt --check && cargo clippy
make lint-go           # go fmt && go vet
make lint-elixir       # mix format --check-formatted && mix credo
```

### Docker

```bash
make docker-build      # Build all Docker images
make docker-up         # Start all services with docker-compose
make docker-down       # Stop all services
make docker-logs       # View service logs
```

### Protocol Buffers

```bash
make proto             # Generate protobuf code from api/proto/
```

## Go Library Architecture

The `libs/go/` directory contains **24+ production-ready packages** with **94 test files** organized in a **workspace structure**:

- **Source**: `libs/go/src/` - All package implementations
- **Tests**: `libs/go/tests/` - Mirrors `src/` structure with property-based tests

### Running Go Library Tests

```bash
# Windows PowerShell
cd libs/go/tests
Get-ChildItem -Directory | ForEach-Object { Push-Location $_.FullName; go test -v ./...; Pop-Location }

# Linux/Mac
cd libs/go/tests
for dir in */; do (cd "$dir" && go test -v ./...); done

# Single package
cd libs/go/tests/resilience && go test -v ./...
cd libs/go/tests/collections && go test -v ./...
```

### Key Go Packages

- **domain** - Type-safe primitives (Email, UUID, ULID, Money, PhoneNumber)
- **errors** - Typed errors with HTTP/gRPC mapping, Go 1.25+ generics (`AsType[T]`, `Must[T]`)
- **functional** - Option/Result/Either/Validated types, iterators (Go 1.23+)
- **fault** - Generic resilience executor interface (Go 1.25+)
- **resilience** - Circuit breaker, retry, bulkhead, rate limiter, timeout
- **collections** - Generic LRU cache, Set, Queue, PriorityQueue with Go 1.23+ iterators
- **validation** - Composable validators with error accumulation
- **http** - Resilient HTTP client with retry/timeout/middleware
- **grpc** - gRPC utilities and error mapping
- **workerpool** - Generic worker pool with priority queue
- **observability** - Structured logging, tracing, correlation IDs
- **security** - Timing-safe comparison, sanitization, PII masking

### Go Version Requirements

- Most packages: **Go 1.25+** (uses generics extensively)
- Iterators: **Go 1.23+** (collections, functional packages)

## SDK Development

The `sdk/` directory contains client SDKs for consuming the auth platform:

### Go SDK (`sdk/go/`)

Complete OAuth 2.1 client with DPoP and PKCE support:

```go
// Basic client creation
c, err := client.New(
    client.WithBaseURL("https://auth.example.com"),
    client.WithClientID("your-client-id"),
)

// Token validation
claims, err := c.ValidateTokenCtx(ctx, accessToken)

// HTTP middleware
mux.Handle("/protected", c.HTTPMiddleware()(handler))

// gRPC interceptors
grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(c)))
```

SDK packages:
- `src/client` - Client creation and configuration
- `src/auth` - PKCE and DPoP implementations
- `src/token` - Token extraction and validation
- `src/middleware` - HTTP/gRPC middleware
- `src/errors` - Error types and helpers
- `src/retry` - Retry policies
- `src/types` - Result, Option, Claims

## Service Architecture

### auth-edge (Rust:8080)
- JWT validation, mTLS enforcement, rate limiting
- First point of contact for all requests

### token (Rust:8081)
- JWT signing, DPoP proof validation, key management
- Handles token generation and rotation

### session-identity (Elixir:8082)
- Session management, OAuth 2.1 flows, event sourcing
- Complete audit trail via event sourcing

### iam-policy (Go:8083)
- RBAC/ABAC with Open Policy Agent (OPA)
- Policy evaluation and enforcement

### mfa (Elixir:8084)
- TOTP, WebAuthn/FIDO2, Passkeys
- Multi-factor authentication flows

## Testing Standards

### Rust Services
- Use `cargo test` for unit tests
- Property-based tests with `proptest` (100+ test cases)
- Run with `cargo test --features proptest -- --ignored`

### Go Services and Libraries
- Use `go test -v -race ./...` for all tests
- Property-based tests with [rapid](https://github.com/flyingmutant/rapid)
- Domain generators in `libs/go/src/testing` package
- Test files located in `libs/go/tests/` mirroring `src/` structure

### Elixir Services
- Use `mix test` for all tests
- Type checking with Dialyzer
- `mix format` and `mix credo` for linting

## Important Patterns

### Go Workspace Structure
The Go libraries use workspace mode (`go.work`):
- Each package in `libs/go/src/` is a separate module
- Each test directory in `libs/go/tests/` is a separate module
- This enables independent versioning and module boundaries

### Error Handling
- **Go**: Use typed errors from `libs/go/src/errors` with HTTP/gRPC mapping
- **SDK**: All errors use `SDKError` type with error codes and helper functions
- Automatic PII redaction and error sanitization

### Functional Programming
Go libraries extensively use functional patterns:
- `Option[T]` for nullable values
- `Result[T]` for fallible operations
- `Either[L, R]` for alternative values
- `Validated[E, A]` for error accumulation
- Pipeline composition for data transformations

### Resilience Patterns
All services use resilience patterns from `libs/go/src/resilience`:
- Circuit breakers with configurable thresholds
- Exponential backoff retry with jitter
- Rate limiting (token bucket algorithm)
- Bulkhead isolation for resource protection
- Configurable timeouts

## Security Considerations

1. **Never commit secrets** - Use Vault for all credentials
2. **OAuth 2.1 with PKCE** - Mandatory for authorization code flow
3. **DPoP tokens** - Sender-constrained tokens prevent token theft
4. **Zero Trust** - All service-to-service communication uses mTLS
5. **Event Sourcing** - Complete audit trail in session-identity service
6. **PII Handling** - Automatic redaction in logs and errors

## Development Workflow

1. **Start infrastructure**: `docker-compose -f deploy/docker/docker-compose.yml up -d postgres redis`
2. **Run linters**: `make lint`
3. **Run tests**: `make test`
4. **Build services**: `make build`
5. **Clean artifacts**: `make clean`

## Common Development Tasks

### Adding a New Go Library Package

1. Create module in `libs/go/src/newpackage/`
2. Create corresponding test module in `libs/go/tests/newpackage/`
3. Add to workspace: `cd libs/go/src && go work use ./newpackage`
4. Add test to workspace: `cd libs/go/tests && go work use ./newpackage`
5. Include property-based tests using `rapid`

### Running a Single Test

```bash
# Rust
cd services/auth-edge && cargo test test_name

# Go
cd services/iam-policy && go test -v -run TestName ./...

# Elixir
cd services/session-identity && mix test test/path/to/test_file.exs:line_number
```

### Generating Protocol Buffers

```bash
make proto
# Or manually:
protoc --proto_path=api/proto \
  --go_out=services/iam-policy \
  --go-grpc_out=services/iam-policy \
  api/proto/auth/*.proto api/proto/infra/*.proto
```

## Documentation

- **ADRs**: `docs/adr/` - Architecture decision records
- **API Docs**: `docs/api/` - API documentation and Postman collections
- **Runbooks**: `docs/runbooks/` - Operational procedures
- **Vault**: `docs/vault/` - Secrets management
- **Linkerd**: `docs/linkerd/` - Service mesh configuration
- **Pact**: `docs/pact/` - Contract testing
- **Passkeys**: `docs/passkeys/` - WebAuthn/FIDO2 implementation
- **CAEP**: `docs/caep/` - Continuous Access Evaluation Protocol

## Prerequisites

- **Rust**: 1.75+
- **Go**: 1.23+ (libraries use Go 1.25+ for generics)
- **Elixir**: 1.16+ with OTP 26+
- **Docker & Docker Compose**
- **protoc**: Protocol Buffers compiler

## Git Workflow

- Main branch: `main`
- Create feature branches from `develop`
- Run `make lint test` before committing
- Squash and merge after approval
