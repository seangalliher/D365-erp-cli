# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
