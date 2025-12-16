# MFA Service Vault Policy
# Requirements: 1.1, 1.2, 1.4

# Manage MFA secrets (TOTP keys, WebAuthn credentials)
path "secret/data/auth-platform/mfa/*" {
  capabilities = ["create", "read", "update", "delete"]
}

# List MFA secrets
path "secret/metadata/auth-platform/mfa/*" {
  capabilities = ["read", "list", "delete"]
}

# Read service configuration
path "secret/data/auth-platform/config/mfa-service/*" {
  capabilities = ["read"]
}

# Dynamic PostgreSQL credentials
path "database/auth-platform/creds/auth-platform-readwrite" {
  capabilities = ["read"]
}

# Dynamic Redis credentials
path "database/auth-platform/creds/auth-platform-redis" {
  capabilities = ["read"]
}

# Use transit for encrypting TOTP secrets at rest
path "transit/auth-platform/encrypt/session-encryption" {
  capabilities = ["update"]
}

path "transit/auth-platform/decrypt/session-encryption" {
  capabilities = ["update"]
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
