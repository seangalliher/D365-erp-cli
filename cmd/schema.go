package cmd

import (
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func init() {
	rootCmd.AddCommand(newSchemaCmd())
}

// newSchemaCmd creates the "schema" command that exports the full CLI command
// tree as a JSON structure for AI agent tool registration.
func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Export CLI command tree as JSON schema for AI agents",
		Long: `Walk the entire Cobra command tree and export a JSON structure describing
every command, its arguments, flags, examples, and guardrails.

This output is designed for AI agent tool registration, allowing agents
to discover all available CLI capabilities programmatically.

Use --full to include additional metadata such as guardrails and examples.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			full, _ := cmd.Flags().GetBool("full")

			schema := buildSchema(rootCmd, full)
			RenderSuccess(cmd, schema, start)
			return nil
		},
	}

	cmd.Flags().Bool("full", false, "Include additional metadata (guardrails, examples)")
	return cmd
}

// buildSchema recursively walks the command tree and builds a SchemaCommand.
func buildSchema(cmd *cobra.Command, full bool) types.SchemaCommand {
	sc := types.SchemaCommand{
		Name:        cmd.Name(),
		Description: cmd.Short,
	}

	// Parse args from Use string.
	sc.Args = extractArgs(cmd)

	// Collect flags.
	sc.Flags = extractFlags(cmd, full)

	// Include examples and guardrails in full mode.
	if full {
		if cmd.Example != "" {
			sc.Examples = []types.SchemaExample{
				{
					Description: "Example usage",
					Command:     strings.TrimSpace(cmd.Example),
				},
			}
		}

		// Extract guardrail hints from the Long description.
		if cmd.Long != "" {
			guardrails := extractGuardrails(cmd.Long)
			if len(guardrails) > 0 {
				sc.Guardrails = guardrails
			}
		}
	}

	// Recurse into subcommands.
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		// Skip help commands.
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		childSchema := buildSchema(sub, full)
		sc.SubCommands = append(sc.SubCommands, childSchema)
	}

	return sc
}

// extractArgs parses positional arguments from the cobra command's Use string.
// Use strings like "find <searchTerm>" indicate a required arg, while
// "[optional]" indicates an optional arg.
func extractArgs(cmd *cobra.Command) []types.SchemaArg {
	useStr := cmd.Use
	parts := strings.Fields(useStr)
	if len(parts) <= 1 {
		return nil
	}

	var args []types.SchemaArg
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "<") && strings.HasSuffix(part, ">") {
			name := strings.Trim(part, "<>")
			args = append(args, types.SchemaArg{
				Name:        name,
				Description: name,
				Required:    true,
				Type:        "string",
			})
		} else if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			name := strings.Trim(part, "[]")
			args = append(args, types.SchemaArg{
				Name:        name,
				Description: name,
				Required:    false,
				Type:        "string",
			})
		}
	}

	return args
}

// extractFlags collects flag definitions from a command.
func extractFlags(cmd *cobra.Command, includePersistent bool) []types.SchemaFlag {
	var flags []types.SchemaFlag

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		sf := types.SchemaFlag{
			Name:        f.Name,
			Short:       f.Shorthand,
			Description: f.Usage,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
		}

		// Check if this flag is marked required via annotations.
		if ann, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			for _, v := range ann {
				if v == "true" {
					sf.Required = true
				}
			}
		}

		flags = append(flags, sf)
	})

	return flags
}

// extractGuardrails looks for guardrail-related keywords in the command's
// long description and returns any guardrail hints found.
func extractGuardrails(long string) []string {
	var guardrails []string
	lines := strings.Split(long, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "guardrail") ||
			strings.Contains(lower, "requires") ||
			strings.Contains(lower, "confirm") ||
			strings.Contains(lower, "destructive") {
			guardrails = append(guardrails, trimmed)
		}
	}
	return guardrails
}
