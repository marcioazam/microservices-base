# Service Mesh Architecture

## Overview

The Resilience Service Operator integrates with Linkerd 2.16+ to provide declarative resilience policies for Kubernetes services.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                          │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                  Linkerd Control Plane                     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │  │
│  │  │ Destination │  │  Identity   │  │   Proxy     │        │  │
│  │  │  Controller │  │  Controller │  │  Injector   │        │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘        │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│  ┌───────────────────────────┼───────────────────────────────┐  │
│  │         Resilience Operator (resilience-system)           │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │              ResiliencePolicy Controller            │  │  │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌────────────┐   │  │  │
│  │  │  │ Reconciler  │  │  Annotation │  │   Status   │   │  │  │
│  │  │  │             │  │   Mapper    │  │  Manager   │   │  │  │
│  │  │  └─────────────┘  └─────────────┘  └────────────┘   │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│  ┌───────────────────────────┼───────────────────────────────┐  │
│  │              Application Namespace                         │  │
│  │  ┌─────────────────┐      │      ┌─────────────────┐      │  │
│  │  │ ResiliencePolicy│──────┼─────▶│    Service      │      │  │
│  │  │   (CRD)         │      │      │  (annotations)  │      │  │
│  │  └─────────────────┘      │      └─────────────────┘      │  │
│  │                           │              │                 │  │
│  │                           │      ┌───────┴───────┐        │  │
│  │                           │      │               │        │  │
│  │  ┌─────────────────┐      │  ┌───┴───┐       ┌───┴───┐   │  │
│  │  │   HTTPRoute     │◀─────┘  │  Pod  │       │  Pod  │   │  │
│  │  │ (retry/timeout) │         │+proxy │       │+proxy │   │  │
│  │  └─────────────────┘         └───────┘       └───────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### ResiliencePolicy Controller

- Watches ResiliencePolicy CRDs
- Reconciles desired state with actual state
- Applies Linkerd annotations to Services
- Creates HTTPRoutes for retry/timeout

### Annotation Mapper

- Maps ResiliencePolicy spec to Linkerd annotations
- Handles circuit breaker, retry, timeout configs
- Removes annotations when features disabled

### Status Manager

- Updates ResiliencePolicy status conditions
- Tracks applied services
- Reports reconciliation state

## Data Flow

1. User creates ResiliencePolicy CR
2. Controller receives reconciliation event
3. Controller looks up target Service
4. Annotation Mapper generates Linkerd annotations
5. Controller applies annotations to Service
6. Controller creates/updates HTTPRoute if needed
7. Status Manager updates policy status
8. Linkerd proxy reads annotations and applies config

## Linkerd Integration

### Circuit Breaker

Uses Linkerd's failure accrual:
- `config.linkerd.io/failure-accrual: consecutive`
- `config.linkerd.io/failure-accrual-consecutive-failures: N`

### Retry

Uses HTTPRoute with annotations:
- `retry.linkerd.io/http: N`
- `retry.linkerd.io/http-status-codes: 5xx,429`
- `retry.linkerd.io/timeout: 5s`

### Timeout

Uses HTTPRoute with annotations:
- `timeout.linkerd.io/request: 30s`
- `timeout.linkerd.io/response: 10s`

## High Availability

- 2+ replicas with leader election
- Pod anti-affinity for spread
- Exponential backoff on failures
- Informer caching for performance

## Security

- Minimal RBAC permissions
- mTLS via Linkerd (automatic)
- CEL validation on CRD inputs
- Non-root container execution
