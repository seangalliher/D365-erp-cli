# Demo Pre-Flight Check Script
# Run this right before recording to verify everything is ready.

Write-Host "=== D365 Demo Pre-Flight Check ===" -ForegroundColor Cyan
Write-Host ""

# 1. CLI version
Write-Host "[1/5] CLI version:" -ForegroundColor Yellow
$version = d365 version 2>&1 | ConvertFrom-Json
Write-Host "  d365 $($version.data.version) ($($version.data.os))" -ForegroundColor Green
Write-Host ""

# 2. Connection status
Write-Host "[2/5] Connection status:" -ForegroundColor Yellow
$status = d365 status 2>&1 | ConvertFrom-Json
if ($status.data.connected) {
    Write-Host "  Connected to $($status.data.environment)" -ForegroundColor Green
} else {
    Write-Host "  NOT CONNECTED - run: d365 connect <url>" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 3. Token expiry
Write-Host "[3/5] Token expiry:" -ForegroundColor Yellow
$expiry = [datetime]::Parse($status.data.token_expiry)
$remaining = $expiry - (Get-Date).ToUniversalTime()
if ($remaining.TotalMinutes -gt 10) {
    Write-Host "  Token valid for $([math]::Round($remaining.TotalMinutes)) minutes" -ForegroundColor Green
} else {
    Write-Host "  Token expires in $([math]::Round($remaining.TotalMinutes)) minutes - re-authenticate!" -ForegroundColor Red
}
Write-Host ""

# 4. ACME doesn't exist
Write-Host "[4/5] Checking ACME doesn't exist:" -ForegroundColor Yellow
$check = d365 data find LegalEntities --query "`$filter=LegalEntityId eq 'ACME'&`$select=LegalEntityId" --timeout 60 2>&1 | ConvertFrom-Json
if ($check.data.value -and $check.data.value.Count -gt 0) {
    Write-Host "  ACME already exists! Run cleanup.ps1 first." -ForegroundColor Red
} else {
    Write-Host "  Clean - no ACME entity found" -ForegroundColor Green
}
Write-Host ""

# 5. Quick connectivity test
Write-Host "[5/5] Connectivity test:" -ForegroundColor Yellow
$test = d365 data find LegalEntities --query "`$top=1&`$select=LegalEntityId" --timeout 60 2>&1 | ConvertFrom-Json
if ($test.success) {
    Write-Host "  D365 responding OK ($($test.metadata.duration_ms)ms)" -ForegroundColor Green
} else {
    Write-Host "  D365 query failed: $($test.error.message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Pre-flight complete ===" -ForegroundColor Cyan
