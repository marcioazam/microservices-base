# Vault Policy for SMS Service
# This policy grants the SMS service read access to its secrets

# Allow reading SMS service secrets
path "secret/data/sms-service/*" {
  capabilities = ["read", "list"]
}

# Allow reading common/shared secrets
path "secret/data/common/*" {
  capabilities = ["read", "list"]
}

# Allow reading database credentials (if using dynamic secrets)
path "database/creds/sms-service" {
  capabilities = ["read"]
}

# Allow token renewal
path "auth/token/renew-self" {
  capabilities = ["update"]
}

# Allow token lookup
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
