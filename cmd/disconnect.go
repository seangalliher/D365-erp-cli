package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/errors"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from the current D365 environment",
	Long: `End the current session and clear stored credentials.

This removes the active session file so subsequent commands will require
a new '.\d365 connect' call.

Examples:
  .\d365 disconnect`,
	RunE: runDisconnect,
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}

func runDisconnect(cmd *cobra.Command, args []string) error {
	start := time.Now()

	if err := config.ClearSession(); err != nil {
		cliErr := errors.Wrap("CONNECTION_ERROR", "failed to clear session", err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	data := map[string]interface{}{
		"disconnected": true,
		"message":      "Successfully disconnected from D365 environment.",
	}

	RenderSuccess(cmd, data, start)
	return nil
}
