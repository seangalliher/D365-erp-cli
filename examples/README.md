# Examples

Ready-to-use examples for common D365 F&O tasks using the D365 CLI.

## Legal Entity Setup

**[`legal-entity-setup/`](legal-entity-setup/)** — Create a new legal entity, chart of accounts, general ledger, and main accounts from scratch.

- `batch.jsonl` — Run the entire setup in one command via `d365 batch`
- `legal-entity.json` — Legal entity definition
- `chart-of-accounts.json` — Chart of accounts definition
- `ledger.json` — General ledger configuration
- `main-accounts.json` — Starter main accounts (Cash, Revenue, Expenses)
- `copilot-prompt.md` — Paste this into GitHub Copilot to run the setup conversationally

### Quick Start

```bash
# Option 1: Run everything at once via batch mode
d365 connect https://your-env.operations.dynamics.com
cat examples/legal-entity-setup/batch.jsonl | d365 batch

# Option 2: Give the prompt to GitHub Copilot and let it drive
# Copy the contents of examples/legal-entity-setup/copilot-prompt.md
# into your Copilot chat
```

## Contributing Examples

Have a useful workflow? PRs welcome! Create a new directory under `examples/` with:
- A `README.md` explaining the scenario
- Data files (`.json`) for each entity
- A `batch.jsonl` for one-shot execution
- Optionally, a `copilot-prompt.md` for AI-driven execution
