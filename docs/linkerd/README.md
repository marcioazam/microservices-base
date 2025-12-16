# Linkerd Service Mesh

## Overview

Linkerd provides automatic mTLS between all Auth Platform services with:
- Zero-config mutual TLS encryption
- Golden metrics (latency, success rate, request volume)
- Distributed tracing with W3C Trace Context
- cert-manager integration for certificate rotation

## Mesh Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                    Linkerd Control Plane                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Identity     │  │ Destination  │  │ Policy       │          │
│  │ Controller   │  │ Controller   │  │ Controller   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Data Plane (Proxies)                          │
│                                                                  │
│  ┌─────────────────┐      mTLS      ┌─────────────────┐        │
│  │ Auth Edge       │◄──────────────►│ Token Service   │        │
│  │ ┌─────────────┐ │                │ ┌─────────────┐ │        │
│  │ │ App         │ │                │ │ App         │ │        │
│  │ └─────────────┘ │                │ └─────────────┘ │        │
│  │ ┌─────────────┐ │                │ ┌─────────────┐ │        │
│  │ │ Proxy       │ │                │ │ Proxy       │ │        │
│  │ └─────────────┘ │                │ └─────────────┘ │        │
│  └─────────────────┘                └─────────────────┘        │
│           │                                  │                  │
│           │              mTLS                │                  │
│           ▼                                  ▼                  │
│  ┌─────────────────┐      mTLS      ┌─────────────────┐        │
│  │ Session Core    │◄──────────────►│ IAM Policy      │        │
│  │ + Proxy         │                │ + Proxy         │        │
│  └─────────────────┘                └─────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

## Certificate Rotation

### Trust Anchor (Root CA)
- Validity: 10 years
- Rotation: 1 year before expiry
- Managed by: cert-manager

### Identity Issuer
- Validity: 48 hours
- Rotation: 25 hours before expiry
- Managed by: cert-manager

### Workload Certificates
- Validity: 24 hours
- Rotation: Automatic by Linkerd

## Debugging Commands

### Check mTLS Status
```bash
# Verify mTLS between services
linkerd viz tap deploy/auth-edge-service -n auth-platform

# Check TLS identity
linkerd viz edges deploy -n auth-platform
```

### View Metrics
```bash
# Golden metrics for a service
linkerd viz stat deploy -n auth-platform

# Top routes by latency
linkerd viz routes deploy/auth-edge-service -n auth-platform
```

### Certificate Status
```bash
# Check identity certificates
linkerd identity -n auth-platform

# Verify cert-manager certificates
kubectl get certificates -n linkerd
```

### Proxy Diagnostics
```bash
# Check proxy logs
kubectl logs deploy/auth-edge-service -n auth-platform -c linkerd-proxy

# Proxy configuration
linkerd viz proxy-config deploy/auth-edge-service -n auth-platform
```

## Alerting

### Error Rate Alert
- Threshold: >1% error rate
- Window: 60 seconds
- Action: Page on-call

### Latency Alert
- Threshold: p99 > 500ms
- Window: 5 minutes
- Action: Notify team

### Certificate Expiry Alert
- Threshold: <1 hour remaining
- Action: Critical page

## Troubleshooting

### Proxy Injection Failed
```bash
# Check injector logs
kubectl logs -n linkerd deploy/linkerd-proxy-injector

# Verify annotation
kubectl get deploy auth-edge-service -n auth-platform -o yaml | grep linkerd
```

### mTLS Handshake Failed
```bash
# Check identity controller
kubectl logs -n linkerd deploy/linkerd-identity

# Verify certificates
kubectl get secret -n linkerd linkerd-identity-issuer -o yaml
```

### High Latency
```bash
# Check proxy resource usage
kubectl top pods -n auth-platform -c linkerd-proxy

# View slow requests
linkerd viz tap deploy/auth-edge-service -n auth-platform --path /slow
```
