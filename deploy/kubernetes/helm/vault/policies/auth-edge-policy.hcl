# Auth Edge Service Vault Policy
# Requirements: 1.1, 1.4

# Read JWT signing keys for token validation
path "secret/data/auth-platform/jwt/*" {
  capabilities = ["read"]
}

# Read service configuration
path "secret/data/auth-platform/config/auth-edge/*" {
  capabilities = ["read"]
}

# Use transit for session encryption
path "transit/auth-platform/encrypt/session-encryption" {
  capabilities = ["update"]
}

path "transit/auth-platform/decrypt/session-encryption" {
  capabilities = ["update"]
}

# Read PKI certificates for mTLS
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
