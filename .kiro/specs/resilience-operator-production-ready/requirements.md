# Requirements Document

## Introduction

Este documento define os requisitos para reestruturar o Resilience Operator, removendo código morto da implementação antiga (microserviço) e preparando o Kubernetes Operator para produção. A mudança envolve renomear `platform/resilience-service` para `platform/resilience-operator` e garantir que toda a estrutura esteja limpa e pronta para deploy.

## Glossary

- **Resilience_Operator**: Kubernetes Operator que gerencia ResiliencePolicy CRDs e configura Linkerd
- **Dead_Code**: Código da implementação antiga (microserviço gRPC) que não é mais utilizado
- **CRD**: Custom Resource Definition do Kubernetes
- **Helm_Chart**: Pacote de deployment para Kubernetes

## Requirements

### Requirement 1: Remove Dead Code

**User Story:** As a developer, I want to remove all dead code from the old microservice implementation, so that the codebase is clean and maintainable.

#### Acceptance Criteria

1. WHEN the cleanup is complete, THE Repository SHALL NOT contain the old `cmd/server/` directory
2. WHEN the cleanup is complete, THE Repository SHALL NOT contain the old `internal/application/`, `internal/domain/`, `internal/infrastructure/`, `internal/presentation/` directories
3. WHEN the cleanup is complete, THE Repository SHALL NOT contain the old `api/proto/` directory
4. WHEN the cleanup is complete, THE Repository SHALL NOT contain obsolete documentation files (ARCHITECTURE_ANALYSIS.md, BUG_ANALYSIS.md, etc.)
5. WHEN the cleanup is complete, THE Repository SHALL NOT contain the old root-level go.mod, Dockerfile, Makefile from the microservice

### Requirement 2: Restructure Directory Layout

**User Story:** As a developer, I want the operator to have a clear directory structure, so that it follows Kubernetes Operator best practices.

#### Acceptance Criteria

1. WHEN restructuring is complete, THE Operator SHALL be located at `platform/resilience-operator/`
2. WHEN restructuring is complete, THE Operator SHALL have the standard Kubebuilder layout (api/, cmd/, internal/, config/)
3. WHEN restructuring is complete, THE Tests SHALL be located at `platform/resilience-operator/tests/`
4. WHEN restructuring is complete, THE Go module path SHALL be updated to reflect the new location

### Requirement 3: Update All References

**User Story:** As a developer, I want all import paths and references to be updated, so that the code compiles and runs correctly.

#### Acceptance Criteria

1. WHEN references are updated, THE Go import paths SHALL use the new module path
2. WHEN references are updated, THE Helm chart SHALL reference the correct image and paths
3. WHEN references are updated, THE Documentation SHALL reflect the new directory structure
4. WHEN references are updated, THE CI/CD configurations SHALL use the correct paths

### Requirement 4: Production-Ready Dockerfile

**User Story:** As a DevOps engineer, I want a production-ready Dockerfile, so that I can deploy the operator securely.

#### Acceptance Criteria

1. THE Dockerfile SHALL use multi-stage build for minimal image size
2. THE Dockerfile SHALL use distroless or scratch base image
3. THE Dockerfile SHALL run as non-root user
4. THE Dockerfile SHALL have proper labels (version, maintainer, etc.)
5. THE Dockerfile SHALL support ARM64 and AMD64 architectures

### Requirement 5: Production Helm Chart

**User Story:** As a DevOps engineer, I want a complete Helm chart, so that I can deploy the operator with proper configuration.

#### Acceptance Criteria

1. THE Helm_Chart SHALL include all necessary templates (deployment, service, RBAC, CRDs)
2. THE Helm_Chart SHALL support configurable resource limits
3. THE Helm_Chart SHALL support pod disruption budgets
4. THE Helm_Chart SHALL support network policies
5. THE Helm_Chart SHALL include ServiceMonitor for Prometheus

### Requirement 6: Comprehensive Testing

**User Story:** As a developer, I want all tests to pass after restructuring, so that I have confidence in the changes.

#### Acceptance Criteria

1. WHEN tests are run, THE Unit tests SHALL pass with 80%+ coverage
2. WHEN tests are run, THE Property tests SHALL pass with 100+ iterations each
3. WHEN tests are run, THE Integration tests SHALL pass
4. THE Test files SHALL be properly organized in the new structure

### Requirement 7: Documentation Update

**User Story:** As a developer, I want updated documentation, so that I can understand and operate the system.

#### Acceptance Criteria

1. THE README SHALL describe the operator purpose and usage
2. THE README SHALL include quickstart instructions
3. THE Documentation SHALL include architecture diagrams
4. THE Documentation SHALL include troubleshooting guide

### Requirement 8: CI/CD Pipeline

**User Story:** As a DevOps engineer, I want CI/CD configuration, so that I can automate builds and deployments.

#### Acceptance Criteria

1. THE CI configuration SHALL build and test the operator
2. THE CI configuration SHALL build and push Docker images
3. THE CI configuration SHALL run linting and security scans
4. THE CD configuration SHALL support Helm deployments

