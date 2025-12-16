# Envoy Gateway - Auth Platform API Gateway

Kubernetes-native API Gateway using Envoy Gateway with Gateway API.

## Overview

This setup uses **Envoy Gateway** as the north-south API Gateway, implementing the **Kubernetes Gateway API** (GA since v1.0). This is the modern, standards-based approach for Kubernetes ingress that replaces the legacy Ingress API.

## Why Envoy Gateway + Gateway API?

### Gateway API Benefits
- **GA and Stable**: Core resources (Gateway, GatewayClass, HTTPRoute) are GA with compatibility guarantees
- **Expressive**: More powerful routing than Ingress (header matching, weighted routing, etc.)
- **Role-Oriented**: Clear separation between infrastructure and application concerns
- **Portable**: Standard API works across implementations

### Envoy Gateway Benefits
- **gRPC/HTTP2 Native**: First-class support for gRPC routing via GRPCRoute
- **High Performance**: Built on Envoy proxy, battle-tested at scale
- **CNCF Ecosystem**: Integrates with OpenTelemetry, cert-manager, etc.
- **Extensible**: Policy attachments for rate limiting, auth, etc.

## Architecture

```
                    Internet
                        │
                        ▼
              ┌─────────────────┐
              │  Cloud LB/NLB   │
              └────────┬────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │     Envoy Gateway            │
        │  (Gateway API Controller)    │
        │                              │
        │  ┌────────────────────────┐  │
        │  │   Gateway Resource     │  │
        │  │  - HTTPS :443          │  │
        │  │  - gRPC  :8443         │  │
        │  │  - Health :8080        │  │
        │  └────────────────────────┘  │
        │                              │
        │  ┌────────────────────────┐  │
        │  │   Policies             │  │
        │  │  - Rate Limiting       │  │
        │  │  - JWT Validation      │  │
        │  │  - CORS                │  │
        │  │  - Circuit Breaker     │  │
        │  └────────────────────────┘  │
        └──────────────┬───────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
   HTTPRoute      GRPCRoute      GRPCRoute
   (REST API)     (Auth Edge)   (Token Svc)
        │              │              │
        ▼              ▼              ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │Session  │   │Auth Edge│   │Token    │
   │Identity │   │Service  │   │Service  │
   └─────────┘   └─────────┘   └─────────┘
```

## Installation

### Prerequisites

```bash
# Install Gateway API CRDs
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

# Install Envoy Gateway
helm install eg oci://docker.io/envoyproxy/gateway-helm \
  --version v1.2.0 \
  -n envoy-gateway-system \
  --create-namespace

# Install cert-manager (for TLS)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
```

### Deploy Auth Platform Gateway

```bash
# Create namespace
kubectl create namespace auth-system

# Apply gateway configuration
kubectl apply -f deployment/kubernetes/gateway/

# Verify
kubectl get gateways -n auth-system
kubectl get httproutes -n auth-system
kubectl get grpcroutes -n auth-system
```

## Configuration Files

| File | Description |
|------|-------------|
| `envoy-gateway.yaml` | GatewayClass and EnvoyProxy configuration |
| `gateway.yaml` | Gateway resource with listeners |
| `http-routes.yaml` | HTTPRoute for REST APIs |
| `grpc-routes.yaml` | GRPCRoute for gRPC services |
| `policies.yaml` | Security, rate limiting, and traffic policies |

## Features

### Rate Limiting
- Global rate limits per client IP
- Stricter limits for sensitive endpoints (token, auth)
- Adaptive limits for failed authentication attempts

### Security
- TLS termination with cert-manager
- CORS configuration
- JWT validation at gateway level
- Security headers injection

### Traffic Management
- Circuit breaker for backend services
- Retry policies with exponential backoff
- Health checks (active and passive)
- Load balancing (Least Request)

### Observability
- Structured JSON access logs
- Prometheus metrics
- OpenTelemetry tracing integration

## Routing

### HTTP Routes
| Path | Service | Description |
|------|---------|-------------|
| `/oauth2/*` | session-identity-core | OAuth 2.1 endpoints |
| `/.well-known/jwks.json` | token-service | JWKS endpoint |
| `/.well-known/openid-configuration` | session-identity-core | OIDC discovery |
| `/mfa/*` | mfa-service | MFA endpoints |
| `/sessions/*` | session-identity-core | Session management |

### gRPC Routes
| Service | Backend |
|---------|---------|
| `auth.edge.AuthEdgeService` | auth-edge-service:8080 |
| `auth.token.TokenService` | token-service:8081 |
| `auth.session.SessionIdentityService` | session-identity-core:8082 |
| `auth.iam.IAMPolicyService` | iam-policy-service:8083 |
| `auth.mfa.MFAService` | mfa-service:8084 |

## East-West Traffic (Service Mesh)

For internal service-to-service (east-west) traffic with mTLS, consider complementing with:

- **Istio**: Full-featured service mesh with mTLS, traffic management
- **Linkerd**: Lightweight service mesh focused on simplicity
- **Cilium**: eBPF-based networking with service mesh capabilities

The Gateway API handles north-south traffic, while service mesh handles east-west.

## Customization

### Adding New Routes

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: my-new-route
  namespace: auth-system
spec:
  parentRefs:
    - name: auth-platform-gateway
      sectionName: https
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /my-endpoint
      backendRefs:
        - name: my-service
          port: 8080
```

### Custom Rate Limits

```yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: custom-rate-limit
spec:
  targetRefs:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
      name: my-new-route
  rateLimit:
    type: Local
    local:
      rules:
        - limit:
            requests: 50
            unit: Second
```

## Troubleshooting

```bash
# Check gateway status
kubectl describe gateway auth-platform-gateway -n auth-system

# Check route status
kubectl describe httproute auth-edge-route -n auth-system

# View Envoy logs
kubectl logs -l gateway.envoyproxy.io/owning-gateway-name=auth-platform-gateway -n auth-system

# Check Envoy config
kubectl port-forward svc/envoy-auth-platform-gateway -n auth-system 19000:19000
curl localhost:19000/config_dump
```

## References

- [Gateway API Documentation](https://gateway-api.sigs.k8s.io/)
- [Envoy Gateway Documentation](https://gateway.envoyproxy.io/)
- [GRPCRoute Specification](https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.GRPCRoute)
