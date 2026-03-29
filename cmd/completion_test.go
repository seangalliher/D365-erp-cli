package cmd

import (
	"strings"
	"testing"
)

func TestCompletionCmd_Structure(t *testing.T) {
	cmd := newCompletionCmd()

	if cmd.Use != "completion <shell>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if len(cmd.ValidArgs) != 4 {
		t.Errorf("expected 4 valid args, got %d", len(cmd.ValidArgs))
	}
}

func TestCompletionCmd_PowerShell(t *testing.T) {
	cmd := newCompletionCmd()
	buf := new(strings.Builder)
	cmd.SetOut(buf)

	// Execute with powershell arg — the completion script is written to rootCmd's stdout
	// but we verify it doesn't error
	rootCmd.SetArgs([]string{"completion", "powershell"})
	err := cmd.RunE(cmd, []string{"powershell"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionCmd_InvalidShell(t *testing.T) {
	cmd := newCompletionCmd()
	err := cmd.RunE(cmd, []string{"nushell"})
	if err == nil {
		t.Fatal("expected error for invalid shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected 'unsupported shell' error, got: %v", err)
	}
}
