// Package daemon provides the IPC protocol and client for communicating
// with the d365 form daemon. The daemon manages stateful form sessions
// via a JSON-over-pipe protocol.
package daemon

import "encoding/json"

// Request is the IPC request envelope sent from CLI to daemon.
type Request struct {
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Args    json.RawMessage `json:"args,omitempty"`
}

// Response is the IPC response envelope returned from daemon to CLI.
type Response struct {
	ID      string          `json:"id"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorDetail    `json:"error,omitempty"`
}

// ErrorDetail carries error info from the daemon.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Well-known daemon commands.
const (
	CmdPing           = "ping"
	CmdShutdown       = "shutdown"
	CmdFormOpen       = "form.open"
	CmdFormClose      = "form.close"
	CmdFormSave       = "form.save"
	CmdFormState      = "form.state"
	CmdFormClick      = "form.click"
	CmdFormSetValues  = "form.set_values"
	CmdFormOpenLookup = "form.open_lookup"
	CmdFormOpenTab    = "form.open_tab"
	CmdFormFilter     = "form.filter"
	CmdFormFilterGrid = "form.filter_grid"
	CmdFormSelectRow  = "form.select_row"
	CmdFormSortGrid   = "form.sort_grid"
	CmdFormFind       = "form.find_controls"
	CmdFormFindMenu   = "form.find_menu"
)

// PingResponse is the response to a ping command.
type PingResponse struct {
	Status    string `json:"status"`
	Uptime    int64  `json:"uptime_seconds"`
	FormOpen  bool   `json:"form_open"`
	FormName  string `json:"form_name,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// FormOpenArgs are the arguments for the form.open command.
type FormOpenArgs struct {
	MenuItemName string `json:"name"`
	MenuItemType string `json:"type"`
	Company      string `json:"company,omitempty"`
}

// FormClickArgs are the arguments for the form.click command.
type FormClickArgs struct {
	ControlName string `json:"controlName"`
	ActionID    string `json:"actionId,omitempty"`
}

// FormSetValuesArgs are the arguments for the form.set_values command.
type FormSetValuesArgs struct {
	Values []ControlValue `json:"setControlValues"`
}

// ControlValue is a name/value pair for setting a control.
type ControlValue struct {
	ControlName string `json:"controlName"`
	Value       string `json:"value"`
}

// FormLookupArgs are the arguments for the form.open_lookup command.
type FormLookupArgs struct {
	LookupName string `json:"lookupName"`
}

// FormTabArgs are the arguments for the form.open_tab command.
type FormTabArgs struct {
	TabName   string `json:"tabName"`
	TabAction string `json:"tabAction"` // Open or Close
}

// FormFilterArgs are the arguments for the form.filter command.
type FormFilterArgs struct {
	ControlName string `json:"controlName"`
	FilterValue string `json:"filterValue"`
}

// FormGridFilterArgs are the arguments for the form.filter_grid command.
type FormGridFilterArgs struct {
	GridName        string `json:"gridName"`
	GridColumnName  string `json:"gridColumnName"`
	GridColumnValue string `json:"gridColumnValue,omitempty"`
}

// FormSelectRowArgs are the arguments for the form.select_row command.
type FormSelectRowArgs struct {
	GridName  string `json:"gridName"`
	RowNumber string `json:"rowNumber"`
	Marking   string `json:"marking,omitempty"`
}

// FormSortGridArgs are the arguments for the form.sort_grid command.
type FormSortGridArgs struct {
	GridName       string `json:"gridName"`
	GridColumnName string `json:"gridColumnName"`
	SortDirection  string `json:"sortDirection"`
}

// FormFindControlsArgs are the arguments for the form.find_controls command.
type FormFindControlsArgs struct {
	SearchTerm string `json:"controlSearchTerm"`
}

// FormFindMenuArgs are the arguments for the form.find_menu command.
type FormFindMenuArgs struct {
	Filter       string `json:"menuItemFilter"`
	Company      string `json:"companyId,omitempty"`
	ResponseSize string `json:"responseSize,omitempty"`
}
