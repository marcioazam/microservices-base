# Auth Platform - Structure Validation Script
# Validates monorepo structure follows state-of-the-art 2025 patterns

$ErrorActionPreference = "Stop"

Write-Host "Validating Auth Platform monorepo structure..." -ForegroundColor Cyan

$requiredDirs = @(
    "api/proto/auth",
    "api/proto/infra",
    "deploy/docker",
    "deploy/kubernetes/gateway",
    "deploy/kubernetes/helm",
    "docs/adr",
    "docs/api",
    "docs/runbooks",
    "libs/go",
    "libs/rust",
    "platform",
    "sdk",
    "services",
    "tools/scripts"
)

$requiredFiles = @(
    "README.md",
    "CONTRIBUTING.md",
    "CHANGELOG.md",
    "Makefile",
    ".github/workflows/ci.yml"
)

$services = @(
    "services/auth-edge",
    "services/token",
    "services/session-identity",
    "services/iam-policy",
    "services/mfa"
)

$errors = @()

# Check required directories
Write-Host "`nChecking required directories..." -ForegroundColor Yellow
foreach ($dir in $requiredDirs) {
    if (Test-Path $dir) {
        Write-Host "  [OK] $dir" -ForegroundColor Green
    } else {
        Write-Host "  [MISSING] $dir" -ForegroundColor Red
        $errors += "Missing directory: $dir"
    }
}

# Check required files
Write-Host "`nChecking required files..." -ForegroundColor Yellow
foreach ($file in $requiredFiles) {
    if (Test-Path $file) {
        Write-Host "  [OK] $file" -ForegroundColor Green
    } else {
        Write-Host "  [MISSING] $file" -ForegroundColor Red
        $errors += "Missing file: $file"
    }
}

# Check services have README
Write-Host "`nChecking services..." -ForegroundColor Yellow
foreach ($svc in $services) {
    if (Test-Path "$svc/README.md") {
        Write-Host "  [OK] $svc (has README)" -ForegroundColor Green
    } else {
        Write-Host "  [WARN] $svc (missing README)" -ForegroundColor Yellow
    }
}

# Check for anti-patterns
Write-Host "`nChecking for anti-patterns..." -ForegroundColor Yellow

# Rust code in Go libs
$rustInGo = Get-ChildItem -Path "libs/go" -Filter "*.rs" -Recurse -ErrorAction SilentlyContinue
if ($rustInGo) {
    Write-Host "  [FAIL] Rust files found in libs/go/" -ForegroundColor Red
    $errors += "Rust files in libs/go/ - should be in libs/rust/"
} else {
    Write-Host "  [OK] No Rust files in libs/go/" -ForegroundColor Green
}

# Summary
Write-Host "`n==================================================" -ForegroundColor Cyan
if ($errors.Count -eq 0) {
    Write-Host "[PASS] Structure validation PASSED" -ForegroundColor Green
    exit 0
} else {
    Write-Host "[FAIL] Structure validation FAILED with $($errors.Count) error(s)" -ForegroundColor Red
    foreach ($e in $errors) {
        Write-Host "  - $e" -ForegroundColor Red
    }
    exit 1
}
