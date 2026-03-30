# Demo Pre-Flight Check Script
# Run this right before recording to verify everything is ready.
#
# Uses $d365 variable so it works whether d365 is on PATH or run locally.
# Override: $d365 = "./d365" or $d365 = "d365"

if (-not $d365) {
    if (Get-Command d365 -ErrorAction SilentlyContinue) {
        $d365 = "d365"
    } elseif (Test-Path "./d365.exe") {
        $d365 = "./d365.exe"
    } elseif (Test-Path "./d365") {
        $d365 = "./d365"
    } else {
        Write-Host "ERROR: d365 binary not found on PATH or in current directory" -ForegroundColor Red
        exit 1
    }
}

Write-Host "=== D365 Demo Pre-Flight Check (using $d365) ===" -ForegroundColor Cyan
Write-Host ""

# 1. CLI version
Write-Host "[1/6] CLI version:" -ForegroundColor Yellow
$version = & $d365 version 2>&1 | ConvertFrom-Json
Write-Host "  d365 $($version.data.version) ($($version.data.os))" -ForegroundColor Green
Write-Host ""

# 2. Connection status
Write-Host "[2/6] Connection status:" -ForegroundColor Yellow
$status = & $d365 status 2>&1 | ConvertFrom-Json
if ($status.data.connected) {
    Write-Host "  Connected to $($status.data.environment)" -ForegroundColor Green
} else {
    Write-Host "  NOT CONNECTED - run: $d365 connect <url>" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 3. Token expiry
Write-Host "[3/6] Token expiry:" -ForegroundColor Yellow
$expiry = [datetime]::Parse($status.data.token_expiry)
$remaining = $expiry - (Get-Date).ToUniversalTime()
if ($remaining.TotalMinutes -gt 10) {
    Write-Host "  Token valid for $([math]::Round($remaining.TotalMinutes)) minutes" -ForegroundColor Green
} else {
    Write-Host "  Token expires in $([math]::Round($remaining.TotalMinutes)) minutes - re-authenticate!" -ForegroundColor Red
}
Write-Host ""

# 4. Current company context
Write-Host "[4/6] Company context:" -ForegroundColor Yellow
$company = & $d365 company get 2>&1 | ConvertFrom-Json
if ($company.data.company -and $company.data.company -ne "") {
    Write-Host "  Current company: $($company.data.company) (will reset for demo)" -ForegroundColor Yellow
    & $d365 company set "" | Out-Null
    Write-Host "  Company context cleared" -ForegroundColor Green
} else {
    Write-Host "  No company set (good - clean state)" -ForegroundColor Green
}
Write-Host ""

# 5. ACME doesn't exist
Write-Host "[5/6] Checking ACME doesn't exist:" -ForegroundColor Yellow
$check = & $d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId' --timeout 60 2>&1 | ConvertFrom-Json
if ($check.data.value -and $check.data.value.Count -gt 0) {
    Write-Host "  ACME already exists! Run cleanup.ps1 first." -ForegroundColor Red
} else {
    Write-Host "  Clean - no ACME entity found" -ForegroundColor Green
}
Write-Host ""

# 6. Quick connectivity test
Write-Host "[6/6] Connectivity test:" -ForegroundColor Yellow
$test = & $d365 data find LegalEntities --query '$top=1&$select=LegalEntityId' --timeout 60 2>&1 | ConvertFrom-Json
if ($test.success) {
    Write-Host "  D365 responding OK ($($test.metadata.duration_ms)ms)" -ForegroundColor Green
} else {
    Write-Host "  D365 query failed: $($test.error.message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Pre-flight complete ===" -ForegroundColor Cyan
