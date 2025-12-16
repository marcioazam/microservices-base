# Session Identity Core Vault Policy
# Requirements: 1.1, 1.2, 1.3, 1.4

# Read service configuration
path "secret/data/auth-platform/config/session-identity/*" {
  capabilities = ["read"]
}

# Dynamic PostgreSQL credentials - Requirements 1.2
path "database/auth-platform/creds/auth-platform-readwrite" {
  capabilities = ["read"]
}

# Dynamic Redis credentials
path "database/auth-platform/creds/auth-platform-redis" {
  capabilities = ["read"]
}

# Use transit for session encryption
path "transit/auth-platform/encrypt/session-encryption" {
  capabilities = ["update"]
}

path "transit/auth-platform/decrypt/session-encryption" {
  capabilities = ["update"]
}

# Read OAuth configuration
path "secret/data/auth-platform/oauth/*" {
  capabilities = ["read"]
}

# Issue service certificates
path "pki/auth-platform-issuer/issue/service-cert" {
  capabilities = ["create", "update"]
}

# Renew own token
path "auth/token/renew-self" {
  capabilities = ["update"]
}

# Lookup own token
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
