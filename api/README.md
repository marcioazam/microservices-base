# Auth Platform API

Modern Protocol Buffer API definitions for the Auth Platform, supporting gRPC, gRPC-Web, Connect-RPC, and REST/HTTP protocols.

## Overview

This directory contains the Protocol Buffer definitions for all Auth Platform services. The APIs are designed following state-of-the-art practices as of December 2025, including:

- **Buf Ecosystem**: Modern protobuf toolchain for linting, breaking change detection, and code generation
- **Protovalidate**: Declarative validation rules using CEL expressions
- **gRPC-Gateway**: REST/HTTP support with OpenAPI generation
- **Connect-RPC**: Browser-native RPC with TypeScript support
- **Multi-language**: Code generation for Go, Rust, TypeScript, and Python

## Directory Structure

```
api/
├── buf.yaml                    # Buf configuration
├── buf.gen.yaml                # Code generation config
├── buf.lock                    # Dependency lock file
├── Makefile                    # Build automation
├── README.md                   # This file
├── proto/
│   ├── auth/
│   │   └── v1/                 # Auth services (versioned)
│   │       ├── auth_edge.proto      # Token validation, DPoP, SPIFFE
│   │       ├── common.proto         # Shared types
│   │       ├── iam_policy.proto     # RBAC/ABAC/ReBAC authorization
│   │       ├── mfa_service.proto    # MFA (TOTP, WebAuthn, Push)
│   │       ├── session_identity.proto # Session management
│   │       └── token_service.proto  # OAuth 2.1 token operations
│   └── infra/
│       └── resilience/
│           └── v1/
│               └── resilience.proto # Resilience policies
├── openapi/                    # Generated OpenAPI specs
│   └── v1/
├── gen/                        # Generated code (gitignored)
│   ├── go/
│   ├── rust/
│   ├── typescript/
│   └── python/
└── tests/                      # Property-based tests
```

## Getting Started

### Prerequisites

- [Buf CLI](https://buf.build/docs/installation) v1.42+
- Go 1.22+ (for Go code generation)
- Node.js 20+ (for TypeScript code generation)
- Python 3.12+ (for Python code generation and tests)

### Installation

```bash
# Install Buf CLI (macOS)
brew install bufbuild/buf/buf

# Install Buf CLI (Linux)
curl -sSL https://github.com/bufbuild/buf/releases/latest/download/buf-Linux-x86_64 -o /usr/local/bin/buf
chmod +x /usr/local/bin/buf

# Verify installation
buf --version
```

### Quick Start

```bash
# Navigate to api directory
cd api

# Update dependencies
buf dep update

# Lint proto files
make lint

# Generate code for all languages
make generate

# Run tests
make test
```

## Available Services

### Auth Edge Service (`auth.v1.AuthEdgeService`)

Token validation and identity services at the edge:

- `ValidateToken` - JWT access token validation
- `IntrospectToken` - RFC 7662 token introspection
- `ValidateDPoP` - DPoP proof validation (RFC 9449)
- `GetServiceIdentity` - SPIFFE/mTLS identity validation

### Token Service (`auth.v1.TokenService`)

OAuth 2.1 token operations:

- `IssueTokens` - Issue access, refresh, and ID tokens
- `RefreshTokens` - Token refresh
- `RevokeToken` - Token revocation (RFC 7009)
- `ExchangeToken` - Token exchange (RFC 8693)
- `GetJWKS` - JSON Web Key Set endpoint
- `PushAuthorizationRequest` - PAR (RFC 9126)

### MFA Service (`auth.v1.MFAService`)

Multi-factor authentication:

- TOTP enrollment and verification
- WebAuthn/Passkey registration and authentication
- Push notification challenges
- Backup codes
- CAEP event streaming

### Session Identity Service (`auth.v1.SessionIdentityService`)

Session management and OAuth flows:

- Session creation, retrieval, and termination
- Risk assessment and step-up authentication
- OAuth 2.1 authorization
- Session event streaming

### IAM Policy Service (`auth.v1.IAMPolicyService`)

Fine-grained access control supporting RBAC, ABAC, and ReBAC patterns:

- `Authorize` - Check if subject can perform action on resource
- `BatchAuthorize` - Batch authorization (up to 100 requests)
- `CreatePermission`, `GetPermission`, `ListPermissions`, `DeletePermission` - Permission CRUD
- `CreateRole`, `GetRole`, `ListRoles`, `UpdateRole`, `DeleteRole` - Role CRUD
- `AssignRole`, `RevokeRole`, `ListRoleAssignments` - Role assignment management
- `CreatePolicy`, `GetPolicy`, `ListPolicies`, `UpdatePolicy`, `DeletePolicy` - Policy CRUD
- `GetPolicyVersion`, `ListPolicyVersions`, `RollbackPolicy` - Policy versioning

Features:
- Detailed authorization decisions with matched policies/roles/permissions
- CEL condition support in policy rules
- Subject, resource, and environment attributes for ABAC
- Policy priority and versioning with rollback capability

### Resilience Service (`infra.resilience.v1.ResilienceService`)

Resilience policy management:

- Circuit breaker configuration
- Retry policies
- Timeout configuration
- Rate limiting
- Bulkhead isolation

## Code Generation

### Generate All Languages

```bash
make generate
```

### Generate Specific Language

```bash
# Go only
make generate-go

# TypeScript only
make generate-ts

# OpenAPI specs only
make generate-openapi
```

### Generated Code Location

| Language   | Output Directory    |
|------------|---------------------|
| Go         | `gen/go/`           |
| Rust       | `gen/rust/`         |
| TypeScript | `gen/typescript/`   |
| Python     | `gen/python/`       |
| OpenAPI    | `openapi/v1/`       |

## Validation

All message fields include protovalidate annotations for declarative validation:

```protobuf
message CreateSessionRequest {
  string user_id = 1 [
    (buf.validate.field).required = true,
    (buf.validate.field).string.uuid = true
  ];
  
  string ip_address = 2 [
    (buf.validate.field).string.ip = true
  ];
}
```

## REST/HTTP Support

All services include `google.api.http` annotations for REST endpoint mapping:

```protobuf
rpc CreateSession(CreateSessionRequest) returns (Session) {
  option (google.api.http) = {
    post: "/v1/sessions"
    body: "*"
  };
}
```

## Development

### Linting

```bash
make lint
```

### Breaking Change Detection

```bash
# Against main branch
make breaking

# Against specific tag
make breaking-tag TAG=v1.0.0
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### CI/CD

The repository includes GitHub Actions workflow (`.github/workflows/proto-ci.yml`) that:

1. Lints proto files on every PR
2. Detects breaking changes against main branch
3. Generates and verifies code compilation
4. Runs property-based tests
5. Validates OpenAPI specs

## Versioning

APIs follow semantic versioning with package-level versioning:

- `auth.v1` - Current stable version
- `auth.v2` - Future major version (when breaking changes needed)

Breaking changes within a major version are not allowed. Use `buf breaking` to verify.

## Contributing

1. Make changes to `.proto` files
2. Run `make lint` to check for issues
3. Run `make breaking` to check for breaking changes
4. Run `make generate` to regenerate code
5. Run `make test` to verify tests pass
6. Submit PR

## License

Copyright 2025 Auth Platform. All rights reserved.
