// Package types provides shared type definitions for the D365 CLI.
package types

import "time"

// Response is the standard JSON envelope for all CLI output.
// Every command returns this structure, ensuring predictable parsing by AI agents.
type Response struct {
	Success  bool        `json:"success"`
	Command  string      `json:"command"`
	Data     interface{} `json:"data,omitempty"`
	Error    *ErrorInfo  `json:"error,omitempty"`
	Metadata *Metadata   `json:"metadata,omitempty"`
}

// ErrorInfo provides structured error details with actionable suggestions.
type ErrorInfo struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Details    string `json:"details,omitempty"`
}

// Metadata provides context about the command execution.
type Metadata struct {
	DurationMs int64  `json:"duration_ms"`
	Company    string `json:"company,omitempty"`
	Environment string `json:"environment,omitempty"`
	Timestamp  string `json:"timestamp"`
	Version    string `json:"version,omitempty"`
}

// NewMetadata creates metadata with the current timestamp and a duration.
func NewMetadata(duration time.Duration) *Metadata {
	return &Metadata{
		DurationMs: duration.Milliseconds(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
}

// SuccessResponse creates a successful response envelope.
func SuccessResponse(command string, data interface{}, meta *Metadata) *Response {
	return &Response{
		Success:  true,
		Command:  command,
		Data:     data,
		Metadata: meta,
	}
}

// ErrorResponse creates an error response envelope.
func ErrorResponse(command string, err *ErrorInfo, meta *Metadata) *Response {
	return &Response{
		Success:  false,
		Command:  command,
		Error:    err,
		Metadata: meta,
	}
}

// ExitCode maps error codes to process exit codes.
const (
	ExitSuccess         = 0
	ExitCommandError    = 1
	ExitConnectionError = 2
	ExitValidationError = 3
)

// Error codes used throughout the CLI.
const (
	ErrCodeConnection       = "CONNECTION_ERROR"
	ErrCodeAuthentication   = "AUTH_ERROR"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodePermission       = "PERMISSION_DENIED"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeServerError      = "SERVER_ERROR"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeSessionRequired  = "SESSION_REQUIRED"
	ErrCodeFormRequired     = "FORM_REQUIRED"
	ErrCodeDaemonError      = "DAEMON_ERROR"
	ErrCodeGuardrailBlock   = "GUARDRAIL_BLOCK"
	ErrCodeGuardrailWarn    = "GUARDRAIL_WARN"
	ErrCodeODataError       = "ODATA_ERROR"
	ErrCodePluginError      = "PLUGIN_ERROR"
)

// ExitCodeForError returns the appropriate process exit code for an error code.
func ExitCodeForError(code string) int {
	switch code {
	case ErrCodeConnection, ErrCodeAuthentication, ErrCodeTimeout, ErrCodeSessionRequired:
		return ExitConnectionError
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeGuardrailBlock:
		return ExitValidationError
	default:
		return ExitCommandError
	}
}

// ConnectionStatus represents the current connection state.
type ConnectionStatus struct {
	Connected   bool   `json:"connected"`
	Environment string `json:"environment,omitempty"`
	User        string `json:"user,omitempty"`
	Company     string `json:"company,omitempty"`
	TokenExpiry string `json:"token_expiry,omitempty"`
	DaemonPID   int    `json:"daemon_pid,omitempty"`
	ActiveForm  string `json:"active_form,omitempty"`
}

// EntityType represents an OData entity type from $metadata.
type EntityType struct {
	Name           string `json:"name"`
	CollectionName string `json:"collection_name"`
	Description    string `json:"description,omitempty"`
	IsPublic       bool   `json:"is_public"`
	IsReadOnly     bool   `json:"is_read_only"`
}

// EntityField represents a field definition within an entity.
type EntityField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	MaxLength   int    `json:"max_length,omitempty"`
	IsKey       bool   `json:"is_key,omitempty"`
	IsReadOnly  bool   `json:"is_read_only,omitempty"`
	Description string `json:"description,omitempty"`
}

// EntityMetadata contains the full metadata for an OData entity.
type EntityMetadata struct {
	EntitySetName  string                 `json:"entity_set_name"`
	Fields         []EntityField          `json:"fields,omitempty"`
	Keys           []string               `json:"keys,omitempty"`
	EnumValues     map[string][]EnumValue `json:"enum_values,omitempty"`
	Relationships  []Relationship         `json:"relationships,omitempty"`
}

// EnumValue represents a single enum member.
type EnumValue struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// Relationship represents a navigation property.
type Relationship struct {
	Name       string `json:"name"`
	Target     string `json:"target"`
	Multiplicity string `json:"multiplicity"`
}

// ActionInfo represents an available D365 action.
type ActionInfo struct {
	Name        string            `json:"name"`
	Label       string            `json:"label,omitempty"`
	Description string            `json:"description,omitempty"`
	Parameters  []ActionParameter `json:"parameters,omitempty"`
}

// ActionParameter describes a parameter for an action.
type ActionParameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	EnumType string `json:"enum_type,omitempty"`
}

// MenuItem represents a D365 menu item (form entry point).
type MenuItem struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type"` // Display or Action
}

// FormState represents the current state of an open form.
type FormState struct {
	FormName string        `json:"form_name"`
	Controls []FormControl `json:"controls,omitempty"`
	IsDirty  bool          `json:"is_dirty"`
}

// FormControl represents a control on a form.
type FormControl struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	Value     string `json:"value,omitempty"`
	IsEnabled bool   `json:"is_enabled"`
	IsLookup  bool   `json:"is_lookup,omitempty"`
}

// GuardrailResult represents the result of a guardrail check.
type GuardrailResult struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"` // error, warn, info
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Blocked    bool   `json:"blocked"`
}

// SchemaCommand represents a single command in the exported CLI schema.
type SchemaCommand struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Args        []SchemaArg       `json:"args,omitempty"`
	Flags       []SchemaFlag      `json:"flags,omitempty"`
	Examples    []SchemaExample   `json:"examples,omitempty"`
	Guardrails  []string          `json:"guardrails,omitempty"`
	SubCommands []SchemaCommand   `json:"sub_commands,omitempty"`
}

// SchemaArg represents a positional argument in the schema.
type SchemaArg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}

// SchemaFlag represents a flag in the schema.
type SchemaFlag struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

// SchemaExample shows a usage example.
type SchemaExample struct {
	Description string `json:"description"`
	Command     string `json:"command"`
}

// BatchCommand represents a single command in batch/pipeline mode.
type BatchCommand struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}
