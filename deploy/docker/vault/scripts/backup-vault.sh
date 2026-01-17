#!/bin/bash
#
# Vault Backup Script
# Creates a complete backup of Vault secrets and configuration
#
# Usage:
#   ./backup-vault.sh                    # Backup to default directory
#   ./backup-vault.sh /path/to/backup    # Backup to specific directory
#   ENCRYPT=true ./backup-vault.sh       # Create encrypted backup
#

set -e

# Configuration
BACKUP_DIR="${1:-./vault-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
ENCRYPT="${ENCRYPT:-false}"
VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║              HashiCorp Vault Backup                        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}[1/6]${NC} Checking prerequisites..."

if ! command -v vault &> /dev/null; then
    echo -e "${RED}✗ Vault CLI not found. Please install vault.${NC}"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${RED}✗ jq not found. Please install jq.${NC}"
    exit 1
fi

if [ -z "$VAULT_TOKEN" ]; then
    echo -e "${RED}✗ VAULT_TOKEN not set${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites checked${NC}"

# Check Vault connectivity
echo -e "${YELLOW}[2/6]${NC} Checking Vault connection..."
if ! vault status > /dev/null 2>&1; then
    echo -e "${RED}✗ Cannot connect to Vault at $VAULT_ADDR${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Connected to Vault at $VAULT_ADDR${NC}"

# Create backup directory
echo -e "${YELLOW}[3/6]${NC} Creating backup directory..."
mkdir -p "$BACKUP_DIR"
EXPORT_DIR="${BACKUP_DIR}/vault-backup-${TIMESTAMP}"
mkdir -p "${EXPORT_DIR}/secrets"
mkdir -p "${EXPORT_DIR}/policies"
echo -e "${GREEN}✓ Created $EXPORT_DIR${NC}"

# Backup secrets
echo -e "${YELLOW}[4/6]${NC} Backing up secrets..."
SECRET_COUNT=0

# Get all secret paths
SECRET_PATHS=$(vault kv list -format=json secret/ 2>/dev/null | jq -r '.[]' 2>/dev/null || echo "")

if [ -n "$SECRET_PATHS" ]; then
    for path in $SECRET_PATHS; do
        echo "  Backing up: secret/$path"
        vault kv get -format=json "secret/$path" > "${EXPORT_DIR}/secrets/${path}.json" 2>/dev/null || true
        ((SECRET_COUNT++)) || true
    done
fi

echo -e "${GREEN}✓ Backed up $SECRET_COUNT secrets${NC}"

# Backup policies
echo -e "${YELLOW}[5/6]${NC} Backing up policies..."
POLICY_COUNT=0

for policy in $(vault policy list 2>/dev/null); do
    if [ "$policy" != "root" ] && [ "$policy" != "default" ]; then
        echo "  Backing up policy: $policy"
        vault policy read "$policy" > "${EXPORT_DIR}/policies/${policy}.hcl" 2>/dev/null || true
        ((POLICY_COUNT++)) || true
    fi
done

echo -e "${GREEN}✓ Backed up $POLICY_COUNT policies${NC}"

# Create metadata file
echo -e "${YELLOW}[6/6]${NC} Creating backup metadata..."
cat > "${EXPORT_DIR}/metadata.json" <<EOF
{
  "timestamp": "$(date -Iseconds)",
  "vault_addr": "$VAULT_ADDR",
  "secret_count": $SECRET_COUNT,
  "policy_count": $POLICY_COUNT,
  "backup_type": "full",
  "encrypted": $ENCRYPT
}
EOF

# Create archive
echo "Creating archive..."
ARCHIVE_NAME="vault-backup-${TIMESTAMP}.tar.gz"
tar -czf "${BACKUP_DIR}/${ARCHIVE_NAME}" -C "$BACKUP_DIR" "vault-backup-${TIMESTAMP}"

# Calculate checksum
sha256sum "${BACKUP_DIR}/${ARCHIVE_NAME}" > "${BACKUP_DIR}/${ARCHIVE_NAME}.sha256"

# Encrypt if requested
if [ "$ENCRYPT" = "true" ]; then
    echo "Encrypting backup..."
    if command -v gpg &> /dev/null; then
        gpg --symmetric --cipher-algo AES256 "${BACKUP_DIR}/${ARCHIVE_NAME}"
        rm "${BACKUP_DIR}/${ARCHIVE_NAME}"
        ARCHIVE_NAME="${ARCHIVE_NAME}.gpg"
    else
        echo -e "${YELLOW}⚠ gpg not found, skipping encryption${NC}"
    fi
fi

# Cleanup temporary directory
rm -rf "$EXPORT_DIR"

# Summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                  Backup Complete!                          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Backup Details:${NC}"
echo -e "  📁 File: ${BLUE}${BACKUP_DIR}/${ARCHIVE_NAME}${NC}"
echo -e "  🔢 Secrets: ${SECRET_COUNT}"
echo -e "  📜 Policies: ${POLICY_COUNT}"
echo -e "  🔒 Encrypted: ${ENCRYPT}"
echo -e "  📊 Checksum: ${BLUE}${BACKUP_DIR}/${ARCHIVE_NAME%.gpg}.sha256${NC}"
echo ""
echo -e "${YELLOW}To restore:${NC}"
echo -e "  ${BLUE}./restore-vault.sh ${BACKUP_DIR}/${ARCHIVE_NAME}${NC}"
echo ""
