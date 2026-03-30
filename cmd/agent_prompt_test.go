package cmd

import (
	"strings"
	"testing"
)

func TestAgentPromptCmd_Structure(t *testing.T) {
	cmd := newAgentPromptCmd()
	if cmd.Use != "agent-prompt" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestBuildAgentPrompt_ContainsKeySections(t *testing.T) {
	prompt := buildAgentPrompt()

	sections := []string{
		"## Overview",
		"## Connection",
		"## Output Schema",
		"## Error Codes",
		"## Guardrails",
		"## Common D365 Entities",
		"## Common Workflows",
		"## PowerShell Tip",
		"## Command Reference",
	}

	for _, section := range sections {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt should contain section %q", section)
		}
	}
}

func TestBuildAgentPrompt_ContainsErrorCodes(t *testing.T) {
	prompt := buildAgentPrompt()

	codes := []string{
		"AUTH_ERROR",
		"SESSION_REQUIRED",
		"NOT_FOUND",
		"VALIDATION_ERROR",
		"ODATA_ERROR",
	}

	for _, code := range codes {
		if !strings.Contains(prompt, code) {
			t.Errorf("prompt should contain error code %q", code)
		}
	}
}

func TestBuildAgentPrompt_ContainsCommands(t *testing.T) {
	prompt := buildAgentPrompt()

	commands := []string{
		"connect",
		"status",
		"data",
		"doctor",
	}

	for _, cmd := range commands {
		if !strings.Contains(prompt, cmd) {
			t.Errorf("prompt should reference command %q", cmd)
		}
	}
}

func TestBuildAgentPrompt_ContainsGuardrails(t *testing.T) {
	prompt := buildAgentPrompt()

	if !strings.Contains(prompt, "cross-company") {
		t.Error("prompt should mention cross-company guardrail")
	}
	if !strings.Contains(prompt, "$select") {
		t.Error("prompt should mention $select guardrail")
	}
	if !strings.Contains(prompt, "--confirm") {
		t.Error("prompt should mention --confirm guardrail")
	}
}

func TestBuildAgentPrompt_ContainsEntityCatalog(t *testing.T) {
	prompt := buildAgentPrompt()

	entities := []string{
		"Customers",
		"Vendors",
		"SalesOrderHeaders",
		"LegalEntities",
		"MainAccounts",
		"LedgerChartOfAccounts",
	}
	for _, e := range entities {
		if !strings.Contains(prompt, e) {
			t.Errorf("prompt should contain entity %q in catalog", e)
		}
	}
}

func TestCommonEntities_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, e := range commonEntities {
		if seen[e.EntitySet] {
			t.Errorf("duplicate entity set name: %s", e.EntitySet)
		}
		seen[e.EntitySet] = true
		if e.Description == "" {
			t.Errorf("entity %s has empty description", e.EntitySet)
		}
		if e.KeyFields == "" {
			t.Errorf("entity %s has empty key fields", e.EntitySet)
		}
	}
}
