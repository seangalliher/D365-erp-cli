package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAgentPromptCmd())
}

func newAgentPromptCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "agent-prompt",
		Short: "Generate a system prompt for AI agents to use this CLI",
		Long: `Output a comprehensive system prompt in markdown that describes the full
CLI capabilities, command reference, output format, error codes, and
best practices.

AI agents can call this once to learn the entire tool. The output is
designed to be pasted into an agent's system prompt or tool description.

Default output is raw markdown to stdout. Use --json to wrap in the
standard JSON envelope.

Example:
  .\d365 agent-prompt
  .\d365 agent-prompt --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			prompt := buildAgentPrompt()

			if asJSON {
				RenderSuccess(cmd, map[string]interface{}{
					"prompt": prompt,
				}, start)
			} else {
				fmt.Fprint(os.Stdout, prompt)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Wrap output in standard JSON envelope")
	return cmd
}

func buildAgentPrompt() string {
	var sb strings.Builder

	sb.WriteString("# D365 ERP CLI — AI Agent System Prompt\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString("You have access to the `d365` CLI tool (\"kubectl for Dynamics 365 Finance & Operations\").\n")
	sb.WriteString("It provides structured JSON output for all commands, making it ideal for programmatic use.\n\n")

	// Connection
	sb.WriteString("## Connection\n\n")
	sb.WriteString("Before using data or form commands, establish a connection:\n")
	sb.WriteString("```\n.\\d365 connect https://<environment>.operations.dynamics.com\n```\n")
	sb.WriteString("Check connection status: `.\\d365 status`\n\n")

	// Output schema
	sb.WriteString("## Output Schema\n\n")
	sb.WriteString("All commands return a JSON envelope:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"success\": true|false,\n")
	sb.WriteString("  \"command\": \"d365 <command>\",\n")
	sb.WriteString("  \"data\": { ... },\n")
	sb.WriteString("  \"error\": { \"code\": \"...\", \"message\": \"...\", \"suggestion\": \"...\" },\n")
	sb.WriteString("  \"metadata\": { \"duration_ms\": 42, \"version\": \"...\", \"company\": \"...\" }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	// Error codes
	sb.WriteString("## Error Codes\n\n")
	sb.WriteString("| Code | Meaning |\n")
	sb.WriteString("|------|--------|\n")
	sb.WriteString("| `AUTH_ERROR` | Authentication failed — re-run `connect` |\n")
	sb.WriteString("| `PERMISSION_DENIED` | Insufficient D365 security role |\n")
	sb.WriteString("| `CONNECTION_ERROR` | Network or DNS issue |\n")
	sb.WriteString("| `SESSION_REQUIRED` | No active session — run `connect` first |\n")
	sb.WriteString("| `NOT_FOUND` | Entity or resource not found |\n")
	sb.WriteString("| `BAD_REQUEST` | Invalid query syntax |\n")
	sb.WriteString("| `VALIDATION_ERROR` | Invalid arguments or flags |\n")
	sb.WriteString("| `ODATA_ERROR` | OData query processing error |\n")
	sb.WriteString("| `TIMEOUT` | Request timed out |\n")
	sb.WriteString("| `SERVER_ERROR` | D365 server error |\n")
	sb.WriteString("| `THROTTLED` | Rate limited (HTTP 429) |\n")
	sb.WriteString("| `DAEMON_ERROR` | Form daemon issue |\n\n")

	// Guardrails
	sb.WriteString("## Guardrails\n\n")
	sb.WriteString("The CLI enforces safety rules:\n")
	sb.WriteString("- `cross-company=true` is auto-injected on data queries unless explicitly set\n")
	sb.WriteString("- Warns if `$select` is missing (performance best practice)\n")
	sb.WriteString("- `data delete` requires `--confirm` flag\n")
	sb.WriteString("- Wide queries (no `$top`) trigger warnings\n")
	sb.WriteString("- Enum values must use symbol names, not integers\n\n")

	// Workflows
	sb.WriteString("## Common Workflows\n\n")
	sb.WriteString("### Query data\n")
	sb.WriteString("```\n")
	sb.WriteString(".\\d365 data find-type Customer                    # Find entity names\n")
	sb.WriteString(".\\d365 data metadata Customers --keys --constraints  # Get entity schema\n")
	sb.WriteString(".\\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Create/Update/Delete data\n")
	sb.WriteString("```\n")
	sb.WriteString(".\\d365 data create Customers --data '{\"CustomerAccount\":\"NEW-001\",\"Name\":\"Test\"}'\n")
	sb.WriteString(".\\d365 data update --data '[{\"ODataPath\":\"Customers(...)\",\"UpdatedFieldValues\":{...}}]'\n")
	sb.WriteString(".\\d365 data delete --paths '[\"Customers(...)\"]' --confirm\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### Form automation\n")
	sb.WriteString("```\n")
	sb.WriteString(".\\d365 form find CustTable       # Find menu item\n")
	sb.WriteString(".\\d365 form open CustTable --type Display\n")
	sb.WriteString(".\\d365 form state                # Read current controls\n")
	sb.WriteString(".\\d365 form set FieldName=Value  # Set values\n")
	sb.WriteString(".\\d365 form save\n")
	sb.WriteString(".\\d365 form close\n")
	sb.WriteString("```\n\n")

	// PowerShell tip
	sb.WriteString("## PowerShell Tip\n\n")
	sb.WriteString("Always use **single quotes** for `--query` values to prevent PowerShell\n")
	sb.WriteString("from interpreting `$select`, `$filter`, `$top` as variables:\n")
	sb.WriteString("```powershell\n")
	sb.WriteString(".\\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'\n")
	sb.WriteString("```\n\n")

	// Dynamic command reference
	sb.WriteString("## Command Reference\n\n")
	writeCommandTree(&sb, rootCmd, 0)

	return sb.String()
}

func writeCommandTree(sb *strings.Builder, cmd *cobra.Command, depth int) {
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}

		indent := strings.Repeat("  ", depth)
		fmt.Fprintf(sb, "%s- **%s**", indent, sub.CommandPath())

		// Add args
		args := extractArgs(sub)
		if len(args) > 0 {
			var argNames []string
			for _, a := range args {
				if a.Required {
					argNames = append(argNames, "<"+a.Name+">")
				} else {
					argNames = append(argNames, "["+a.Name+"]")
				}
			}
			sb.WriteString(" " + strings.Join(argNames, " "))
		}

		sb.WriteString(" — " + sub.Short + "\n")

		// Add key flags
		flags := extractFlags(sub, false)
		if len(flags) > 0 {
			for _, f := range flags {
				required := ""
				if f.Required {
					required = " *(required)*"
				}
				fmt.Fprintf(sb, "%s  - `--%s` (%s): %s%s\n", indent, f.Name, f.Type, f.Description, required)
			}
		}

		// Recurse
		if sub.HasSubCommands() {
			writeCommandTree(sb, sub, depth+1)
		}
	}
}
