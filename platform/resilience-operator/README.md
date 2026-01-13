# Resilience Operator

Kubernetes Operator for managing resilience policies via Linkerd Service Mesh.

## Overview

The Resilience Operator provides a declarative way to configure circuit breakers, retries, and timeouts for Kubernetes services using Linkerd annotations.

## Features

- **Circuit Breaker**: Automatic failure isolation with configurable thresholds
- **Retry**: Automatic retries with configurable attempts and status codes
- **Timeout**: Request and response timeout configuration
- **Rate Limiting**: (Future) Request rate limiting

## Quick Start

```bash
# Install CRDs
make install

# Deploy operator
make deploy IMG=resilience-operator:latest

# Create a policy
kubectl apply -f config/samples/resilience_v1_resiliencepolicy.yaml
```

## Example Policy

```yaml
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: api-resilience
spec:
  targetRef:
    name: api-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  retry:
    enabled: true
    maxAttempts: 3
    retryableStatusCodes: "5xx,429"
  timeout:
    enabled: true
    requestTimeout: "30s"
```

## Development

```bash
# Run tests
make test

# Run locally
make run

# Build image
make docker-build IMG=resilience-operator:latest
```

## Architecture

The operator watches `ResiliencePolicy` CRDs and:
1. Applies circuit breaker config via Service annotations
2. Creates HTTPRoutes for retry/timeout configuration
3. Manages cleanup via finalizers

## Requirements

- Kubernetes 1.28+
- Linkerd 2.16+
- Gateway API v1.2+
