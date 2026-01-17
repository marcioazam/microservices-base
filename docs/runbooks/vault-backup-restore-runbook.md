# Vault Backup and Restore Runbook

This runbook provides procedures for backing up and restoring HashiCorp Vault data.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Backup Procedures](#backup-procedures)
4. [Restore Procedures](#restore-procedures)
5. [Automated Backups](#automated-backups)
6. [Disaster Recovery](#disaster-recovery)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)

## Overview

### What Gets Backed Up

- **Secrets**: All KV secrets stored in Vault
- **Policies**: Access control policies
- **Authentication methods**: Configured auth backends
- **Audit logs**: (if configured)

### Backup Methods

| Method | Use Case | Pros | Cons |
|--------|----------|------|------|
| Raft Snapshots | Production HA clusters | Complete, consistent | Requires Raft storage |
| KV Export | Development, simple setups | Easy, portable | Only KV data |
| Consul Snapshots | Consul backend | Complete state | Requires Consul |
| File System | Single-node dev | Simple | Not consistent |

## Prerequisites

### Tools Required

```bash
# Vault CLI
vault version

# jq for JSON processing
jq --version

# AWS CLI (for S3 backups)
aws --version
```

### Access Required

```bash
# Root token or backup policy
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_TOKEN="<backup-token>"

# Verify access
vault status
vault token lookup
```

### Backup Policy (Recommended)

```hcl
# vault-backup-policy.hcl
path "sys/storage/raft/snapshot" {
  capabilities = ["read"]
}

path "secret/*" {
  capabilities = ["read", "list"]
}

path "sys/policies/*" {
  capabilities = ["read", "list"]
}
```

## Backup Procedures

### Method 1: Raft Snapshot (Recommended for Production)

```bash
#!/bin/bash
# backup-vault-raft.sh

set -e

BACKUP_DIR="${BACKUP_DIR:-./vault-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
SNAPSHOT_FILE="${BACKUP_DIR}/vault-snapshot-${TIMESTAMP}.snap"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Take Raft snapshot
echo "Taking Raft snapshot..."
vault operator raft snapshot save "$SNAPSHOT_FILE"

# Verify snapshot
echo "Verifying snapshot..."
vault operator raft snapshot inspect "$SNAPSHOT_FILE"

# Compress
echo "Compressing..."
gzip "$SNAPSHOT_FILE"

# Calculate checksum
sha256sum "${SNAPSHOT_FILE}.gz" > "${SNAPSHOT_FILE}.gz.sha256"

echo "Backup completed: ${SNAPSHOT_FILE}.gz"
echo "Checksum: $(cat ${SNAPSHOT_FILE}.gz.sha256)"
```

### Method 2: KV Secrets Export

```bash
#!/bin/bash
# backup-vault-kv.sh

set -e

BACKUP_DIR="${BACKUP_DIR:-./vault-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/vault-kv-${TIMESTAMP}.json"

mkdir -p "$BACKUP_DIR"

echo "Exporting KV secrets..."

# Get all secret paths
PATHS=$(vault kv list -format=json secret/ 2>/dev/null | jq -r '.[]' || echo "")

# Export each secret
echo "{" > "$BACKUP_FILE"
echo '  "timestamp": "'$(date -Iseconds)'",' >> "$BACKUP_FILE"
echo '  "vault_addr": "'$VAULT_ADDR'",' >> "$BACKUP_FILE"
echo '  "secrets": {' >> "$BACKUP_FILE"

FIRST=true
for path in $PATHS; do
    if [ "$FIRST" = true ]; then
        FIRST=false
    else
        echo "," >> "$BACKUP_FILE"
    fi

    echo "  Backing up: secret/$path"
    SECRET_DATA=$(vault kv get -format=json "secret/$path" 2>/dev/null | jq '.data.data')
    echo -n "    \"$path\": $SECRET_DATA" >> "$BACKUP_FILE"
done

echo "" >> "$BACKUP_FILE"
echo "  }" >> "$BACKUP_FILE"
echo "}" >> "$BACKUP_FILE"

# Encrypt backup (recommended)
echo "Encrypting backup..."
gpg --symmetric --cipher-algo AES256 "$BACKUP_FILE"
rm "$BACKUP_FILE"

echo "Backup completed: ${BACKUP_FILE}.gpg"
```

### Method 3: Full Export (Development)

```bash
#!/bin/bash
# backup-vault-full.sh

set -e

BACKUP_DIR="${BACKUP_DIR:-./vault-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
EXPORT_DIR="${BACKUP_DIR}/vault-export-${TIMESTAMP}"

mkdir -p "$EXPORT_DIR"

# Export secrets
echo "Exporting secrets..."
vault kv get -format=json secret/sms-service > "${EXPORT_DIR}/sms-service.json" 2>/dev/null || true
vault kv get -format=json secret/common > "${EXPORT_DIR}/common.json" 2>/dev/null || true

# Export policies
echo "Exporting policies..."
mkdir -p "${EXPORT_DIR}/policies"
for policy in $(vault policy list); do
    if [ "$policy" != "root" ] && [ "$policy" != "default" ]; then
        vault policy read "$policy" > "${EXPORT_DIR}/policies/${policy}.hcl"
    fi
done

# Export auth methods
echo "Exporting auth configuration..."
vault auth list -format=json > "${EXPORT_DIR}/auth-methods.json"

# Create archive
echo "Creating archive..."
tar -czvf "${EXPORT_DIR}.tar.gz" -C "$BACKUP_DIR" "vault-export-${TIMESTAMP}"
rm -rf "$EXPORT_DIR"

# Encrypt
gpg --symmetric --cipher-algo AES256 "${EXPORT_DIR}.tar.gz"
rm "${EXPORT_DIR}.tar.gz"

echo "Backup completed: ${EXPORT_DIR}.tar.gz.gpg"
```

## Restore Procedures

### Method 1: Raft Snapshot Restore

```bash
#!/bin/bash
# restore-vault-raft.sh

set -e

SNAPSHOT_FILE="$1"

if [ -z "$SNAPSHOT_FILE" ]; then
    echo "Usage: $0 <snapshot-file>"
    exit 1
fi

# Decompress if needed
if [[ "$SNAPSHOT_FILE" == *.gz ]]; then
    echo "Decompressing snapshot..."
    gunzip -k "$SNAPSHOT_FILE"
    SNAPSHOT_FILE="${SNAPSHOT_FILE%.gz}"
fi

# Verify checksum if available
if [ -f "${SNAPSHOT_FILE}.sha256" ]; then
    echo "Verifying checksum..."
    sha256sum -c "${SNAPSHOT_FILE}.sha256"
fi

# Inspect snapshot
echo "Inspecting snapshot..."
vault operator raft snapshot inspect "$SNAPSHOT_FILE"

# Confirm restore
read -p "This will OVERWRITE current Vault data. Continue? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

# Restore snapshot
echo "Restoring snapshot..."
vault operator raft snapshot restore -force "$SNAPSHOT_FILE"

echo "Restore completed. Vault will restart."
```

### Method 2: KV Secrets Restore

```bash
#!/bin/bash
# restore-vault-kv.sh

set -e

BACKUP_FILE="$1"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file.json.gpg>"
    exit 1
fi

# Decrypt if encrypted
if [[ "$BACKUP_FILE" == *.gpg ]]; then
    echo "Decrypting backup..."
    gpg --decrypt "$BACKUP_FILE" > "${BACKUP_FILE%.gpg}"
    BACKUP_FILE="${BACKUP_FILE%.gpg}"
fi

# Parse and restore secrets
echo "Restoring secrets..."

# Get list of paths
PATHS=$(jq -r '.secrets | keys[]' "$BACKUP_FILE")

for path in $PATHS; do
    echo "Restoring: secret/$path"
    SECRET_DATA=$(jq -c ".secrets[\"$path\"]" "$BACKUP_FILE")
    echo "$SECRET_DATA" | vault kv put "secret/$path" -
done

# Clean up decrypted file
rm -f "${BACKUP_FILE}"

echo "Restore completed."
```

### Method 3: Full Restore (Development)

```bash
#!/bin/bash
# restore-vault-full.sh

set -e

ARCHIVE="$1"

if [ -z "$ARCHIVE" ]; then
    echo "Usage: $0 <archive.tar.gz.gpg>"
    exit 1
fi

# Decrypt
echo "Decrypting archive..."
gpg --decrypt "$ARCHIVE" > "${ARCHIVE%.gpg}"
ARCHIVE="${ARCHIVE%.gpg}"

# Extract
echo "Extracting archive..."
EXPORT_DIR=$(tar -tzf "$ARCHIVE" | head -1 | cut -f1 -d"/")
tar -xzf "$ARCHIVE"

# Restore policies first
echo "Restoring policies..."
for policy_file in "${EXPORT_DIR}/policies/"*.hcl; do
    if [ -f "$policy_file" ]; then
        policy_name=$(basename "$policy_file" .hcl)
        echo "  Restoring policy: $policy_name"
        vault policy write "$policy_name" "$policy_file"
    fi
done

# Restore secrets
echo "Restoring secrets..."
for secret_file in "${EXPORT_DIR}/"*.json; do
    if [ -f "$secret_file" ] && [ "$(basename $secret_file)" != "auth-methods.json" ]; then
        secret_name=$(basename "$secret_file" .json)
        echo "  Restoring secret: $secret_name"
        SECRET_DATA=$(jq -c '.data.data' "$secret_file")
        echo "$SECRET_DATA" | vault kv put "secret/$secret_name" -
    fi
done

# Clean up
rm -rf "$EXPORT_DIR" "$ARCHIVE"

echo "Restore completed."
```

## Automated Backups

### Kubernetes CronJob

```yaml
# vault-backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: vault-backup
  namespace: vault
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: vault-backup
          containers:
          - name: backup
            image: hashicorp/vault:1.15.0
            command:
            - /bin/sh
            - -c
            - |
              set -e
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)
              vault operator raft snapshot save /backup/vault-${TIMESTAMP}.snap
              gzip /backup/vault-${TIMESTAMP}.snap
              # Upload to S3
              aws s3 cp /backup/vault-${TIMESTAMP}.snap.gz s3://${BACKUP_BUCKET}/vault/
              # Clean old local backups
              find /backup -name "*.snap.gz" -mtime +7 -delete
            env:
            - name: VAULT_ADDR
              value: "https://vault:8200"
            - name: VAULT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: vault-backup-token
                  key: token
            - name: BACKUP_BUCKET
              value: "my-backup-bucket"
            volumeMounts:
            - name: backup-volume
              mountPath: /backup
          volumes:
          - name: backup-volume
            persistentVolumeClaim:
              claimName: vault-backup-pvc
          restartPolicy: OnFailure
```

### PowerShell Scheduled Task (Windows)

```powershell
# Schedule-VaultBackup.ps1

$Action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument @"
-ExecutionPolicy Bypass -File C:\scripts\Backup-Vault.ps1
"@

$Trigger = New-ScheduledTaskTrigger -Daily -At "02:00"
$Settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -DontStopOnIdleEnd

Register-ScheduledTask -TaskName "VaultBackup" -Action $Action -Trigger $Trigger -Settings $Settings -User "SYSTEM"
```

## Disaster Recovery

### Complete Cluster Loss

1. **Deploy new Vault cluster**
   ```bash
   # Deploy infrastructure
   terraform apply -target=module.vault

   # Initialize new cluster
   vault operator init -key-shares=5 -key-threshold=3
   ```

2. **Restore from backup**
   ```bash
   # Unseal the cluster
   vault operator unseal <key1>
   vault operator unseal <key2>
   vault operator unseal <key3>

   # Restore snapshot
   vault operator raft snapshot restore -force /backup/latest.snap
   ```

3. **Verify restoration**
   ```bash
   vault status
   vault kv list secret/
   vault policy list
   ```

### Single Node Recovery

```bash
# Stop Vault
systemctl stop vault

# Restore data directory
rm -rf /opt/vault/data/*
tar -xzf /backup/vault-data-backup.tar.gz -C /opt/vault/data/

# Start Vault
systemctl start vault

# Unseal
vault operator unseal
```

## Verification

### Backup Verification Script

```bash
#!/bin/bash
# verify-vault-backup.sh

set -e

SNAPSHOT_FILE="$1"

if [ -z "$SNAPSHOT_FILE" ]; then
    echo "Usage: $0 <snapshot-file>"
    exit 1
fi

echo "=== Vault Backup Verification ==="
echo ""

# Check file exists and size
echo "1. File Check"
if [ -f "$SNAPSHOT_FILE" ]; then
    SIZE=$(du -h "$SNAPSHOT_FILE" | cut -f1)
    echo "   ✓ File exists: $SNAPSHOT_FILE ($SIZE)"
else
    echo "   ✗ File not found: $SNAPSHOT_FILE"
    exit 1
fi

# Verify checksum
echo ""
echo "2. Checksum Verification"
if [ -f "${SNAPSHOT_FILE}.sha256" ]; then
    if sha256sum -c "${SNAPSHOT_FILE}.sha256" > /dev/null 2>&1; then
        echo "   ✓ Checksum valid"
    else
        echo "   ✗ Checksum mismatch!"
        exit 1
    fi
else
    echo "   ⚠ No checksum file found"
fi

# Inspect snapshot
echo ""
echo "3. Snapshot Inspection"
if vault operator raft snapshot inspect "$SNAPSHOT_FILE" > /dev/null 2>&1; then
    echo "   ✓ Snapshot is valid"
    vault operator raft snapshot inspect "$SNAPSHOT_FILE" | grep -E "^(ID|Size|Index|Term):"
else
    echo "   ✗ Snapshot inspection failed"
    exit 1
fi

echo ""
echo "=== Verification Complete ==="
```

### Restore Test (Non-Production)

```bash
#!/bin/bash
# test-vault-restore.sh

# Start temporary Vault for testing
docker run -d --name vault-test \
    -p 8201:8200 \
    -e 'VAULT_DEV_ROOT_TOKEN_ID=test-root-token' \
    hashicorp/vault:1.15.0

sleep 5

export VAULT_ADDR="http://localhost:8201"
export VAULT_TOKEN="test-root-token"

# Attempt restore
./restore-vault-kv.sh /backup/latest.json.gpg

# Verify secrets
vault kv list secret/
vault kv get secret/sms-service

# Cleanup
docker rm -f vault-test
```

## Troubleshooting

### Common Issues

#### Backup Fails with Permission Denied

```bash
# Check token permissions
vault token capabilities sys/storage/raft/snapshot

# Required: ["read"]
```

#### Restore Fails - Cluster Not Leader

```bash
# Find leader
vault operator raft list-peers

# Run restore on leader node
VAULT_ADDR="https://vault-leader:8200" vault operator raft snapshot restore ...
```

#### Snapshot Too Large

```bash
# Check current size
vault operator raft snapshot save /tmp/test.snap
du -h /tmp/test.snap

# Enable compression
gzip -9 /tmp/test.snap
```

#### Encrypted Backup - Wrong Passphrase

```bash
# Decrypt with correct passphrase
gpg --decrypt --batch --passphrase-file /secure/passphrase backup.gpg
```

### Recovery Contacts

| Role | Contact | Escalation |
|------|---------|------------|
| Vault Admin | vault-team@example.com | PagerDuty |
| Security Team | security@example.com | Slack #security |
| On-Call | oncall@example.com | PagerDuty |

### SLA Targets

| Metric | Target |
|--------|--------|
| RPO (Recovery Point Objective) | 24 hours |
| RTO (Recovery Time Objective) | 4 hours |
| Backup Retention | 30 days |
| Backup Verification | Weekly |
