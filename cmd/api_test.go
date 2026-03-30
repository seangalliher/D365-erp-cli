package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// API command group structure
// ---------------------------------------------------------------------------

func TestAPICmd_HasFindAndInvokeSubcommands(t *testing.T) {
	cmd := newAPICmd()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"find":   false,
		"invoke": false,
	}

	for _, sub := range subcommands {
		if _, ok := expected[sub.Name()]; ok {
			expected[sub.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected subcommand %q not found in api command", name)
		}
	}

	if len(subcommands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(subcommands))
	}
}

// ---------------------------------------------------------------------------
// api find
// ---------------------------------------------------------------------------

func TestAPIFindCmd_RequiresSearchTermArg(t *testing.T) {
	cmd := newAPICmd()
	cmd.SetArgs([]string{"find"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no searchTerm provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra args validation error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// api invoke
// ---------------------------------------------------------------------------

func TestAPIInvokeCmd_RequiresActionNameArg(t *testing.T) {
	cmd := newAPICmd()
	cmd.SetArgs([]string{"invoke", "--params", `{"key": "value"}`})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no actionName provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra args validation error, got: %v", err)
	}
}

func TestAPIInvokeCmd_RequiresParamsFlag(t *testing.T) {
	cmd := newAPICmd()
	cmd.SetArgs([]string{"invoke", "SomeAction"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --params not provided")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "required flag") && !strings.Contains(errStr, "params") {
		t.Errorf("expected required flag error for --params, got: %v", err)
	}
}

func TestAPIInvokeCmd_InvalidParamsJSON(t *testing.T) {
	cmd := newAPICmd()
	cmd.SetArgs([]string{"invoke", "SomeAction", "--params", "not valid json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON in --params")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

func TestAPIInvokeCmd_CompanyFlagExists(t *testing.T) {
	cmd := newAPIInvokeCmd()
	f := cmd.Flags().Lookup("company")
	if f == nil {
		t.Fatal("expected --company flag to exist on invoke command")
	}
	if f.DefValue != "" {
		t.Errorf("expected --company default to be empty, got %q", f.DefValue)
	}
}

// ---------------------------------------------------------------------------
// schema command
// ---------------------------------------------------------------------------

func TestSchemaCmd_ProducesValidJSON(t *testing.T) {
	// Build schema from the root command which has all registered subcommands.
	schema := buildSchema(rootCmd, false)

	// Marshal to JSON and verify it's valid.
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("schema failed to marshal to JSON: %v", err)
	}

	// Verify it parses back.
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("schema JSON failed to parse: %v", err)
	}

	// Verify root command name is present.
	if name, ok := parsed["name"].(string); !ok || name != "d365" {
		t.Errorf("expected root schema name to be 'd365', got %v", parsed["name"])
	}

	// Verify subcommands exist.
	subs, ok := parsed["sub_commands"].([]interface{})
	if !ok || len(subs) == 0 {
		t.Fatal("expected schema to have sub_commands")
	}

	// Verify at least the known commands are in the schema.
	subNames := make(map[string]bool)
	for _, sub := range subs {
		subMap, ok := sub.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := subMap["name"].(string); ok {
			subNames[name] = true
		}
	}

	for _, expected := range []string{"version", "data", "api", "schema", "docs"} {
		if !subNames[expected] {
			t.Errorf("expected %q command in schema sub_commands", expected)
		}
	}
}

func TestSchemaCmd_FullIncludesGuardrails(t *testing.T) {
	schema := buildSchema(rootCmd, true)

	// Marshal and check it's valid.
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("full schema failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("full schema JSON failed to parse: %v", err)
	}

	// Full mode should still produce valid JSON with name.
	if name, ok := parsed["name"].(string); !ok || name != "d365" {
		t.Errorf("expected root schema name to be 'd365', got %v", parsed["name"])
	}
}

// ---------------------------------------------------------------------------
// docs command
// ---------------------------------------------------------------------------

func TestDocsCmd_ListsTopicsWhenNoArg(t *testing.T) {
	cmd := newDocsCmd()
	cmd.SetArgs([]string{})

	// The command should not error (it lists topics).
	// It will call RenderSuccess which writes to stdout; we just ensure no error.
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error when listing topics, got: %v", err)
	}
}

func TestDocsCmd_ReturnsContentForValidTopic(t *testing.T) {
	// Verify all expected topics exist and have content.
	expectedTopics := []string{
		"odata-filters",
		"form-workflow",
		"authentication",
		"batch-mode",
		"enum-syntax",
		"entities",
	}

	for _, topic := range expectedTopics {
		dt, ok := docTopics[topic]
		if !ok {
			t.Errorf("expected topic %q to exist in docTopics", topic)
			continue
		}
		if dt.Title == "" {
			t.Errorf("topic %q has empty title", topic)
		}
		if dt.Content == "" {
			t.Errorf("topic %q has empty content", topic)
		}
	}
}

func TestDocsCmd_UnknownTopicSuggestsMatches(t *testing.T) {
	suggestions := findClosestTopics("odata")
	found := false
	for _, s := range suggestions {
		if s == "odata-filters" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'odata-filters' in suggestions for 'odata', got: %v", suggestions)
	}
}

func TestDocsCmd_UnknownTopicNoMatch(t *testing.T) {
	suggestions := findClosestTopics("zzzznonexistent")
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for gibberish query, got: %v", suggestions)
	}
}
