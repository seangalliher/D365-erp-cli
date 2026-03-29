# Demo Cleanup Script
# Run this to remove ACME entity data before re-recording the demo.
# Requires: d365 connected to your D365 environment

Write-Host "=== D365 Demo Cleanup ===" -ForegroundColor Cyan
Write-Host ""

# Switch to ACME company first (needed to delete company-scoped data)
Write-Host "[1/4] Switching to ACME company..." -ForegroundColor Yellow
d365 company set ACME

# Delete main accounts
Write-Host "[2/4] Deleting main accounts..." -ForegroundColor Yellow
$accounts = @("110100", "120100", "200100", "300100", "400100", "500100", "600100", "610100")
foreach ($acct in $accounts) {
    Write-Host "  Deleting MainAccount $acct"
    d365 data delete --paths "[""MainAccounts(dataAreaId='ACME',MainAccountId='$acct',ChartOfAccounts='ACME-COA')""]" --confirm 2>$null
}

# Delete ledger
Write-Host "[3/4] Deleting ledger and chart of accounts..." -ForegroundColor Yellow
d365 data delete --paths "[""Ledgers(dataAreaId='ACME')""]" --confirm 2>$null
d365 data delete --paths "[""LedgerChartOfAccounts(ChartOfAccounts='ACME-COA')""]" --confirm 2>$null

# Switch back to default company and delete legal entity
Write-Host "[4/4] Deleting ACME legal entity..." -ForegroundColor Yellow
d365 company set ""
d365 data delete --paths "[""LegalEntities(LegalEntityId='ACME')""]" --confirm 2>$null

Write-Host ""
Write-Host "=== Cleanup complete ===" -ForegroundColor Green
Write-Host "Verify with: d365 data find LegalEntities --query `"`$filter=LegalEntityId eq 'ACME'&`$select=LegalEntityId,Name`"" --timeout 60"
