# Demo Recording Prep

## Pre-Flight Checklist

Run through this before hitting record.

### 1. Verify CLI is built and on PATH
```powershell
d365 version
```
Expected: shows version, Go, and OS info.

### 2. Verify connection to D365 environment
```powershell
d365 status
```
Expected: `"connected": true` with your environment URL. If the token is expired:
```powershell
d365 connect https://fy26-h2-standard.operations.dynamics.com
```

### 3. Verify ACME doesn't already exist
```powershell
d365 data find LegalEntities --query "$filter=LegalEntityId eq 'ACME'&$select=LegalEntityId,Name" --timeout 60
```
Expected: empty `value` array (no results). If ACME exists from a prior run, use the cleanup script:
```powershell
.\examples\legal-entity-setup\demo\cleanup.ps1
```

### 4. Verify you're NOT in the ACME company
```powershell
d365 company get
```
Expected: company should be empty or a different company (e.g., USMF).

### 5. Test a simple query works
```powershell
d365 data find LegalEntities --query "$top=2&$select=LegalEntityId,Name" --timeout 60
```
Expected: returns 2 legal entities with names.

---

## Demo Script (Copilot Flow)

### Opening (show the repo)
1. Open browser to `https://github.com/seangalliher/D365-erp-cli`
2. Briefly show the README and the Architecture diagram

### Setup (terminal)
3. Open terminal, run `d365 status` to show you're connected
4. Run `d365 doctor` to show the diagnostic output

### Copilot Demo
5. Open GitHub Copilot (in VS Code or terminal)
6. Paste this prompt:

> Create a new legal entity called ACME Corp with ID ACME in my D365 environment, then set up its chart of accounts and general ledger with starter accounts for Cash, AR, AP, Retained Earnings, Revenue, COGS, OpEx, and Payroll.

7. Let Copilot execute each step — it will use `d365` CLI commands
8. Watch it:
   - Create the legal entity
   - Switch company context
   - Create chart of accounts
   - Configure the general ledger
   - Create 8 main accounts
   - Verify with a final query

### Closing
9. Show the final query result (table format if interactive)
10. Show `d365 company get` confirming you're in ACME
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

- Intro + status/doctor: ~30 seconds
- Copilot prompt + execution: ~2-3 minutes (depends on D365 response times)
- Closing verification: ~30 seconds
- **Total: ~3-4 minutes**
