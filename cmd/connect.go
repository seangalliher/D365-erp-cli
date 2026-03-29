package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/auth"
	"github.com/seangalliher/d365-erp-cli/internal/config"
	"github.com/seangalliher/d365-erp-cli/internal/errors"
)

var connectCmd = &cobra.Command{
	Use:   "connect <environment-url>",
	Short: "Connect to a D365 Finance & Operations environment",
	Long: `Authenticate and establish a session with a D365 Finance & Operations environment.

Quick start (if you have Azure CLI installed and logged in):

  .\d365 connect https://your-env.operations.dynamics.com

That's it. The CLI uses your existing Azure CLI credentials by default.

Authentication methods:

  az-cli              Reuses your 'az login' session (default — no extra flags needed)
  browser             Interactive browser popup (requires --tenant, --client-id)
  device-code         Prints a code to enter at microsoft.com (requires --tenant, --client-id)
  client-credentials  Service principal / app registration (requires --tenant, --client-id, --client-secret)
  managed-identity    For code running on Azure VMs, App Service, etc.

Note: --tenant is your Azure AD tenant ID (a GUID found in Azure Portal > Microsoft Entra ID),
NOT the D365 environment ID from Power Platform admin center.

Examples:
  .\d365 connect https://contoso.operations.dynamics.com
  .\d365 connect https://contoso.operations.dynamics.com --company USMF
  .\d365 connect https://contoso.operations.dynamics.com --auth browser --tenant <azure-ad-tenant-id> --client-id <app-id>
  .\d365 connect https://contoso.operations.dynamics.com --auth client-credentials --tenant <azure-ad-tenant-id> --client-id <app-id> --client-secret <secret>`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConnect,
}

var (
	connectAuthMethod   string
	connectClientID     string
	connectClientSecret string
	connectTenant       string
)

func init() {
	f := connectCmd.Flags()
	f.StringVar(&connectAuthMethod, "auth", "az-cli", "Authentication method: az-cli, browser, device-code, client-credentials, managed-identity")
	f.StringVar(&connectClientID, "client-id", "", "Azure AD application (client) ID")
	f.StringVar(&connectClientSecret, "client-secret", "", "Client secret for client-credentials auth")
	f.StringVar(&connectTenant, "tenant", "", "Azure AD tenant ID (not the D365 environment ID — find it in Azure Portal > Microsoft Entra ID)")

	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	start := time.Now()

	var environmentURL string
	if len(args) > 0 {
		environmentURL = args[0]
	} else {
		val, err := promptForArg("Environment URL", "e.g., https://contoso.operations.dynamics.com")
		if err != nil {
			return fmt.Errorf("%s", err.Error())
		}
		environmentURL = val
	}

	// Fall back to environment variables for auth parameters.
	if connectClientID == "" {
		connectClientID = config.GetEnvOrDefault(config.EnvClientID, "")
	}
	if connectClientSecret == "" {
		connectClientSecret = config.GetEnvOrDefault(config.EnvClientSecret, "")
	}
	if connectTenant == "" {
		connectTenant = config.GetEnvOrDefault(config.EnvTenantID, "")
	}
	if connectAuthMethod == "az-cli" {
		if m := config.GetEnvOrDefault(config.EnvAuthMethod, ""); m != "" {
			connectAuthMethod = m
		}
	}

	// Create the authenticator for the chosen method.
	provCfg := auth.ProviderConfig{
		TenantID:     connectTenant,
		ClientID:     connectClientID,
		ClientSecret: connectClientSecret,
		Environment:  environmentURL,
	}

	authenticator, err := auth.NewAuthenticator(connectAuthMethod, provCfg)
	if err != nil {
		cliErr := errors.AsCLIError(err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	// Validate the connection by acquiring a token.
	ctx := context.Background()
	token, err := authenticator.GetD365Token(ctx)
	if err != nil {
		cliErr := errors.AsCLIError(err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	// Build and save the session.
	sess := &config.Session{
		Connected:   true,
		Environment: environmentURL,
		Company:     GetCompany(),
		TokenExpiry: token.ExpiresOn.UTC().Format(time.RFC3339),
		ProfileName: flagProfile,
	}

	if err := config.SaveSession(sess); err != nil {
		cliErr := errors.Wrap("CONNECTION_ERROR", "failed to save session", err)
		RenderError(cmd, cliErr.ToErrorInfo(), start)
		return nil
	}

	data := map[string]interface{}{
		"connected":    true,
		"environment":  sess.Environment,
		"auth_method":  authenticator.Method(),
		"company":      sess.Company,
		"user":         sess.User,
		"token_expiry": sess.TokenExpiry,
	}

	RenderSuccess(cmd, data, start)
	return nil
}
