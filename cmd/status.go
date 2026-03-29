package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/errors"
	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current connection and session status",
	Long: `Display the current D365 connection state including environment URL,
authenticated user, active company, token expiry, and daemon status.

Examples:
  .\d365 status
  .\d365 status --output json`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	start := time.Now()

	sess, err := config.LoadSession()
	if err != nil {
		cliErr := errors.AsCLIError(err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	status := types.ConnectionStatus{
		Connected:   sess.Connected,
		Environment: sess.Environment,
		User:        sess.User,
		Company:     sess.Company,
		TokenExpiry: sess.TokenExpiry,
		DaemonPID:   sess.DaemonPID,
	}

	// Override company with the resolved value from flags/env/config.
	if company := GetCompany(); company != "" {
		status.Company = company
	}

	RenderSuccess(cmd, status, start)
	return nil
}
