# Demo Cleanup Script
# Run this to remove ACME entity data before re-recording the demo.
# Requires: d365 connected to your D365 environment
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

Write-Host "=== D365 Demo Cleanup (using $d365) ===" -ForegroundColor Cyan
Write-Host ""

# Switch to ACME company first (needed to delete company-scoped data)
Write-Host "[1/4] Switching to ACME company..." -ForegroundColor Yellow
& $d365 company set ACME

# Delete main accounts
Write-Host "[2/4] Deleting main accounts..." -ForegroundColor Yellow
$accounts = @("110100", "120100", "200100", "210100", "300100", "310100", "400100", "410100", "500100", "510100", "600100", "610100", "620100")
foreach ($acct in $accounts) {
    Write-Host "  Deleting MainAccount $acct"
    & $d365 data delete --paths "[""MainAccounts(dataAreaId='ACME',MainAccountId='$acct',ChartOfAccounts='ACME-COA')""]" --confirm 2>$null
}

# Delete ledger
Write-Host "[3/4] Deleting ledger and chart of accounts..." -ForegroundColor Yellow
& $d365 data delete --paths "[""Ledgers(dataAreaId='ACME')""]" --confirm 2>$null
& $d365 data delete --paths "[""LedgerChartOfAccounts(ChartOfAccounts='ACME-COA')""]" --confirm 2>$null

# Switch back to default company and delete legal entity
Write-Host "[4/4] Deleting ACME legal entity..." -ForegroundColor Yellow
& $d365 company set ""
& $d365 data delete --paths "[""LegalEntities(LegalEntityId='ACME')""]" --confirm 2>$null

Write-Host ""
Write-Host "=== Cleanup complete ===" -ForegroundColor Green
Write-Host "Verify with: $d365 data find LegalEntities --query '`$filter=LegalEntityId eq ''ACME''&`$select=LegalEntityId,Name' --timeout 60"
