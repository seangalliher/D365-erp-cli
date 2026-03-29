package batch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func TestProcessStream_Success(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{
			Success: true,
			Command: cmd.Command,
			Data:    map[string]string{"result": "ok"},
		}, nil
	}

	input := `{"command":"data.find","args":{"entity":"Customers"}}
{"command":"data.find","args":{"entity":"Vendors"}}
`

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, false)
	summary, err := executor.ProcessStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Total != 2 {
		t.Errorf("expected total=2, got %d", summary.Total)
	}
	if summary.Succeeded != 2 {
		t.Errorf("expected succeeded=2, got %d", summary.Succeeded)
	}
	if summary.Errors != 0 {
		t.Errorf("expected errors=0, got %d", summary.Errors)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 output lines, got %d", len(lines))
	}

	var result Result
	if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
		t.Fatalf("unmarshal first result: %v", err)
	}
	if !result.Success {
		t.Error("expected first result to succeed")
	}
}

func TestProcessStream_ParseError(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{Success: true, Command: cmd.Command}, nil
	}

	input := `not json
{"command":"data.find"}
`

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, false)
	summary, err := executor.ProcessStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Total != 2 {
		t.Errorf("expected total=2, got %d", summary.Total)
	}
	if summary.Errors != 1 {
		t.Errorf("expected errors=1, got %d", summary.Errors)
	}
	if summary.Succeeded != 1 {
		t.Errorf("expected succeeded=1, got %d", summary.Succeeded)
	}
}

func TestProcessStream_StopOnError(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{
			Success: false,
			Command: cmd.Command,
			Error:   &types.ErrorInfo{Code: "ERR", Message: "fail"},
		}, nil
	}

	input := `{"command":"cmd1"}
{"command":"cmd2"}
`

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, true)
	_, err := executor.ProcessStream(strings.NewReader(input))
	if err == nil {
		t.Error("expected error with stopOnErr=true")
	}

	// Should have only processed the first command
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 output line (stop on error), got %d", len(lines))
	}
}

func TestProcessStream_EmptyLines(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{Success: true, Command: cmd.Command}, nil
	}

	input := `
{"command":"cmd1"}

{"command":"cmd2"}

`

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, false)
	summary, err := executor.ProcessStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("expected total=2 (skipping empty lines), got %d", summary.Total)
	}
}

func TestProcessStream_WithIDs(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{Success: true, Command: cmd.Command}, nil
	}

	input := `{"id":"req-1","command":"cmd1"}
`

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, false)
	_, err := executor.ProcessStream(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ID != "req-1" {
		t.Errorf("expected ID 'req-1', got %q", result.ID)
	}
}

func TestProcessStream_EmptyInput(t *testing.T) {
	t.Parallel()

	handler := func(cmd *Command) (*types.Response, error) {
		return &types.Response{Success: true}, nil
	}

	var output bytes.Buffer
	executor := NewExecutor(handler, &output, false)
	summary, err := executor.ProcessStream(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("expected total=0, got %d", summary.Total)
	}
}

func TestSummary(t *testing.T) {
	t.Parallel()

	s := &Summary{Total: 10, Succeeded: 7, Errors: 3}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	var decoded Summary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if decoded.Total != 10 {
		t.Errorf("expected total=10, got %d", decoded.Total)
	}
}
