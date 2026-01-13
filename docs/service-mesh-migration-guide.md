# Service Mesh Migration Guide

## Overview

This guide covers migrating services to the Linkerd service mesh with ResiliencePolicy support.

## Prerequisites

- Kubernetes 1.30+ cluster
- Linkerd 2.16+ installed
- Resilience Operator deployed
- kubectl configured

## Migration Strategies

### 1. Namespace-Level Injection (Recommended)

Enable injection for entire namespace:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    linkerd.io/inject: enabled
```

All new pods in this namespace will have sidecars injected.

### 2. Deployment-Level Injection

Enable injection for specific deployments:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  template:
    metadata:
      annotations:
        linkerd.io/inject: enabled
```

### 3. Opt-Out Pattern

Enable namespace injection, opt-out specific pods:

```yaml
# Namespace with injection
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    linkerd.io/inject: enabled
---
# Pod that opts out
apiVersion: v1
kind: Pod
metadata:
  annotations:
    linkerd.io/inject: disabled
```

## Step-by-Step Migration

### Step 1: Verify Linkerd Installation

```bash
linkerd check
linkerd viz check
```

### Step 2: Create Meshed Namespace

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: my-service-ns
  annotations:
    linkerd.io/inject: enabled
EOF
```

### Step 3: Deploy Service

```bash
kubectl apply -f deployment.yaml -n my-service-ns
```

### Step 4: Verify Sidecar Injection

```bash
kubectl get pods -n my-service-ns -o jsonpath='{.items[*].spec.containers[*].name}'
# Should show: my-container linkerd-proxy
```

### Step 5: Create ResiliencePolicy

```bash
kubectl apply -f - <<EOF
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: my-service-resilience
  namespace: my-service-ns
spec:
  targetRef:
    name: my-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  timeout:
    enabled: true
    requestTimeout: "30s"
EOF
```

### Step 6: Verify Policy Applied

```bash
kubectl get respol -n my-service-ns
kubectl get svc my-service -n my-service-ns -o yaml | grep linkerd
```

## Gradual Rollout

### Canary Deployment

1. Deploy canary with mesh:
```bash
kubectl apply -f canary-deployment.yaml
```

2. Route percentage of traffic:
```yaml
apiVersion: split.smi-spec.io/v1alpha1
kind: TrafficSplit
metadata:
  name: my-service-split
spec:
  service: my-service
  backends:
    - service: my-service-stable
      weight: 90
    - service: my-service-canary
      weight: 10
```

3. Monitor and increase traffic gradually.

## Rollback

### Remove from Mesh

```bash
# Remove namespace injection
kubectl annotate namespace my-namespace linkerd.io/inject-

# Restart pods to remove sidecars
kubectl rollout restart deployment -n my-namespace
```

### Remove ResiliencePolicy

```bash
kubectl delete respol my-service-resilience -n my-namespace
```

## Monitoring Migration

```bash
# Check mesh status
linkerd viz stat deploy -n my-namespace

# Check traffic
linkerd viz tap deploy/my-service -n my-namespace

# Check mTLS
linkerd viz edges deploy -n my-namespace
```

## Troubleshooting

### Sidecar Not Injected

1. Check namespace annotation
2. Check pod annotation (not disabled)
3. Verify Linkerd injector is running

### Policy Not Applied

1. Check ResiliencePolicy status
2. Verify target service exists
3. Check operator logs
