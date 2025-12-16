# HashiCorp Vault Configuration for Auth Platform

This Helm chart deploys and configures HashiCorp Vault for the Auth Platform with:
- HA mode with Raft storage (3 replicas)
- Kubernetes authentication
- KV v2, Database, PKI, and Transit secrets engines
- Audit logging enabled
- Prometheus metrics

## Prerequisites

- Kubernetes 1.25+
- Helm 3.0+
- cert-manager (for TLS certificates)

## Installation

```bash
# Add HashiCorp Helm repository
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

# Install Vault
helm install vault . -n vault --create-namespace

# Initialize Vault (first time only)
kubectl exec -n vault vault-0 -- vault operator init
```

## Configuration

See `values.yaml` for all configuration options.

## Requirements Covered

- 1.1: Kubernetes authentication without K8s Secrets
- 1.2: Dynamic database credentials with 1h TTL
- 1.3: Automatic secret renewal at 80% TTL
- 1.4: Audit logging with accessor identity
- 2.1: PKI certificates with 72h max validity
