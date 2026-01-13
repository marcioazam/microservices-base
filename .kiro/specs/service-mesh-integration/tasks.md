# Implementation Plan: Service Mesh Integration

## Overview

This implementation plan transforms the Resilience Service into a Kubernetes Operator that integrates with Linkerd 2.16+ service mesh. The plan follows a phased approach over 8-10 weeks, with property-based testing validation at each stage.

## Tasks

- [x] 1. Phase 1: Foundation Setup (Week 1)
  - [x] 1.1 Kubernetes cluster setup and validation
    - Provision Kubernetes 1.30+ cluster (kind/k3d for dev, EKS/GKE for prod)
    - Install cert-manager for certificate management
    - Configure kubectl access and verify cluster resources
    - Validate minimum 4 CPU, 8GB RAM available
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 1.2 Install Linkerd 2.16+ service mesh
    - Install Linkerd CLI and validate cluster compatibility
    - Install Linkerd CRDs and control plane
    - Install Linkerd Viz extension for observability
    - Verify all Linkerd checks passing
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 1.3 Install Gateway API CRDs
    - Install Gateway API CRDs v1.1.0+
    - Verify HTTPRoute and GRPCRoute CRDs available
    - _Requirements: 2.5_

- [x] 2. Phase 2: Operator Scaffold (Week 2)
  - [x] 2.1 Initialize operator project with Kubebuilder
    - Install Kubebuilder 3.14+
    - Initialize operator in `platform/resilience-service/operator`
    - Create ResiliencePolicy API scaffold
    - Verify project builds successfully
    - _Requirements: 4.1_

  - [x] 2.2 Define ResiliencePolicy CRD types
    - Implement ResiliencePolicySpec with all config types
    - Implement TargetReference for service targeting
    - Implement CircuitBreakerConfig, RetryConfig, TimeoutConfig
    - Add Kubebuilder markers for validation and printing
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

  - [x] 2.3 Generate and install CRDs
    - Run `make manifests` to generate CRD YAML
    - Install CRDs to cluster with `make install`
    - Verify CRDs with `kubectl explain resiliencepolicy.spec`
    - Create sample ResiliencePolicy manifests
    - _Requirements: 3.6_

- [x] 3. Phase 3: Controller Implementation (Week 3-4)
  - [x] 3.1 Implement reconciliation controller
    - Implement Reconcile function with idempotent logic
    - Add finalizer pattern for cleanup
    - Implement target service lookup
    - Add owner reference management
    - _Requirements: 4.2, 4.3, 4.4, 4.5_

  - [x] 3.2 Write property test for reconciliation idempotency
    - **Property 1: Reconciliation Idempotency**
    - **Validates: Requirements 4.3**

  - [x] 3.3 Implement circuit breaker configuration
    - Create Linkerd annotation mapper for circuit breaker
    - Apply failure-accrual annotations to target Service
    - Handle enable/disable toggle
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [x] 3.4 Write property test for circuit breaker annotations
    - **Property 3: Circuit Breaker Annotation Consistency**
    - **Validates: Requirements 5.1, 5.2**

  - [x] 3.5 Implement retry and timeout configuration
    - Create HTTPRoute generator for retry/timeout
    - Apply retry annotations (maxAttempts, statusCodes, timeout)
    - Apply timeout annotations (request, response)
    - Set owner reference on HTTPRoute
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.3, 7.4_

  - [x] 3.6 Write property tests for retry and timeout
    - **Property 4: Retry Configuration Mapping**
    - **Property 5: Timeout Configuration Mapping**
    - **Validates: Requirements 6.1-6.4, 7.1-7.3**

  - [x] 3.7 Implement deletion handling
    - Remove Service annotations on policy deletion
    - HTTPRoute cleanup via owner reference
    - Remove finalizer after cleanup
    - _Requirements: 4.4, 4.5_

  - [x] 3.8 Write property test for finalizer cleanup
    - **Property 2: Finalizer Cleanup Completeness**
    - **Validates: Requirements 4.4, 4.5**

- [x] 4. Phase 4: Status and Observability (Week 5)
  - [x] 4.1 Implement status manager
    - Update status conditions after reconciliation
    - Set Ready, Failed, TargetServiceNotFound conditions
    - Track appliedToServices and lastUpdateTime
    - _Requirements: 4.6, 4.7_

  - [x] 4.2 Write property test for status accuracy
    - **Property 7: Status Condition Accuracy**
    - **Validates: Requirements 4.6, 4.7**

  - [x] 4.3 Implement metrics and logging
    - Add Prometheus metrics for reconciliation
    - Add structured logging with correlation IDs
    - Emit events for policy applications
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_

  - [x] 4.4 Write property test for owner references
    - **Property 6: Owner Reference Integrity**
    - **Validates: Requirements 4.4**

- [x] 5. Checkpoint - Verify operator core functionality

- [x] 6. Phase 5: High Availability (Week 6)
  - [x] 6.1 Implement leader election
    - Configure controller-runtime leader election
    - Set lease duration and renew deadline
    - Test failover behavior
    - _Requirements: 11.1, 11.2_

  - [x] 6.2 Write property test for leader election
    - **Property 10: Leader Election Consistency**
    - **Validates: Requirements 11.1, 11.2**

  - [x] 6.3 Implement informer caching
    - Configure shared informer caches
    - Optimize watch filters for performance
    - _Requirements: 11.3_

  - [x] 6.4 Implement exponential backoff
    - Configure backoff for failed reconciliations
    - Set max retry attempts and intervals
    - _Requirements: 11.4_

- [x] 7. Phase 6: Security Hardening (Week 7)
  - [x] 7.1 Configure minimal RBAC
    - Define ClusterRole with least privilege
    - Create ServiceAccount and binding
    - Verify no excessive permissions
    - _Requirements: 12.3_

  - [x] 7.2 Implement input validation
    - Add CEL validation rules to CRD
    - Validate all policy fields
    - Prevent injection attacks
    - _Requirements: 12.4_

  - [x] 7.3 Write property test for input validation
    - **Property 8: Target Service Validation**
    - **Validates: Requirements 4.7**

  - [x] 7.4 Verify mTLS configuration
    - Confirm Linkerd mTLS is enabled by default
    - Document certificate rotation
    - _Requirements: 12.1, 12.2_

- [x] 8. Checkpoint - Verify security and HA

- [x] 9. Phase 7: Testing Suite (Week 8)
  - [x] 9.1 Write unit tests for controller
    - Test reconciliation logic with mock client
    - Test annotation mapper functions
    - Test HTTPRoute generator
    - Test status manager
    - _Requirements: 13.1_

  - [x] 9.2 Write integration tests with envtest
    - Test full reconciliation cycle
    - Test finalizer cleanup
    - Test status updates
    - Test owner reference propagation
    - _Requirements: 13.2_

  - [x] 9.3 Write property test for annotation removal
    - **Property 9: Annotation Removal on Disable**
    - **Validates: Requirements 4.3**

  - [x] 9.4 Write end-to-end tests
    - Deploy test services with Linkerd sidecars
    - Create ResiliencePolicy and verify annotations
    - Test circuit breaker activation
    - Test retry behavior
    - _Requirements: 13.4_

  - [x] 9.5 Verify code coverage
    - Run coverage report
    - Ensure 80%+ coverage
    - _Requirements: 13.5_

- [x] 10. Phase 8: Deployment and Documentation (Week 9-10)
  - [x] 10.1 Create Helm chart
    - Define Chart.yaml and values.yaml
    - Create deployment, service, RBAC templates
    - Include CRD templates
    - _Requirements: 14.1_

  - [x] 10.2 Create example manifests
    - ResiliencePolicy examples for common patterns
    - Namespace injection examples
    - Service migration examples
    - _Requirements: 14.2_

  - [x] 10.3 Write operational runbook
    - Installation procedures
    - Troubleshooting guide
    - Common operations
    - _Requirements: 14.3_

  - [x] 10.4 Write architecture documentation
    - Architecture diagrams
    - Component descriptions
    - Data flow documentation
    - _Requirements: 14.4_

- [x] 11. Phase 9: Service Migration (Week 10+)
  - [x] 11.1 Migrate IAM Policy Service to mesh
    - Add linkerd.io/inject annotation to namespace
    - Verify sidecar injection
    - Create ResiliencePolicy for iam-policy-service
    - Verify traffic flows through mesh
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 10.1, 10.2, 10.3_

  - [x] 11.2 Create migration guide
    - Document per-namespace injection
    - Document per-deployment override
    - Document gradual rollout strategy
    - _Requirements: 10.4_

- [x] 12. Final checkpoint - Full system validation

## Notes

- All tasks are required for production-ready service mesh integration
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (100+ iterations each)
- Unit tests validate specific examples and edge cases
- Timeline: 8-10 weeks for full implementation
- Can be parallelized with multiple engineers

## Dependencies

- Kubernetes 1.30+ cluster
- Linkerd 2.16+ (edge release recommended)
- Gateway API CRDs v1.1.0+
- Kubebuilder 3.14+
- Go 1.24+
- pgregory.net/rapid for property testing

## Completion Summary

All 12 phases completed successfully:
- Operator scaffold with Kubebuilder
- ResiliencePolicy CRD with CEL validation
- Controller with reconciliation, finalizers, owner references
- Linkerd annotation mapper for circuit breaker, retry, timeout
- Status manager with conditions
- Prometheus metrics
- Leader election and HA
- RBAC with least privilege
- mTLS documentation
- Property tests (10 properties, 100+ iterations each)
- Unit tests, integration tests, e2e tests
- Helm chart for deployment
- Example manifests
- Operational runbook
- Architecture documentation
- IAM Policy Service migration manifests
- Migration guide
