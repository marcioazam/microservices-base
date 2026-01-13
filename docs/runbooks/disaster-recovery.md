# Disaster Recovery Procedures

## Overview

This document outlines disaster recovery procedures for the Auth Platform deployment infrastructure.

## Recovery Time Objectives

| Scenario | RTO | RPO |
|----------|-----|-----|
| Single pod failure | < 1 min | 0 |
| Node failure | < 5 min | 0 |
| AZ failure | < 15 min | 0 |
| Region failure | < 1 hour | < 5 min |
| Data corruption | < 4 hours | < 1 hour |

## Backup Procedures

### 1. Kubernetes Resources

```bash
# Export all resources in namespace
kubectl get all -n auth-system -o yaml > backup/auth-system-all.yaml

# Export secrets (encrypted)
kubectl get secrets -n auth-system -o yaml | \
  kubeseal --format yaml > backup/auth-system-secrets-sealed.yaml

# Export ConfigMaps
kubectl get configmaps -n auth-system -o yaml > backup/auth-system-configmaps.yaml
```

### 2. Persistent Data

```bash
# Database backup (PostgreSQL)
kubectl exec -n data-system postgres-0 -- \
  pg_dump -U postgres auth_db > backup/auth_db_$(date +%Y%m%d).sql

# Redis backup
kubectl exec -n data-system redis-0 -- redis-cli BGSAVE
kubectl cp data-system/redis-0:/data/dump.rdb backup/redis_$(date +%Y%m%d).rdb
```

### 3. ArgoCD State

```bash
# Export ArgoCD applications
kubectl get applications -n argocd -o yaml > backup/argocd-apps.yaml

# Export ArgoCD projects
kubectl get appprojects -n argocd -o yaml > backup/argocd-projects.yaml
```

## Recovery Procedures

### Scenario 1: Single Service Recovery

```bash
# Delete and recreate deployment
kubectl delete deployment auth-edge-service -n auth-system
kubectl apply -k deploy/kubernetes/overlays/prod

# Or force ArgoCD sync
argocd app sync auth-platform-prod --force
```

### Scenario 2: Namespace Recovery

```bash
# Recreate namespace
kubectl create namespace auth-system

# Apply all resources
kubectl apply -k deploy/kubernetes/overlays/prod

# Restore secrets from sealed secrets
kubectl apply -f backup/auth-system-secrets-sealed.yaml
```

### Scenario 3: Cluster Recovery

```bash
# 1. Provision new cluster
# 2. Install prerequisites
helm install linkerd linkerd/linkerd2 -n linkerd
helm install argocd argo/argo-cd -n argocd

# 3. Restore ArgoCD state
kubectl apply -f backup/argocd-apps.yaml

# 4. ArgoCD will sync all applications
argocd app sync --all
```

### Scenario 4: Data Recovery

```bash
# Restore PostgreSQL
kubectl exec -i -n data-system postgres-0 -- \
  psql -U postgres auth_db < backup/auth_db_20241225.sql

# Restore Redis
kubectl cp backup/redis_20241225.rdb data-system/redis-0:/data/dump.rdb
kubectl exec -n data-system redis-0 -- redis-cli DEBUG RELOAD
```

## Failover Procedures

### Active-Passive Failover

```bash
# 1. Verify secondary cluster health
kubectl --context=secondary get nodes

# 2. Update DNS to point to secondary
# (via Route53/CloudFlare API)

# 3. Scale up secondary
kubectl --context=secondary scale deployment --all --replicas=3 -n auth-system

# 4. Monitor traffic shift
linkerd stat deploy -n auth-system --context=secondary
```

## Verification Checklist

After recovery, verify:

- [ ] All pods running and healthy
- [ ] Services responding to health checks
- [ ] Database connections working
- [ ] Cache connectivity verified
- [ ] External integrations functional
- [ ] Metrics flowing to Prometheus
- [ ] Logs appearing in Loki
- [ ] Traces visible in Tempo
- [ ] Alerts configured and firing correctly

## Contact Information

| Role | Contact |
|------|---------|
| Platform Team | platform@example.com |
| On-Call | PagerDuty escalation |
| Security | security@example.com |
