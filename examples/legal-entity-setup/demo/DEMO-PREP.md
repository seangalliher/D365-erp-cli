# Demo Recording Prep

## Intro Script (30-45 seconds)

> If you've ever configured a legal entity in D365 Finance & Operations, you know the drill — dozens of forms, hundreds of clicks, and a lot of alt-tabbing between docs and the browser. What if you could do it all from the command line — or better yet, just tell an AI agent to do it for you?
>
> Now, you might be thinking "why not just use the D365 MCP server?" MCP is great when you're inside a tool that supports it — VS Code, Claude Desktop — but it's a protocol, not a product. It can't be scripted, piped, or scheduled. You can't run it in CI, you can't hand it to an ops team, and you can't build a shell pipeline with it.
>
> D365 ERP CLI gives you a real tool. It's kubectl for Dynamics 365 — 25+ commands, structured JSON output, AI guardrails built in — and it works everywhere: your terminal, a shell script, a CI/CD pipeline, or as the tool layer underneath an AI agent like Copilot.
>
> Today I'm going to show you GitHub Copilot creating an entire legal entity with a configured general ledger, from scratch, using nothing but this CLI.

---

## Pre-Flight Checklist

Run through this before hitting record.

### Option A: Automated (recommended)

```powershell
./examples/legal-entity-setup/demo/preflight.ps1
```

This checks CLI version, connection, token expiry, company context, ACME state, and connectivity.

### Option B: Manual

#### 1. Verify CLI is built
```powershell
./d365 version
```
Expected: shows version, Go, and OS info.

#### 2. Verify connection to D365 environment
```powershell
./d365 status
```
Expected: `"connected": true` with your environment URL. If the token is expired:
```powershell
./d365 connect https://your-env.operations.dynamics.com
```

#### 3. Verify ACME doesn't already exist
```powershell
./d365 data find LegalEntities --query '$filter=LegalEntityId eq ''ACME''&$select=LegalEntityId,Name' --timeout 60
```
Expected: empty `value` array. If ACME exists from a prior run:
```powershell
./examples/legal-entity-setup/demo/cleanup.ps1
```

#### 4. Verify company context is clean
```powershell
./d365 company get
```
Expected: company should be empty or a different company (e.g., USMF). The preflight script clears this automatically.

#### 5. Test a simple query works
```powershell
./d365 data find LegalEntities --query '$top=2&$select=LegalEntityId,Name' --timeout 60
```
Expected: returns 2 legal entities with names.

---

## Demo Script (Copilot Flow)

### Intro (30-45 seconds)
1. Deliver the intro script above (or a natural version of it)

### Show the Repo
2. Open browser to `https://github.com/seangalliher/D365-erp-cli`
3. Briefly show the README and the Architecture diagram

### Setup (terminal)
3. Open terminal, run `./d365 status` to show you're connected
4. Run `./d365 version` to show the CLI version

### Copilot Demo
5. Open GitHub Copilot (in VS Code or terminal)
6. Paste the contents of `examples/legal-entity-setup/copilot-prompt.md`
7. Let Copilot execute each step — it will use `./d365` CLI commands
8. Watch it:
   - Check that ACME doesn't exist (query first)
   - Create the legal entity
   - Verify the legal entity was created
   - Switch company context to ACME
   - Create chart of accounts
   - Configure the general ledger
   - Create 8 main accounts one at a time
   - Verify with a final query showing all accounts

### Closing
9. Show the final query result (table format if interactive)
10. Show `./d365 company get` confirming you're in ACME
11. Open browser to the D365 environment to show the entity was actually created

---

## Tips for a Clean Recording

- **Use a clean terminal** — clear scrollback before starting
- **Increase font size** — at least 16pt for readability
- **Use a dark theme** — easier to read on video
- **Close notifications** — no popups during recording
- **Set timeout higher** — some D365 calls are slow on first hit:
  ```powershell
  # If Copilot doesn't set --timeout, you can set a default env var
  $env:D365_TIMEOUT = "60"
  ```
- **Token expiry** — re-authenticate right before recording so you have a fresh token
- **Practice once** — run through the full flow before recording to warm up the D365 cache

---

## Timing Estimate

- Intro + status/version: ~30 seconds
- Copilot prompt + execution: ~2-3 minutes (depends on D365 response times)
- Closing verification: ~30 seconds
- **Total: ~3-4 minutes**
