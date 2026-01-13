# Auth Platform Deployment Architecture

## Overview

The Auth Platform uses a modern GitOps-based deployment architecture with the following key components:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Developer Workflow                            │
├─────────────────────────────────────────────────────────────────────┤
│  Git Push → GitHub Actions → Validation → ArgoCD → Kubernetes       │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                               │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │   Gateway   │  │   Linkerd   │  │    OTel     │                 │
│  │    API      │  │   (mTLS)    │  │  Collector  │                 │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │
│         │                │                │                         │
│  ┌──────▼────────────────▼────────────────▼──────┐                 │
│  │              Auth Platform Services            │                 │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐         │                 │
│  │  │  Auth   │ │  Token  │ │ Session │  ...    │                 │
│  │  │  Edge   │ │ Service │ │ Identity│         │                 │
│  │  └─────────┘ └─────────┘ └─────────┘         │                 │
│  └───────────────────────────────────────────────┘                 │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                    Observability Stack                               │
├─────────────────────────────────────────────────────────────────────┤
│  Prometheus → Grafana    Loki → Grafana    Tempo → Grafana          │
│  (Metrics)               (Logs)            (Traces)                  │
└─────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
deploy/
├── docker/
│   ├── docker-compose.yml          # Base compose (V2 format)
│   ├── docker-compose.dev.yml      # Development overrides
│   ├── docker-compose.staging.yml  # Staging overrides
│   ├── docker-compose.prod.yml     # Production overrides
│   ├── .env.example                # Environment template
│   └── observability/
│       └── otel-collector-config.yaml
├── kubernetes/
│   ├── base/                       # Kustomize base manifests
│   ├── overlays/
│   │   ├── dev/                    # Development overlay
│   │   ├── staging/                # Staging overlay
│   │   └── prod/                   # Production overlay
│   ├── helm/
│   │   └── auth-platform/          # Helm chart
│   ├── gateway/                    # Gateway API resources
│   ├── argocd/                     # ArgoCD Applications
│   └── observability/              # Observability stack
└── tests/                          # Property-based tests
```

## Key Components

### 1. Docker Compose (Local Development)
- V2 format with extensions for DRY configuration
- Environment-specific overrides
- Health checks with `service_healthy` conditions
- Security hardening (no-new-privileges, read-only)

### 2. Kubernetes (Production)
- PSS Restricted profile compliance
- Kustomize overlays for environment separation
- Helm charts for templated deployments
- Gateway API for traffic management

### 3. Service Mesh (Linkerd)
- Automatic mTLS between services
- Retry policies via ServiceProfiles
- Circuit breaker via failure accrual
- Traffic splitting for canary deployments

### 4. Observability
- OpenTelemetry Collector for unified telemetry
- Prometheus for metrics with remote write
- Loki for log aggregation
- Tempo for distributed tracing
- Grafana for visualization

## Security Features

| Feature | Implementation |
|---------|---------------|
| Pod Security | PSS Restricted profile |
| Network Isolation | NetworkPolicy per service |
| Secret Management | ExternalSecrets + Vault |
| mTLS | Linkerd automatic encryption |
| Image Security | No `latest` tags in production |

## Deployment Environments

| Environment | Replicas | Image Tags | Features |
|-------------|----------|------------|----------|
| Development | 1 | `dev` | Debug logging, hot reload |
| Staging | 2 | `staging` | Production-like, testing |
| Production | 3+ | Semver | HA, autoscaling, PDB |
