# Resilience Operator Runbook

## Overview

This runbook covers operational procedures for the Resilience Operator.

## Installation

### Prerequisites

- Kubernetes 1.30+
- Linkerd 2.16+ installed
- Gateway API CRDs v1.1.0+
- Helm 3.x

### Install with Helm

```bash
# Add repository
helm repo add auth-platform https://charts.auth-platform.github.com

# Install operator
helm install resilience-operator auth-platform/resilience-operator \
  --namespace resilience-system \
  --create-namespace

# Verify installation
kubectl get pods -n resilience-system
```

### Install CRDs Manually

```bash
kubectl apply -f deploy/kubernetes/service-mesh/helm/resilience-operator/crds/
```

## Common Operations

### Create a ResiliencePolicy

```bash
kubectl apply -f - <<EOF
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: my-policy
  namespace: default
spec:
  targetRef:
    name: my-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
EOF
```

### Check Policy Status

```bash
kubectl get resiliencepolicy -A
kubectl describe resiliencepolicy my-policy
```

### View Operator Logs

```bash
kubectl logs -n resilience-system -l app.kubernetes.io/name=resilience-operator -f
```

## Troubleshooting

### Policy Not Applied

1. Check policy status:
   ```bash
   kubectl get respol my-policy -o yaml
   ```

2. Verify target service exists:
   ```bash
   kubectl get svc my-service
   ```

3. Check operator logs for errors:
   ```bash
   kubectl logs -n resilience-system deploy/resilience-operator --tail=100
   ```

### Service Annotations Missing

1. Verify service annotations:
   ```bash
   kubectl get svc my-service -o jsonpath='{.metadata.annotations}'
   ```

2. Check if policy is Ready:
   ```bash
   kubectl get respol my-policy -o jsonpath='{.status.conditions}'
   ```

### Operator Not Starting

1. Check pod status:
   ```bash
   kubectl get pods -n resilience-system
   kubectl describe pod -n resilience-system -l app.kubernetes.io/name=resilience-operator
   ```

2. Verify RBAC:
   ```bash
   kubectl auth can-i --list --as=system:serviceaccount:resilience-system:resilience-operator
   ```

## Metrics

### Key Metrics

| Metric | Description |
|--------|-------------|
| `resilience_reconciliations_total` | Total reconciliations |
| `resilience_reconciliation_errors_total` | Reconciliation errors |
| `resilience_reconciliation_duration_seconds` | Reconciliation duration |
| `resilience_policies_total` | Total policies by status |

### Prometheus Queries

```promql
# Reconciliation rate
rate(resilience_reconciliations_total[5m])

# Error rate
rate(resilience_reconciliation_errors_total[5m]) / rate(resilience_reconciliations_total[5m])

# P99 reconciliation latency
histogram_quantile(0.99, rate(resilience_reconciliation_duration_seconds_bucket[5m]))
```

## Alerts

### Critical Alerts

- **ResilienceOperatorDown**: Operator not running
- **HighReconciliationErrorRate**: >5% error rate for 5 minutes
- **ReconciliationLatencyHigh**: P99 >10s for 5 minutes

## Upgrade Procedure

```bash
# Backup CRDs
kubectl get crd resiliencepolicies.resilience.auth-platform.github.com -o yaml > backup-crd.yaml

# Upgrade
helm upgrade resilience-operator auth-platform/resilience-operator \
  --namespace resilience-system

# Verify
kubectl rollout status deployment/resilience-operator -n resilience-system
```

## Rollback

```bash
helm rollback resilience-operator -n resilience-system
```
