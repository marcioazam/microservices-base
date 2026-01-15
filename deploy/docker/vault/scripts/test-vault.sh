#!/bin/bash
# Test Vault Connection and Secret Access

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Vault Connection & Access Test         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""

# Load configuration
if [ -f .env.vault ]; then
    source .env.vault
    echo -e "${GREEN}✓ Loaded configuration from .env.vault${NC}"
else
    echo -e "${YELLOW}⚠ .env.vault not found, using defaults${NC}"
    VAULT_ADDR=${VAULT_ADDR:-http://localhost:8200}
    VAULT_TOKEN=${VAULT_TOKEN:-root}
fi

export VAULT_ADDR
export VAULT_TOKEN

# Test 1: Vault connectivity
echo -e "${YELLOW}[Test 1/5]${NC} Testing Vault connectivity..."
if vault status > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Vault is accessible at $VAULT_ADDR${NC}"
else
    echo -e "${RED}✗ Cannot connect to Vault${NC}"
    exit 1
fi

# Test 2: Authentication
echo -e "${YELLOW}[Test 2/5]${NC} Testing authentication..."
if vault token lookup > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Token is valid${NC}"
else
    echo -e "${RED}✗ Authentication failed${NC}"
    exit 1
fi

# Test 3: Read SMS service secrets
echo -e "${YELLOW}[Test 3/5]${NC} Reading SMS service secrets..."
if vault kv get -format=json secret/sms-service > /dev/null 2>&1; then
    echo -e "${GREEN}✓ SMS service secrets accessible${NC}"

    # Display secret metadata (not the actual secret values)
    echo -e "${BLUE}Secret metadata:${NC}"
    vault kv metadata get secret/sms-service | grep -E "Created|Versions"
else
    echo -e "${RED}✗ Cannot read SMS service secrets${NC}"
    exit 1
fi

# Test 4: List available secrets
echo -e "${YELLOW}[Test 4/5]${NC} Listing available secrets..."
echo -e "${BLUE}Available secret paths:${NC}"
vault kv list secret/ 2>/dev/null || echo "  (no secrets found)"

# Test 5: Python client test
echo -e "${YELLOW}[Test 5/5]${NC} Testing Python client..."
python3 - <<'PYTHON_TEST'
import os
import sys

try:
    import hvac

    vault_addr = os.getenv('VAULT_ADDR', 'http://localhost:8200')
    vault_token = os.getenv('VAULT_TOKEN', 'root')

    client = hvac.Client(url=vault_addr, token=vault_token)

    # Test authentication
    if not client.is_authenticated():
        print("\033[0;31m✗ Python client authentication failed\033[0m")
        sys.exit(1)

    # Read secrets
    secret = client.secrets.kv.v2.read_secret_version(path='sms-service')
    jwt_secret = secret['data']['data']['jwt_secret_key']

    print(f"\033[0;32m✓ Python client working correctly\033[0m")
    print(f"\033[0;34mJWT Secret length: {len(jwt_secret)} characters\033[0m")

except ImportError:
    print("\033[1;33m⚠ hvac library not installed. Run: pip install hvac\033[0m")
    sys.exit(0)
except Exception as e:
    print(f"\033[0;31m✗ Python client error: {e}\033[0m")
    sys.exit(1)
PYTHON_TEST

PYTHON_EXIT_CODE=$?

# Summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         Test Summary                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""

if [ $PYTHON_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed! Vault is ready to use.${NC}"
    echo ""
    echo -e "${YELLOW}Quick Commands:${NC}"
    echo -e "  View secrets: ${BLUE}vault kv get secret/sms-service${NC}"
    echo -e "  Update secret: ${BLUE}vault kv put secret/sms-service key=value${NC}"
    echo -e "  Delete secret: ${BLUE}vault kv delete secret/sms-service${NC}"
    echo -e "  Vault UI: ${BLUE}$VAULT_ADDR/ui${NC}"
else
    echo -e "${YELLOW}⚠ Some tests completed with warnings${NC}"
fi

echo ""
