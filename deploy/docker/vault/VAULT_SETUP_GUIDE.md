# HashiCorp Vault - Complete Setup Guide

## 📚 Table of Contents

1. [Introduction](#introduction)
2. [Quick Start](#quick-start)
3. [Architecture](#architecture)
4. [Installation](#installation)
5. [Configuration](#configuration)
6. [Usage](#usage)
7. [Security Best Practices](#security-best-practices)
8. [Troubleshooting](#troubleshooting)
9. [Production Deployment](#production-deployment)

---

## 🎯 Introduction

This guide covers the complete setup and usage of HashiCorp Vault for secrets management in the microservices platform.

**What is Vault?**
- Secure secrets storage (API keys, passwords, certificates)
- Dynamic secrets generation
- Data encryption
- Access control and audit logging
- Token-based authentication

**Why use Vault?**
- ✅ **Security**: Encrypted secrets at rest and in transit
- ✅ **Centralization**: One place for all secrets
- ✅ **Audit**: Complete audit trail of secret access
- ✅ **Rotation**: Automatic secret rotation
- ✅ **Compliance**: HIPAA, PCI-DSS, GDPR ready

---

## 🚀 Quick Start

### Step 1: Start Vault

```bash
# Navigate to Vault directory
cd deploy/docker/vault

# Start Vault server
docker-compose -f docker-compose.vault.yml up -d

# Check status
docker-compose -f docker-compose.vault.yml ps
```

**Expected Output:**
```
NAME            STATUS   PORTS
vault-server    Up       0.0.0.0:8200->8200/tcp
```

### Step 2: Initialize Vault

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Run initialization script
./scripts/init-vault.sh
```

**Expected Output:**
```
╔════════════════════════════════════════════════════════════╗
║   HashiCorp Vault - Secrets Management Initialization    ║
╚════════════════════════════════════════════════════════════╝

[1/8] Checking Vault connection...
✓ Vault is running at http://localhost:8200

[2/8] Waiting for Vault to be ready...
✓ Vault is ready

... (additional setup steps)

╔════════════════════════════════════════════════════════════╗
║                  Setup Complete! 🎉                        ║
╚════════════════════════════════════════════════════════════╝

Vault Configuration:
  📍 URL: http://localhost:8200
  🌐 UI:  http://localhost:8200/ui
  🔑 Root Token: root
```

### Step 3: Test Vault

```bash
# Test connection and secret access
./scripts/test-vault.sh
```

### Step 4: Use Vault in Your Application

```python
# Load configuration from .env.vault
source .env.vault

# Python example
from src.config.vault_settings import get_settings_with_vault

settings = get_settings_with_vault()
jwt_secret = settings.jwt_secret_key  # Loaded from Vault!
```

---

## 🏗️ Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Applications                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ SMS Svc  │  │ Auth Svc │  │ Email Svc│             │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
│       │             │             │                     │
│       └─────────────┼─────────────┘                     │
│                     │                                   │
│              ┌──────▼──────┐                            │
│              │  Vault API  │                            │
│              │  :8200      │                            │
│              └──────┬──────┘                            │
│                     │                                   │
│              ┌──────▼──────┐                            │
│              │   Storage   │                            │
│              │  (File/DB)  │                            │
│              └─────────────┘                            │
└─────────────────────────────────────────────────────────┘
```

### Secret Flow

```
1. Application Startup
   ↓
2. Load VAULT_ADDR and VAULT_TOKEN from environment
   ↓
3. Initialize VaultClient
   ↓
4. Authenticate with Vault
   ↓
5. Read secrets from configured path
   ↓
6. Merge secrets with environment config
   ↓
7. Application ready with secure secrets
```

### Directory Structure

```
deploy/docker/vault/
├── docker-compose.vault.yml    # Vault server configuration
├── config/
│   └── vault.hcl              # Production configuration
├── policies/
│   └── sms-service-policy.hcl # Access control policies
├── scripts/
│   ├── init-vault.sh          # Initialization script
│   └── test-vault.sh          # Testing script
└── VAULT_SETUP_GUIDE.md       # This guide
```

---

## 💾 Installation

### Prerequisites

- Docker & Docker Compose
- Python 3.11+ (for applications using Vault)
- Bash shell (for scripts)

### Install Vault CLI (Optional)

#### Linux
```bash
wget https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_linux_amd64.zip
unzip vault_1.15.0_linux_amd64.zip
sudo mv vault /usr/local/bin/
vault --version
```

#### macOS
```bash
brew install vault
vault --version
```

#### Windows
```powershell
choco install vault
vault --version
```

### Install Python Dependencies

```bash
cd services/sms-service
pip install hvac  # HashiCorp Vault Python client
```

---

## ⚙️ Configuration

### Environment Variables

Create `.env.vault` file (generated by init script):

```bash
# Vault Configuration
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=your_service_token_here
VAULT_NAMESPACE=
VAULT_SKIP_VERIFY=false

# SMS Service Configuration
SMS_SERVICE_VAULT_PATH=secret/sms-service
```

### Application Integration

**Option 1: Automatic (Recommended)**

```python
# src/config/vault_settings.py automatically loads secrets
from src.config.vault_settings import get_settings_with_vault

settings = get_settings_with_vault()
# Secrets are loaded from Vault automatically
```

**Option 2: Manual**

```python
from src.shared.vault_client import get_vault_client

vault = get_vault_client()
jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")
```

### Docker Compose Integration

```yaml
# docker-compose.yml
services:
  sms-service:
    environment:
      - VAULT_ADDR=http://vault:8200
      - VAULT_TOKEN=${VAULT_TOKEN}  # From .env file
      - VAULT_ENABLED=true
    depends_on:
      - vault
```

---

## 📖 Usage

### 1. Viewing Secrets

```bash
# Using Vault CLI
vault kv get secret/sms-service

# View specific key
vault kv get -field=jwt_secret_key secret/sms-service

# View as JSON
vault kv get -format=json secret/sms-service
```

### 2. Adding/Updating Secrets

```bash
# Add new secret
vault kv put secret/sms-service new_key="new_value"

# Update existing secret
vault kv put secret/sms-service jwt_secret_key="updated_secret"

# Add multiple secrets at once
vault kv put secret/sms-service \
    jwt_secret_key="secret1" \
    api_key="secret2" \
    webhook_secret="secret3"
```

### 3. Deleting Secrets

```bash
# Delete specific version
vault kv delete secret/sms-service

# Delete all versions (metadata)
vault kv metadata delete secret/sms-service

# Undelete (restore)
vault kv undelete -versions=1 secret/sms-service
```

### 4. Python Client Usage

```python
from src.shared.vault_client import VaultClient

# Initialize client
vault = VaultClient()

# Read single secret
jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")

# Read all secrets
all_secrets = vault.get_secret("sms-service")

# Read multiple secrets
secrets = vault.get_secrets_batch(
    "sms-service",
    ["jwt_secret_key", "twilio_auth_token"]
)

# Write secrets
vault.put_secret("sms-service/temp", {
    "temp_key": "temp_value"
})

# Check authentication
if vault.is_authenticated():
    print("Connected to Vault!")

# Get token TTL
ttl = vault.get_token_ttl()
print(f"Token expires in {ttl} seconds")

# Renew token
vault.renew_token()
```

### 5. Vault UI

Access the web interface:

```
URL: http://localhost:8200/ui
Token: root (or your service token)
```

**UI Features:**
- Browse secrets
- Create/update/delete secrets
- View audit logs
- Manage policies
- Monitor token leases

---

## 🔒 Security Best Practices

### 1. Token Management

```bash
# Create limited-privilege tokens for services
vault token create \
    -policy=sms-service \
    -ttl=720h \
    -renewable=true \
    -display-name="sms-service-prod"

# Revoke tokens when no longer needed
vault token revoke <token_id>

# List active tokens
vault list auth/token/accessors
```

### 2. Access Control Policies

```hcl
# policies/sms-service-policy.hcl
path "secret/data/sms-service/*" {
  capabilities = ["read", "list"]
}

path "secret/data/common/*" {
  capabilities = ["read", "list"]
}
```

Apply policy:
```bash
vault policy write sms-service /vault/policies/sms-service-policy.hcl
```

### 3. Audit Logging

```bash
# Enable audit logging
vault audit enable file file_path=/vault/logs/audit.log

# View audit logs
docker exec vault-server cat /vault/logs/audit.log
```

### 4. Secret Rotation

```bash
# Rotate JWT secret
NEW_SECRET=$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')
vault kv put secret/sms-service jwt_secret_key="$NEW_SECRET"

# Application should reload secrets periodically
# or use Vault's secret leasing feature
```

### 5. Environment Separation

```bash
# Development
vault kv put secret/dev/sms-service ...

# Staging
vault kv put secret/staging/sms-service ...

# Production
vault kv put secret/prod/sms-service ...
```

---

## 🐛 Troubleshooting

### Issue 1: Cannot Connect to Vault

**Symptoms:**
```
Error: Get "http://localhost:8200/v1/sys/health": dial tcp connect: connection refused
```

**Solutions:**
```bash
# Check if Vault is running
docker ps | grep vault

# Check Vault logs
docker logs vault-server

# Restart Vault
docker-compose -f docker-compose.vault.yml restart

# Check network connectivity
curl http://localhost:8200/v1/sys/health
```

### Issue 2: Authentication Failed

**Symptoms:**
```
VaultError: Failed to authenticate with Vault
```

**Solutions:**
```bash
# Verify token is valid
vault token lookup

# Check token in environment
echo $VAULT_TOKEN

# Re-create token
vault token create -policy=sms-service
```

### Issue 3: Secret Not Found

**Symptoms:**
```
InvalidPath: Secret path not found: sms-service
```

**Solutions:**
```bash
# List available secrets
vault kv list secret/

# Check exact path
vault kv get secret/sms-service

# Re-run initialization
./scripts/init-vault.sh
```

### Issue 4: Permission Denied

**Symptoms:**
```
Error: permission denied
```

**Solutions:**
```bash
# Check token policies
vault token lookup

# Verify policy allows access
vault policy read sms-service

# Use root token temporarily for debugging
export VAULT_TOKEN=root
```

### Issue 5: Python hvac Not Installed

**Symptoms:**
```
ModuleNotFoundError: No module named 'hvac'
```

**Solution:**
```bash
pip install hvac
```

---

## 🚀 Production Deployment

### Production Checklist

- [ ] Use TLS/HTTPS for Vault API
- [ ] Configure persistent storage backend (PostgreSQL, Consul)
- [ ] Enable auto-unseal with cloud KMS
- [ ] Set up High Availability (HA) cluster
- [ ] Enable audit logging
- [ ] Configure proper authentication (LDAP, OIDC, AWS IAM)
- [ ] Implement secret rotation policy
- [ ] Set up monitoring and alerting
- [ ] Document disaster recovery procedures
- [ ] Regular backup of Vault data

### Production Configuration Example

```hcl
# vault.hcl
listener "tcp" {
  address       = "0.0.0.0:8200"
  tls_cert_file = "/vault/config/tls/vault.crt"
  tls_key_file  = "/vault/config/tls/vault.key"
}

storage "postgresql" {
  connection_url = "postgres://vault:password@postgres:5432/vault?sslmode=require"
}

seal "awskms" {
  region     = "us-east-1"
  kms_key_id = "your-kms-key-id"
}

api_addr = "https://vault.yourdomain.com:8200"
cluster_addr = "https://vault.yourdomain.com:8201"

ui = true
```

### Docker Compose Production

```yaml
version: '3.8'

services:
  vault:
    image: hashicorp/vault:1.15.0
    restart: always
    ports:
      - "8200:8200"
    environment:
      VAULT_ADDR: https://vault:8200
    volumes:
      - ./config/vault.hcl:/vault/config/vault.hcl:ro
      - ./tls:/vault/config/tls:ro
      - vault-data:/vault/data
    command: server -config=/vault/config/vault.hcl
    cap_add:
      - IPC_LOCK
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: vault
spec:
  serviceName: vault
  replicas: 3
  selector:
    matchLabels:
      app: vault
  template:
    metadata:
      labels:
        app: vault
    spec:
      containers:
      - name: vault
        image: hashicorp/vault:1.15.0
        ports:
        - containerPort: 8200
          name: api
        - containerPort: 8201
          name: cluster
        env:
        - name: VAULT_ADDR
          value: "https://vault:8200"
        volumeMounts:
        - name: config
          mountPath: /vault/config
        - name: data
          mountPath: /vault/data
```

---

## 📞 Support

### Documentation Links

- [Vault Official Documentation](https://www.vaultproject.io/docs)
- [Vault API Reference](https://www.vaultproject.io/api-docs)
- [hvac Python Client](https://hvac.readthedocs.io/)
- [Best Practices](https://learn.hashicorp.com/collections/vault/best-practices)

### Commands Reference

```bash
# Status
vault status

# Health check
curl http://localhost:8200/v1/sys/health

# Read secret
vault kv get secret/sms-service

# Write secret
vault kv put secret/sms-service key=value

# Delete secret
vault kv delete secret/sms-service

# List secrets
vault kv list secret/

# Token operations
vault token lookup
vault token renew
vault token revoke <token>

# Policy operations
vault policy list
vault policy read <policy-name>
vault policy write <policy-name> policy.hcl
```

---

## 🎓 Additional Resources

- **Tutorial**: [Learn Vault](https://learn.hashicorp.com/vault)
- **Video**: [Vault Getting Started](https://www.youtube.com/watch?v=VYfl-DpZ5wM)
- **Book**: [HashiCorp Vault: The Definitive Guide](https://www.vaultproject.io/docs/internals)
- **Community**: [Vault Forum](https://discuss.hashicorp.com/c/vault)

---

**Last Updated:** 2026-01-14
**Version:** 1.0.0
**Maintained by:** DevOps Team
