# Design Document

## Overview

Este documento descreve o design para reestruturar o Resilience Operator, removendo código morto e preparando para produção. A mudança principal é mover de `platform/resilience-service/operator/` para `platform/resilience-operator/` com uma estrutura limpa.

## Architecture

### Current Structure (Problematic)

```
platform/resilience-service/           ← Nome confuso (não é um service)
├── cmd/server/                        ← CÓDIGO MORTO
├── internal/                          ← CÓDIGO MORTO (microserviço antigo)
│   ├── application/
│   ├── domain/
│   ├── infrastructure/
│   └── presentation/
├── api/proto/                         ← CÓDIGO MORTO
├── operator/                          ← Código ativo (aninhado)
│   ├── api/v1/
│   ├── cmd/
│   ├── internal/
│   └── config/
├── tests/                             ← Mistura de testes ativos e mortos
├── go.mod                             ← Módulo antigo
├── Dockerfile                         ← Dockerfile antigo
└── *.md                               ← Documentação obsoleta
```

### Target Structure (Clean)

```
platform/resilience-operator/          ← Nome claro
├── api/
│   └── v1/                            ← CRD types
│       ├── groupversion_info.go
│       ├── resiliencepolicy_types.go
│       └── zz_generated.deepcopy.go
├── cmd/
│   └── main.go                        ← Operator entrypoint
├── internal/
│   ├── controller/                    ← Reconciliation logic
│   ├── linkerd/                       ← Linkerd annotation mapper
│   ├── status/                        ← Status manager
│   └── metrics/                       ← Prometheus metrics
├── config/
│   ├── crd/                           ← CRD manifests
│   ├── rbac/                          ← RBAC manifests
│   ├── manager/                       ← Deployment manifests
│   └── samples/                       ← Example CRs
├── tests/
│   ├── unit/                          ← Unit tests
│   ├── integration/                   ← Integration tests
│   ├── property/                      ← Property-based tests
│   └── e2e/                           ← End-to-end tests
├── Dockerfile                         ← Production Dockerfile
├── Makefile                           ← Build automation
├── go.mod                             ← Go module
├── PROJECT                            ← Kubebuilder project file
└── README.md                          ← Documentation
```

## Components and Interfaces

### Go Module

```go
module github.com/auth-platform/platform/resilience-operator

go 1.24

require (
    k8s.io/apimachinery v0.32.0
    k8s.io/client-go v0.32.0
    sigs.k8s.io/controller-runtime v0.20.0
    sigs.k8s.io/gateway-api v1.2.0
    pgregory.net/rapid v1.1.0
)
```

### Dockerfile (Multi-stage, Production-Ready)

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o manager cmd/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
```

## Data Models

### ResiliencePolicy CRD (unchanged)

```yaml
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: example
spec:
  targetRef:
    name: my-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  retry:
    enabled: true
    maxAttempts: 3
  timeout:
    enabled: true
    requestTimeout: "30s"
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system.*

### Property 1: Module Path Consistency

*For any* Go source file in the operator, all import paths SHALL use the new module path `github.com/auth-platform/platform/resilience-operator`.

**Validates: Requirements 3.1**

### Property 2: No Dead Code References

*For any* file in the new structure, there SHALL be no references to the old `resilience-service` paths.

**Validates: Requirements 1.1, 1.2, 1.3**

### Property 3: Test Coverage Preservation

*For any* test file moved to the new structure, the test SHALL pass with the same behavior as before.

**Validates: Requirements 6.1, 6.2, 6.3**

### Property 4: Dockerfile Security

*For any* container built from the Dockerfile, the process SHALL run as non-root user (UID 65532).

**Validates: Requirements 4.3**

## Error Handling

- Build failures due to incorrect import paths → Fix import paths
- Test failures after move → Update test imports and paths
- Helm deployment failures → Update chart references

## Testing Strategy

### Unit Tests
- Test all controller functions
- Test annotation mapper
- Test status manager

### Property Tests
- Reconciliation idempotency
- Annotation consistency
- Status accuracy

### Integration Tests
- Full reconciliation cycle with envtest
- CRD validation

### E2E Tests
- Deploy to kind cluster
- Create ResiliencePolicy
- Verify Linkerd annotations

