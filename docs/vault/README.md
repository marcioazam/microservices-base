# HashiCorp Vault Integration

## Overview

HashiCorp Vault provides centralized secrets management for the Auth Platform with:
- Dynamic database credentials with automatic rotation
- PKI certificate management for Linkerd
- Kubernetes authentication for zero-trust access
- Comprehensive audit logging

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Vault HA Cluster (3 nodes)                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ KV v2 Engine │  │ Database     │  │ PKI Engine   │          │
│  │              │  │ Engine       │  │              │          │
│  │ • JWT Keys   │  │ • PostgreSQL │  │ • Root CA    │          │
│  │ • API Keys   │  │ • Redis      │  │ • Issuer CA  │          │
│  │ • Configs    │  │              │  │ • Certs      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ K8s Auth     │  │ Transit      │  │ Audit Log    │          │
│  │ Method       │  │ Engine       │  │              │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Auth Platform Services                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ Auth Edge   │  │ Token       │  │ Session     │              │
│  │ + Agent     │  │ + Agent     │  │ + Agent     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration Reference

### Secrets Engines

| Engine | Path | Purpose |
|--------|------|---------|
| KV v2 | `secret/auth-platform/*` | Static secrets (JWT keys, configs) |
| Database | `database/auth-platform/*` | Dynamic PostgreSQL/Redis credentials |
| PKI | `pki/auth-platform-*` | Certificate management |
| Transit | `transit/auth-platform/*` | Encryption-as-a-service |

### Authentication Roles

| Role | Service Account | Policies |
|------|-----------------|----------|
| auth-edge-service | auth-edge-service | auth-edge-policy |
| token-service | token-service | token-service-policy |
| session-identity-core | session-identity-core | session-identity-policy |
| iam-policy-service | iam-policy-service | iam-policy-policy |
| mfa-service | mfa-service | mfa-service-policy |

### Dynamic Credentials TTL

| Credential Type | Default TTL | Max TTL |
|-----------------|-------------|---------|
| PostgreSQL readonly | 1 hour | 24 hours |
| PostgreSQL readwrite | 1 hour | 24 hours |
| Redis | 1 hour | 24 hours |

## Troubleshooting

### Common Issues

#### Service Cannot Authenticate
```bash
# Check service account exists
kubectl get sa -n auth-platform

# Verify Vault role configuration
vault read auth/kubernetes/role/auth-edge-service

# Check Vault logs
kubectl logs -n vault vault-0
```

#### Secret Not Found
```bash
# Verify secret path
vault kv get secret/auth-platform/jwt/signing-key

# Check policy permissions
vault policy read auth-edge-policy
```

#### Lease Renewal Failed
```bash
# Check lease status
vault lease lookup <lease_id>

# Force renewal
vault lease renew <lease_id>
```

### Health Checks

```bash
# Vault status
vault status

# Check seal status
vault operator seal-status

# Audit log verification
kubectl exec -n vault vault-0 -- cat /vault/audit/audit.log | tail -10
```

## Operations

### Secret Rotation

```bash
# Rotate JWT signing key
vault kv put secret/auth-platform/jwt/signing-key \
  algorithm="ES256" \
  key_id="key-2025-02" \
  private_key="..." \
  public_key="..."

# Rotate database root credentials
vault write -force database/auth-platform/rotate-root/auth-platform-postgres
```

### Backup and Recovery

```bash
# Create snapshot
vault operator raft snapshot save backup.snap

# Restore from snapshot
vault operator raft snapshot restore backup.snap
```
