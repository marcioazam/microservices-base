#Requires -Version 5.1
<#
.SYNOPSIS
    HashiCorp Vault Initialization Script for Windows

.DESCRIPTION
    This script initializes Vault and sets up secrets for all services.
    Equivalent to init-vault.sh for Windows environments.

.EXAMPLE
    .\init-vault.ps1

.EXAMPLE
    .\init-vault.ps1 -VaultAddr "http://localhost:8200" -VaultToken "root"

.NOTES
    Author: Microservices Platform Team
    Date: 2026-01-16
#>

param(
    [string]$VaultAddr = $env:VAULT_ADDR,
    [string]$VaultToken = $env:VAULT_TOKEN
)

# Set defaults if not provided
if (-not $VaultAddr) { $VaultAddr = "http://localhost:8200" }
if (-not $VaultToken) { $VaultToken = "root" }

# Colors for output
$Colors = @{
    Red    = "Red"
    Green  = "Green"
    Yellow = "Yellow"
    Blue   = "Cyan"
}

function Write-Step {
    param([string]$Step, [string]$Message)
    Write-Host "[$Step] " -ForegroundColor $Colors.Yellow -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor $Colors.Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor $Colors.Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor $Colors.Yellow
}

function Generate-SecureSecret {
    # Generate 48 bytes of cryptographically secure random data, base64url encoded
    $bytes = New-Object byte[] 48
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    return [Convert]::ToBase64String($bytes) -replace '\+', '-' -replace '/', '_' -replace '=', ''
}

function Test-VaultHealth {
    param([string]$Addr)
    try {
        $response = Invoke-RestMethod -Uri "$Addr/v1/sys/health" -Method Get -TimeoutSec 5 -ErrorAction Stop
        return $true
    }
    catch {
        return $false
    }
}

function Invoke-VaultAPI {
    param(
        [string]$Method,
        [string]$Path,
        [hashtable]$Body = @{},
        [string]$Token
    )

    $headers = @{
        "X-Vault-Token" = $Token
        "Content-Type" = "application/json"
    }

    $uri = "$VaultAddr/v1/$Path"

    try {
        if ($Method -eq "GET") {
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers -ErrorAction Stop
        }
        else {
            $jsonBody = $Body | ConvertTo-Json -Depth 10
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers -Body $jsonBody -ErrorAction Stop
        }
        return $response
    }
    catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if ($statusCode -eq 400 -and $Path -like "*secrets/enable*") {
            # Already enabled
            return @{ already_enabled = $true }
        }
        throw $_
    }
}

# Banner
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Blue
Write-Host "║   HashiCorp Vault - Secrets Management Initialization      ║" -ForegroundColor $Colors.Blue
Write-Host "║                    (Windows PowerShell)                    ║" -ForegroundColor $Colors.Blue
Write-Host "╚════════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Blue
Write-Host ""

# Step 1: Check Vault connection
Write-Step "1/8" "Checking Vault connection..."
if (-not (Test-VaultHealth -Addr $VaultAddr)) {
    Write-Error "Vault is not accessible at $VaultAddr"
    Write-Warning "Start Vault with: docker-compose -f docker-compose.vault.yml up -d"
    exit 1
}
Write-Success "Vault is running at $VaultAddr"

# Set environment variables
$env:VAULT_ADDR = $VaultAddr
$env:VAULT_TOKEN = $VaultToken

# Step 2: Wait for Vault to be ready
Write-Step "2/8" "Waiting for Vault to be ready..."
$maxRetries = 30
$ready = $false
for ($i = 0; $i -lt $maxRetries; $i++) {
    try {
        $status = Invoke-VaultAPI -Method GET -Path "sys/seal-status" -Token $VaultToken
        if (-not $status.sealed) {
            $ready = $true
            break
        }
    }
    catch {
        Write-Host "." -NoNewline
        Start-Sleep -Seconds 1
    }
}

if (-not $ready) {
    Write-Error "Vault is not ready after $maxRetries seconds"
    exit 1
}
Write-Success "Vault is ready"

# Step 3: Enable KV secrets engine v2
Write-Step "3/8" "Enabling KV secrets engine..."
try {
    $mountPayload = @{
        type = "kv"
        options = @{
            version = "2"
        }
    }
    Invoke-VaultAPI -Method POST -Path "sys/mounts/secret" -Body $mountPayload -Token $VaultToken | Out-Null
    Write-Success "KV secrets engine enabled"
}
catch {
    if ($_.Exception.Message -like "*path is already in use*") {
        Write-Warning "KV secrets engine already enabled"
    }
    else {
        Write-Warning "KV secrets engine may already be enabled"
    }
}

# Step 4: Generate secure secrets
Write-Step "4/8" "Generating cryptographically secure secrets..."

$JwtSecret = Generate-SecureSecret
$TwilioAuthToken = if ($env:TWILIO_AUTH_TOKEN) { $env:TWILIO_AUTH_TOKEN } else { Generate-SecureSecret }
$TwilioWebhookSecret = if ($env:TWILIO_WEBHOOK_SECRET) { $env:TWILIO_WEBHOOK_SECRET } else { Generate-SecureSecret }
$MessagebirdApiKey = if ($env:MESSAGEBIRD_API_KEY) { $env:MESSAGEBIRD_API_KEY } else { Generate-SecureSecret }
$MessagebirdWebhookSecret = if ($env:MESSAGEBIRD_WEBHOOK_SECRET) { $env:MESSAGEBIRD_WEBHOOK_SECRET } else { Generate-SecureSecret }
$DbPassword = Generate-SecureSecret
$RedisPassword = Generate-SecureSecret
$RabbitmqPassword = Generate-SecureSecret

Write-Success "Secure secrets generated"

# Step 5: Store SMS service secrets
Write-Step "5/8" "Storing SMS service secrets in Vault..."
$smsSecrets = @{
    data = @{
        jwt_secret_key = $JwtSecret
        twilio_auth_token = $TwilioAuthToken
        twilio_webhook_secret = $TwilioWebhookSecret
        messagebird_api_key = $MessagebirdApiKey
        messagebird_webhook_secret = $MessagebirdWebhookSecret
        database_password = $DbPassword
    }
}

try {
    Invoke-VaultAPI -Method POST -Path "secret/data/sms-service" -Body $smsSecrets -Token $VaultToken | Out-Null
    Write-Success "SMS service secrets stored"
}
catch {
    Write-Error "Failed to store SMS service secrets: $_"
    exit 1
}

# Step 6: Store common secrets
Write-Step "6/8" "Storing common secrets..."
$commonSecrets = @{
    data = @{
        postgres_password = $DbPassword
        redis_password = $RedisPassword
        rabbitmq_password = $RabbitmqPassword
    }
}

try {
    Invoke-VaultAPI -Method POST -Path "secret/data/common" -Body $commonSecrets -Token $VaultToken | Out-Null
    Write-Success "Common secrets stored"
}
catch {
    Write-Error "Failed to store common secrets: $_"
    exit 1
}

# Step 7: Create policy for SMS service
Write-Step "7/8" "Creating access policies..."
$policyHcl = @"
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
"@

try {
    $policyPayload = @{
        policy = $policyHcl
    }
    Invoke-VaultAPI -Method PUT -Path "sys/policies/acl/sms-service" -Body $policyPayload -Token $VaultToken | Out-Null
    Write-Success "Policies created"
}
catch {
    Write-Error "Failed to create policy: $_"
    exit 1
}

# Step 8: Create token for SMS service
Write-Step "8/8" "Creating service token..."
$tokenPayload = @{
    policies = @("sms-service")
    display_name = "sms-service"
    ttl = "720h"
    renewable = $true
}

try {
    $tokenResponse = Invoke-VaultAPI -Method POST -Path "auth/token/create" -Body $tokenPayload -Token $VaultToken
    $SmsToken = $tokenResponse.auth.client_token
    Write-Success "Service token created"
}
catch {
    Write-Error "Failed to create service token: $_"
    exit 1
}

# Save configuration to .env.vault file
Write-Step "" "Saving configuration to .env.vault..."
$envContent = @"
# Vault Configuration
# Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
VAULT_ADDR=$VaultAddr
VAULT_TOKEN=$SmsToken
VAULT_NAMESPACE=
VAULT_SKIP_VERIFY=false

# SMS Service - Read secrets from Vault
SMS_SERVICE_VAULT_PATH=secret/sms-service
"@

$envContent | Out-File -FilePath ".env.vault" -Encoding utf8 -Force
Write-Success "Configuration saved to .env.vault"

# Display summary
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Blue
Write-Host "║                  Setup Complete!                           ║" -ForegroundColor $Colors.Blue
Write-Host "╚════════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Blue
Write-Host ""

Write-Host "Vault Configuration:" -ForegroundColor $Colors.Green
Write-Host "  URL: " -NoNewline; Write-Host $VaultAddr -ForegroundColor $Colors.Blue
Write-Host "  UI:  " -NoNewline; Write-Host "$VaultAddr/ui" -ForegroundColor $Colors.Blue
Write-Host "  Root Token: " -NoNewline; Write-Host $VaultToken -ForegroundColor $Colors.Yellow
Write-Host ""

Write-Host "SMS Service Token:" -ForegroundColor $Colors.Green
Write-Host "  Token: " -NoNewline; Write-Host $SmsToken -ForegroundColor $Colors.Yellow
Write-Host "  Saved to: " -NoNewline; Write-Host ".env.vault" -ForegroundColor $Colors.Blue
Write-Host ""

Write-Host "Secrets Stored:" -ForegroundColor $Colors.Green
Write-Host "  ✓ secret/sms-service/jwt_secret_key"
Write-Host "  ✓ secret/sms-service/twilio_auth_token"
Write-Host "  ✓ secret/sms-service/twilio_webhook_secret"
Write-Host "  ✓ secret/sms-service/messagebird_api_key"
Write-Host "  ✓ secret/sms-service/messagebird_webhook_secret"
Write-Host "  ✓ secret/sms-service/database_password"
Write-Host "  ✓ secret/common/* (shared secrets)"
Write-Host ""

Write-Host "Next Steps:" -ForegroundColor $Colors.Yellow
Write-Host "  1. Load environment: " -NoNewline
Write-Host "`$env:VAULT_ADDR='$VaultAddr'; `$env:VAULT_TOKEN='$SmsToken'" -ForegroundColor $Colors.Blue
Write-Host "  2. Test Vault access: " -NoNewline
Write-Host ".\scripts\test-vault.ps1" -ForegroundColor $Colors.Blue
Write-Host "  3. Start your services: " -NoNewline
Write-Host "docker-compose up -d" -ForegroundColor $Colors.Blue
Write-Host ""

Write-Host "View secrets:" -ForegroundColor $Colors.Yellow
Write-Host "  vault kv get secret/sms-service" -ForegroundColor $Colors.Blue
Write-Host ""

Write-Host "IMPORTANT SECURITY NOTES:" -ForegroundColor $Colors.Red
Write-Host "  - Keep your root token secure!"
Write-Host "  - Never commit .env.vault to version control!"
Write-Host "  - Use Vault policies for fine-grained access control"
Write-Host "  - Enable audit logging in production"
Write-Host "  - Use auto-unseal with cloud KMS in production"
Write-Host ""
