# AI Agent Integration

D365 CLI is designed as the primary interface for AI agents interacting with Dynamics 365 Finance & Operations. Every command returns structured JSON, making it easy for agents to parse results, handle errors, and chain operations.

## Getting Started with an AI Agent

### 1. Generate the System Prompt

```bash
d365 agent-prompt            # Raw markdown — paste into agent config
d365 agent-prompt --json     # JSON envelope for programmatic use
```

The system prompt includes:
- Complete command reference with syntax and examples
- Entity catalog (24 common D365 entities with key fields)
- Guardrail descriptions so the agent understands safety rules
- Common workflow patterns (check-before-create, verify-after-create)

### 2. Register CLI Tools

Export the full CLI schema for tool registration:

```bash
d365 schema --full | jq .                                       # bash + jq
d365 schema --full | ConvertFrom-Json | ConvertTo-Json -Depth 10 # PowerShell
```

The schema includes command names, arguments, flags, descriptions, and examples — everything an AI tool registry needs.

## Example: Creating a Legal Entity with GitHub Copilot

> **Prompt:** *"Create a new legal entity called ACME Corp with ID ACME, then set up its chart of accounts and general ledger."*

Copilot uses the D365 CLI to execute each step automatically:

```bash
# Step 1 — Check if ACME already exists (creates are NOT idempotent)
$ d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId,Name' --timeout 60
✓ Empty result — safe to create

# Step 2 — Create the legal entity
$ d365 data create LegalEntities --data '{
    "LegalEntityId": "ACME",
    "Name": "ACME Corp",
    "CompanyType": "Organization",
    "AddressCountryRegion": "USA"
  }' --timeout 60
✓ Created LegalEntities(dataAreaId='ACME')

# Step 3 — Verify it was created
$ d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId,Name' --timeout 60
✓ Found: ACME — ACME Corp

# Step 4 — Switch context to the new company
$ d365 company set ACME
✓ Company set to ACME

# Step 5 — Create chart of accounts
$ d365 data create LedgerChartOfAccounts --data '{
    "ChartOfAccounts": "ACME-COA",
    "Description": "ACME Corp Chart of Accounts"
  }' --timeout 60
✓ Created LedgerChartOfAccounts('ACME-COA')

# Step 6 — Configure the general ledger
$ d365 data create Ledgers --data '{
    "ChartOfAccounts": "ACME-COA",
    "Name": "ACME General Ledger",
    "AccountingCurrency": "USD",
    "ReportingCurrency": "USD",
    "FiscalCalendar": "Standard"
  }' --timeout 60
✓ Created Ledgers for ACME

# Step 7 — Create main accounts
$ d365 data create MainAccounts --data '{"MainAccountId":"110100","Name":"Cash - Operating","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"120100","Name":"Accounts Receivable","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"200100","Name":"Accounts Payable","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"300100","Name":"Retained Earnings","MainAccountType":"BalanceSheet","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"400100","Name":"Revenue","MainAccountType":"Revenue","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"500100","Name":"Cost of Goods Sold","MainAccountType":"Expense","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"600100","Name":"Operating Expenses","MainAccountType":"Expense","ChartOfAccounts":"ACME-COA"}' --timeout 60
$ d365 data create MainAccounts --data '{"MainAccountId":"610100","Name":"Payroll Expenses","MainAccountType":"Expense","ChartOfAccounts":"ACME-COA"}' --timeout 60

# Step 8 — Verify the setup
$ d365 data find MainAccounts --query '$filter=ChartOfAccounts eq ''ACME-COA''&$select=MainAccountId,Name,MainAccountType' --timeout 60
✓ 8 accounts created
```

Every step returns structured JSON, so the agent can inspect results, handle errors, and chain operations — no screenshots, no clicking, no guesswork.

> **Try it yourself:** See [`examples/legal-entity-setup/`](../examples/legal-entity-setup/) for ready-to-use prompts, batch scripts, and demo tooling.

## Standard JSON Envelope

Every command returns this structure:

```json
{
  "success": true,
  "command": "d365 data find",
  "data": { ... },
  "error": null,
  "metadata": {
    "duration_ms": 142,
    "company": "USMF",
    "timestamp": "2026-01-01T00:00:00Z"
  }
}
```

On error:

```json
{
  "success": false,
  "command": "d365 data create",
  "data": null,
  "error": {
    "code": "ODATA_ERROR",
    "message": "Resource not found for the segment 'InvalidEntity'.",
    "suggestion": "Check the entity name with: d365 data find-type <search>"
  },
  "metadata": { ... }
}
```

Agents should always check `success` before processing `data`.

## Batch/Pipeline Mode

Send multiple commands via JSONL on stdin:

```bash
echo '{"command":"data find","args":{"entity":"Customers","query":"$top=3"}}
{"command":"data find","args":{"entity":"Vendors","query":"$top=3"}}' | d365 batch
```

PowerShell:
```powershell
@'
{"command":"data find","args":{"entity":"Customers","query":"$top=3"}}
{"command":"data find","args":{"entity":"Vendors","query":"$top=3"}}
'@ | d365 batch
```

## Guardrails

Built-in safety checks that protect AI agents from common mistakes:

| Rule | What it does |
|------|-------------|
| **cross-company** | Auto-injects `cross-company=true` when filtering by `dataAreaId` |
| **select-recommended** | Warns when `$select` is missing from queries |
| **delete-confirm** | Blocks deletes without `--confirm` flag |
| **wide-query** | Warns on queries without `$top` or `$filter` |
| **enum-format** | Warns when numeric values are used for enum fields |

Guardrails emit warnings in the JSON response `metadata.warnings` array so agents can self-correct.

## Best Practices for AI Agents

1. **Check before create** — D365 creates are NOT idempotent. Always query first.
2. **Verify after create** — Re-query to confirm the record was created successfully.
3. **Use `--timeout 60`** — D365 can be slow, especially on first requests.
4. **Use single quotes** for `--query` values to avoid shell variable expansion.
5. **Use `$select`** — Only request the fields you need. Reduces payload and avoids guardrail warnings.
6. **Read error suggestions** — The `error.suggestion` field tells the agent what to do next.
