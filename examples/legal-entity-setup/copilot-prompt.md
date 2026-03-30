# D365 Legal Entity Setup

You have access to the `d365` CLI tool for managing Dynamics 365 Finance & Operations.
Use `./d365` (or `d365` if it's on your PATH) to run commands.

## Important Rules

- **Use ONLY the `./d365` CLI tool** — do NOT use MCP tools, MCP servers, or any other D365 integration. All operations must go through `./d365` commands.
- **Always use single quotes** for `--query` values (OData `$` parameters conflict with shell variable expansion). To embed OData string literals, use `'"'"'value'"'"'` (bash) — do NOT use `''` (that's PowerShell syntax).
- **Creates are NOT idempotent** — always query first to check if a record already exists before creating it
- **Verify after every create** — re-query the record to confirm it was created successfully
- **Use the exact entity set names** listed below — do not guess or search for them
- **Include `--timeout 60`** on data commands — D365 can be slow on first requests
- **Use `--keys` when checking metadata** — `./d365 data metadata <Entity> --keys` returns only key fields, avoiding large responses

## Entity Reference

Use these exact entity set names — no need to call `find-type`:

| Entity Set | Description | Key Fields |
|-----------|-------------|------------|
| LegalEntities | Legal entities / companies | LegalEntityId |
| ChartOfAccounts | Chart of accounts definitions | ChartOfAccounts |
| Ledgers | Ledger configuration per company | LegalEntityId |
| MainAccounts | Chart of accounts main accounts | dataAreaId, MainAccountId, ChartOfAccounts |

## Task

Create a new legal entity called **ACME Corp** with the company ID **ACME**, then set up its general ledger with a basic chart of accounts.

## Steps

### 1. Create the legal entity

First check it doesn't exist:
```
./d365 data find LegalEntities --query '$filter=LegalEntityId eq '"'"'ACME'"'"'&$select=LegalEntityId,Name' --timeout 60
```

If the result `value` array is empty, create it (use minimal fields only — extra fields like CompanyType or AddressCountryRegion cause errors):
```
./d365 data create LegalEntities --data '{"LegalEntityId":"ACME","Name":"ACME Corp"}' --timeout 60
```

Verify it was created:
```
./d365 data find LegalEntities --query '$filter=LegalEntityId eq '"'"'ACME'"'"'&$select=LegalEntityId,Name' --timeout 60
```

### 2. Switch to the new company

```
./d365 company set ACME
```

### 3. Create a chart of accounts

```
./d365 data create ChartOfAccounts --data '{"ChartOfAccounts":"ACME-COA","Description":"ACME Corp Chart of Accounts"}' --timeout 60
```

Verify:
```
./d365 data find ChartOfAccounts --query '$filter=ChartOfAccounts eq '"'"'ACME-COA'"'"'&$select=ChartOfAccounts,Description' --timeout 60
```

### 4. Configure the general ledger

The Ledgers entity requires `LegalEntityId` explicitly. The `Name` field is read-only and cannot be set on insert.
```
./d365 data create Ledgers --data '{"LegalEntityId":"ACME","ChartOfAccounts":"ACME-COA","AccountingCurrency":"USD","ReportingCurrency":"USD","FiscalCalendar":"Standard"}' --timeout 60
```

### 5. Create main accounts

Create each account one at a time. All accounts use `ChartOfAccounts: ACME-COA`.

| Account ID | Name                | Type       |
|-----------|---------------------|------------|
| 110100    | Cash                 | Asset      |
| 120100    | Accounts Receivable  | Asset      |
| 210100    | Accounts Payable     | Liability  |
| 310100    | Retained Earnings    | Equity     |
| 410100    | Revenue              | Revenue    |
| 510100    | Cost of Goods Sold   | Expense    |
| 610100    | Operating Expenses   | Expense    |
| 620100    | Payroll Expense      | Expense    |

Example for the first account:
```
./d365 data create MainAccounts --data '{"MainAccountId":"110100","Name":"Cash","MainAccountType":"Asset","ChartOfAccounts":"ACME-COA"}' --timeout 60
```

### 6. Verify the setup

Query all accounts under the chart of accounts:
```
./d365 data find MainAccounts --query '$filter=ChartOfAccounts eq '"'"'ACME-COA'"'"'&$select=MainAccountId,Name,MainAccountType' --timeout 60
```

Expected: 8 main accounts returned.

## Error Recovery

- If a create returns an error, read the error message carefully — it often tells you exactly what's wrong
- If you get a 409 Conflict, the record already exists — move to the next step
- If you get a timeout, retry with `--timeout 120`
- Use `./d365 data metadata <EntitySet> --keys` to inspect key fields (use `--keys` to keep the response small)
