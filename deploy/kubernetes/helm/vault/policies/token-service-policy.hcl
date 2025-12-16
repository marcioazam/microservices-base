# Token Service Vault Policy
# Requirements: 1.1, 1.2, 1.4

# Full access to JWT signing keys
path "secret/data/auth-platform/jwt/*" {
  capabilities = ["create", "read", "update"]
}

# Manage JWT key metadata
path "secret/metadata/auth-platform/jwt/*" {
  capabilities = ["read", "list"]
}

# Read service configuration
path "secret/data/auth-platform/config/token-service/*" {
  capabilities = ["read"]
}

# Use transit for JWT signing (ES256)
path "transit/auth-platform/sign/jwt-signing" {
  capabilities = ["update"]
}

path "transit/auth-platform/verify/jwt-signing" {
  capabilities = ["update"]
}

# Read transit key info for JWKS
path "transit/auth-platform/keys/jwt-signing" {
  capabilities = ["read"]
}

# Issue service certificates
path "pki/auth-platform-issuer/issue/service-cert" {
  capabilities = ["create", "update"]
}

# Redis dynamic credentials
path "database/auth-platform/creds/auth-platform-redis" {
  capabilities = ["read"]
}

# Renew own token
path "auth/token/renew-self" {
  capabilities = ["update"]
}

# Lookup own token
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
