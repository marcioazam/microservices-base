#!/bin/bash
#
# Vault Restore Script
# Restores Vault secrets and configuration from backup
#
# Usage:
#   ./restore-vault.sh backup-file.tar.gz
#   ./restore-vault.sh backup-file.tar.gz.gpg
#

set -e

# Configuration
BACKUP_FILE="$1"
VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
TEMP_DIR=$(mktemp -d)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║              HashiCorp Vault Restore                       ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Validate input
if [ -z "$BACKUP_FILE" ]; then
    echo -e "${RED}Usage: $0 <backup-file>${NC}"
    echo ""
    echo "Examples:"
    echo "  $0 vault-backup-20240115.tar.gz"
    echo "  $0 vault-backup-20240115.tar.gz.gpg"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}✗ Backup file not found: $BACKUP_FILE${NC}"
    exit 1
fi

# Check prerequisites
echo -e "${YELLOW}[1/6]${NC} Checking prerequisites..."

if ! command -v vault &> /dev/null; then
    echo -e "${RED}✗ Vault CLI not found${NC}"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${RED}✗ jq not found${NC}"
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
echo -e "${GREEN}✓ Connected to Vault${NC}"

# Decrypt if encrypted
echo -e "${YELLOW}[3/6]${NC} Preparing backup file..."
ARCHIVE_FILE="$BACKUP_FILE"

if [[ "$BACKUP_FILE" == *.gpg ]]; then
    echo "  Decrypting backup..."
    if ! command -v gpg &> /dev/null; then
        echo -e "${RED}✗ gpg not found, cannot decrypt${NC}"
        exit 1
    fi
    ARCHIVE_FILE="${TEMP_DIR}/backup.tar.gz"
    gpg --decrypt "$BACKUP_FILE" > "$ARCHIVE_FILE"
    echo -e "${GREEN}  ✓ Decrypted${NC}"
fi

# Verify checksum if available
CHECKSUM_FILE="${BACKUP_FILE%.gpg}.sha256"
if [ -f "$CHECKSUM_FILE" ]; then
    echo "  Verifying checksum..."
    ORIGINAL_ARCHIVE="${BACKUP_FILE%.gpg}"
    if sha256sum -c "$CHECKSUM_FILE" > /dev/null 2>&1; then
        echo -e "${GREEN}  ✓ Checksum valid${NC}"
    else
        echo -e "${YELLOW}  ⚠ Checksum mismatch (file may have been modified)${NC}"
        read -p "Continue anyway? (yes/no): " CONTINUE
        if [ "$CONTINUE" != "yes" ]; then
            exit 1
        fi
    fi
fi

echo -e "${GREEN}✓ Backup file ready${NC}"

# Extract archive
echo -e "${YELLOW}[4/6]${NC} Extracting backup..."
tar -xzf "$ARCHIVE_FILE" -C "$TEMP_DIR"
BACKUP_DIR=$(ls -d ${TEMP_DIR}/vault-backup-* 2>/dev/null | head -1)

if [ -z "$BACKUP_DIR" ] || [ ! -d "$BACKUP_DIR" ]; then
    echo -e "${RED}✗ Invalid backup archive structure${NC}"
    exit 1
fi

# Show metadata
if [ -f "${BACKUP_DIR}/metadata.json" ]; then
    echo ""
    echo "  Backup Information:"
    echo "  - Timestamp: $(jq -r '.timestamp' ${BACKUP_DIR}/metadata.json)"
    echo "  - Secrets: $(jq -r '.secret_count' ${BACKUP_DIR}/metadata.json)"
    echo "  - Policies: $(jq -r '.policy_count' ${BACKUP_DIR}/metadata.json)"
fi

echo -e "${GREEN}✓ Extracted${NC}"

# Confirm restore
echo ""
echo -e "${YELLOW}⚠  WARNING: This will overwrite existing secrets and policies!${NC}"
read -p "Are you sure you want to restore? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

# Restore policies
echo -e "${YELLOW}[5/6]${NC} Restoring policies..."
POLICY_COUNT=0

if [ -d "${BACKUP_DIR}/policies" ]; then
    for policy_file in "${BACKUP_DIR}/policies/"*.hcl; do
        if [ -f "$policy_file" ]; then
            policy_name=$(basename "$policy_file" .hcl)
            echo "  Restoring policy: $policy_name"
            vault policy write "$policy_name" "$policy_file"
            ((POLICY_COUNT++)) || true
        fi
    done
fi

echo -e "${GREEN}✓ Restored $POLICY_COUNT policies${NC}"

# Restore secrets
echo -e "${YELLOW}[6/6]${NC} Restoring secrets..."
SECRET_COUNT=0

if [ -d "${BACKUP_DIR}/secrets" ]; then
    for secret_file in "${BACKUP_DIR}/secrets/"*.json; do
        if [ -f "$secret_file" ]; then
            secret_name=$(basename "$secret_file" .json)
            echo "  Restoring secret: $secret_name"

            # Extract just the data
            SECRET_DATA=$(jq -c '.data.data' "$secret_file")

            # Write to Vault
            echo "$SECRET_DATA" | vault kv put "secret/$secret_name" -
            ((SECRET_COUNT++)) || true
        fi
    done
fi

echo -e "${GREEN}✓ Restored $SECRET_COUNT secrets${NC}"

# Summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                  Restore Complete!                         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Restore Summary:${NC}"
echo -e "  📜 Policies restored: ${POLICY_COUNT}"
echo -e "  🔐 Secrets restored: ${SECRET_COUNT}"
echo ""
echo -e "${YELLOW}Verification:${NC}"
echo -e "  ${BLUE}vault kv list secret/${NC}"
echo -e "  ${BLUE}vault policy list${NC}"
echo ""
