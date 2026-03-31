// Package errors provides structured error types for the D365 CLI.
// All errors include machine-readable codes, human-readable messages,
// and actionable suggestions for resolution.
package errors

import (
	"fmt"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

// CLIError is a structured error with a code and optional suggestion.
type CLIError struct {
	Code       string
	Message    string
	Suggestion string
	Cause      error
}

func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *CLIError) Unwrap() error {
	return e.Cause
}

// ToErrorInfo converts a CLIError to a types.ErrorInfo for JSON output.
func (e *CLIError) ToErrorInfo() *types.ErrorInfo {
	info := &types.ErrorInfo{
		Code:       e.Code,
		Message:    e.Message,
		Suggestion: e.Suggestion,
	}
	if e.Cause != nil {
		info.Details = e.Cause.Error()
	}
	return info
}

// ExitCode returns the process exit code for this error.
func (e *CLIError) ExitCode() int {
	return types.ExitCodeForError(e.Code)
}

// New creates a new CLIError.
func New(code, message string) *CLIError {
	return &CLIError{Code: code, Message: message}
}

// Newf creates a new CLIError with a formatted message.
func Newf(code, format string, args ...interface{}) *CLIError {
	return &CLIError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap wraps an existing error with a CLIError.
func Wrap(code string, message string, cause error) *CLIError {
	return &CLIError{Code: code, Message: message, Cause: cause}
}

// WithSuggestion adds a suggestion to a CLIError.
func (e *CLIError) WithSuggestion(s string) *CLIError {
	e.Suggestion = s
	return e
}

// Common error constructors.

func ConnectionError(msg string, cause error) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeConnection,
		Message:    msg,
		Cause:      cause,
		Suggestion: "Check the environment URL and your network connection. Use 'd365 status' to check connectivity.",
	}
}

func AuthError(msg string, cause error) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeAuthentication,
		Message:    msg,
		Cause:      cause,
		Suggestion: "Run 'd365 connect <url>' to authenticate. Check your credentials or try a different auth method with --auth.",
	}
}

func ValidationError(msg string) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeValidation,
		Message:    msg,
		Suggestion: "Check command arguments and flags. Run '.\\d365 <command> --help' for usage.",
	}
}

func NotFoundError(resource, identifier string) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found: %s", resource, identifier),
		Suggestion: "Run '.\\d365 data find-type <keyword>' to search for the correct entity name.",
	}
}

func SessionRequired() *CLIError {
	return &CLIError{
		Code:       types.ErrCodeSessionRequired,
		Message:    "no active session",
		Suggestion: "Run 'd365 connect <environment-url>' to establish a connection first.",
	}
}

func FormRequired() *CLIError {
	return &CLIError{
		Code:       types.ErrCodeFormRequired,
		Message:    "no form is currently open",
		Suggestion: "Run 'd365 form open <menuItemName> --type Display' to open a form first.",
	}
}

func DaemonError(msg string, cause error) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeDaemonError,
		Message:    msg,
		Cause:      cause,
		Suggestion: "Try 'd365 daemon restart' or check daemon logs.",
	}
}

func ODataError(msg string, cause error) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeODataError,
		Message:    msg,
		Cause:      cause,
		Suggestion: "Check your OData query syntax. Run '.\\d365 docs odata-filters' for reference.",
	}
}

func InputError(msg string) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeInvalidInput,
		Message:    msg,
		Suggestion: "Check the input format. Run '.\\d365 <command> --help' for expected syntax.",
	}
}

func TimeoutError(operation string) *CLIError {
	return &CLIError{
		Code:       types.ErrCodeTimeout,
		Message:    fmt.Sprintf("operation timed out: %s", operation),
		Suggestion: "The D365 server may be slow. Try again or increase timeout with --timeout.",
	}
}

// AuthConfigError creates an error for missing auth configuration after a
// successful connect. The field name is mapped to its environment variable
// so the suggestion is immediately actionable.
func AuthConfigError(method, missingField string) *CLIError {
	envVar := envVarForField(missingField)
	return &CLIError{
		Code:    types.ErrCodeAuthentication,
		Message: fmt.Sprintf("your saved profile uses %q auth, but %s is not set", method, missingField),
		Suggestion: fmt.Sprintf(
			"Set the %s environment variable, or reconnect:\n  d365 connect <url> --auth %s",
			envVar, method),
	}
}

// envVarForField maps an auth field name to its D365_* environment variable.
func envVarForField(field string) string {
	switch field {
	case "client-secret":
		return "D365_CLIENT_SECRET"
	case "tenant":
		return "D365_TENANT_ID"
	case "client-id":
		return "D365_CLIENT_ID"
	default:
		return "D365_" + field
	}
}

// AsCLIError attempts to convert an error to a CLIError.
// Returns the CLIError if successful, or wraps the error if not.
func AsCLIError(err error) *CLIError {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr
	}
	return Wrap(types.ErrCodeConnection, err.Error(), err)
}
