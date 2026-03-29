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
- Connected via `d365 connect`

## Usage

### Option 1: Batch Mode (One Command)

```bash
d365 connect https://your-env.operations.dynamics.com
cat batch.jsonl | d365 batch
```

### Option 2: Step by Step

```bash
# Create the legal entity
d365 data create LegalEntities --data @legal-entity.json

# Switch to the new company
d365 company set ACME

# Create chart of accounts
d365 data create LedgerChartOfAccounts --data @chart-of-accounts.json

# Configure the general ledger
d365 data create Ledgers --data @ledger.json

# Add main accounts (one per line in the file)
d365 data create MainAccounts --data @main-accounts.json
```

### Option 3: AI Agent (GitHub Copilot)

Copy the contents of [`copilot-prompt.md`](copilot-prompt.md) into your GitHub Copilot chat. The agent will use the D365 CLI to execute each step, inspect results, and handle errors automatically.

## Customization

Edit the JSON files to match your organization:

- **`legal-entity.json`** — Change `LegalEntityId`, `Name`, and `AddressCountryRegion`
- **`chart-of-accounts.json`** — Change `ChartOfAccounts` ID and `Description`
- **`ledger.json`** — Change currencies, fiscal calendar
- **`main-accounts.json`** — Add/remove accounts to match your GL structure

## Demo

<!-- TODO: Replace with actual video link -->
[![Demo Video](https://img.shields.io/badge/Demo-Watch%20Video-red?style=for-the-badge&logo=youtube)](https://github.com/seangalliher/D365-erp-cli)
