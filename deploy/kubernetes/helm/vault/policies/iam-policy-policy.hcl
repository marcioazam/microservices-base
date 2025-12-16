# IAM Policy Service Vault Policy
# Requirements: 1.1, 1.2, 1.4

# Read policy configuration
path "secret/data/auth-platform/config/iam-policy/*" {
  capabilities = ["read"]
}

# Read-only database access for policy lookups
path "database/auth-platform/creds/auth-platform-readonly" {
  capabilities = ["read"]
}

# Read RBAC/ABAC policy definitions
path "secret/data/auth-platform/policies/*" {
  capabilities = ["read", "list"]
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
