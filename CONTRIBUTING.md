# Contributing to d365

Thank you for your interest in contributing to d365! This document provides guidelines and instructions for contributing.

## Prerequisites

- **Go 1.26+** — [golang.org/dl](https://golang.org/dl/)
- **GNU Make** (optional, for `make` targets)
- **golangci-lint** (optional, for linting) — [installation](https://golangci-lint.run/welcome/install/)

## Getting Started

```bash
# Fork and clone the repository
git clone https://github.com/<your-username>/d365.git
cd d365

# Install dependencies
go mod download

# Build
go build .

# Run tests
go test ./... -parallel 8
```

## Development Workflow

1. **Create a branch** from `main` for your change:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the code style guidelines below.

3. **Write tests** for new functionality. All tests should use `t.Parallel()`.

4. **Run checks locally** before pushing:
   ```bash
   go test ./... -parallel 8
   go vet ./...
   gofmt -l .
   ```

5. **Push and open a pull request** against `main`.

## Code Style

- Format with `gofmt` (enforced by CI)
- Follow standard Go conventions ([Effective Go](https://go.dev/doc/effective_go))
- Use the factory pattern `newXxxCmd() *cobra.Command` for new commands
- All tests must call `t.Parallel()`
- Prefer table-driven tests
- Errors should use the structured `CLIError` type from `internal/errors/`
- Commands should return structured JSON via `RenderSuccess()` / `RenderError()`

## Project Structure

```
cmd/           — Cobra command definitions (one file per command group)
internal/      — Private implementation packages
  auth/        — Azure AD authentication flows
  batch/       — JSONL batch executor
  client/      — HTTP + OData client
  config/      — Configuration and session management
  daemon/      — IPC daemon for form sessions
  errors/      — Structured error types
  guardrails/  — AI safety rules
  output/      — Renderer (JSON, table, CSV, raw)
pkg/types/     — Shared types (exported)
```

## Commit Messages

Use clear, descriptive commit messages:

```
<type>: <short description>

<optional body explaining the "why">
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`

Examples:
- `feat: add CSV export for data find`
- `fix: handle empty $filter in query parser`
- `docs: update quickstart guide`

## Adding a New Command

1. Create or extend a file in `cmd/` (e.g., `cmd/mycommand.go`)
2. Use the factory pattern: `func newMyCmd() *cobra.Command { ... }`
3. Register in the parent command's `init()` or `AddCommand()` call
4. Add corresponding tests in `cmd/mycommand_test.go`
5. Update `README.md` command reference

## Running Tests

```bash
# All tests
go test ./... -parallel 8

# Specific package
go test ./internal/guardrails/ -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Questions?

Open a [discussion](https://github.com/seangalliher/d365-erp-cli/discussions) or [issue](https://github.com/seangalliher/d365-erp-cli/issues) if you have questions about contributing.
