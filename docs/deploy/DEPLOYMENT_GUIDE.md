# Auth Platform Deployment Guide

## Prerequisites

- Docker 24.0+ with Compose V2
- kubectl 1.28+
- Helm 3.13+
- Kustomize 5.0+
- ArgoCD CLI (optional)

## Local Development

### Quick Start

```bash
# Copy environment template
cp deploy/docker/.env.example deploy/docker/.env

# Edit environment variables
vim deploy/docker/.env

# Start all services
docker compose -f deploy/docker/docker-compose.yml \
  -f deploy/docker/docker-compose.dev.yml up -d

# Check service health
docker compose ps

# View logs
docker compose logs -f auth-edge-service
```

### Development Overrides

The `docker-compose.dev.yml` provides:
- Debug ports exposed
- Verbose logging (DEBUG level)
- Volume mounts for hot reload
- Reduced resource limits

## Staging Deployment

### Using Kustomize

```bash
# Preview staging manifests
kustomize build deploy/kubernetes/overlays/staging

# Apply to cluster
kubectl apply -k deploy/kubernetes/overlays/staging

# Verify deployment
kubectl get pods -n auth-system-staging
```

### Using ArgoCD

```bash
# Apply ArgoCD Application
kubectl apply -f deploy/kubernetes/argocd/staging-application.yaml

# Check sync status
argocd app get auth-platform-staging

# Force sync if needed
argocd app sync auth-platform-staging
```

## Production Deployment

### Pre-Deployment Checklist

- [ ] All property tests pass
- [ ] Security scanning completed
- [ ] Image tags are semver (not `latest`)
- [ ] Resource limits configured
- [ ] PDB and HPA enabled
- [ ] Network policies applied
- [ ] Secrets configured in Vault

### Using Kustomize

```bash
# Preview production manifests
kustomize build deploy/kubernetes/overlays/prod

# Apply with dry-run first
kubectl apply -k deploy/kubernetes/overlays/prod --dry-run=server

# Apply to cluster
kubectl apply -k deploy/kubernetes/overlays/prod

# Verify rollout
kubectl rollout status deployment/auth-edge-service -n auth-system
```

### Using Helm

```bash
# Install/upgrade with production values
helm upgrade --install auth-platform \
  deploy/kubernetes/helm/auth-platform \
  --namespace auth-system \
  --create-namespace \
  --set global.environment=production \
  --set authEdgeService.image.tag=v1.0.0 \
  --set tokenService.image.tag=v1.0.0 \
  --set sessionIdentityCore.image.tag=v1.0.0 \
  --set iamPolicyService.image.tag=v1.0.0 \
  --set mfaService.image.tag=v1.0.0

# Verify deployment
helm status auth-platform -n auth-system
```

## Rollback Procedures

### Kustomize/kubectl

```bash
# View rollout history
kubectl rollout history deployment/auth-edge-service -n auth-system

# Rollback to previous revision
kubectl rollout undo deployment/auth-edge-service -n auth-system

# Rollback to specific revision
kubectl rollout undo deployment/auth-edge-service -n auth-system --to-revision=2
```

### Helm

```bash
# View release history
helm history auth-platform -n auth-system

# Rollback to previous release
helm rollback auth-platform -n auth-system

# Rollback to specific revision
helm rollback auth-platform 2 -n auth-system
```

### ArgoCD

```bash
# View sync history
argocd app history auth-platform-prod

# Rollback to previous sync
argocd app rollback auth-platform-prod
```

## Validation Commands

```bash
# Validate Docker Compose
docker compose -f deploy/docker/docker-compose.yml config

# Lint Helm chart
helm lint deploy/kubernetes/helm/auth-platform

# Validate Kubernetes manifests
kubectl apply -k deploy/kubernetes/overlays/prod --dry-run=server

# Run property tests
pytest deploy/tests/ -v
```
