# Deployment Troubleshooting Runbook

## Common Issues and Resolutions

### 1. Pod Stuck in Pending State

**Symptoms:**
- Pod status shows `Pending`
- Events show scheduling issues

**Diagnosis:**
```bash
kubectl describe pod <pod-name> -n auth-system
kubectl get events -n auth-system --sort-by='.lastTimestamp'
```

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| Insufficient resources | Scale cluster or reduce requests |
| Node selector mismatch | Check nodeSelector in deployment |
| PVC not bound | Verify StorageClass and PVC |
| Taints/tolerations | Add appropriate tolerations |

### 2. CrashLoopBackOff

**Symptoms:**
- Pod restarts repeatedly
- Status shows `CrashLoopBackOff`

**Diagnosis:**
```bash
kubectl logs <pod-name> -n auth-system --previous
kubectl describe pod <pod-name> -n auth-system
```

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| Missing config/secrets | Verify ConfigMaps and Secrets exist |
| Database connection | Check DB connectivity and credentials |
| OOM killed | Increase memory limits |
| Probe failures | Adjust probe timing or endpoints |

### 3. Service Unavailable (503)

**Symptoms:**
- Gateway returns 503 errors
- Services appear healthy

**Diagnosis:**
```bash
kubectl get endpoints -n auth-system
kubectl get svc -n auth-system
linkerd stat deploy -n auth-system
```

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| No ready endpoints | Check pod readiness probes |
| NetworkPolicy blocking | Verify NetworkPolicy rules |
| Service selector mismatch | Check labels match |
| Circuit breaker open | Wait for recovery or check upstream |

### 4. High Latency

**Symptoms:**
- P99 latency exceeds SLO
- Alerts firing

**Diagnosis:**
```bash
# Check Linkerd metrics
linkerd stat deploy -n auth-system

# Check resource usage
kubectl top pods -n auth-system

# Check HPA status
kubectl get hpa -n auth-system
```

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| CPU throttling | Increase CPU limits |
| Memory pressure | Increase memory or optimize |
| Connection pool exhaustion | Tune pool settings |
| Downstream dependency | Check dependency health |

### 5. ArgoCD Sync Failed

**Symptoms:**
- Application shows `OutOfSync` or `Degraded`
- Sync operation fails

**Diagnosis:**
```bash
argocd app get auth-platform-prod
argocd app sync auth-platform-prod --dry-run
kubectl get events -n auth-system
```

**Common Causes & Solutions:**

| Cause | Solution |
|-------|----------|
| Invalid manifests | Run `kustomize build` locally |
| RBAC issues | Check ArgoCD service account |
| Resource conflicts | Check for manual changes |
| Webhook validation | Check admission webhooks |

## Diagnostic Commands

```bash
# Overall cluster health
kubectl get nodes
kubectl top nodes

# Namespace overview
kubectl get all -n auth-system

# Pod details
kubectl describe pod <pod> -n auth-system
kubectl logs <pod> -n auth-system -f

# Service mesh status
linkerd check
linkerd stat deploy -n auth-system
linkerd tap deploy/<service> -n auth-system

# Network debugging
kubectl run debug --rm -it --image=nicolaka/netshoot -- /bin/bash
```

## Escalation Path

1. **L1**: Check runbook, restart pods if needed
2. **L2**: Investigate logs, check dependencies
3. **L3**: Engage platform team, consider rollback
4. **P1**: Immediate rollback, incident bridge
