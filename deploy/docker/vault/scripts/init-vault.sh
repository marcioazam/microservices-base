#!/bin/bash
# HashiCorp Vault Initialization Script
# This script initializes Vault and sets up secrets for all services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   HashiCorp Vault - Secrets Management Initialization    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
VAULT_ADDR=${VAULT_ADDR:-http://localhost:8200}
VAULT_TOKEN=${VAULT_TOKEN:-root}

# Check if Vault is running
echo -e "${YELLOW}[1/8]${NC} Checking Vault connection..."
if ! curl -s "$VAULT_ADDR/v1/sys/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ Vault is not accessible at $VAULT_ADDR${NC}"
    echo -e "${YELLOW}Start Vault with: docker-compose -f docker-compose.vault.yml up -d${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Vault is running at $VAULT_ADDR${NC}"

# Export Vault configuration
export VAULT_ADDR
export VAULT_TOKEN

# Wait for Vault to be ready
echo -e "${YELLOW}[2/8]${NC} Waiting for Vault to be ready..."
for i in {1..30}; do
    if vault status > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Vault is ready${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# Enable KV secrets engine v2
echo -e "${YELLOW}[3/8]${NC} Enabling KV secrets engine..."
vault secrets enable -version=2 -path=secret kv 2>/dev/null || echo -e "${YELLOW}⚠ KV secrets engine already enabled${NC}"
echo -e "${GREEN}✓ KV secrets engine ready${NC}"

# Generate secure secrets
echo -e "${YELLOW}[4/8]${NC} Generating cryptographically secure secrets..."

# Function to generate secure secret
generate_secret() {
    python3 -c 'import secrets; print(secrets.token_urlsafe(48))'
}

# Generate all secrets
JWT_SECRET=$(generate_secret)
TWILIO_AUTH_TOKEN=${TWILIO_AUTH_TOKEN:-$(generate_secret)}
TWILIO_WEBHOOK_SECRET=${TWILIO_WEBHOOK_SECRET:-$(generate_secret)}
MESSAGEBIRD_API_KEY=${MESSAGEBIRD_API_KEY:-$(generate_secret)}
MESSAGEBIRD_WEBHOOK_SECRET=${MESSAGEBIRD_WEBHOOK_SECRET:-$(generate_secret)}
DB_PASSWORD=$(generate_secret)

echo -e "${GREEN}✓ Secure secrets generated${NC}"

# Store SMS service secrets
echo -e "${YELLOW}[5/8]${NC} Storing SMS service secrets in Vault..."
vault kv put secret/sms-service \
    jwt_secret_key="$JWT_SECRET" \
    twilio_auth_token="$TWILIO_AUTH_TOKEN" \
    twilio_webhook_secret="$TWILIO_WEBHOOK_SECRET" \
    messagebird_api_key="$MESSAGEBIRD_API_KEY" \
    messagebird_webhook_secret="$MESSAGEBIRD_WEBHOOK_SECRET" \
    database_password="$DB_PASSWORD" \
    > /dev/null

echo -e "${GREEN}✓ SMS service secrets stored${NC}"

# Store common/shared secrets
echo -e "${YELLOW}[6/8]${NC} Storing common secrets..."
vault kv put secret/common \
    postgres_password="$DB_PASSWORD" \
    redis_password="$(generate_secret)" \
    rabbitmq_password="$(generate_secret)" \
    > /dev/null

echo -e "${GREEN}✓ Common secrets stored${NC}"

# Create policy for SMS service
echo -e "${YELLOW}[7/8]${NC} Creating access policies..."
vault policy write sms-service /vault/policies/sms-service-policy.hcl 2>/dev/null || \
vault policy write sms-service - <<EOF
# SMS Service Policy
path "secret/data/sms-service/*" {
  capabilities = ["read", "list"]
}
path "secret/data/common/*" {
  capabilities = ["read", "list"]
}
path "auth/token/renew-self" {
  capabilities = ["update"]
}
EOF

echo -e "${GREEN}✓ Policies created${NC}"

# Create token for SMS service
echo -e "${YELLOW}[8/8]${NC} Creating service token..."
SMS_TOKEN=$(vault token create \
    -policy=sms-service \
    -display-name="sms-service" \
    -ttl=720h \
    -renewable=true \
    -format=json | python3 -c "import sys, json; print(json.load(sys.stdin)['auth']['client_token'])")

echo -e "${GREEN}✓ Service token created${NC}"

# Save configuration to .env file
echo -e "${YELLOW}Saving configuration to .env.vault...${NC}"
cat > .env.vault <<EOF
# Vault Configuration
# Generated: $(date)
VAULT_ADDR=$VAULT_ADDR
VAULT_TOKEN=$SMS_TOKEN
VAULT_NAMESPACE=
VAULT_SKIP_VERIFY=false

# SMS Service - Read secrets from Vault
SMS_SERVICE_VAULT_PATH=secret/sms-service
EOF

echo -e "${GREEN}✓ Configuration saved to .env.vault${NC}"

# Display summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                  Setup Complete! 🎉                        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Vault Configuration:${NC}"
echo -e "  📍 URL: ${BLUE}$VAULT_ADDR${NC}"
echo -e "  🌐 UI:  ${BLUE}$VAULT_ADDR/ui${NC}"
echo -e "  🔑 Root Token: ${YELLOW}$VAULT_TOKEN${NC}"
echo ""
echo -e "${GREEN}SMS Service Token:${NC}"
echo -e "  🎫 Token: ${YELLOW}$SMS_TOKEN${NC}"
echo -e "  📄 Saved to: ${BLUE}.env.vault${NC}"
echo ""
echo -e "${GREEN}Secrets Stored:${NC}"
echo -e "  ✓ secret/sms-service/jwt_secret_key"
echo -e "  ✓ secret/sms-service/twilio_auth_token"
echo -e "  ✓ secret/sms-service/twilio_webhook_secret"
echo -e "  ✓ secret/sms-service/messagebird_api_key"
echo -e "  ✓ secret/sms-service/messagebird_webhook_secret"
echo -e "  ✓ secret/sms-service/database_password"
echo -e "  ✓ secret/common/* (shared secrets)"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo -e "  1. Source the environment: ${BLUE}source .env.vault${NC}"
echo -e "  2. Test Vault access: ${BLUE}./scripts/test-vault.sh${NC}"
echo -e "  3. Start your services: ${BLUE}docker-compose up -d${NC}"
echo ""
echo -e "${YELLOW}View secrets:${NC}"
echo -e "  ${BLUE}vault kv get secret/sms-service${NC}"
echo ""
echo -e "${RED}IMPORTANT SECURITY NOTES:${NC}"
echo -e "  ⚠  Keep your root token secure!"
echo -e "  ⚠  Never commit .env.vault to version control!"
echo -e "  ⚠  Use Vault policies for fine-grained access control"
echo -e "  ⚠  Enable audit logging in production"
echo -e "  ⚠  Use auto-unseal with cloud KMS in production"
echo ""
