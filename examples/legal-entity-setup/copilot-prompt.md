# D365 Legal Entity Setup

You have access to the `d365` CLI tool for managing Dynamics 365 Finance & Operations.

## Task

Create a new legal entity called **ACME Corp** with the company ID **ACME**, then set up its general ledger with a basic chart of accounts.

## Steps

1. **Create the legal entity** using `d365 data create LegalEntities` with:
   - LegalEntityId: ACME
   - Name: ACME Corp
   - CompanyType: Organization
   - AddressCountryRegion: USA

2. **Switch to the new company** using `d365 company set ACME`

3. **Create a chart of accounts** using `d365 data create LedgerChartOfAccounts` with:
   - ChartOfAccounts: ACME-COA
   - Description: ACME Corp Chart of Accounts

4. **Configure the general ledger** using `d365 data create Ledgers` with:
   - ChartOfAccounts: ACME-COA
   - Name: ACME General Ledger
   - AccountingCurrency: USD
   - ReportingCurrency: USD
   - FiscalCalendar: Standard

5. **Create main accounts** using `d365 data create MainAccounts` for each:

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

6. **Verify** the setup by querying: `d365 data find MainAccounts --query "$filter=ChartOfAccounts eq 'ACME-COA'&$select=MainAccountId,Name,MainAccountType"`

## Notes

- If any step fails, check the error response and fix the issue before continuing
- Use `d365 data metadata <entity>` if you need to inspect available fields
- Use `d365 data find-type <search>` to discover entity names
