# Command Reference

Complete reference for all `d365` CLI commands.

## Connection

```bash
d365 connect <url>           # Connect with interactive browser auth
d365 connect <url> --auth device-code   # Device-code flow (no browser)
d365 connect <url> --auth client-credentials --client-id <id> --client-secret <secret>
d365 connect <url> --auth az-cli        # Use existing Azure CLI session
d365 connect <url> --auth managed-identity  # Managed identity (Azure VMs)
d365 disconnect              # End session
d365 status                  # Show connection status, token expiry
d365 company get             # Get current company context
d365 company set USMF        # Switch company context
```

## Data (OData)

All data commands support `--timeout <seconds>` (default: 30).

### find-type — Discover entity types

```bash
d365 data find-type customer             # Single-word search
d365 data find-type "chart of accounts"  # Multi-word search (PascalCase-aware)
d365 data find-type ledger --top 5       # Limit results
```

### metadata — Inspect entity schema

```bash
d365 data metadata Customers                    # Basic field list
d365 data metadata Customers --keys             # Include key fields
d365 data metadata Customers --constraints      # Include field constraints
d365 data metadata Customers --enums            # Include enum values
```

### find — Query entities

> **PowerShell users:** Always use **single quotes** for `--query` values to prevent `$` variable expansion.

```bash
d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'
d365 data find Customers --query '$filter=Name eq ''Contoso''&$select=CustomerAccount,Name'
d365 data find SalesOrderHeaders --query '$expand=SalesOrderLines&$top=3'
```

### create — Create an entity

```bash
d365 data create Customers --data '{"CustomerAccount":"CUST-001","Name":"New Customer","CustomerGroupId":"10"}'
d365 data create LegalEntities --data '{"LegalEntityId":"ACME","Name":"ACME Corp","CompanyType":"Organization"}'
```

### update — Update entities

```bash
d365 data update --data '[{"ODataPath":"Customers(dataAreaId='\''USMF'\'',CustomerAccount='\''CUST-001'\'')","UpdatedFieldValues":{"Name":"Updated Name"}}]'
```

### delete — Delete entities

```bash
d365 data delete --paths '["Customers(dataAreaId='\''USMF'\'',CustomerAccount='\''CUST-001'\'')"]' --confirm
```

The `--confirm` flag is required. Without it, the command is blocked by the delete-confirm guardrail.

## API (Actions)

```bash
d365 api find <search>                # Find available actions by name
d365 api invoke <action> --params '{"key":"value"}'  # Invoke an action
```

## Forms (Stateful)

Form commands drive the actual D365 web client via a Playwright-powered background daemon. The daemon launches a Chromium browser, authenticates to your D365 environment, and executes form interactions exactly as a human user would — clicking buttons, typing into fields, and reading form state from the DOM.

**How it works:**
1. The daemon auto-starts on first `d365 form` command
2. On first run, a visible browser opens for AAD login (SSO cookies are cached for subsequent runs)
3. Each CLI command sends a request to the daemon via IPC, which drives the browser
4. Form state (controls, values, grids) is extracted from the D365 DOM and returned as JSON

```bash
# Navigation
d365 form find <search>               # Find menu items by name
d365 form open CustTable --type Display   # Open a form
d365 form close                        # Close current form
d365 form save                         # Save current form
d365 form state                        # Get all form controls and values

# Interaction
d365 form click <control>             # Click a button or control
d365 form set Name=Value              # Set field values
d365 form set Name=Value Other=Value2 # Set multiple fields
d365 form lookup <control>            # Open a lookup dropdown
d365 form tab <tab> --action Open     # Open/close a tab
d365 form find-controls <search>      # Search for controls by name

# Grid operations
d365 form filter <control> <value>    # Filter form by control
d365 form grid-filter <col> <val> --grid Grid1     # Filter grid column
d365 form grid-select <row> --grid Grid1           # Select grid row
d365 form grid-sort <col> --grid Grid1 --direction Descending
```

### Example: Create a Customer via Form

```bash
# Start daemon (auto-starts on first form command)
d365 daemon start

# Open the Customers form
d365 form open CustTable --type Display

# Click New to open the create dialog
d365 form click NewCustomer

# Fill in the dialog fields
d365 form set DynamicHeader_AccountNum="CUST001"
d365 form set Org_Name="Contoso Ltd"
d365 form set DynamicDetail_CustGroup="10"

# Save (clicks the Save button on the dialog)
d365 form save

# Verify via OData
d365 data find Customers --query '$filter=CustomerAccount eq ''CUST001'''
```

### Cookie Caching

The daemon caches D365 session cookies to `~/.d365cli/session-cookies.json`. On subsequent daemon starts, these cookies are restored so the browser can skip the AAD login flow entirely — reducing startup from ~2 minutes to ~5 seconds.

## Utilities

```bash
d365 quickstart              # Interactive guided setup for new users
d365 doctor                  # Diagnostic health checks (config, auth, DNS, connectivity)
d365 agent-prompt            # Generate AI agent system prompt (markdown)
d365 agent-prompt --json     # Same, wrapped in JSON envelope
d365 completion powershell   # Generate shell completion script
d365 schema                  # Export CLI schema for AI tool registration
d365 schema --full           # Full schema with examples
d365 docs <topic>            # Built-in documentation (topics: odata, auth, guardrails, entities)
d365 batch                   # JSONL batch mode (reads from stdin)
d365 daemon start            # Start form daemon manually
d365 daemon stop             # Stop form daemon
d365 daemon status           # Check daemon status
d365 daemon restart          # Restart daemon
d365 version                 # Show version, Go, OS info
```

## Global Flags

Available on every command:

```
-o, --output   Output format: json, table, csv, raw (default: auto-detected by TTY)
    --company  Company/legal entity override (e.g., USMF)
    --profile  Configuration profile
-q, --quiet    Suppress non-essential output
    --no-color Disable colored output
-v, --verbose  Verbose logging to stderr
    --ci       CI mode (implies --output json --quiet --no-color)
    --timeout  Request timeout in seconds (default: 30)
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Command error (bad request, entity not found, etc.) |
| 2 | Connection/auth/timeout error |
| 3 | Validation/input error |
