package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/client"
	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

func init() {
	rootCmd.AddCommand(newAPICmd())
}

// newAPICmd creates the "api" command group with find and invoke subcommands.
func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Discover and invoke D365 service actions",
		Long: `Commands for discovering and executing D365 custom service actions.

Use "api find" to search for available actions by keyword, and
"api invoke" to execute a named action with parameters.

Designed for AI agents and automation pipelines.`,
	}

	cmd.AddCommand(
		newAPIFindCmd(),
		newAPIInvokeCmd(),
	)

	return cmd
}

// ---------------------------------------------------------------------------
// d365 api find <searchTerm>
// ---------------------------------------------------------------------------

func newAPIFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find <searchTerm>",
		Short: "Search for available D365 actions by keyword",
		Long: `Search for action menu items by keyword. Returns matching actions with
their names, labels, descriptions, and parameter definitions.

The search is performed against the D365 custom service discovery endpoint.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			searchTerm := args[0]

			sess, err := RequireSession()
			if err != nil {
				e := clierrors.SessionRequired()
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			tp, err := getTokenProvider(sess.Environment)
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			var result interface{}
			err = withSpinner("Searching actions...", func() error {
				var findErr error
				result, findErr = apiFindActions(cmd.Context(), sess.Environment, tp, searchTerm)
				return findErr
			})
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, result, start)
			return nil
		},
	}

	return cmd
}

// apiFindActions calls the D365 action discovery endpoint to search for actions.
func apiFindActions(ctx context.Context, envURL string, tp client.TokenProvider, searchTerm string) (interface{}, error) {
	c := client.NewClient(envURL, tp, time.Duration(flagTimeout)*time.Second)
	c.SetVerbose(flagVerbose)

	payload := map[string]string{"searchTerm": searchTerm}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, clierrors.ODataError("failed to marshal search request", err)
	}

	resp, err := c.Post(ctx, "api/services/find", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, clierrors.ODataError("failed to read action search response", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If not valid JSON, return as raw string
		return string(respBody), nil
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// d365 api invoke <actionName> --params '{}' [--company <id>]
// ---------------------------------------------------------------------------

func newAPIInvokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke <actionName>",
		Short: "Execute a named D365 action",
		Long: `Execute a named D365 custom service action with the given parameters.

The --params flag accepts a JSON string with the action's parameters.
Use --company to override the default company context for the invocation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			actionName := args[0]
			paramsJSON, _ := cmd.Flags().GetString("params")
			companyOverride, _ := cmd.Flags().GetString("company")

			// Validate the params JSON.
			var params json.RawMessage
			if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
				return fmt.Errorf("invalid JSON in --params: %w", err)
			}

			sess, err := RequireSession()
			if err != nil {
				e := clierrors.SessionRequired()
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			tp, err := getTokenProvider(sess.Environment)
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			// Determine effective company.
			company := companyOverride
			if company == "" {
				company = GetCompany()
			}

			var result interface{}
			err = withSpinner("Invoking action...", func() error {
				var invokeErr error
				result, invokeErr = apiInvokeAction(cmd.Context(), sess.Environment, tp, actionName, params, company)
				return invokeErr
			})
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, result, start)
			return nil
		},
	}

	cmd.Flags().String("params", "", "JSON string with action parameters (required)")
	_ = cmd.MarkFlagRequired("params")
	cmd.Flags().String("company", "", "Company/legal entity override for this action invocation")
	return cmd
}

// apiInvokeAction calls the D365 action execution endpoint.
func apiInvokeAction(ctx context.Context, envURL string, tp client.TokenProvider, actionName string, params json.RawMessage, company string) (interface{}, error) {
	c := client.NewClient(envURL, tp, time.Duration(flagTimeout)*time.Second)
	c.SetVerbose(flagVerbose)

	payload := map[string]interface{}{
		"name":       actionName,
		"parameters": params,
	}
	if company != "" {
		payload["companyId"] = company
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, clierrors.ODataError("failed to marshal action request", err)
	}

	resp, err := c.Post(ctx, "api/services/invoke", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, clierrors.ODataError("failed to read action response", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}
	return result, nil
}
