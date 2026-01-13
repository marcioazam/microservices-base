# Linkerd mTLS Configuration

## Overview

Linkerd 2.16+ provides automatic mutual TLS (mTLS) for all meshed traffic. This document describes the mTLS configuration for the Resilience Service Operator integration.

## Default mTLS Behavior

Linkerd enables mTLS by default for all traffic between meshed pods:

- **Automatic Certificate Provisioning**: Linkerd automatically provisions TLS certificates for each proxy
- **Certificate Rotation**: Certificates are automatically rotated every 24 hours
- **Zero Configuration**: No application changes required for mTLS

## Certificate Hierarchy

```
Trust Anchor (Root CA)
    └── Identity Issuer (Intermediate CA)
        └── Workload Certificates (Per-proxy)
```

### Trust Anchor

- Self-signed root certificate
- Default validity: 10 years
- Stored in `linkerd-identity-trust-roots` ConfigMap

### Identity Issuer

- Intermediate CA signed by trust anchor
- Default validity: 1 year
- Stored in `linkerd-identity-issuer` Secret
- Managed by cert-manager in production

### Workload Certificates

- Per-proxy certificates
- Default validity: 24 hours
- Automatically rotated by Linkerd identity service

## Verification Commands

### Check mTLS Status

```bash
# Verify Linkerd identity is healthy
linkerd check --proxy

# Check certificate expiration
linkerd identity

# View mTLS stats for a namespace
linkerd viz stat deploy -n <namespace>
```

### Verify Traffic is Encrypted

```bash
# Check if traffic is secured (look for 🔒 icon)
linkerd viz tap deploy/<deployment> -n <namespace>

# View TLS stats
linkerd viz edges deploy -n <namespace>
```

## Production Configuration

### Using cert-manager for Certificate Management

```yaml
# cert-manager ClusterIssuer for Linkerd
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: linkerd-trust-anchor
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: linkerd-identity-issuer
  namespace: linkerd
spec:
  secretName: linkerd-identity-issuer
  duration: 8760h # 1 year
  renewBefore: 720h # 30 days
  issuerRef:
    name: linkerd-trust-anchor
    kind: ClusterIssuer
  commonName: identity.linkerd.cluster.local
  dnsNames:
    - identity.linkerd.cluster.local
  isCA: true
  privateKey:
    algorithm: ECDSA
  usages:
    - cert sign
    - crl sign
    - server auth
    - client auth
```

### Certificate Rotation Monitoring

Monitor certificate expiration with Prometheus:

```yaml
# Alert for certificate expiration
groups:
  - name: linkerd-mtls
    rules:
      - alert: LinkerdCertificateExpiringSoon
        expr: |
          linkerd_identity_cert_expiration_timestamp_seconds - time() < 86400 * 7
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Linkerd certificate expiring within 7 days"
```

## Security Considerations

### Requirements Met

- **Requirement 12.1**: mTLS enabled by default for all meshed traffic
- **Requirement 12.2**: Certificate rotation handled automatically

### Best Practices

1. **Use cert-manager in production** for automated certificate lifecycle management
2. **Monitor certificate expiration** with Prometheus alerts
3. **Rotate trust anchor** before expiration (typically every 5-10 years)
4. **Use external CA** for multi-cluster deployments

### Disabling mTLS (Not Recommended)

For debugging purposes only:

```yaml
# Skip mTLS for specific port (NOT for production)
apiVersion: v1
kind: Service
metadata:
  annotations:
    config.linkerd.io/skip-outbound-ports: "3306"
```

## Troubleshooting

### Certificate Issues

```bash
# Check identity controller logs
kubectl logs -n linkerd deploy/linkerd-identity

# Verify proxy certificates
linkerd viz tap deploy/<deployment> --to deploy/<other> | grep tls
```

### Common Issues

1. **Clock Skew**: Ensure NTP is configured on all nodes
2. **Certificate Expired**: Check cert-manager logs and certificate status
3. **Identity Not Ready**: Verify identity controller is healthy

## References

- [Linkerd mTLS Documentation](https://linkerd.io/2/features/automatic-mtls/)
- [Linkerd Identity](https://linkerd.io/2/reference/architecture/#identity)
- [cert-manager Integration](https://linkerd.io/2/tasks/automatically-rotating-control-plane-tls-credentials/)
