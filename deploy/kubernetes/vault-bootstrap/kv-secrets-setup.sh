#!/bin/bash
# KV v2 Secrets Engine Setup Script
# Requirements: 1.1 - Centralized secrets management

set -e

VAULT_ADDR="${VAULT_ADDR:-https://vault.vault.svc:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-}"

echo "Setting up KV v2 secrets engine..."

# Enable KV v2 at secret/auth-platform
vault secrets enable -path=secret/auth-platform -version=2 kv 2>/dev/null || echo "KV engine already enabled"

# Configure KV engine settings
vault write secret/auth-platform/config max_versions=10 cas_required=false delete_version_after="0s"

# Create initial secret structure
echo "Creating initial secret structure..."

# JWT signing keys placeholder
vault kv put secret/auth-platform/jwt/signing-key \
  algorithm="ES256" \
  key_id="key-2025-01" \
  created_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  private_key="PLACEHOLDER_GENERATE_IN_PRODUCTION" \
  public_key="PLACEHOLDER_GENERATE_IN_PRODUCTION"

# Service configurations
vault kv put secret/auth-platform/config/auth-edge/default \
  jwk_cache_ttl_seconds=3600 \
  circuit_breaker_failure_threshold=5 \
  circuit_breaker_success_threshold=3 \
  circuit_breaker_timeout_seconds=30

vault kv put secret/auth-platform/config/token-service/default \
  access_token_ttl_seconds=900 \
  refresh_token_ttl_days=7 \
  signing_algorithm="ES256"

vault kv put secret/auth-platform/config/session-identity/default \
  session_ttl_hours=24 \
  risk_score_threshold=0.7

vault kv put secret/auth-platform/config/iam-policy/default \
  policy_reload_interval_seconds=60 \
  decision_cache_ttl_seconds=300

vault kv put secret/auth-platform/config/mfa-service/default \
  totp_window_size=1 \
  webauthn_timeout=60000 \
  push_timeout=60

echo "KV v2 secrets engine setup complete!"
