package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/seangalliher/d365-erp-cli/internal/daemon"
	clierrors "github.com/seangalliher/d365-erp-cli/internal/errors"
)

func init() {
	formCmd := &cobra.Command{
		Use:   "form",
		Short: "Manage D365 form sessions (stateful)",
		Long: `The form command group manages stateful D365 form interactions.

These commands communicate with a background daemon that maintains
form session state. The daemon is auto-started on first use.

Workflow:
  1. .\d365 form find <search>        # Find menu items
  2. .\d365 form open <name> --type Display  # Open a form
  3. .\d365 form click <control>      # Interact with controls
  4. .\d365 form set <name>=<value>   # Set field values
  5. .\d365 form save                 # Save changes
  6. .\d365 form close                # Close the form`,
	}

	formCmd.AddCommand(
		newFormFindCmd(),
		newFormOpenCmd(),
		newFormCloseCmd(),
		newFormSaveCmd(),
		newFormStateCmd(),
		newFormClickCmd(),
		newFormSetCmd(),
		newFormLookupCmd(),
		newFormTabCmd(),
		newFormFilterCmd(),
		newFormGridFilterCmd(),
		newFormGridSelectCmd(),
		newFormGridSortCmd(),
		newFormFindControlsCmd(),
	)

	rootCmd.AddCommand(formCmd)
}

// requireDaemon ensures the daemon is running and returns a connected client.
func requireDaemon() (*daemon.Client, error) {
	if err := daemon.EnsureDaemon(); err != nil {
		return nil, err
	}
	return daemon.Connect()
}

// sendFormCommand sends a command to the daemon and returns the response data.
func sendFormCommand(command string, args interface{}) (json.RawMessage, error) {
	client, err := requireDaemon()
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	var resp *daemon.Response
	err = withSpinner("Processing...", func() error {
		var sendErr error
		resp, sendErr = client.SendCommand(command, args)
		return sendErr
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// formSuccess renders a successful form command response.
func formSuccess(cmd *cobra.Command, data json.RawMessage, start time.Time) {
	var parsed interface{}
	if data != nil {
		_ = json.Unmarshal(data, &parsed)
	}
	RenderSuccess(cmd, parsed, start)
}

// formError renders a form command error.
func formError(cmd *cobra.Command, err error, start time.Time) {
	cliErr := clierrors.AsCLIError(err)
	RenderError(cmd, cliErr.ToErrorInfo(), start)
}

// --- form find ---

func newFormFindCmd() *cobra.Command {
	var company string
	var responseSize string

	cmd := &cobra.Command{
		Use:   "find <search-term>",
		Short: "Find menu items (forms) by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			data, err := sendFormCommand(daemon.CmdFormFindMenu, &daemon.FormFindMenuArgs{
				Filter:       args[0],
				Company:      resolveCompany(company),
				ResponseSize: responseSize,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&company, "company", "", "Company ID")
	cmd.Flags().StringVar(&responseSize, "response-size", "50", "Max results to return")
	return cmd
}

// --- form open ---

func newFormOpenCmd() *cobra.Command {
	var menuType string
	var company string

	cmd := &cobra.Command{
		Use:   "open <menu-item-name>",
		Short: "Open a form by menu item name",
		Long: `Opens a D365 form via its menu item. The daemon is auto-started if needed.

Examples:
  .\d365 form open CustTable --type Display
  .\d365 form open LedgerJournalTable --type Display --company USMF`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			data, err := sendFormCommand(daemon.CmdFormOpen, &daemon.FormOpenArgs{
				MenuItemName: args[0],
				MenuItemType: menuType,
				Company:      resolveCompany(company),
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&menuType, "type", "Display", "Menu item type: Display or Action")
	cmd.Flags().StringVar(&company, "company", "", "Company ID override")
	return cmd
}

// --- form close ---

func newFormCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Close the currently open form",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormClose, nil)
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// --- form save ---

func newFormSaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "save",
		Short: "Save the current form",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormSave, nil)
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// --- form state ---

func newFormStateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "state",
		Short: "Get the current form state",
		Long:  "Returns the form state including control names, values, types, and enabled status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormState, nil)
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// --- form click ---

func newFormClickCmd() *cobra.Command {
	var actionID string

	cmd := &cobra.Command{
		Use:   "click <control-name>",
		Short: "Click a control on the form",
		Long: `Clicks a control such as a button or tab. For controls with
sub-actions (list boxes, tree nodes), use --action to specify the action.

Examples:
  .\d365 form click PostButton
  .\d365 form click StatusDropdown --action "Completed"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormClick, &daemon.FormClickArgs{
				ControlName: args[0],
				ActionID:    actionID,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&actionID, "action", "", "Action ID for the control (list/tree items)")
	return cmd
}

// --- form set ---

func newFormSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>=<value> [<name>=<value>...]",
		Short: "Set one or more control values",
		Long: `Sets values on form controls. Provide name=value pairs as arguments.

For lookup controls, use '.\d365 form lookup' instead.

Examples:
  .\d365 form set CustomerAccount=US-001
  .\d365 form set FirstName=John LastName=Doe City="New York"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			values, err := parseControlValues(args)
			if err != nil {
				formError(cmd, clierrors.InputError(err.Error()), start)
				return nil
			}

			data, rErr := sendFormCommand(daemon.CmdFormSetValues, &daemon.FormSetValuesArgs{
				Values: values,
			})
			if rErr != nil {
				formError(cmd, rErr, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

func parseControlValues(args []string) ([]daemon.ControlValue, error) {
	var values []daemon.ControlValue
	for _, arg := range args {
		idx := findEquals(arg)
		if idx < 0 {
			return nil, fmt.Errorf("invalid format %q: expected name=value", arg)
		}
		values = append(values, daemon.ControlValue{
			ControlName: arg[:idx],
			Value:       arg[idx+1:],
		})
	}
	return values, nil
}

func findEquals(s string) int {
	for i, c := range s {
		if c == '=' {
			return i
		}
	}
	return -1
}

// --- form lookup ---

func newFormLookupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lookup <control-name>",
		Short: "Open a lookup on a control",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormOpenLookup, &daemon.FormLookupArgs{
				LookupName: args[0],
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// --- form tab ---

func newFormTabCmd() *cobra.Command {
	var action string

	cmd := &cobra.Command{
		Use:   "tab <tab-name>",
		Short: "Open or close a tab on the form",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormOpenTab, &daemon.FormTabArgs{
				TabName:   args[0],
				TabAction: action,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&action, "action", "Open", "Tab action: Open or Close")
	return cmd
}

// --- form filter ---

func newFormFilterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "filter <control-name> <value>",
		Short: "Apply a filter on the form",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormFilter, &daemon.FormFilterArgs{
				ControlName: args[0],
				FilterValue: args[1],
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// --- form grid-filter ---

func newFormGridFilterCmd() *cobra.Command {
	var gridName string

	cmd := &cobra.Command{
		Use:   "grid-filter <column-name> [value]",
		Short: "Filter a grid by column",
		Long: `Filters a grid column. Pass an empty value to clear the filter.

Examples:
  .\d365 form grid-filter CustomerAccount 'US-*' --grid MainGrid
  .\d365 form grid-filter Status '' --grid MainGrid  # clear filter`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			value := ""
			if len(args) > 1 {
				value = args[1]
			}
			data, err := sendFormCommand(daemon.CmdFormFilterGrid, &daemon.FormGridFilterArgs{
				GridName:        gridName,
				GridColumnName:  args[0],
				GridColumnValue: value,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&gridName, "grid", "", "Grid control name (required)")
	_ = cmd.MarkFlagRequired("grid")
	return cmd
}

// --- form grid-select ---

func newFormGridSelectCmd() *cobra.Command {
	var gridName string
	var marking string

	cmd := &cobra.Command{
		Use:   "grid-select <row-number>",
		Short: "Select a row in a grid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormSelectRow, &daemon.FormSelectRowArgs{
				GridName:  gridName,
				RowNumber: args[0],
				Marking:   marking,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&gridName, "grid", "", "Grid control name (required)")
	cmd.Flags().StringVar(&marking, "marking", "", "Mark row: Marked or Unmarked")
	_ = cmd.MarkFlagRequired("grid")
	return cmd
}

// --- form grid-sort ---

func newFormGridSortCmd() *cobra.Command {
	var gridName string
	var direction string

	cmd := &cobra.Command{
		Use:   "grid-sort <column-name>",
		Short: "Sort a grid by column",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormSortGrid, &daemon.FormSortGridArgs{
				GridName:       gridName,
				GridColumnName: args[0],
				SortDirection:  direction,
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
	cmd.Flags().StringVar(&gridName, "grid", "", "Grid control name (required)")
	cmd.Flags().StringVar(&direction, "direction", "Ascending", "Sort direction: Ascending or Descending")
	_ = cmd.MarkFlagRequired("grid")
	return cmd
}

// --- form find-controls ---

func newFormFindControlsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find-controls <search-term>",
		Short: "Search for controls on the current form",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			data, err := sendFormCommand(daemon.CmdFormFind, &daemon.FormFindControlsArgs{
				SearchTerm: args[0],
			})
			if err != nil {
				formError(cmd, err, start)
				return nil
			}
			formSuccess(cmd, data, start)
			return nil
		},
	}
}

// resolveCompany returns the provided company or falls back to the global company.
func resolveCompany(override string) string {
	if override != "" {
		return override
	}
	return GetCompany()
}
