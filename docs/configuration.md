# Configuration

## Getting Started

### Step 1: Connect

The simplest way to get started is with Azure CLI — if you already have it installed:

```bash
az login
d365 connect https://your-env.operations.dynamics.com
```

That's it. The CLI reuses your existing Azure CLI credentials by default — no extra flags needed.

### Step 2: Run a command

```bash
d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'
```

PowerShell users: always use **single quotes** for `--query` values so `$select`, `$filter`, etc. are not treated as PowerShell variables.

### Step 3: Explore

```bash
d365 data find-type "sales order"   # Find entity names
d365 data metadata SalesOrders      # See available fields
d365 doctor                         # Run diagnostics if something seems wrong
```

---

## Authentication Methods

| Method | Flag | Use Case | Extra Setup |
|--------|------|----------|-------------|
| Azure CLI | `--auth az-cli` **(default)** | Getting started, interactive dev | Just run `az login` first |
| Browser | `--auth browser` | Interactive browser popup | Requires `--tenant`, `--client-id` |
| Device Code | `--auth device-code` | Remote / SSH sessions | Requires `--tenant`, `--client-id` |
| Client Credentials | `--auth client-credentials` | CI/CD, automation, service principals | Requires `--tenant`, `--client-id`, `--client-secret` |
| Managed Identity | `--auth managed-identity` | Azure VMs, App Service | No flags needed |

### Azure CLI (default — recommended for getting started)

```bash
az login
d365 connect https://your-env.operations.dynamics.com
```

Nothing else needed. If you have the Azure CLI installed and logged in, this works out of the box.

### Client Credentials (CI/CD and automation)

```bash
d365 connect https://your-env.operations.dynamics.com \
  --auth client-credentials \
  --tenant <azure-ad-tenant-id> \
  --client-id <app-client-id> \
  --client-secret <secret>
```

Or via environment variables:

```bash
export D365_AUTH_METHOD=client-credentials
export D365_CLIENT_ID=your-client-id
export D365_CLIENT_SECRET=your-secret
export D365_TENANT_ID=your-tenant-id
d365 connect https://your-env.operations.dynamics.com
```

## How Auth Works After Connect

When you run `d365 connect`, the CLI saves your connection settings so subsequent commands work automatically:

| Saved to profile | NOT saved (security) |
|-------------------|----------------------|
| Auth method | Client secret |
| Tenant ID | |
| Client ID | |
| Environment URL | |

**If you use `client-credentials` auth**, you must also set the `D365_CLIENT_SECRET` environment variable — the CLI will not save secrets to disk. You'll see a reminder after connecting.

```powershell
# PowerShell
$env:D365_CLIENT_SECRET = "your-secret"
```

```bash
# Bash / Linux / macOS
export D365_CLIENT_SECRET="your-secret"
```

Once the environment variable is set, all subsequent commands (`data find`, `data create`, etc.) will authenticate automatically.

---

## Config File

D365 CLI stores configuration in a JSON file:

```
~/.d365cli/config.json              # Linux / macOS
$env:USERPROFILE\.d365cli\config.json   # Windows / PowerShell
```

The config file is created automatically on first `d365 connect`. You can also create it manually:

```json
{
  "default_profile": "dev",
  "profiles": {
    "dev": {
      "environment": "https://dev.operations.dynamics.com",
      "auth_method": "az-cli",
      "company": "USMF"
    },
    "prod": {
      "environment": "https://prod.operations.dynamics.com",
      "auth_method": "client-credentials",
      "client_id": "your-client-id",
      "tenant_id": "your-tenant-id",
      "company": ""
    }
  }
}
```

## Environment Variables

Override any config file setting with environment variables:

| Variable | Description |
|----------|-------------|
| `D365_URL` | Default environment URL |
| `D365_COMPANY` | Default company context |
| `D365_CLIENT_ID` | Service principal client ID |
| `D365_CLIENT_SECRET` | Service principal secret (**not saved to disk**) |
| `D365_TENANT_ID` | Azure AD tenant ID |
| `D365_AUTH_METHOD` | Auth method: `az-cli`, `browser`, `device-code`, `client-credentials`, `managed-identity` |
| `D365_PROFILE` | Active profile name |
| `D365_OUTPUT` | Output format: `json`, `table`, `csv`, `raw` |
| `D365_CI` | CI mode: enables JSON output, quiet mode, no color |
| `D365_TIMEOUT` | Default request timeout in seconds |

Environment variables take precedence over config file values.

## Global Flags

Available on every command:

```
-o, --output   Output format: json, table, csv, raw
               Default: auto-detected (JSON when piped, table when interactive)
    --company  Company/legal entity override (e.g., USMF)
    --profile  Configuration profile to use
-q, --quiet    Suppress non-essential output
    --no-color Disable colored output
-v, --verbose  Verbose logging to stderr
    --ci       CI mode (implies --output json --quiet --no-color)
    --timeout  Request timeout in seconds (default: 30)
```

## Profiles

Use profiles to switch between environments:

```bash
d365 connect https://dev.operations.dynamics.com --profile dev
d365 connect https://prod.operations.dynamics.com --profile prod

# Switch between them
d365 status --profile dev
d365 data find Customers --profile prod --query '$top=1'
```

## Daemon Configuration

The form daemon runs as a background process for stateful form sessions:

- **Windows:** TCP port 51365
- **Linux/macOS:** Unix socket `~/.d365cli/daemon.sock`

The daemon auto-starts on first `d365 form` command and stops after an idle timeout (default: 10 minutes).

```bash
d365 daemon status    # Check if running
d365 daemon start     # Start manually
d365 daemon stop      # Stop
d365 daemon restart   # Restart
```

## Troubleshooting

### "Commands fail after a successful connect"

If you connected with `--auth client-credentials`, the client secret is not saved to disk (for security). Set it as an environment variable:

```bash
export D365_CLIENT_SECRET="your-secret"   # Bash
$env:D365_CLIENT_SECRET = "your-secret"   # PowerShell
```

Then your commands will work. Run `d365 doctor` to verify everything is configured correctly.

### "tenant is required for browser auth"

This means the CLI doesn't know which auth method you intended to use. It happens when the auth method from your previous connect wasn't saved (older CLI versions). Fix it by reconnecting:

```bash
d365 connect https://your-env.operations.dynamics.com --auth client-credentials \
  --tenant <id> --client-id <id> --client-secret <secret>
```

### "Token expired"

Tokens expire after a period of time. Simply reconnect:

```bash
d365 connect https://your-env.operations.dynamics.com
```

### "I want to change the active company"

```bash
d365 company set USMF           # Set default company
d365 data find Customers --company ACME   # One-off override
```
