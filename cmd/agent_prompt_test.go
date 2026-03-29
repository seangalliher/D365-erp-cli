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
