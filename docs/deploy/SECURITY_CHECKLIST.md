# Security Hardening Checklist

## Pod Security Standards (PSS)

### Restricted Profile Compliance

- [x] `runAsNonRoot: true` - Pods run as non-root user
- [x] `runAsUser: 65534` - Explicit non-root UID
- [x] `runAsGroup: 65534` - Explicit non-root GID
- [x] `fsGroup: 65534` - File system group set
- [x] `seccompProfile.type: RuntimeDefault` - Seccomp enabled
- [x] `allowPrivilegeEscalation: false` - No privilege escalation
- [x] `readOnlyRootFilesystem: true` - Read-only root FS
- [x] `capabilities.drop: [ALL]` - All capabilities dropped

### Verification Commands

```bash
# Check pod security context
kubectl get pod <pod> -n auth-system -o jsonpath='{.spec.securityContext}'

# Check container security context
kubectl get pod <pod> -n auth-system -o jsonpath='{.spec.containers[0].securityContext}'

# Validate against PSS
kubectl label --dry-run=server --overwrite ns auth-system \
  pod-security.kubernetes.io/enforce=restricted
```

## Network Security

### Network Policies

- [x] Default deny ingress policy
- [x] Explicit allow rules per service
- [x] Egress restrictions to known endpoints
- [x] Namespace isolation

### Verification

```bash
# List network policies
kubectl get networkpolicy -n auth-system

# Test connectivity
kubectl run test --rm -it --image=busybox -- wget -qO- http://auth-edge-service:8080/health
```

## Secret Management

### Best Practices

- [x] No secrets in ConfigMaps
- [x] ExternalSecrets for Vault integration
- [x] SealedSecrets for GitOps
- [x] Secret rotation enabled
- [x] No secrets in environment variables (use volume mounts)

### Verification

```bash
# Check for secrets in env vars
kubectl get deployment -n auth-system -o jsonpath='{.items[*].spec.template.spec.containers[*].env[*].valueFrom.secretKeyRef}'

# Verify ExternalSecrets
kubectl get externalsecrets -n auth-system
```

## Image Security

### Requirements

- [x] No `latest` tags in production
- [x] Images from trusted registries only
- [x] Image pull secrets configured
- [x] Vulnerability scanning in CI/CD

### Verification

```bash
# Check image tags
kubectl get pods -n auth-system -o jsonpath='{.items[*].spec.containers[*].image}'

# Scan images with Trivy
trivy image auth-platform/auth-edge-service:v1.0.0
```

## Service Mesh Security

### Linkerd mTLS

- [x] Namespace injection enabled
- [x] All pods have proxy sidecar
- [x] mTLS enforced between services
- [x] Authorization policies defined

### Verification

```bash
# Check Linkerd injection
kubectl get ns auth-system -o jsonpath='{.metadata.annotations.linkerd\.io/inject}'

# Verify mTLS
linkerd viz edges -n auth-system
linkerd viz stat deploy -n auth-system
```

## Resource Limits

### Requirements

- [x] CPU limits set for all containers
- [x] Memory limits set for all containers
- [x] CPU requests set for all containers
- [x] Memory requests set for all containers

### Verification

```bash
# Check resource limits
kubectl get pods -n auth-system -o jsonpath='{.items[*].spec.containers[*].resources}'
```

## Audit Logging

### Requirements

- [x] Kubernetes audit logging enabled
- [x] Application audit logs to Loki
- [x] Security events to SIEM
- [x] Log retention policies configured

## Compliance Summary

| Category | Status | Notes |
|----------|--------|-------|
| PSS Restricted | ✅ | All pods compliant |
| Network Isolation | ✅ | NetworkPolicies applied |
| Secret Management | ✅ | Vault + ExternalSecrets |
| Image Security | ✅ | Semver tags, scanning |
| mTLS | ✅ | Linkerd enforced |
| Resource Limits | ✅ | All containers limited |
| Audit Logging | ✅ | Centralized in Loki |
