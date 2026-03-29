package cmd

import (
	"testing"
)

func TestQuickstartCmd_Structure(t *testing.T) {
	cmd := newQuickstartCmd()
	if cmd.Use != "quickstart" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestQuickstartCmd_HasLongDescription(t *testing.T) {
	cmd := newQuickstartCmd()
	if cmd.Long == "" {
		t.Error("quickstart should have a Long description")
	}
}
