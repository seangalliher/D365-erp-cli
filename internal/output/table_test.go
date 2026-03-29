package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableWriter_BasicRender(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := NewTableWriter(&buf, []string{"Name", "ID"})
	tw.AddRow([]string{"Contoso", "1"})
	tw.AddRow([]string{"Fabrikam", "2"})
	tw.Render()

	output := buf.String()

	if !strings.Contains(output, "Name") {
		t.Error("expected header 'Name' in output")
	}
	if !strings.Contains(output, "Contoso") {
		t.Error("expected 'Contoso' in output")
	}
	if !strings.Contains(output, "Fabrikam") {
		t.Error("expected 'Fabrikam' in output")
	}
	if !strings.Contains(output, "(2 rows)") {
		t.Error("expected '(2 rows)' count")
	}
	if !strings.Contains(output, "---") {
		t.Error("expected separator line")
	}
}

func TestTableWriter_EmptyHeaders(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := NewTableWriter(&buf, []string{})
	tw.Render()

	if buf.String() != "" {
		t.Errorf("expected no output for empty headers, got %q", buf.String())
	}
}

func TestTableWriter_LongValues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := NewTableWriter(&buf, []string{"Description"})
	longVal := strings.Repeat("x", 100)
	tw.AddRow([]string{longVal})
	tw.Render()

	output := buf.String()
	// Long values should be truncated with "..."
	if !strings.Contains(output, "...") {
		t.Error("expected long values to be truncated with '...'")
	}
}

func TestTableWriter_ZeroRows(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tw := NewTableWriter(&buf, []string{"A", "B"})
	tw.Render()

	if !strings.Contains(buf.String(), "(0 rows)") {
		t.Errorf("expected '(0 rows)', got %q", buf.String())
	}
}
