#!/bin/bash
# PKI Secrets Engine Setup Script
# Requirements: 2.1, 2.2, 2.3, 2.4

set -e

VAULT_ADDR="${VAULT_ADDR:-https://vault.vault.svc:8200}"

echo "Setting up PKI secrets engine..."

# ============================================
# Root CA Setup - Requirements 2.1
# ============================================
echo "Setting up Root CA..."

# Enable PKI engine for root CA
vault secrets enable -path=pki/auth-platform-root pki 2>/dev/null || echo "Root PKI engine already enabled"

# Tune for 10 year max TTL
vault secrets tune -max-lease-ttl=87600h pki/auth-platform-root

# Generate root CA certificate (10 years)
vault write -format=json pki/auth-platform-root/root/generate/internal \
    common_name="Auth Platform Root CA" \
    issuer_name="auth-platform-root-2025" \
    ttl=87600h \
    key_type=ec \
    key_bits=256 > /tmp/root-ca.json

echo "Root CA generated successfully"

# Configure CRL and OCSP URLs
vault write pki/auth-platform-root/config/urls \
    issuing_certificates="https://vault.vault.svc:8200/v1/pki/auth-platform-root/ca" \
    crl_distribution_points="https://vault.vault.svc:8200/v1/pki/auth-platform-root/crl" \
    ocsp_servers="https://vault.vault.svc:8200/v1/pki/auth-platform-root/ocsp"

# ============================================
# Intermediate CA Setup - Requirements 2.4
# ============================================
echo "Setting up Intermediate CA for Linkerd..."

# Enable PKI engine for intermediate CA
vault secrets enable -path=pki/auth-platform-issuer pki 2>/dev/null || echo "Issuer PKI engine already enabled"

# Tune for 1 year max TTL
vault secrets tune -max-lease-ttl=8760h pki/auth-platform-issuer

# Generate intermediate CSR
vault write -format=json pki/auth-platform-issuer/intermediate/generate/internal \
    common_name="Auth Platform Issuer CA" \
    issuer_name="auth-platform-issuer-2025" \
    key_type=ec \
    key_bits=256 > /tmp/intermediate-csr.json

# Extract CSR
CSR=$(cat /tmp/intermediate-csr.json | jq -r '.data.csr')

# Sign intermediate with root CA
vault write -format=json pki/auth-platform-root/root/sign-intermediate \
    csr="$CSR" \
    format=pem_bundle \
    ttl=8760h > /tmp/intermediate-cert.json

# Import signed certificate
CERT=$(cat /tmp/intermediate-cert.json | jq -r '.data.certificate')
vault write pki/auth-platform-issuer/intermediate/set-signed certificate="$CERT"

echo "Intermediate CA signed and imported"

# Configure CRL and OCSP URLs for intermediate
vault write pki/auth-platform-issuer/config/urls \
    issuing_certificates="https://vault.vault.svc:8200/v1/pki/auth-platform-issuer/ca" \
    crl_distribution_points="https://vault.vault.svc:8200/v1/pki/auth-platform-issuer/crl" \
    ocsp_servers="https://vault.vault.svc:8200/v1/pki/auth-platform-issuer/ocsp"

# ============================================
# Certificate Roles - Requirements 2.1 (72h max)
# ============================================
echo "Creating certificate roles..."

# Service certificate role (72 hours max TTL)
vault write pki/auth-platform-issuer/roles/service-cert \
    allowed_domains="auth-platform.svc.cluster.local,auth-platform.local" \
    allow_subdomains=true \
    allow_bare_domains=false \
    max_ttl=72h \
    ttl=24h \
    key_type=ec \
    key_bits=256 \
    require_cn=true \
    allowed_uri_sans="spiffe://auth-platform.local/*" \
    enforce_hostnames=true \
    server_flag=true \
    client_flag=true

# Linkerd identity issuer role (48 hours for identity issuer)
vault write pki/auth-platform-issuer/roles/linkerd-identity \
    allowed_domains="identity.linkerd.cluster.local" \
    allow_subdomains=false \
    allow_bare_domains=true \
    max_ttl=48h \
    ttl=24h \
    key_type=ec \
    key_bits=256 \
    basic_constraints_valid_for_non_ca=false \
    key_usage="DigitalSignature,KeyEncipherment,KeyAgreement,CertSign" \
    ext_key_usage="ServerAuth,ClientAuth"

echo "PKI secrets engine setup complete!"

# Cleanup
rm -f /tmp/root-ca.json /tmp/intermediate-csr.json /tmp/intermediate-cert.json
