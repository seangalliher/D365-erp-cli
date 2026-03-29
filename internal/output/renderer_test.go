package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"table", FormatTable, false},
		{"csv", FormatCSV, false},
		{"raw", FormatRaw, false},
		{"", FormatJSON, false},
		{"xml", "", true},
		{"yaml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatJSON, false)

	resp := &types.Response{
		Success: true,
		Command: "test",
		Data:    map[string]string{"key": "value"},
		Metadata: &types.Metadata{
			DurationMs: 42,
			Timestamp:  "2026-01-01T00:00:00Z",
		},
	}

	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var result types.Response
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if !result.Success {
		t.Error("expected success=true in JSON output")
	}
	if result.Command != "test" {
		t.Errorf("expected command='test', got %q", result.Command)
	}
}

func TestRenderJSON_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatJSON, false)

	resp := types.ErrorResponse("test", &types.ErrorInfo{
		Code:       "TEST_ERR",
		Message:    "something broke",
		Suggestion: "fix it",
	}, &types.Metadata{DurationMs: 10, Timestamp: "2026-01-01T00:00:00Z"})

	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var result types.Response
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result.Success {
		t.Error("expected success=false")
	}
	if result.Error.Code != "TEST_ERR" {
		t.Errorf("expected error code 'TEST_ERR', got %q", result.Error.Code)
	}
}

func TestRenderTable_Map(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatTable, false)

	resp := &types.Response{
		Success: true,
		Command: "test",
		Data:    map[string]interface{}{"name": "Contoso", "id": "US-001"},
	}

	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Contoso") {
		t.Errorf("table output should contain 'Contoso', got: %s", output)
	}
}

func TestRenderTable_Slice(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatTable, false)

	resp := &types.Response{
		Success: true,
		Command: "test",
		Data: []interface{}{
			map[string]interface{}{"name": "Contoso", "id": "1"},
			map[string]interface{}{"name": "Fabrikam", "id": "2"},
		},
	}

	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Contoso") {
		t.Error("expected table to contain 'Contoso'")
	}
	if !strings.Contains(output, "Fabrikam") {
		t.Error("expected table to contain 'Fabrikam'")
	}
	if !strings.Contains(output, "(2 rows)") {
		t.Error("expected '(2 rows)' count")
	}
}

func TestRenderTable_EmptySlice(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatTable, false)

	resp := &types.Response{Success: true, Data: []interface{}{}}
	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(buf.String(), "(no results)") {
		t.Errorf("expected '(no results)', got %q", buf.String())
	}
}

func TestRenderTable_NilData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatTable, false)

	resp := &types.Response{Success: true, Data: nil}
	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected 'OK' for nil data, got %q", buf.String())
	}
}

func TestRenderTable_NilDataQuiet(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatTable, true)

	resp := &types.Response{Success: true, Data: nil}
	_ = r.Render(resp)

	if buf.String() != "" {
		t.Errorf("expected no output in quiet mode, got %q", buf.String())
	}
}

func TestRenderTable_Error(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	r := NewRenderer(&out, &errOut, FormatTable, false)

	resp := types.ErrorResponse("test", &types.ErrorInfo{
		Code:       "ERR",
		Message:    "broken",
		Suggestion: "fix it",
	}, nil)

	_ = r.Render(resp)

	if !strings.Contains(errOut.String(), "broken") {
		t.Errorf("expected error in stderr, got %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "fix it") {
		t.Errorf("expected suggestion in stderr, got %q", errOut.String())
	}
}

func TestRenderCSV(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatCSV, false)

	resp := &types.Response{
		Success: true,
		Data: []interface{}{
			map[string]interface{}{"name": "Contoso", "id": "1"},
			map[string]interface{}{"name": "Fabrikam", "id": "2"},
		},
	}

	if err := r.Render(resp); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("expected 3 CSV lines, got %d", len(lines))
	}
}

func TestRenderRaw_String(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatRaw, false)

	resp := &types.Response{Success: true, Data: "raw data here"}
	_ = r.Render(resp)

	if buf.String() != "raw data here" {
		t.Errorf("expected 'raw data here', got %q", buf.String())
	}
}

func TestRenderError_ExitCode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewRenderer(&buf, &bytes.Buffer{}, FormatJSON, false)

	resp := types.ErrorResponse("test", &types.ErrorInfo{
		Code:    types.ErrCodeValidation,
		Message: "bad input",
	}, nil)

	exitCode := r.RenderError(resp)
	if exitCode != types.ExitValidationError {
		t.Errorf("expected exit code %d, got %d", types.ExitValidationError, exitCode)
	}
}

func TestRendererFormat(t *testing.T) {
	t.Parallel()

	r := NewRenderer(nil, nil, FormatCSV, false)
	if r.Format() != FormatCSV {
		t.Errorf("expected FormatCSV, got %v", r.Format())
	}
}
