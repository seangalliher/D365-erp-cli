package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/auth"
	"github.com/seangalliher/d365-erp-cli/internal/client"
	"github.com/seangalliher/d365-erp-cli/internal/config"
	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

func init() {
	rootCmd.AddCommand(newDataCmd())
}

// newDataCmd creates the "data" command group with all subcommands.
func newDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Manage D365 data entities via OData",
		Long: `Commands for querying, creating, updating, and deleting D365 data entities
via OData. Designed for AI agents and automation pipelines.

All commands produce structured JSON output and include guardrails for
safe operation (cross-company auto-injection, $select warnings, delete
confirmation).

PowerShell users: use single quotes for --query values to prevent
PowerShell from treating $ as variable interpolation:

  .\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'`,
	}

	cmd.AddCommand(
		newFindTypeCmd(),
		newMetadataCmd(),
		newFindCmd(),
		newCreateCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
	)

	return cmd
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getTokenProvider creates a client.TokenProvider by constructing an
// Authenticator from the current config/profile and environment variables.
func getTokenProvider(envURL string) (client.TokenProvider, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	profile := cfg.ActiveProfile(flagProfile)

	method := config.GetEnvOrDefault(config.EnvAuthMethod, profile.AuthMethod)
	if method == "" {
		method = auth.MethodAzCLI
	}

	authCfg := auth.ProviderConfig{
		TenantID:     config.GetEnvOrDefault(config.EnvTenantID, profile.TenantID),
		ClientID:     config.GetEnvOrDefault(config.EnvClientID, profile.ClientID),
		ClientSecret: config.GetEnvOrDefault(config.EnvClientSecret, ""),
		Environment:  envURL,
	}

	authenticator, err := auth.NewAuthenticator(method, authCfg)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context) (string, error) {
		tok, err := authenticator.GetD365Token(ctx)
		if err != nil {
			return "", err
		}
		return tok.AccessToken, nil
	}, nil
}

// parseRawQueryOptions parses a raw OData query options string (e.g.
// "$filter=Name eq 'Contoso'&$select=CustomerAccount,Name") into a
// client.QueryOptions struct. If cross-company is not explicitly present
// in the raw string, the returned CrossCompany field is nil so that
// BuildQueryString will auto-inject cross-company=true.
func parseRawQueryOptions(raw string) (client.QueryOptions, error) {
	if raw == "" {
		return client.QueryOptions{}, nil
	}

	vals, err := url.ParseQuery(raw)
	if err != nil {
		return client.QueryOptions{}, fmt.Errorf("invalid query string: %w", err)
	}

	opts := client.QueryOptions{
		Filter:  vals.Get("$filter"),
		Select:  vals.Get("$select"),
		Expand:  vals.Get("$expand"),
		OrderBy: vals.Get("$orderby"),
	}
	if v := vals.Get("$top"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return client.QueryOptions{}, fmt.Errorf("invalid $top value %q: must be a number", v)
		}
		opts.Top = n
	}
	if v := vals.Get("$skip"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return client.QueryOptions{}, fmt.Errorf("invalid $skip value %q: must be a number", v)
		}
		opts.Skip = n
	}
	if vals.Get("$count") == "true" {
		opts.Count = true
	}
	if v := vals.Get("cross-company"); v != "" {
		b := v == "true"
		opts.CrossCompany = &b
	}

	return opts, nil
}

// updateEntry represents a single entity update in the batch format accepted
// by the "data update" command.
type updateEntry struct {
	ODataPath          string                 `json:"ODataPath"`
	UpdatedFieldValues map[string]interface{} `json:"UpdatedFieldValues"`
}

// ---------------------------------------------------------------------------
// d365 data find-type <searchTerm> [--top N]
// ---------------------------------------------------------------------------

func newFindTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find-type <searchTerm>",
		Short: "Search OData entity types by keyword",
		Long: `Search the $metadata document for EntityType definitions whose name
contains the search term (case-insensitive). Returns up to --top results
with the matching EntityType name and corresponding EntitySet name.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			searchTerm := args[0]
			top, _ := cmd.Flags().GetInt("top")

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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			matches, err := oc.FindEntityTypes(cmd.Context(), searchTerm, top)
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, matches, start)
			return nil
		},
	}

	cmd.Flags().Int("top", 10, "Maximum number of results to return")
	return cmd
}

// ---------------------------------------------------------------------------
// d365 data metadata <entitySetName> [--enums] [--keys] [--constraints] [--relationships]
// ---------------------------------------------------------------------------

func newMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata <entitySetName>",
		Short: "Get entity metadata from the $metadata document",
		Long: `Retrieve the OData $metadata document for the environment. The entity set
name argument identifies the entity of interest. Use the boolean flags to
indicate which metadata sections are desired.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			entitySetName := args[0]
			enums, _ := cmd.Flags().GetBool("enums")
			keys, _ := cmd.Flags().GetBool("keys")
			constraints, _ := cmd.Flags().GetBool("constraints")
			relationships, _ := cmd.Flags().GetBool("relationships")

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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			metadata, err := oc.GetMetadata(cmd.Context())
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			data := map[string]interface{}{
				"entity_set_name": entitySetName,
				"raw_metadata":    metadata,
				"include": map[string]bool{
					"enums":         enums,
					"keys":          keys,
					"constraints":   constraints,
					"relationships": relationships,
				},
			}

			RenderSuccess(cmd, data, start)
			return nil
		},
	}

	cmd.Flags().Bool("enums", false, "Include enum value definitions")
	cmd.Flags().Bool("keys", false, "Include key field information")
	cmd.Flags().Bool("constraints", false, "Include field constraints (read-only, max length, etc.)")
	cmd.Flags().Bool("relationships", false, "Include navigation property relationships")
	return cmd
}

// ---------------------------------------------------------------------------
// d365 data find <odataPath> [--query <queryOptions>]
// ---------------------------------------------------------------------------

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find <odataPath>",
		Short: "Query entities with OData",
		Long: `Query D365 data entities using OData query options.

AI guardrails:
  - Warns to stderr if $select is not specified (performance best practice).
  - Auto-injects cross-company=true unless explicitly set in the query.

PowerShell: use single quotes for --query to avoid $ being interpreted:
  .\d365 data find Customers --query '$top=5&$select=CustomerAccount,Name'`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			var odataPath string
			if len(args) > 0 {
				odataPath = args[0]
			} else {
				val, err := promptForArg("Entity path", "e.g., Customers or SalesOrderHeaders")
				if err != nil {
					return fmt.Errorf("%s", err.Error())
				}
				odataPath = val
			}
			queryStr, _ := cmd.Flags().GetString("query")

			opts, err := parseRawQueryOptions(queryStr)
			if err != nil {
				e := clierrors.ValidationError(err.Error())
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			// AI guardrail: warn if $select is missing from query.
			if opts.Select == "" {
				fmt.Fprintln(os.Stderr,
					"Warning: $select not specified in query. Consider adding $select to limit returned fields and improve performance.")
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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			result, err := oc.QueryEntities(cmd.Context(), odataPath, opts)
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, result, start)
			return nil
		},
	}

	cmd.Flags().String("query", "", `OData query options (e.g., "$filter=Name eq 'Contoso'&$select=CustomerAccount,Name")`)
	return cmd
}

// ---------------------------------------------------------------------------
// d365 data create <odataPath> --data <json>
// ---------------------------------------------------------------------------

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <odataPath>",
		Short: "Create an entity via OData POST",
		Long:  `Create a new entity instance by POSTing JSON data to the specified OData path.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			odataPath := args[0]
			dataJSON, _ := cmd.Flags().GetString("data")

			// Validate JSON before connecting.
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
				return fmt.Errorf("invalid JSON in --data: %w", err)
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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			result, err := oc.CreateEntity(cmd.Context(), odataPath, data)
			if err != nil {
				e := clierrors.AsCLIError(err)
				RenderError(cmd, e.ToErrorInfo(), start)
				return nil
			}

			RenderSuccess(cmd, result, start)
			return nil
		},
	}

	cmd.Flags().String("data", "", "JSON object with entity field values")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

// ---------------------------------------------------------------------------
// d365 data update --data <json>
// ---------------------------------------------------------------------------

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update entities via OData PATCH",
		Long: `Update one or more entities by sending PATCH requests.

The --data flag accepts a JSON array of objects, each with:
  - ODataPath: the entity path including key
    (e.g., "Customers(dataAreaId='USMF',CustomerAccount='US-001')")
  - UpdatedFieldValues: a map of field names to new values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			dataJSON, _ := cmd.Flags().GetString("data")

			// Validate JSON before connecting.
			var updates []updateEntry
			if err := json.Unmarshal([]byte(dataJSON), &updates); err != nil {
				return fmt.Errorf("invalid JSON in --data: expected array of {ODataPath, UpdatedFieldValues}: %w", err)
			}
			if len(updates) == 0 {
				return fmt.Errorf("--data must contain at least one update entry")
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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			var results []map[string]interface{}
			for _, u := range updates {
				if err := oc.UpdateEntity(cmd.Context(), u.ODataPath, u.UpdatedFieldValues); err != nil {
					e := clierrors.AsCLIError(err)
					RenderError(cmd, e.ToErrorInfo(), start)
					return nil
				}
				results = append(results, map[string]interface{}{
					"ODataPath": u.ODataPath,
					"status":    "updated",
				})
			}

			RenderSuccess(cmd, map[string]interface{}{
				"updated": len(results),
				"results": results,
			}, start)
			return nil
		},
	}

	cmd.Flags().String("data", "", `JSON array of {"ODataPath":"...","UpdatedFieldValues":{...}} objects`)
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

// ---------------------------------------------------------------------------
// d365 data delete --paths <json> [--confirm]
// ---------------------------------------------------------------------------

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete entities via OData DELETE",
		Long: `Delete one or more entities by sending DELETE requests.

REQUIRES the --confirm flag as a guardrail for this destructive operation.
Without --confirm the command exits with a validation error.

The --paths flag accepts a JSON array of OData paths to delete.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			// Guardrail: require explicit --confirm for destructive operations.
			confirm, _ := cmd.Flags().GetBool("confirm")
			if !confirm {
				return fmt.Errorf("--confirm flag is required for delete operations (guardrail for destructive operation)")
			}

			pathsJSON, _ := cmd.Flags().GetString("paths")

			// Validate JSON before connecting.
			var paths []string
			if err := json.Unmarshal([]byte(pathsJSON), &paths); err != nil {
				return fmt.Errorf("invalid JSON in --paths: expected array of OData path strings: %w", err)
			}
			if len(paths) == 0 {
				return fmt.Errorf("--paths must contain at least one OData path")
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

			oc := client.NewODataClient(sess.Environment, tp, time.Duration(flagTimeout)*time.Second)
			oc.SetVerbose(flagVerbose)

			var results []map[string]interface{}
			for _, p := range paths {
				if err := oc.DeleteEntity(cmd.Context(), p); err != nil {
					e := clierrors.AsCLIError(err)
					RenderError(cmd, e.ToErrorInfo(), start)
					return nil
				}
				results = append(results, map[string]interface{}{
					"ODataPath": p,
					"status":    "deleted",
				})
			}

			RenderSuccess(cmd, map[string]interface{}{
				"deleted": len(results),
				"results": results,
			}, start)
			return nil
		},
	}

	cmd.Flags().String("paths", "", `JSON array of OData paths to delete`)
	cmd.Flags().Bool("confirm", false, "Required flag to confirm destructive delete operation")
	_ = cmd.MarkFlagRequired("paths")
	return cmd
}
