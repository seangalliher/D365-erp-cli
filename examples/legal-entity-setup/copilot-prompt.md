# D365 Legal Entity Setup

You have access to the `d365` CLI tool for managing Dynamics 365 Finance & Operations.
Use `./d365` (or `d365` if it's on your PATH) to run commands.

## Important Rules

- **Always use single quotes** for `--query` values (OData `$` parameters conflict with shell variable expansion)
- **Creates are NOT idempotent** — always query first to check if a record already exists before creating it
- **Verify after every create** — re-query the record to confirm it was created successfully
- **Use the exact entity set names** listed below — do not guess or search for them
- **Include `--timeout 60`** on data commands — D365 can be slow on first requests

## Entity Reference

Use these exact entity set names — no need to call `find-type`:

| Entity Set | Description | Key Fields |
|-----------|-------------|------------|
| LegalEntities | Legal entities / companies | LegalEntityId |
| LedgerChartOfAccounts | Chart of accounts definitions | ChartOfAccounts |
| Ledgers | Ledger configuration per company | dataAreaId |
| MainAccounts | Chart of accounts main accounts | dataAreaId, MainAccountId, ChartOfAccounts |

## Task

Create a new legal entity called **ACME Corp** with the company ID **ACME**, then set up its general ledger with a basic chart of accounts.

## Steps

### 1. Create the legal entity

First check it doesn't exist:
```
./d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId,Name' --timeout 60
```

If the result `value` array is empty, create it:
```
./d365 data create LegalEntities --data '{"LegalEntityId":"ACME","Name":"ACME Corp","CompanyType":"Organization","AddressCountryRegion":"USA"}' --timeout 60
```

Verify it was created:
```
./d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId,Name' --timeout 60
```

### 2. Switch to the new company

```
./d365 company set ACME
```

### 3. Create a chart of accounts

```
./d365 data create LedgerChartOfAccounts --data '{"ChartOfAccounts":"ACME-COA","Description":"ACME Corp Chart of Accounts"}' --timeout 60
```

Verify:
```
./d365 data find LedgerChartOfAccounts --query '$filter=ChartOfAccounts eq ''ACME-COA''&$select=ChartOfAccounts,Description' --timeout 60
```

### 4. Configure the general ledger

```
./d365 data create Ledgers --data '{"ChartOfAccounts":"ACME-COA","Name":"ACME General Ledger","AccountingCurrency":"USD","ReportingCurrency":"USD","FiscalCalendar":"Standard"}' --timeout 60
```

### 5. Create main accounts

Create each account one at a time. All accounts use `ChartOfAccounts: ACME-COA`.

| Account ID | Name                | Type         |
|-----------|---------------------|--------------|
| 110100    | Cash - Operating     | BalanceSheet |
| 120100    | Accounts Receivable  | BalanceSheet |
| 200100    | Accounts Payable     | BalanceSheet |
| 300100    | Retained Earnings    | BalanceSheet |
| 400100    | Revenue              | Revenue      |
| 500100    | Cost of Goods Sold   | Expense      |
| 600100    | Operating Expenses   | Expense      |
| 610100    | Payroll Expenses     | Expense      |

Example for the first account:
```
./d365 data create MainAccounts --data '{"MainAccountId":"110100","Name":"Cash - Operating","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
```

### 6. Verify the setup

Query all accounts under the chart of accounts:
```
./d365 data find MainAccounts --query '$filter=ChartOfAccounts eq ''ACME-COA''&$select=MainAccountId,Name,MainAccountType' --timeout 60
```

Expected: 8 main accounts returned.

## Error Recovery

- If a create returns an error, read the error message carefully — it often tells you exactly what's wrong
- If you get a 409 Conflict, the record already exists — move to the next step
- If you get a timeout, retry with `--timeout 120`
- Use `./d365 data metadata <EntitySet>` to inspect available fields if needed
