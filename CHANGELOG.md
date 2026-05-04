# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Form daemon powered by Playwright** — headless Chromium browser automation replaces the previous stub handler
  - Creates, reads, and interacts with D365 forms via the actual web client
  - Playwright's `Type()` with character-by-character input for Knockout.js data binding
  - Playwright's `Click()` with `Force` option for D365 action pane and dialog buttons
  - Session cookie caching (`~/.d365cli/session-cookies.json`) — skips AAD login on daemon restart (~5s vs ~2min)
  - Persistent browser profile (`~/.d365cli/browser-profile/`) for SSO cookie reuse
  - Eager browser warmup at daemon start — login completes before accepting commands
  - Save verification with infolog/dialog error detection
  - Dialog dismissal before navigation (handles "Discard changes?" prompts)
  - Shell noise filtering in form state extraction (excludes D365 chrome controls)
  - ReactList grid column detection for D365's modern grid component
- Client read deadline increased to 5 minutes (from 60s) for initial login + form load
- Accept loop fix — daemon no longer spins on closed connection errors

### Changed
- Replaced `go-rod/rod` with `playwright-community/playwright-go` for browser automation
  - Playwright handles actionability checks, modal overlays, and Knockout.js bindings natively
  - No more Windows Defender false positives (rod's `leakless.exe` issue)
- Form state extraction scoped to `[data-dyn-role="Form"]` element, filtering out 100+ shell noise controls

### Previous
- Connection management: `connect`, `disconnect`, `status`, `company get/set`
- OData operations: `find-type`, `metadata`, `find`, `create`, `update`, `delete`
- API discovery and invocation: `api find`, `api invoke`
- Form automation: `form open`, `form close`, `form save`, `form state`, `form click`, `form set`, `form lookup`, `form tab`, `form filter`, `form grid-filter`, `form grid-select`, `form grid-sort`, `form find-controls`
- Background daemon for form sessions with auto-start and idle timeout
- Multiple output formats: JSON (default), table, CSV, raw with TTY auto-detection
- Batch mode for JSONL pipeline processing
- AI guardrails: cross-company enforcement, $select reminder, delete confirmation, wide-query warning, enum format validation
- Shell completion for bash, zsh, fish, and PowerShell
- `doctor` command for environment diagnostics
- `agent-prompt` command for AI agent system prompt generation
- `quickstart` command for interactive first-run setup
- Contextual error suggestions
- 5 authentication flows: browser, device-code, client-credentials, az-cli, managed-identity
