# Service Mesh Infrastructure Setup

## Prerequisites

- Kubernetes 1.30+
- kubectl configured
- Helm 3.x (optional)

## Quick Start

### 1. Create Kind Cluster (Development)

```bash
kind create cluster --name service-mesh-dev --config kind-config.yaml
```

### 2. Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=300s
```

### 3. Install Linkerd 2.16+

```bash
# Install CLI
curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install-edge | sh
export PATH=$PATH:$HOME/.linkerd2/bin

# Pre-check
linkerd check --pre

# Install CRDs and control plane
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -

# Verify
linkerd check
```

### 4. Install Gateway API CRDs

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml
```

### 5. Install Linkerd Viz

```bash
linkerd viz install | kubectl apply -f -
linkerd viz check
```

## Validation

```bash
# Check cluster
kubectl version --short
kubectl get nodes

# Check Linkerd
linkerd check

# Check Gateway API
kubectl get crds | grep gateway
```
