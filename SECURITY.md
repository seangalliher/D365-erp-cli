# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in d365, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email the maintainers or use [GitHub's private vulnerability reporting](https://github.com/seangalliher/d365-erp-cli/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge receipt within 48 hours and aim to provide a fix or mitigation within 7 days for critical issues.

## Scope

This policy covers the `d365` CLI tool itself. It does not cover the Dynamics 365 F&O platform — report those issues directly to Microsoft.

## Credential Handling

The d365 CLI stores authentication tokens locally in `~/.d365/session.json`. Tokens are never logged or transmitted to third parties. The CLI uses the Azure Identity SDK for all authentication flows.
