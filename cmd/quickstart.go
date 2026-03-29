package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/auth"
	"github.com/seangalliher/d365-erp-cli/internal/config"
	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
	"github.com/seangalliher/d365-erp-cli/internal/output"
)

func init() {
	rootCmd.AddCommand(newQuickstartCmd())
}

func newQuickstartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quickstart",
		Short: "Interactive guided setup for first-time users",
		Long: `Walk through connecting to a D365 environment step by step.

This command is interactive and requires a terminal (TTY).
It guides you through:
  1. Entering your environment URL
  2. Authenticating via Azure CLI
  3. Setting a default company (optional)
  4. Verifying the connection

Example:
  .\d365 quickstart`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			if !output.IsTerminal(os.Stdout) {
				e := clierrors.ValidationError("quickstart requires an interactive terminal (TTY)")
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			result, err := runQuickstart()
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, result, start)
			return nil
		},
	}
}

func runQuickstart() (map[string]interface{}, error) {
	w := os.Stderr

	// Step 1: Banner
	fmt.Fprint(w, banner())
	fmt.Fprintln(w, "  Welcome to the D365 ERP CLI quickstart!")
	fmt.Fprintln(w)

	// Step 2: Check existing connection
	sess, err := config.LoadSession()
	if err == nil && sess.Connected && sess.Environment != "" {
		fmt.Fprintf(w, "  You are already connected to: %s\n", sess.Environment)
		fmt.Fprintln(w)

		answer, err := promptForArgFrom(os.Stdin, w, "Continue with existing connection? (y/n)", "")
		if err != nil || (answer != "n" && answer != "N" && answer != "no") {
			fmt.Fprintln(w, "  Using existing connection.")
			fmt.Fprintln(w)
			printWhatsNext(w, sess.Environment, sess.Company)
			return map[string]interface{}{
				"status":      "existing_connection",
				"environment": sess.Environment,
				"company":     sess.Company,
			}, nil
		}
	}

	// Step 3: Get environment URL
	fmt.Fprintln(w, "  Step 1: Enter your D365 Finance & Operations environment URL")
	fmt.Fprintln(w)
	envURL, err := promptForArgFrom(os.Stdin, w, "Environment URL", "e.g., https://contoso.operations.dynamics.com")
	if err != nil {
		return nil, fmt.Errorf("environment URL is required: %w", err)
	}
	fmt.Fprintln(w)

	// Step 4: Connect via az-cli
	fmt.Fprintln(w, "  Step 2: Authenticating via Azure CLI...")
	provCfg := auth.ProviderConfig{
		Environment: envURL,
	}
	authenticator, err := auth.NewAuthenticator(auth.MethodAzCLI, provCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	token, err := authenticator.GetD365Token(context.Background())
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Fprintln(w, "  Authenticated successfully!")
	fmt.Fprintln(w)

	// Step 5: Company (optional)
	fmt.Fprintln(w, "  Step 3: Set a default company (optional)")
	company, _ := promptForArgFrom(os.Stdin, w, "Company ID", "e.g., USMF — press Enter to skip")
	fmt.Fprintln(w)

	// Step 6: Save session
	newSess := &config.Session{
		Connected:   true,
		Environment: envURL,
		Company:     company,
		TokenExpiry: token.ExpiresOn.UTC().Format(time.RFC3339),
	}
	if err := config.SaveSession(newSess); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Fprintln(w, "  Connection saved!")
	fmt.Fprintln(w)

	// Step 7: What's next
	printWhatsNext(w, envURL, company)

	return map[string]interface{}{
		"status":      "connected",
		"environment": envURL,
		"company":     company,
		"auth_method": "az-cli",
	}, nil
}

func printWhatsNext(w *os.File, envURL, company string) {
	fmt.Fprintln(w, "  What's next:")
	fmt.Fprintln(w, "  ───────────")
	fmt.Fprintln(w, "  Check status:     .\\d365 status")
	fmt.Fprintln(w, "  Find entities:    .\\d365 data find-type Customer")
	fmt.Fprintln(w, "  Query data:       .\\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'")
	fmt.Fprintln(w, "  Entity metadata:  .\\d365 data metadata Customers --keys --constraints")
	fmt.Fprintln(w, "  Open a form:      .\\d365 form find CustTable")
	fmt.Fprintln(w, "  Run diagnostics:  .\\d365 doctor")
	fmt.Fprintln(w, "  Full CLI schema:  .\\d365 schema --full")
	fmt.Fprintln(w, "  Agent prompt:     .\\d365 agent-prompt")
	fmt.Fprintln(w)
}
