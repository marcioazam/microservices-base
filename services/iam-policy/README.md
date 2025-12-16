# IAM Policy Service

Policy decision point (PDP) service implementing RBAC and ABAC using Open Policy Agent (OPA).

## Overview

The IAM Policy Service provides:

- **RBAC**: Role-Based Access Control with hierarchical roles
- **ABAC**: Attribute-Based Access Control via Rego policies
- **Hot Reload**: Policy updates without service restart
- **gRPC Interface**: High-performance authorization decisions

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    IAM Policy Service                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Policy Engine (OPA) │  │ RBAC Module                 │   │
│  │ - Rego evaluation   │  │ - Role hierarchy            │   │
│  │ - Policy hot reload │  │ - Permission inheritance    │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ gRPC Handlers       │  │ Config                      │   │
│  │ - Authorization     │  │ - YAML-based                │   │
│  │ - Policy management │  │ - Environment vars          │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **Language**: Go 1.21
- **Policy Engine**: Open Policy Agent (OPA) v0.60
- **RPC**: gRPC
- **Policy Language**: Rego

## Policies

### RBAC (`policies/rbac.rego`)
Role-based access control with hierarchical role inheritance.

### ABAC (`policies/abac.rego`)
Attribute-based policies for fine-grained access control.

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Service bind address | `0.0.0.0` |
| `PORT` | Service port | `50054` |
| `POLICY_PATH` | Path to Rego policy files | `./policies` |

## Running

```bash
# Development
go run cmd/server/main.go

# Production
go build -o iam-policy-service cmd/server/main.go
./iam-policy-service
```

## Testing

```bash
go test ./...
```

## API

See `proto/iam_policy.proto` for the complete gRPC service definition.

### Key Endpoints

- `Authorize`: Evaluates authorization request against policies
- `GetPermissions`: Returns permissions for a subject
- `CheckPermission`: Checks specific permission

## Policy Hot Reload

The service watches the policy directory for changes and automatically reloads policies without restart. This enables:

- Zero-downtime policy updates
- A/B testing of policies
- Gradual policy rollouts
