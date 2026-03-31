# D365 — CLI for Dynamics 365 Finance & Operations

[![CI](https://github.com/seangalliher/d365-erp-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/seangalliher/d365-erp-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/seangalliher/d365-erp-cli)](https://goreportcard.com/report/github.com/seangalliher/d365-erp-cli)

**kubectl for D365** — a structured, scriptable CLI for Dynamics 365 Finance & Operations. Designed as the primary interface for AI agents, with full human usability.

<img width="376" height="107" alt="image" src="https://github.com/user-attachments/assets/1c453612-2b4d-486b-a517-cf2d47c2ee6c" />

[![Demo Video](https://img.shields.io/badge/Demo-Watch%20Video-red?style=for-the-badge&logo=youtube)](https://youtu.be/OcQeyeWyn5w?si=UJhxnAhLluCmZezP)

## Features

- **25+ commands** covering Data (OData), API (actions), Form (stateful UI), and diagnostics
- **Structured JSON output** on every command — predictable parsing for AI agents
- **TTY auto-detection** — JSON when piped, tables when interactive
- **5 auth methods** — browser, device-code, client-credentials, az-cli, managed-identity
- **AI guardrails** — cross-company auto-injection, `$select` warnings, enum validation, delete confirmation
- **AI system prompt** — `d365 agent-prompt` generates a complete tool description for agents
- **Self-diagnosing** — `d365 doctor` checks config, auth, DNS, connectivity, and daemon health
- **Batch mode** — JSONL pipeline for multi-command execution
- **Background daemon** — stateful form sessions with auto-start and idle timeout
- **Cross-platform** — Windows, macOS, Linux (amd64 + arm64)

## Installation

### Download pre-built binary (recommended)

Grab the latest release for your platform from [GitHub Releases](https://github.com/seangalliher/d365-erp-cli/releases):

| Platform | File |
|----------|------|
| Windows (x64) | `d365_*_windows_amd64.zip` |
| Windows (ARM) | `d365_*_windows_arm64.zip` |
| macOS (Apple Silicon) | `d365_*_darwin_arm64.tar.gz` |
| macOS (Intel) | `d365_*_darwin_amd64.tar.gz` |
| Linux (x64) | `d365_*_linux_amd64.tar.gz` |
| Linux (ARM) | `d365_*_linux_arm64.tar.gz` |

Extract the archive and place `d365` (or `d365.exe`) somewhere on your PATH.

### Install with Go

```bash
go install github.com/seangalliher/d365-erp-cli@latest
```

### Build from source

```bash
git clone https://github.com/seangalliher/d365-erp-cli.git
cd d365-erp-cli
go build -o d365.exe .    # Windows
# go build -o d365 .      # macOS / Linux
```

## Quick Start

```bash
# Interactive guided setup
d365 quickstart

# Or connect manually
d365 connect https://your-env.operations.dynamics.com

# Query data
d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'

# Check status
d365 status
```

> **PowerShell users:** OData parameters like `$top`, `$select`, and `$filter` start with `$`, which PowerShell treats as variables. Always wrap `--query` values in **single quotes** to prevent expansion.

## Commands

| Category | Commands | Description |
|----------|----------|-------------|
| **Connection** | `connect`, `disconnect`, `status`, `company` | Authentication and session management |
| **Data** | `find-type`, `metadata`, `find`, `create`, `update`, `delete` | OData CRUD operations |
| **API** | `find`, `invoke` | Discover and invoke D365 actions |
| **Forms** | `open`, `close`, `state`, `click`, `set`, `lookup`, `tab`, `filter`, `grid-*` | Stateful form automation |
| **Utilities** | `quickstart`, `doctor`, `agent-prompt`, `schema`, `docs`, `batch`, `version` | Setup, diagnostics, AI integration |

See [docs/commands.md](docs/commands.md) for the full command reference.

## AI Agent Integration

D365 CLI is built for AI agents. Generate a system prompt, register tools, and let agents drive D365 operations:

```bash
d365 agent-prompt            # Generate system prompt for your agent
d365 schema --full           # Export CLI schema for tool registration
```

The system prompt includes a catalog of 24 common D365 entities, guardrail descriptions, and workflow patterns (check-before-create, verify-after-create) so agents work correctly out of the box.

See [docs/ai-agents.md](docs/ai-agents.md) for the full integration guide, including a step-by-step Copilot example.

> **Live example:** See [`examples/legal-entity-setup/`](examples/legal-entity-setup/) for a complete walkthrough — creating a legal entity with chart of accounts and general ledger using an AI agent.

## Why CLI over MCP?

The D365 MCP server works well inside tools that support it (VS Code, Claude Desktop), but it has structural costs that add up for AI agents:

| | MCP Server | D365 CLI |
|---|---|---|
| **Tool definitions** | 21 tools injected into context on *every message* (~2,900 tokens/turn) | 1 tool (Bash) — ~100 tokens/turn |
| **Entity discovery** | Agent must call `find-type` → `metadata` → create (3+ round trips) | Entity catalog baked into system prompt (0 round trips) |
| **Metadata responses** | Full entity schema returned (~2,000-5,000 tokens each) | `--keys` flag returns only key fields |
| **Per-turn overhead** | ~2,900 tokens | ~100 tokens |
| **Scriptable** | No — protocol-bound to host tool | Yes — shell scripts, CI/CD, pipelines |
| **Schedulable** | No | Yes — cron, batch, automation |

### Token savings over a typical workflow

The CLI's one-time system prompt (`d365 agent-prompt`) costs ~2,500 tokens. After that, each turn only adds ~100 tokens for the Bash tool definition — compared to ~2,900 tokens per turn for MCP tool schemas.

| Workflow length | MCP input tokens | CLI input tokens | Savings |
|---|---|---|---|
| 10 turns | ~29,000 | ~3,500 | **88%** |
| 15 turns | ~44,000 | ~4,000 | **91%** |
| 20 turns | ~58,000 | ~4,500 | **92%** |

> These estimates cover tool definition overhead only. The real savings are larger because MCP's extra discovery round trips (find-type, metadata) generate conversation history that compounds across turns — typically adding another 5,000-15,000 tokens per workflow.

## Architecture

```mermaid
graph TB
    User([User / AI Agent]) --> CLI

    subgraph CLI["d365 CLI"]
        direction TB
        Cobra[Cobra Commands<br/><i>cmd/</i>]
        Guardrails[Guardrails<br/><i>AI safety rules</i>]
        Output[Output Renderer<br/><i>JSON · table · CSV · raw</i>]
        Cobra --> Guardrails --> Output
    end

    subgraph Core["Core Services"]
        direction TB
        Auth[Auth<br/><i>5 Azure AD flows</i>]
        Client[OData Client<br/><i>HTTP + retry</i>]
        Config[Config<br/><i>profiles · sessions</i>]
        Batch[Batch<br/><i>JSONL pipeline</i>]
        Errors[Errors<br/><i>structured + suggestions</i>]
    end

    subgraph Daemon["Form Daemon"]
        direction TB
        Server[IPC Server<br/><i>TCP :51365</i>]
        DClient[Daemon Client]
        Server --- DClient
    end

    CLI --> Core
    CLI --> Daemon

    Client --> D365[(D365 F&O<br/>OData / API)]
    Auth --> Entra[Microsoft Entra ID]
    Server --> D365

    classDef primary fill:#2563eb,stroke:#1d4ed8,color:#fff
    classDef service fill:#7c3aed,stroke:#6d28d9,color:#fff
    classDef external fill:#059669,stroke:#047857,color:#fff
    classDef user fill:#d97706,stroke:#b45309,color:#fff

    class CLI primary
    class Core,Daemon service
    class D365,Entra external
    class User user
```

### Package Layout

```
d365 (single binary)
├── cmd/           Cobra command definitions
├── internal/
│   ├── auth/      Azure AD authentication (5 flows)
│   ├── batch/     JSONL batch/pipeline executor
│   ├── client/    HTTP + OData client with retry
│   ├── config/    Configuration and session management
│   ├── daemon/    IPC server/client for form sessions
│   ├── errors/    Structured error types with suggestions
│   ├── guardrails/ AI safety rules engine
│   └── output/    Renderer (JSON, table, CSV, raw) + TTY detection
└── pkg/types/     Shared types (Response, ErrorInfo, etc.)
```

## Configuration

See [docs/configuration.md](docs/configuration.md) for config files, environment variables, auth methods, and profiles.

## Development

```bash
make build          # Build binary
make test           # Run tests (parallel)
make test-coverage  # Tests with coverage report
make lint           # Run linters
make cross-build    # Build for all platforms
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guidelines.

## License

MIT — see [LICENSE](LICENSE).

## Disclaimer

This project is a personal research project and is not affiliated with, endorsed by, or associated with my employer or any other organization. The views, code, and opinions expressed in this repository are solely my own and do not represent the views of any company or entity I am or have been associated with. This software is provided as-is for educational and research purposes.
