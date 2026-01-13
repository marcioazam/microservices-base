#!/bin/bash
# Service Mesh Cluster Setup Script
# Requirements: kubectl, kind, helm (optional)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-service-mesh-dev}"

echo "=== Service Mesh Infrastructure Setup ==="

# 1. Create Kind cluster
echo "[1/5] Creating Kind cluster..."
if kind get clusters | grep -q "$CLUSTER_NAME"; then
    echo "Cluster $CLUSTER_NAME already exists"
else
    kind create cluster --name "$CLUSTER_NAME" --config "$SCRIPT_DIR/kind-config.yaml"
fi

# 2. Install cert-manager
echo "[2/5] Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=300s

# 3. Install Linkerd
echo "[3/5] Installing Linkerd 2.16+..."
if ! command -v linkerd &> /dev/null; then
    curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install-edge | sh
    export PATH=$PATH:$HOME/.linkerd2/bin
fi

linkerd check --pre
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -
linkerd check

# 4. Install Gateway API CRDs
echo "[4/5] Installing Gateway API CRDs v1.1.0..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# 5. Install Linkerd Viz
echo "[5/5] Installing Linkerd Viz..."
linkerd viz install | kubectl apply -f -
linkerd viz check

echo "=== Setup Complete ==="
echo "Cluster: $CLUSTER_NAME"
echo "Linkerd: $(linkerd version --client --short)"
echo ""
echo "Next steps:"
echo "  1. Deploy resilience operator: kubectl apply -f operator/"
echo "  2. Create ResiliencePolicy: kubectl apply -f samples/"
