package cmd

import (
	"strings"
	"testing"
)

func TestPromptForArgFrom_ValidInput(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("https://contoso.operations.dynamics.com\n")
	w := new(strings.Builder)

	val, err := promptForArgFrom(r, w, "Environment URL", "e.g., https://contoso.operations.dynamics.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "https://contoso.operations.dynamics.com" {
		t.Errorf("expected URL, got %q", val)
	}

	output := w.String()
	if !strings.Contains(output, "Environment URL") {
		t.Errorf("prompt should contain argument name, got %q", output)
	}
}

func TestPromptForArgFrom_EmptyInput(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("\n")
	w := new(strings.Builder)

	_, err := promptForArgFrom(r, w, "Company ID", "e.g., USMF")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("expected missing argument error, got: %v", err)
	}
}

func TestPromptForArgFrom_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("  USMF  \n")
	w := new(strings.Builder)

	val, err := promptForArgFrom(r, w, "Company", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "USMF" {
		t.Errorf("expected trimmed value 'USMF', got %q", val)
	}
}

func TestPromptForArgFrom_ShowsHint(t *testing.T) {
	t.Parallel()

	r := strings.NewReader("test\n")
	w := new(strings.Builder)

	_, _ = promptForArgFrom(r, w, "URL", "https://...")
	output := w.String()
	if !strings.Contains(output, "https://...") {
		t.Errorf("prompt should show hint, got %q", output)
	}
}
