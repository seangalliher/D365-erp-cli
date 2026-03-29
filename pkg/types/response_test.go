package types

import (
	"testing"
	"time"
)

func TestNewMetadata(t *testing.T) {
	t.Parallel()

	start := time.Now()
	time.Sleep(5 * time.Millisecond)
	meta := NewMetadata(time.Since(start))

	if meta.DurationMs < 5 {
		t.Errorf("expected duration >= 5ms, got %d", meta.DurationMs)
	}
	if meta.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestSuccessResponse(t *testing.T) {
	t.Parallel()

	data := map[string]string{"key": "value"}
	meta := &Metadata{DurationMs: 100, Timestamp: "2026-01-01T00:00:00Z"}
	resp := SuccessResponse("test cmd", data, meta)

	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Command != "test cmd" {
		t.Errorf("expected command='test cmd', got %q", resp.Command)
	}
	if resp.Error != nil {
		t.Error("expected error=nil for success response")
	}
	if resp.Metadata.DurationMs != 100 {
		t.Errorf("expected duration=100, got %d", resp.Metadata.DurationMs)
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	errInfo := &ErrorInfo{
		Code:       ErrCodeValidation,
		Message:    "test error",
		Suggestion: "fix it",
	}
	meta := &Metadata{DurationMs: 50, Timestamp: "2026-01-01T00:00:00Z"}
	resp := ErrorResponse("test cmd", errInfo, meta)

	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error.Code != ErrCodeValidation {
		t.Errorf("expected code=%q, got %q", ErrCodeValidation, resp.Error.Code)
	}
	if resp.Error.Suggestion != "fix it" {
		t.Errorf("expected suggestion='fix it', got %q", resp.Error.Suggestion)
	}
}

func TestExitCodeForError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code     string
		wantExit int
	}{
		{ErrCodeConnection, ExitConnectionError},
		{ErrCodeAuthentication, ExitConnectionError},
		{ErrCodeTimeout, ExitConnectionError},
		{ErrCodeValidation, ExitValidationError},
		{ErrCodeInvalidInput, ExitValidationError},
		{ErrCodeGuardrailBlock, ExitValidationError},
		{ErrCodeNotFound, ExitCommandError},
		{ErrCodeServerError, ExitCommandError},
		{ErrCodeODataError, ExitCommandError},
		{"UNKNOWN_CODE", ExitCommandError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()
			got := ExitCodeForError(tt.code)
			if got != tt.wantExit {
				t.Errorf("ExitCodeForError(%q) = %d, want %d", tt.code, got, tt.wantExit)
			}
		})
	}
}
