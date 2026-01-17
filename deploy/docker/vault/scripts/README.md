# Vault Scripts

This directory contains scripts for initializing and managing HashiCorp Vault.

## Available Scripts

| Script | Platform | Description |
|--------|----------|-------------|
| `init-vault.sh` | Linux/macOS | Initialize Vault and set up secrets |
| `init-vault.ps1` | Windows | Initialize Vault and set up secrets |
| `test-vault.sh` | Linux/macOS | Test Vault connection and secrets |
| `test-vault.ps1` | Windows | Test Vault connection and secrets |

## Prerequisites

1. **Docker** must be running
2. **Vault container** must be started:
   ```bash
   docker-compose -f ../docker-compose.vault.yml up -d
   ```

## Quick Start

### Windows (PowerShell)

```powershell
# Navigate to the vault directory
cd deploy\docker\vault

# Start Vault container
docker-compose -f docker-compose.vault.yml up -d

# Wait a few seconds for Vault to start
Start-Sleep -Seconds 5

# Initialize Vault (creates secrets and policies)
.\scripts\init-vault.ps1

# Test the connection
.\scripts\test-vault.ps1

# Load environment variables
$envContent = Get-Content .env.vault
foreach ($line in $envContent) {
    if ($line -match "^([^#][^=]+)=(.*)$") {
        [Environment]::SetEnvironmentVariable($matches[1], $matches[2], "Process")
    }
}
```

### Linux/macOS (Bash)

```bash
# Navigate to the vault directory
cd deploy/docker/vault

# Start Vault container
docker-compose -f docker-compose.vault.yml up -d

# Wait a few seconds for Vault to start
sleep 5

# Initialize Vault (creates secrets and policies)
./scripts/init-vault.sh

# Test the connection
./scripts/test-vault.sh

# Load environment variables
source .env.vault
```

## What the Scripts Do

### init-vault.sh / init-vault.ps1

1. **Check Vault Health** - Verifies Vault is running and accessible
2. **Enable KV Secrets Engine** - Enables KV v2 secrets engine at `secret/`
3. **Generate Secrets** - Creates cryptographically secure secrets for:
   - JWT signing key
   - Twilio credentials
   - MessageBird credentials
   - Database password
   - Redis password
   - RabbitMQ password
4. **Store Secrets** - Saves secrets to Vault at:
   - `secret/sms-service/*`
   - `secret/common/*`
5. **Create Policies** - Sets up access policies for services
6. **Generate Service Token** - Creates a renewable token for SMS service
7. **Save Configuration** - Writes `.env.vault` with Vault configuration

### test-vault.sh / test-vault.ps1

1. **Health Check** - Verifies Vault is healthy
2. **Token Validation** - Checks token is valid and shows TTL
3. **Read SMS Secrets** - Verifies SMS service secrets are accessible
4. **Read Common Secrets** - Verifies common secrets are accessible
5. **Token Renewal** - Tests that the token can be renewed

## Environment Variables

After running the init script, the following environment variables are set in `.env.vault`:

| Variable | Description |
|----------|-------------|
| `VAULT_ADDR` | Vault server URL (default: `http://localhost:8200`) |
| `VAULT_TOKEN` | Service token for authentication |
| `VAULT_NAMESPACE` | Vault namespace (empty for open source) |
| `VAULT_SKIP_VERIFY` | Skip TLS verification (false for production) |

## Secrets Stored

### SMS Service (`secret/sms-service`)

| Key | Description |
|-----|-------------|
| `jwt_secret_key` | JWT signing key (64 characters) |
| `twilio_auth_token` | Twilio authentication token |
| `twilio_webhook_secret` | Twilio webhook signing secret |
| `messagebird_api_key` | MessageBird API key |
| `messagebird_webhook_secret` | MessageBird webhook secret |
| `database_password` | PostgreSQL password |

### Common (`secret/common`)

| Key | Description |
|-----|-------------|
| `postgres_password` | PostgreSQL password |
| `redis_password` | Redis password |
| `rabbitmq_password` | RabbitMQ password |

## Troubleshooting

### Vault Not Starting

```bash
# Check container status
docker ps -a | grep vault

# View container logs
docker logs vault-server
```

### Permission Denied (Linux/macOS)

```bash
# Make scripts executable
chmod +x scripts/*.sh
```

### PowerShell Execution Policy (Windows)

```powershell
# Allow script execution (run as Administrator)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Connection Refused

Ensure Vault container is running and ports are mapped correctly:

```bash
docker-compose -f docker-compose.vault.yml ps
```

### Token Expired

Re-run the init script to generate a new token:

```bash
# Linux/macOS
./scripts/init-vault.sh

# Windows
.\scripts\init-vault.ps1
```

## Security Notes

1. **Never commit `.env.vault`** - Contains sensitive tokens
2. **Use production configuration** for non-dev environments:
   - Enable TLS
   - Use auto-unseal with cloud KMS
   - Enable audit logging
   - Use fine-grained policies
3. **Rotate tokens regularly** - The default token TTL is 720 hours (30 days)
4. **Use namespaces** in enterprise for multi-tenancy
