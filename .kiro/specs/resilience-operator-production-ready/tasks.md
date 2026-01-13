# Implementation Plan: Resilience Operator Production Ready

## Overview

Este plano reestrutura o Resilience Operator, remove código morto e prepara para produção. A execução deve ser feita em ordem para evitar quebras.

## Tasks

- [x] 1. Phase 1: Backup and Analysis
  - [x] 1.1 Document current file structure
    - List all files in platform/resilience-service/
    - Identify which files are active vs dead code
    - Create backup reference
    - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. Phase 2: Create New Structure
  - [x] 2.1 Create new directory platform/resilience-operator/
    - Create directory structure (api/, cmd/, internal/, config/, tests/)
    - _Requirements: 2.1, 2.2_

  - [x] 2.2 Move operator code to new location
    - Move api/v1/ to platform/resilience-operator/api/v1/
    - Move cmd/main.go to platform/resilience-operator/cmd/
    - Move internal/ to platform/resilience-operator/internal/
    - Move config/ to platform/resilience-operator/config/
    - _Requirements: 2.1, 2.2_

  - [x] 2.3 Move and consolidate tests
    - Move relevant property tests to platform/resilience-operator/tests/property/
    - Move relevant unit tests to platform/resilience-operator/tests/unit/
    - Move relevant integration tests to platform/resilience-operator/tests/integration/
    - Move relevant e2e tests to platform/resilience-operator/tests/e2e/
    - _Requirements: 2.3, 6.1, 6.2, 6.3_

- [x] 3. Phase 3: Update Go Module
  - [x] 3.1 Create new go.mod with correct module path
    - Module: github.com/auth-platform/platform/resilience-operator
    - Update all dependencies
    - _Requirements: 2.4, 3.1_

  - [x] 3.2 Update all import paths in source files
    - Update imports in api/v1/*.go
    - Update imports in cmd/main.go
    - Update imports in internal/**/*.go
    - _Requirements: 3.1_

  - [x] 3.3 Update all import paths in test files
    - Update imports in tests/**/*.go
    - _Requirements: 3.1, 6.4_

- [x] 4. Phase 4: Production Dockerfile
  - [x] 4.1 Create production-ready Dockerfile
    - Multi-stage build
    - Distroless base image
    - Non-root user
    - ARM64/AMD64 support
    - Proper labels
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 4.2 Create .dockerignore
    - Exclude tests, docs, unnecessary files
    - _Requirements: 4.1_

- [x] 5. Phase 5: Update Helm Chart
  - [x] 5.1 Update Helm chart references
    - Update image repository path
    - Update any hardcoded paths
    - _Requirements: 3.2, 5.1_

  - [x] 5.2 Add PodDisruptionBudget template
    - Ensure HA during upgrades
    - _Requirements: 5.3_

  - [x] 5.3 Add NetworkPolicy template
    - Restrict network access
    - _Requirements: 5.4_

  - [x] 5.4 Add ServiceMonitor template
    - Prometheus integration
    - _Requirements: 5.5_

- [x] 6. Phase 6: Update Makefile and Build
  - [x] 6.1 Create new Makefile
    - Build targets
    - Test targets
    - Docker targets
    - Helm targets
    - _Requirements: 3.1_

  - [x] 6.2 Update PROJECT file
    - Correct domain and repo
    - _Requirements: 2.2_

- [x] 7. Phase 7: Documentation
  - [x] 7.1 Create comprehensive README.md
    - Purpose and overview
    - Quickstart guide
    - Configuration options
    - _Requirements: 7.1, 7.2_

  - [x] 7.2 Update architecture documentation
    - Update paths in docs/service-mesh-architecture.md
    - Update paths in docs/runbooks/
    - _Requirements: 7.3, 7.4_

- [x] 8. Phase 8: CI/CD Configuration
  - [x] 8.1 Create GitHub Actions workflow
    - Build and test
    - Lint and security scan
    - Docker build and push
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 9. Phase 9: Remove Dead Code
  - [x] 9.1 Remove old platform/resilience-service/ directory
    - Delete entire directory after confirming new structure works
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 10. Phase 10: Verification
  - [x] 10.1 Run all tests
    - go test ./...
    - Verify 80%+ coverage
    - _Requirements: 6.1, 6.2, 6.3_

  - [x] 10.2 Build Docker image
    - docker build -t resilience-operator:test .
    - Verify image runs
    - _Requirements: 4.1_

  - [x] 10.3 Helm lint
    - helm lint deploy/kubernetes/service-mesh/helm/resilience-operator/
    - _Requirements: 5.1_

- [x] 11. Final Checkpoint
  - Verify no dead code remains
  - Verify all tests pass
  - Verify Docker builds
  - Verify Helm chart is valid

## Notes

- Execute phases in order to avoid breaking changes
- Phase 9 (remove dead code) should only be done after Phase 10 verification passes
- All tasks are required for production readiness
- Backup important files before deletion

## Files to Delete (Dead Code)

```
platform/resilience-service/cmd/server/
platform/resilience-service/internal/application/
platform/resilience-service/internal/domain/
platform/resilience-service/internal/infrastructure/
platform/resilience-service/internal/presentation/
platform/resilience-service/api/proto/
platform/resilience-service/configs/
platform/resilience-service/bin/
platform/resilience-service/pkg/
platform/resilience-service/go.mod (root level)
platform/resilience-service/go.sum (root level)
platform/resilience-service/Dockerfile (root level)
platform/resilience-service/Makefile (root level)
platform/resilience-service/coverage.out
platform/resilience-service/*.md (obsolete docs)
```

## Files to Keep and Move

```
platform/resilience-service/operator/* → platform/resilience-operator/
platform/resilience-service/tests/property/operator_*.go → platform/resilience-operator/tests/property/
platform/resilience-service/tests/unit/controller_test.go → platform/resilience-operator/tests/unit/
platform/resilience-service/tests/integration/controller_integration_test.go → platform/resilience-operator/tests/integration/
platform/resilience-service/tests/e2e/resilience_policy_e2e_test.go → platform/resilience-operator/tests/e2e/
```

