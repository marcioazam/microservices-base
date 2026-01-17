#Requires -Version 5.1
<#
.SYNOPSIS
    Vault Backup Script for Windows

.DESCRIPTION
    Creates a complete backup of Vault secrets and configuration.

.PARAMETER BackupDir
    Directory to store backups (default: .\vault-backups)

.PARAMETER Encrypt
    Encrypt the backup with a passphrase

.EXAMPLE
    .\backup-vault.ps1
    .\backup-vault.ps1 -BackupDir "D:\backups"
    .\backup-vault.ps1 -Encrypt

.NOTES
    Requires: Vault CLI, PowerShell 5.1+
#>

param(
    [string]$BackupDir = ".\vault-backups",
    [switch]$Encrypt
)

# Configuration
$VaultAddr = if ($env:VAULT_ADDR) { $env:VAULT_ADDR } else { "http://localhost:8200" }
$VaultToken = $env:VAULT_TOKEN
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

# Colors
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
    Write-Host "OK $Message" -ForegroundColor $Colors.Green
}

function Write-Fail {
    param([string]$Message)
    Write-Host "X $Message" -ForegroundColor $Colors.Red
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
Write-Host "=== HashiCorp Vault Backup ===" -ForegroundColor $Colors.Blue
Write-Host ""

# Check prerequisites
Write-Step "1/6" "Checking prerequisites..."

if (-not $VaultToken) {
    Write-Fail "VAULT_TOKEN not set"
    exit 1
}

# Test Vault connection
try {
    $health = Invoke-RestMethod -Uri "$VaultAddr/v1/sys/health" -Method Get -TimeoutSec 5 -ErrorAction Stop
    Write-Success "Connected to Vault at $VaultAddr"
}
catch {
    Write-Fail "Cannot connect to Vault at $VaultAddr"
    exit 1
}

# Create backup directory
Write-Step "2/6" "Creating backup directory..."
$ExportDir = Join-Path $BackupDir "vault-backup-$Timestamp"
New-Item -ItemType Directory -Path $ExportDir -Force | Out-Null
New-Item -ItemType Directory -Path "$ExportDir\secrets" -Force | Out-Null
New-Item -ItemType Directory -Path "$ExportDir\policies" -Force | Out-Null
Write-Success "Created $ExportDir"

# Backup secrets
Write-Step "3/6" "Backing up secrets..."
$SecretCount = 0

$secretList = Invoke-VaultAPI -Method GET -Path "secret/metadata?list=true" -Token $VaultToken
if ($secretList.Success -and $secretList.Data.data.keys) {
    foreach ($secretPath in $secretList.Data.data.keys) {
        $secretPath = $secretPath.TrimEnd('/')
        Write-Host "  Backing up: secret/$secretPath"

        $secret = Invoke-VaultAPI -Method GET -Path "secret/data/$secretPath" -Token $VaultToken
        if ($secret.Success) {
            $secret.Data | ConvertTo-Json -Depth 10 | Out-File "$ExportDir\secrets\$secretPath.json" -Encoding utf8
            $SecretCount++
        }
    }
}
Write-Success "Backed up $SecretCount secrets"

# Backup policies
Write-Step "4/6" "Backing up policies..."
$PolicyCount = 0

$policyList = Invoke-VaultAPI -Method GET -Path "sys/policies/acl?list=true" -Token $VaultToken
if ($policyList.Success -and $policyList.Data.data.keys) {
    foreach ($policy in $policyList.Data.data.keys) {
        if ($policy -ne "root" -and $policy -ne "default") {
            Write-Host "  Backing up policy: $policy"

            $policyData = Invoke-VaultAPI -Method GET -Path "sys/policies/acl/$policy" -Token $VaultToken
            if ($policyData.Success) {
                $policyData.Data.data.policy | Out-File "$ExportDir\policies\$policy.hcl" -Encoding utf8
                $PolicyCount++
            }
        }
    }
}
Write-Success "Backed up $PolicyCount policies"

# Create metadata
Write-Step "5/6" "Creating metadata..."
$metadata = @{
    timestamp = (Get-Date -Format "o")
    vault_addr = $VaultAddr
    secret_count = $SecretCount
    policy_count = $PolicyCount
    backup_type = "full"
    encrypted = $Encrypt.IsPresent
}
$metadata | ConvertTo-Json | Out-File "$ExportDir\metadata.json" -Encoding utf8
Write-Success "Metadata created"

# Create archive
Write-Step "6/6" "Creating archive..."
$ArchiveName = "vault-backup-$Timestamp.zip"
$ArchivePath = Join-Path $BackupDir $ArchiveName

Compress-Archive -Path $ExportDir -DestinationPath $ArchivePath -Force

# Calculate hash
$hash = Get-FileHash -Path $ArchivePath -Algorithm SHA256
$hash.Hash | Out-File "$ArchivePath.sha256" -Encoding utf8

# Encrypt if requested
if ($Encrypt) {
    Write-Host "  Encryption requested - Please use external tool (7-Zip, GPG)"
    Write-Host "  Example: 7z a -p -mhe=on $ArchivePath.7z $ArchivePath"
}

# Cleanup temp directory
Remove-Item -Path $ExportDir -Recurse -Force

# Summary
Write-Host ""
Write-Host "=== Backup Complete! ===" -ForegroundColor $Colors.Blue
Write-Host ""
Write-Host "Backup Details:" -ForegroundColor $Colors.Green
Write-Host "  File: $ArchivePath"
Write-Host "  Secrets: $SecretCount"
Write-Host "  Policies: $PolicyCount"
Write-Host "  Checksum: $ArchivePath.sha256"
Write-Host ""
Write-Host "To restore:" -ForegroundColor $Colors.Yellow
Write-Host "  .\restore-vault.ps1 -BackupFile `"$ArchivePath`""
Write-Host ""
