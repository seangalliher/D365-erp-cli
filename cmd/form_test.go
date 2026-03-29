package cmd

import (
	"testing"

	"github.com/seangalliher/d365-erp-cli/internal/daemon"
	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func TestParseControlValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{"single", []string{"Name=John"}, 1, false},
		{"multiple", []string{"Name=John", "City=Seattle"}, 2, false},
		{"with spaces", []string{"City=New York"}, 1, false},
		{"empty value", []string{"Name="}, 1, false},
		{"no equals", []string{"InvalidArg"}, 0, true},
		{"equals in value", []string{"Filter=A=B"}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseControlValues(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.want {
				t.Errorf("got %d values, want %d", len(result), tt.want)
			}
		})
	}
}

func TestParseControlValues_Content(t *testing.T) {
	t.Parallel()

	result, err := parseControlValues([]string{"Name=John", "AccountNum=US-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result[0].ControlName != "Name" {
		t.Errorf("expected ControlName 'Name', got %q", result[0].ControlName)
	}
	if result[0].Value != "John" {
		t.Errorf("expected Value 'John', got %q", result[0].Value)
	}
	if result[1].ControlName != "AccountNum" {
		t.Errorf("expected ControlName 'AccountNum', got %q", result[1].ControlName)
	}
	if result[1].Value != "US-001" {
		t.Errorf("expected Value 'US-001', got %q", result[1].Value)
	}
}

func TestFindEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{"a=b", 1},
		{"name=value", 4},
		{"no-equals", -1},
		{"=leading", 0},
		{"a=b=c", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := findEquals(tt.input)
			if got != tt.want {
				t.Errorf("findEquals(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveCompany(t *testing.T) {
	t.Parallel()

	// Override takes precedence
	result := resolveCompany("USMF")
	if result != "USMF" {
		t.Errorf("expected 'USMF', got %q", result)
	}

	// Empty falls back to GetCompany()
	result = resolveCompany("")
	// We can't predict GetCompany() output in tests, just ensure no panic
	_ = result
}

func formDaemonError(code, msg string) *types.ErrorInfo {
	return &types.ErrorInfo{
		Code:    code,
		Message: msg,
	}
}

func TestFormDaemonError(t *testing.T) {
	t.Parallel()

	errInfo := formDaemonError("TEST_ERR", "something failed")
	if errInfo.Code != "TEST_ERR" {
		t.Errorf("expected code 'TEST_ERR', got %q", errInfo.Code)
	}
	if errInfo.Message != "something failed" {
		t.Errorf("expected message 'something failed', got %q", errInfo.Message)
	}
}

func TestDaemonProtocolConstants(t *testing.T) {
	t.Parallel()

	// Verify form commands map to expected daemon protocol commands
	expected := map[string]string{
		"find":          daemon.CmdFormFindMenu,
		"open":          daemon.CmdFormOpen,
		"close":         daemon.CmdFormClose,
		"save":          daemon.CmdFormSave,
		"state":         daemon.CmdFormState,
		"click":         daemon.CmdFormClick,
		"set":           daemon.CmdFormSetValues,
		"lookup":        daemon.CmdFormOpenLookup,
		"tab":           daemon.CmdFormOpenTab,
		"filter":        daemon.CmdFormFilter,
		"grid-filter":   daemon.CmdFormFilterGrid,
		"grid-select":   daemon.CmdFormSelectRow,
		"grid-sort":     daemon.CmdFormSortGrid,
		"find-controls": daemon.CmdFormFind,
	}

	for name, cmd := range expected {
		if cmd == "" {
			t.Errorf("command %q maps to empty daemon command", name)
		}
	}
}
