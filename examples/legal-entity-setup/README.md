# Legal Entity Setup Example

Create a new legal entity with a fully configured general ledger — from scratch, in under a minute.

## What This Does

1. Creates a new legal entity (ACME Corp)
2. Creates a chart of accounts
3. Configures the general ledger with USD currency
4. Adds starter main accounts (Cash, Revenue, Expenses)
5. Verifies the setup

## Prerequisites

- D365 F&O environment with admin access
- Connected via `./d365 connect`

## Usage

### Option 1: Batch Mode (One Command)

```powershell
./d365 connect https://your-env.operations.dynamics.com
Get-Content batch.jsonl | ./d365 batch
```

### Option 2: Step by Step

```powershell
# Create the legal entity
./d365 data create LegalEntities --data '{"LegalEntityId":"ACME","Name":"ACME Corp","CompanyType":"Organization","AddressCountryRegion":"USA"}' --timeout 60

# Switch to the new company
./d365 company set ACME

# Create chart of accounts
./d365 data create LedgerChartOfAccounts --data '{"ChartOfAccounts":"ACME-COA","Description":"ACME Corp Chart of Accounts"}' --timeout 60

# Configure the general ledger
./d365 data create Ledgers --data '{"ChartOfAccounts":"ACME-COA","Name":"ACME General Ledger","AccountingCurrency":"USD","ReportingCurrency":"USD","FiscalCalendar":"Standard"}' --timeout 60

# Add main accounts
./d365 data create MainAccounts --data '{"MainAccountId":"110100","Name":"Cash - Operating","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
```

### Option 3: AI Agent (GitHub Copilot)

Copy the contents of [`copilot-prompt.md`](copilot-prompt.md) into your GitHub Copilot chat. The agent will use the D365 CLI to execute each step, verify results, and handle errors automatically.

The prompt includes:
- Exact entity set names (no guessing needed)
- Check-before-create pattern to prevent duplicates
- Verify-after-create to confirm success
- Error recovery guidance

## Demo

### Pre-flight & Cleanup Scripts

Located in [`demo/`](demo/):

- **`preflight.ps1`** — Validates CLI, connection, token, and clean state before recording
- **`cleanup.ps1`** — Removes all ACME test data (legal entity, ledger, accounts)
- **`DEMO-PREP.md`** — Full demo script with intro narration and recording tips

```powershell
# Before recording
./examples/legal-entity-setup/demo/preflight.ps1

# After recording (or to reset)
./examples/legal-entity-setup/demo/cleanup.ps1
```

## Customization

Edit the prompt or JSON payloads to match your organization:

- **Company ID** — Change `ACME` to your company code
- **Chart of Accounts** — Change `ACME-COA` to your COA ID
- **Currency** — Change `USD` to your accounting/reporting currency
- **Main Accounts** — Add/remove accounts to match your GL structure
