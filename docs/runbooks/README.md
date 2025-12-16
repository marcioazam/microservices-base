# Operational Runbooks

## Vault Incident Response

### Vault Sealed

**Symptoms:** Services cannot retrieve secrets, authentication failures

**Resolution:**
```bash
# 1. Check seal status
vault status

# 2. Get unseal keys from secure storage
# 3. Unseal Vault (requires 3 of 5 keys)
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>

# 4. Verify services reconnect
kubectl logs -n auth-platform deploy/auth-edge-service | grep -i vault
```

### Vault Unavailable

**Symptoms:** 5xx errors from services, secret retrieval timeouts

**Resolution:**
```bash
# 1. Check Vault pods
kubectl get pods -n vault

# 2. Check Vault logs
kubectl logs -n vault vault-0

# 3. If leader election issue, check Raft status
kubectl exec -n vault vault-0 -- vault operator raft list-peers

# 4. Services should use cached credentials (5 min grace period)
# 5. If Vault down > 5 min, consider failover or restore from backup
```

### Secret Rotation Failure

**Symptoms:** Lease renewal errors, credential expiration

**Resolution:**
```bash
# 1. Check lease status
vault lease lookup <lease_id>

# 2. Force renewal
vault lease renew -increment=1h <lease_id>

# 3. If renewal fails, revoke and get new credentials
vault lease revoke <lease_id>

# 4. Service will automatically request new credentials
```

---

## Linkerd Troubleshooting

### mTLS Handshake Failures

**Symptoms:** Connection refused between services, TLS errors

**Resolution:**
```bash
# 1. Check identity controller
kubectl logs -n linkerd deploy/linkerd-identity

# 2. Verify certificates
kubectl get certificates -n linkerd

# 3. Check certificate expiry
linkerd check --proxy

# 4. If certificates expired, trigger rotation
kubectl delete secret -n linkerd linkerd-identity-issuer
# cert-manager will regenerate
```

### High Proxy Latency

**Symptoms:** p99 latency > 2ms, slow service responses

**Resolution:**
```bash
# 1. Check proxy resource usage
kubectl top pods -n auth-platform -c linkerd-proxy

# 2. Increase proxy resources if needed
kubectl patch deploy auth-edge-service -n auth-platform --patch '
spec:
  template:
    metadata:
      annotations:
        config.linkerd.io/proxy-cpu-request: "200m"
        config.linkerd.io/proxy-memory-request: "50Mi"
'

# 3. Check for connection pooling issues
linkerd viz tap deploy/auth-edge-service -n auth-platform
```

### Proxy Injection Not Working

**Symptoms:** Pods running without sidecar, no mTLS

**Resolution:**
```bash
# 1. Check annotation
kubectl get deploy auth-edge-service -n auth-platform -o yaml | grep linkerd

# 2. Verify injector is running
kubectl get pods -n linkerd | grep injector

# 3. Check injector logs
kubectl logs -n linkerd deploy/linkerd-proxy-injector

# 4. Force re-injection
kubectl rollout restart deploy/auth-edge-service -n auth-platform
```

---

## Contract Verification Failures

### Consumer Contract Fails

**Symptoms:** CI pipeline blocked, contract mismatch

**Resolution:**
```bash
# 1. View failed verification
pact-broker describe-version \
  --pacticipant token-service \
  --version $GIT_SHA

# 2. Compare expected vs actual
# Check Pact Broker UI for diff

# 3. Options:
#    a) Fix provider to match contract
#    b) Update consumer expectations
#    c) Mark as WIP if in development

# 4. Re-run verification
cargo test --test pact_provider_tests
```

### Can-I-Deploy Blocked

**Symptoms:** Deployment halted, matrix shows failures

**Resolution:**
```bash
# 1. View deployment matrix
pact-broker matrix \
  --pacticipant auth-edge-service \
  --version $GIT_SHA \
  --to-environment production

# 2. Identify failing contracts

# 3. Options:
#    a) Fix and verify contracts
#    b) Deploy provider first if consumer depends on new feature
#    c) Use --ignore flag for known issues (not recommended)

# 4. After fix, re-check
pact-broker can-i-deploy \
  --pacticipant auth-edge-service \
  --version $GIT_SHA \
  --to-environment production
```

### Pact Broker Unavailable

**Symptoms:** Contract publish fails, verification skipped

**Resolution:**
```bash
# 1. Check Pact Broker pods
kubectl get pods -n pact

# 2. Check database connection
kubectl logs -n pact deploy/pact-broker | grep -i database

# 3. Restart if needed
kubectl rollout restart deploy/pact-broker -n pact

# 4. Verify health
curl https://pact-broker.auth-platform.local/diagnostic/status/heartbeat
```

---

## Emergency Procedures

### Full Platform Outage

1. Check Vault status - services need secrets
2. Check Linkerd control plane - services need mTLS
3. Check database connectivity
4. Review recent deployments for rollback candidates

### Security Incident

1. Rotate all secrets immediately
2. Revoke all Vault leases
3. Regenerate Linkerd certificates
4. Review audit logs
5. Notify security team

### Data Recovery

1. Restore Vault from snapshot
2. Verify secret integrity
3. Rotate compromised credentials
4. Re-verify all contracts
