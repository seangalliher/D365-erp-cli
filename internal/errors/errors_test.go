package errors

import (
	"fmt"
	"testing"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func TestCLIError_Error(t *testing.T) {
	t.Parallel()

	err := New(types.ErrCodeValidation, "field is required")
	if err.Error() != "[VALIDATION_ERROR] field is required" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestCLIError_ErrorWithCause(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("network timeout")
	err := Wrap(types.ErrCodeConnection, "failed to connect", cause)

	expected := "[CONNECTION_ERROR] failed to connect: network timeout"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("root cause")
	err := Wrap(types.ErrCodeConnection, "wrapper", cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestCLIError_ToErrorInfo(t *testing.T) {
	t.Parallel()

	err := New(types.ErrCodeValidation, "bad input").WithSuggestion("check the data")
	info := err.ToErrorInfo()

	if info.Code != types.ErrCodeValidation {
		t.Errorf("expected code %q, got %q", types.ErrCodeValidation, info.Code)
	}
	if info.Message != "bad input" {
		t.Errorf("expected message 'bad input', got %q", info.Message)
	}
	if info.Suggestion != "check the data" {
		t.Errorf("expected suggestion 'check the data', got %q", info.Suggestion)
	}
}

func TestCLIError_ExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *CLIError
		wantCode int
	}{
		{"connection", ConnectionError("fail", nil), types.ExitConnectionError},
		{"auth", AuthError("fail", nil), types.ExitConnectionError},
		{"validation", ValidationError("fail"), types.ExitValidationError},
		{"not found", NotFoundError("entity", "X"), types.ExitCommandError},
		{"session required", SessionRequired(), types.ExitConnectionError},
		{"form required", FormRequired(), types.ExitCommandError},
		{"timeout", TimeoutError("query"), types.ExitConnectionError},
		{"input error", InputError("bad"), types.ExitValidationError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.ExitCode(); got != tt.wantCode {
				t.Errorf("ExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestCommonErrors_HaveSuggestions(t *testing.T) {
	t.Parallel()

	errs := []*CLIError{
		ConnectionError("fail", nil),
		AuthError("fail", nil),
		SessionRequired(),
		FormRequired(),
		DaemonError("fail", nil),
		TimeoutError("op"),
		ValidationError("fail"),
		NotFoundError("entity", "X"),
		ODataError("fail", nil),
		InputError("fail"),
	}

	for _, err := range errs {
		if err.Suggestion == "" {
			t.Errorf("error %q (code=%s) should have a suggestion", err.Message, err.Code)
		}
	}
}

func TestNewf(t *testing.T) {
	t.Parallel()

	err := Newf(types.ErrCodeValidation, "field %q must be %d chars", "name", 10)
	expected := `[VALIDATION_ERROR] field "name" must be 10 chars`
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAsCLIError(t *testing.T) {
	t.Parallel()

	// Regular error gets wrapped
	regular := fmt.Errorf("some error")
	cli := AsCLIError(regular)
	if cli.Code != types.ErrCodeConnection {
		t.Errorf("expected CONNECTION_ERROR code, got %q", cli.Code)
	}

	// CLIError passes through
	original := ValidationError("already cli")
	cli = AsCLIError(original)
	if cli != original {
		t.Error("expected same CLIError instance")
	}
}
