#!/bin/bash
# Database Secrets Engine Setup Script
# Requirements: 1.2 - Dynamic database credentials with 1h TTL

set -e

VAULT_ADDR="${VAULT_ADDR:-https://vault.vault.svc:8200}"

echo "Setting up Database secrets engine..."

# Enable database secrets engine
vault secrets enable -path=database/auth-platform database 2>/dev/null || echo "Database engine already enabled"

# Configure PostgreSQL connection
echo "Configuring PostgreSQL connection..."
vault write database/auth-platform/config/auth-platform-postgres \
    plugin_name=postgresql-database-plugin \
    allowed_roles="auth-platform-readonly,auth-platform-readwrite" \
    connection_url="postgresql://{{username}}:{{password}}@postgres.auth-platform.svc:5432/auth_platform?sslmode=require" \
    username="vault-admin" \
    password="${POSTGRES_VAULT_PASSWORD}"

# Create readonly role - Requirements 1.2 (1h TTL)
echo "Creating readonly role..."
vault write database/auth-platform/roles/auth-platform-readonly \
    db_name=auth-platform-postgres \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    revocation_statements="DROP ROLE IF EXISTS \"{{name}}\";" \
    default_ttl="1h" \
    max_ttl="24h"

# Create readwrite role - Requirements 1.2 (1h TTL)
echo "Creating readwrite role..."
vault write database/auth-platform/roles/auth-platform-readwrite \
    db_name=auth-platform-postgres \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    revocation_statements="DROP ROLE IF EXISTS \"{{name}}\";" \
    default_ttl="1h" \
    max_ttl="24h"

echo "PostgreSQL database engine setup complete!"
