package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/errors"
)

var companyCmd = &cobra.Command{
	Use:   "company",
	Short: "Get or set the active D365 company (legal entity)",
	Long: `Manage the active company (legal entity) used for D365 operations.

Use 'company get' to display the current company and 'company set' to change it.

Examples:
  .\d365 company get
  .\d365 company set USMF`,
}

var companyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the current active company",
	Long: `Display the currently active D365 company (legal entity).

The company is resolved from (highest to lowest priority):
  1. --company flag
  2. D365_COMPANY environment variable
  3. Active session
  4. Configuration profile

Examples:
  .\d365 company get`,
	RunE: runCompanyGet,
}

var companySetCmd = &cobra.Command{
	Use:   "set <company-id>",
	Short: "Set the active company",
	Long: `Update the active D365 company (legal entity) in the current session.

The company ID is the short code for the legal entity (e.g., USMF, DEMF, DAT).

Examples:
  .\d365 company set USMF
  .\d365 company set DEMF`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCompanySet,
}

func init() {
	companyCmd.AddCommand(companyGetCmd)
	companyCmd.AddCommand(companySetCmd)
	rootCmd.AddCommand(companyCmd)
}

func runCompanyGet(cmd *cobra.Command, args []string) error {
	start := time.Now()

	company := GetCompany()

	data := map[string]interface{}{
		"company": company,
	}

	RenderSuccess(cmd, data, start)
	return nil
}

func runCompanySet(cmd *cobra.Command, args []string) error {
	start := time.Now()

	var companyID string
	if len(args) > 0 {
		companyID = args[0]
	} else {
		val, err := promptForArg("Company ID", "e.g., USMF, DEMF, DAT")
		if err != nil {
			return fmt.Errorf("%s", err.Error())
		}
		companyID = val
	}

	sess, err := config.LoadSession()
	if err != nil {
		cliErr := errors.AsCLIError(err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	if !sess.Connected {
		cliErr := errors.SessionRequired()
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	sess.Company = companyID

	if err := config.SaveSession(sess); err != nil {
		cliErr := errors.Wrap("CONNECTION_ERROR", "failed to save session", err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	data := map[string]interface{}{
		"company": companyID,
		"message": "Company updated successfully.",
	}

	RenderSuccess(cmd, data, start)
	return nil
}
