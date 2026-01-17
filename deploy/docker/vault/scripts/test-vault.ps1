#Requires -Version 5.1
<#
.SYNOPSIS
    Test HashiCorp Vault Connection and Secrets

.DESCRIPTION
    This script tests the Vault connection and verifies that secrets
    can be read correctly. Use this after running init-vault.ps1.

.EXAMPLE
    .\test-vault.ps1

.EXAMPLE
    .\test-vault.ps1 -VaultAddr "http://localhost:8200"

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
if (-not $VaultToken) {
    # Try to load from .env.vault
    if (Test-Path ".env.vault") {
        $envContent = Get-Content ".env.vault" | Where-Object { $_ -match "^VAULT_TOKEN=" }
        if ($envContent) {
            $VaultToken = ($envContent -split "=", 2)[1]
        }
    }
    if (-not $VaultToken) {
        $VaultToken = "root"
    }
}

# Colors
$Colors = @{
    Red    = "Red"
    Green  = "Green"
    Yellow = "Yellow"
    Blue   = "Cyan"
}

function Write-TestResult {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Details = ""
    )

    if ($Passed) {
        Write-Host "  ✓ " -ForegroundColor $Colors.Green -NoNewline
        Write-Host "$Name" -NoNewline
        if ($Details) {
            Write-Host " - $Details" -ForegroundColor $Colors.Blue
        }
        else {
            Write-Host ""
        }
    }
    else {
        Write-Host "  ✗ " -ForegroundColor $Colors.Red -NoNewline
        Write-Host "$Name" -NoNewline
        if ($Details) {
            Write-Host " - $Details" -ForegroundColor $Colors.Yellow
        }
        else {
            Write-Host ""
        }
    }
    return $Passed
}

function Invoke-VaultAPI {
    param(
        [string]$Method,
        [string]$Path,
        [string]$Token
    )

    $headers = @{
        "X-Vault-Token" = $Token
    }

    $uri = "$VaultAddr/v1/$Path"

    try {
        $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers -ErrorAction Stop
        return @{ Success = $true; Data = $response }
    }
    catch {
        return @{ Success = $false; Error = $_.Exception.Message }
    }
}

# Banner
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Blue
Write-Host "║            HashiCorp Vault - Connection Test               ║" -ForegroundColor $Colors.Blue
Write-Host "╚════════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Blue
Write-Host ""

$totalTests = 0
$passedTests = 0

# Test 1: Vault Health
Write-Host "Test 1: Vault Health Check" -ForegroundColor $Colors.Yellow
$totalTests++
try {
    $health = Invoke-RestMethod -Uri "$VaultAddr/v1/sys/health" -Method Get -TimeoutSec 5 -ErrorAction Stop
    if (Write-TestResult -Name "Vault is running" -Passed $true -Details "Version $($health.version)") {
        $passedTests++
    }
}
catch {
    Write-TestResult -Name "Vault is running" -Passed $false -Details "Cannot connect to $VaultAddr"
    Write-Host ""
    Write-Host "Vault is not accessible. Please start Vault first:" -ForegroundColor $Colors.Red
    Write-Host "  docker-compose -f docker-compose.vault.yml up -d" -ForegroundColor $Colors.Blue
    exit 1
}

# Test 2: Token Authentication
Write-Host ""
Write-Host "Test 2: Token Authentication" -ForegroundColor $Colors.Yellow
$totalTests++
$tokenLookup = Invoke-VaultAPI -Method GET -Path "auth/token/lookup-self" -Token $VaultToken
if ($tokenLookup.Success) {
    $ttl = $tokenLookup.Data.data.ttl
    $ttlHours = [math]::Round($ttl / 3600, 1)
    if (Write-TestResult -Name "Token is valid" -Passed $true -Details "TTL: $ttlHours hours") {
        $passedTests++
    }
}
else {
    Write-TestResult -Name "Token is valid" -Passed $false -Details $tokenLookup.Error
}

# Test 3: Read SMS Service Secrets
Write-Host ""
Write-Host "Test 3: Read SMS Service Secrets" -ForegroundColor $Colors.Yellow
$smsSecrets = Invoke-VaultAPI -Method GET -Path "secret/data/sms-service" -Token $VaultToken

if ($smsSecrets.Success) {
    $totalTests++
    $data = $smsSecrets.Data.data.data

    $requiredKeys = @(
        "jwt_secret_key",
        "twilio_auth_token",
        "twilio_webhook_secret",
        "messagebird_api_key",
        "messagebird_webhook_secret",
        "database_password"
    )

    $allPresent = $true
    foreach ($key in $requiredKeys) {
        if (-not $data.$key) {
            $allPresent = $false
            break
        }
    }

    if (Write-TestResult -Name "SMS service secrets readable" -Passed $allPresent -Details "$($requiredKeys.Count) secrets found") {
        $passedTests++
    }

    # Show masked secrets
    Write-Host ""
    Write-Host "  Secrets (masked):" -ForegroundColor $Colors.Blue
    foreach ($key in $requiredKeys) {
        $value = $data.$key
        if ($value) {
            $masked = $value.Substring(0, [Math]::Min(8, $value.Length)) + "..." + $value.Substring([Math]::Max(0, $value.Length - 4))
            Write-Host "    $key = $masked"
        }
    }
}
else {
    $totalTests++
    Write-TestResult -Name "SMS service secrets readable" -Passed $false -Details $smsSecrets.Error
}

# Test 4: Read Common Secrets
Write-Host ""
Write-Host "Test 4: Read Common Secrets" -ForegroundColor $Colors.Yellow
$totalTests++
$commonSecrets = Invoke-VaultAPI -Method GET -Path "secret/data/common" -Token $VaultToken

if ($commonSecrets.Success) {
    $data = $commonSecrets.Data.data.data
    $keyCount = ($data | Get-Member -MemberType NoteProperty).Count
    if (Write-TestResult -Name "Common secrets readable" -Passed $true -Details "$keyCount secrets found") {
        $passedTests++
    }
}
else {
    Write-TestResult -Name "Common secrets readable" -Passed $false -Details $commonSecrets.Error
}

# Test 5: Token Renewal
Write-Host ""
Write-Host "Test 5: Token Renewal Capability" -ForegroundColor $Colors.Yellow
$totalTests++

try {
    $headers = @{
        "X-Vault-Token" = $VaultToken
        "Content-Type" = "application/json"
    }
    $renewResponse = Invoke-RestMethod -Uri "$VaultAddr/v1/auth/token/renew-self" -Method POST -Headers $headers -ErrorAction Stop
    $newTtl = [math]::Round($renewResponse.auth.lease_duration / 3600, 1)
    if (Write-TestResult -Name "Token can be renewed" -Passed $true -Details "New TTL: $newTtl hours") {
        $passedTests++
    }
}
catch {
    Write-TestResult -Name "Token can be renewed" -Passed $false -Details $_.Exception.Message
}

# Summary
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Blue
Write-Host "║                     Test Summary                           ║" -ForegroundColor $Colors.Blue
Write-Host "╚════════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Blue
Write-Host ""

if ($passedTests -eq $totalTests) {
    Write-Host "  Result: " -NoNewline
    Write-Host "ALL TESTS PASSED ($passedTests/$totalTests)" -ForegroundColor $Colors.Green
    Write-Host ""
    Write-Host "  Vault is properly configured and ready for use!" -ForegroundColor $Colors.Green
    $exitCode = 0
}
else {
    Write-Host "  Result: " -NoNewline
    Write-Host "$passedTests/$totalTests TESTS PASSED" -ForegroundColor $Colors.Yellow
    Write-Host ""
    Write-Host "  Some tests failed. Please check the configuration." -ForegroundColor $Colors.Yellow
    $exitCode = 1
}

Write-Host ""
Write-Host "Configuration:" -ForegroundColor $Colors.Blue
Write-Host "  VAULT_ADDR:  $VaultAddr"
Write-Host "  VAULT_TOKEN: $($VaultToken.Substring(0, [Math]::Min(10, $VaultToken.Length)))..."
Write-Host ""

exit $exitCode
