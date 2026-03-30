# Configuration

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
      "auth_method": "browser",
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
| `D365_CLIENT_SECRET` | Service principal secret |
| `D365_TENANT_ID` | Azure AD tenant ID |
| `D365_AUTH_METHOD` | Auth method: `browser`, `device-code`, `client-credentials`, `az-cli`, `managed-identity` |
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

## Authentication Methods

| Method | Flag | Use Case |
|--------|------|----------|
| Browser | `--auth browser` (default) | Interactive development |
| Device Code | `--auth device-code` | Remote / SSH sessions |
| Client Credentials | `--auth client-credentials` | CI/CD, automation |
| Azure CLI | `--auth az-cli` | Reuse existing `az login` session |
| Managed Identity | `--auth managed-identity` | Azure VMs, App Service |

### Browser (default)

```bash
d365 connect https://your-env.operations.dynamics.com
```

Opens a browser for Microsoft Entra ID login.

### Client Credentials (CI/CD)

```bash
d365 connect https://your-env.operations.dynamics.com \
  --auth client-credentials \
  --client-id $D365_CLIENT_ID \
  --client-secret $D365_CLIENT_SECRET \
  --tenant-id $D365_TENANT_ID
```

Or via environment variables:

```bash
export D365_URL=https://your-env.operations.dynamics.com
export D365_AUTH_METHOD=client-credentials
export D365_CLIENT_ID=your-client-id
export D365_CLIENT_SECRET=your-secret
export D365_TENANT_ID=your-tenant-id
d365 connect
```

### Profiles

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
