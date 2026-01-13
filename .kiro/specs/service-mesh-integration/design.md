# Design Document: Service Mesh Integration

## Overview

This design document describes the architecture for implementing a Service Mesh using Linkerd 2.16+ with the Resilience Service transformed into a Kubernetes Operator. The operator manages ResiliencePolicy custom resources and translates them into Linkerd configurations via Gateway API (HTTPRoute/GRPCRoute) and Service annotations.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes Control Plane                   │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         Resilience Operator (controller-runtime)        │ │
│  │                                                          │ │
│  │  ┌──────────────┐      ┌──────────────────────────┐   │ │
│  │  │ ResiliencePolicy │ → │  Reconciliation Loop     │   │ │
│  │  │     CRD       │      │  (watch → reconcile)     │   │ │
│  │  └──────────────┘      └──────────────────────────┘   │ │
│  │                                  ↓                      │ │
│  │                    ┌─────────────────────────┐         │ │
│  │                    │  Gateway API Resources  │         │ │
│  │                    │  - HTTPRoute            │         │ │
│  │                    │  - Service Annotations  │         │ │
│  │                    └─────────────────────────┘         │ │
│  └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                     Linkerd Control Plane                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Destination │  │   Identity  │  │  Proxy Injector     │ │
│  │  Controller │  │  Controller │  │  (Webhook)          │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│   IAM Policy     │    │   Auth Service   │    │   Other Service  │
│  ┌────────────┐  │    │  ┌────────────┐  │    │  ┌────────────┐  │
│  │ linkerd2-  │  │    │  │ linkerd2-  │  │    │  │ linkerd2-  │  │
│  │   proxy    │◄─┼────┼─►│   proxy    │◄─┼────┼─►│   proxy    │  │
│  │ (sidecar)  │  │    │  │ (sidecar)  │  │    │  │ (sidecar)  │  │
│  └─────┬──────┘  │    │  └─────┬──────┘  │    │  └─────┬──────┘  │
│        │         │    │        │         │    │        │         │
│  ┌─────▼──────┐  │    │  ┌─────▼──────┐  │    │  ┌─────▼──────┐  │
│  │ App Logic  │  │    │  │ App Logic  │  │    │  │ App Logic  │  │
│  └────────────┘  │    │  └────────────┘  │    │  └────────────┘  │
└──────────────────┘    └──────────────────┘    └──────────────────┘
```

### Data Flow

1. Platform engineer creates ResiliencePolicy CR
2. Operator watches for ResiliencePolicy changes
3. Operator reconciles by:
   - Applying circuit breaker config via Service annotations
   - Creating/updating HTTPRoute with retry/timeout annotations
4. Linkerd control plane reads configurations
5. Linkerd proxies enforce policies on all traffic

## Components and Interfaces

### Component 1: ResiliencePolicy CRD

Custom Resource Definition for declaring resilience policies.

```go
// api/v1/resiliencepolicy_types.go
type ResiliencePolicySpec struct {
    TargetRef      TargetReference      `json:"targetRef"`
    CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`
    Retry          *RetryConfig          `json:"retry,omitempty"`
    Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
    RateLimit      *RateLimitConfig      `json:"rateLimit,omitempty"`
}

type TargetReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
    Port      *int32 `json:"port,omitempty"`
}

type CircuitBreakerConfig struct {
    Enabled          bool  `json:"enabled"`
    FailureThreshold int32 `json:"failureThreshold"`
}

type RetryConfig struct {
    Enabled              bool   `json:"enabled"`
    MaxAttempts          int32  `json:"maxAttempts"`
    RetryableStatusCodes string `json:"retryableStatusCodes,omitempty"`
    RetryTimeout         string `json:"retryTimeout,omitempty"`
}

type TimeoutConfig struct {
    Enabled         bool   `json:"enabled"`
    RequestTimeout  string `json:"requestTimeout"`
    ResponseTimeout string `json:"responseTimeout,omitempty"`
}
```

### Component 2: Reconciliation Controller

Controller that watches ResiliencePolicy and applies configurations.

```go
// internal/controller/resiliencepolicy_controller.go
type ResiliencePolicyReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Logger logr.Logger
}

func (r *ResiliencePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
func (r *ResiliencePolicyReconciler) applyCircuitBreaker(ctx context.Context, policy *v1.ResiliencePolicy, service *corev1.Service) error
func (r *ResiliencePolicyReconciler) applyRetryAndTimeout(ctx context.Context, policy *v1.ResiliencePolicy, service *corev1.Service) error
func (r *ResiliencePolicyReconciler) handleDeletion(ctx context.Context, policy *v1.ResiliencePolicy) (ctrl.Result, error)
```

### Component 3: Linkerd Annotation Mapper

Maps ResiliencePolicy to Linkerd-specific annotations.

```go
// internal/linkerd/annotations.go
type AnnotationMapper struct{}

func (m *AnnotationMapper) CircuitBreakerAnnotations(config *CircuitBreakerConfig) map[string]string
func (m *AnnotationMapper) RetryAnnotations(config *RetryConfig) map[string]string
func (m *AnnotationMapper) TimeoutAnnotations(config *TimeoutConfig) map[string]string
```

**Linkerd Annotation Mapping:**

| ResiliencePolicy Field | Linkerd Annotation |
|------------------------|-------------------|
| circuitBreaker.failureThreshold | config.linkerd.io/failure-accrual-consecutive-failures |
| retry.maxAttempts | retry.linkerd.io/http |
| retry.retryableStatusCodes | retry.linkerd.io/http-status-codes |
| retry.retryTimeout | retry.linkerd.io/timeout |
| timeout.requestTimeout | timeout.linkerd.io/request |
| timeout.responseTimeout | timeout.linkerd.io/response |

### Component 4: HTTPRoute Generator

Generates Gateway API HTTPRoute resources for retry/timeout policies.

```go
// internal/gateway/httproute.go
type HTTPRouteGenerator struct {
    client client.Client
    scheme *runtime.Scheme
}

func (g *HTTPRouteGenerator) GenerateHTTPRoute(policy *v1.ResiliencePolicy, service *corev1.Service) *gatewayv1.HTTPRoute
func (g *HTTPRouteGenerator) CreateOrUpdate(ctx context.Context, route *gatewayv1.HTTPRoute) error
```

### Component 5: Status Manager

Manages ResiliencePolicy status conditions.

```go
// internal/status/manager.go
type StatusManager struct {
    client client.Client
}

func (m *StatusManager) SetReady(ctx context.Context, policy *v1.ResiliencePolicy) error
func (m *StatusManager) SetFailed(ctx context.Context, policy *v1.ResiliencePolicy, reason, message string) error
func (m *StatusManager) SetTargetNotFound(ctx context.Context, policy *v1.ResiliencePolicy, target string) error
```

## Data Models

### ResiliencePolicy Example

```yaml
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: iam-policy-service-resilience
  namespace: production
spec:
  targetRef:
    name: iam-policy-service
    namespace: production
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  retry:
    enabled: true
    maxAttempts: 3
    retryableStatusCodes: "5xx,429"
    retryTimeout: "1s"
  timeout:
    enabled: true
    requestTimeout: "10s"
    responseTimeout: "5s"
status:
  conditions:
    - type: Ready
      status: "True"
      reason: Applied
      message: "Resilience policy successfully applied"
  observedGeneration: 1
  appliedToServices:
    - "production/iam-policy-service"
  lastUpdateTime: "2025-12-22T10:00:00Z"
```

### Generated Service Annotations

```yaml
apiVersion: v1
kind: Service
metadata:
  name: iam-policy-service
  namespace: production
  annotations:
    config.linkerd.io/failure-accrual: "consecutive"
    config.linkerd.io/failure-accrual-consecutive-failures: "5"
```

### Generated HTTPRoute

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: iam-policy-service-resilience
  namespace: production
  annotations:
    retry.linkerd.io/http: "3"
    retry.linkerd.io/http-status-codes: "5xx,429"
    retry.linkerd.io/timeout: "1s"
    timeout.linkerd.io/request: "10s"
    timeout.linkerd.io/response: "5s"
  ownerReferences:
    - apiVersion: resilience.auth-platform.github.com/v1
      kind: ResiliencePolicy
      name: iam-policy-service-resilience
spec:
  parentRefs:
    - name: iam-policy-service
      kind: Service
  rules:
    - backendRefs:
        - name: iam-policy-service
          port: 50051
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do.*

### Property 1: Reconciliation Idempotency

*For any* ResiliencePolicy and any number of reconciliation runs, the resulting Service annotations and HTTPRoute should be identical.

**Validates: Requirements 4.3**

### Property 2: Finalizer Cleanup Completeness

*For any* deleted ResiliencePolicy, all applied Service annotations and owned HTTPRoutes should be removed.

**Validates: Requirements 4.4, 4.5**

### Property 3: Circuit Breaker Annotation Consistency

*For any* ResiliencePolicy with circuitBreaker enabled, the target Service should have exactly the correct Linkerd failure-accrual annotations.

**Validates: Requirements 5.1, 5.2**

### Property 4: Retry Configuration Mapping

*For any* ResiliencePolicy with retry enabled, the generated HTTPRoute should contain all retry annotations with correct values.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4**

### Property 5: Timeout Configuration Mapping

*For any* ResiliencePolicy with timeout enabled, the generated HTTPRoute should contain all timeout annotations with correct values.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 6: Owner Reference Integrity

*For any* HTTPRoute created by the operator, it should have an owner reference to the source ResiliencePolicy.

**Validates: Requirements 4.4**

### Property 7: Status Condition Accuracy

*For any* ResiliencePolicy after reconciliation, the status conditions should accurately reflect the current state (Ready, Failed, TargetNotFound).

**Validates: Requirements 4.6, 4.7**

### Property 8: Target Service Validation

*For any* ResiliencePolicy referencing a non-existent service, the status should be set to TargetServiceNotFound and no configurations should be applied.

**Validates: Requirements 4.7**

### Property 9: Annotation Removal on Disable

*For any* ResiliencePolicy where a feature (circuitBreaker, retry, timeout) is disabled, the corresponding annotations should be removed from the target.

**Validates: Requirements 4.3**

### Property 10: Leader Election Consistency

*For any* operator deployment with multiple replicas, only one replica should be actively reconciling at any time.

**Validates: Requirements 11.1, 11.2**

## Error Handling

### Error Categories

| Error Type | Handling Strategy | Status Condition |
|------------|-------------------|------------------|
| Target service not found | Requeue after 30s | TargetServiceNotFound |
| API server unavailable | Exponential backoff | Reconciling |
| Invalid policy spec | Reject with validation error | Invalid |
| HTTPRoute creation failed | Retry with backoff | Failed |
| Annotation update failed | Retry with backoff | Failed |

### Reconciliation Error Flow

```go
func (r *ResiliencePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch policy - NotFound is terminal
    // 2. Handle deletion - cleanup and remove finalizer
    // 3. Get target service - requeue if not found
    // 4. Apply configurations - retry on transient errors
    // 5. Update status - always attempt
}
```

## Testing Strategy

### Unit Tests

- Controller reconciliation logic with mock client
- Annotation mapper functions
- HTTPRoute generator
- Status manager

### Integration Tests (envtest)

- Full reconciliation cycle with fake Kubernetes API
- Finalizer cleanup verification
- Status condition updates
- Owner reference propagation

### Property-Based Tests (rapid)

- Idempotency verification
- Annotation consistency
- Configuration mapping correctness

### End-to-End Tests

- Real Linkerd cluster with test services
- Traffic verification through mesh
- Circuit breaker activation
- Retry behavior validation

### Test Configuration

- Property tests: 100+ iterations each
- Integration tests: envtest with Gateway API CRDs
- E2E tests: kind cluster with Linkerd installed

## Deployment

### Operator Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: resilience-operator
  namespace: resilience-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: resilience-operator
  template:
    spec:
      containers:
        - name: operator
          image: resilience-operator:latest
          args:
            - --leader-elect=true
            - --metrics-bind-address=:8080
            - --health-probe-bind-address=:8081
          resources:
            limits:
              cpu: 500m
              memory: 256Mi
            requests:
              cpu: 100m
              memory: 128Mi
```

### RBAC Configuration

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: resilience-operator
rules:
  - apiGroups: ["resilience.auth-platform.github.com"]
    resources: ["resiliencepolicies"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["resilience.auth-platform.github.com"]
    resources: ["resiliencepolicies/status"]
    verbs: ["get", "update", "patch"]
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["httproutes", "grpcroutes"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get", "list", "watch", "update", "patch"]
```

### Helm Chart Structure

```
charts/resilience-operator/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── serviceaccount.yaml
│   ├── clusterrole.yaml
│   ├── clusterrolebinding.yaml
│   └── crds/
│       └── resiliencepolicy.yaml
```
