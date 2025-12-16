# ADR-001: Monorepo Structure for Auth Platform

**Status:** Accepted  
**Date:** 2025-12-16  
**Decision Makers:** Platform Team

## Context

The Auth Platform is a multi-service authentication and authorization system built with multiple languages (Rust, Go, Elixir). We needed a clear, scalable folder structure that:

1. Separates concerns clearly (domain services vs platform services vs shared libs)
2. Supports multiple languages without confusion
3. Follows industry best practices (NX, Bazel, Turborepo patterns)
4. Enables efficient CI/CD with change detection

## Decision

We adopt the following monorepo structure:

```
auth-platform/
├── api/                    # API contracts (proto, OpenAPI)
│   └── proto/
│       ├── auth/           # Domain service protos
│       └── infra/          # Infrastructure protos
├── deploy/                 # Deployment configurations
│   ├── docker/             # Docker configs per service
│   └── kubernetes/         # K8s manifests, Helm charts
├── docs/                   # Documentation
│   ├── adr/                # Architecture Decision Records
│   ├── api/                # API docs, Postman collections
│   └── runbooks/           # Operational runbooks
├── libs/                   # Shared libraries by language
│   ├── go/                 # Go shared libs
│   └── rust/               # Rust shared libs
├── platform/               # Platform/Infrastructure services
│   └── resilience-service/
├── sdk/                    # Client SDKs
│   ├── go/
│   ├── python/
│   └── typescript/
├── services/               # Domain microservices
│   ├── auth-edge/          # Rust
│   ├── iam-policy/         # Go
│   ├── mfa/                # Elixir
│   ├── session-identity/   # Elixir
│   └── token/              # Rust
└── tools/                  # Build tools and scripts
```

## Consequences

### Positive
- Clear separation between domain services (`services/`) and platform services (`platform/`)
- Language-specific shared libraries prevent confusion (`libs/go/`, `libs/rust/`)
- Centralized API contracts enable contract-first development
- Standard structure familiar to developers from NX/Bazel ecosystems

### Negative
- Requires updating import paths when migrating from previous structure
- CI/CD workflows need path-based change detection

### Neutral
- Documentation centralized in `docs/` with subdirectories by type

## Alternatives Considered

1. **Flat structure** - All services at root level. Rejected: doesn't scale.
2. **Language-first** (`rust/`, `go/`, `elixir/`) - Rejected: mixes concerns.
3. **Domain-first** (`auth/`, `identity/`) - Rejected: unclear service boundaries.

## References

- [NX Monorepo Structure](https://nx.dev/concepts/decisions/folder-structure)
- [Graphite - How we organize our monorepo](https://graphite.com/blog/how-we-organize-our-monorepo-to-ship-fast)
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
