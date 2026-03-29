package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCompletionCmd())
}

// newCompletionCmd creates the "completion" command for generating shell completions.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for d365.

Supported shells: powershell, bash, zsh, fish

PowerShell:
  .\d365 completion powershell | Out-String | Invoke-Expression

  To load completions for every new session, add the output to your profile:
  .\d365 completion powershell >> $PROFILE

Bash:
  source <(d365 completion bash)

Zsh:
  d365 completion zsh > "${fpath[1]}/_d365"

Fish:
  d365 completion fish | source`,
		ValidArgs: []string{"powershell", "bash", "zsh", "fish"},
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell %q: must be one of powershell, bash, zsh, fish", args[0])
			}
		},
	}

	return cmd
}
