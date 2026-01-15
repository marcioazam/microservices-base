# HashiCorp Vault Production Configuration
# This is an example production configuration for Vault
# Copy and customize for your environment

# API listener
listener "tcp" {
  address       = "0.0.0.0:8200"
  tls_disable   = 0
  tls_cert_file = "/vault/config/tls/vault.crt"
  tls_key_file  = "/vault/config/tls/vault.key"
}

# Storage backend - PostgreSQL (recommended for production)
storage "postgresql" {
  connection_url = "postgres://vault:vault_password@postgres:5432/vault?sslmode=disable"
  table          = "vault_kv_store"
  max_parallel   = 128
}

# Alternative: Consul storage backend (for HA setup)
# storage "consul" {
#   address = "consul:8500"
#   path    = "vault/"
# }

# Telemetry
telemetry {
  prometheus_retention_time = "30s"
  disable_hostname          = false
}

# API settings
api_addr = "https://vault:8200"
cluster_addr = "https://vault:8201"

# UI
ui = true

# Seal configuration - AWS KMS (auto-unseal for production)
# seal "awskms" {
#   region     = "us-east-1"
#   kms_key_id = "your-kms-key-id"
# }

# Seal configuration - Azure Key Vault
# seal "azurekeyvault" {
#   tenant_id      = "your-tenant-id"
#   client_id      = "your-client-id"
#   client_secret  = "your-client-secret"
#   vault_name     = "your-vault-name"
#   key_name       = "your-key-name"
# }

# Disable mlock (not recommended for production, but may be needed in containers)
# disable_mlock = true

# Maximum request duration
max_lease_ttl = "768h"
default_lease_ttl = "768h"
