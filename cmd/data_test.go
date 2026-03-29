package cmd

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Command structure
// ---------------------------------------------------------------------------

func TestDataCmd_HasAllSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	subcommands := cmd.Commands()

	expected := map[string]bool{
		"find-type": false,
		"metadata":  false,
		"find":      false,
		"create":    false,
		"update":    false,
		"delete":    false,
	}

	for _, sub := range subcommands {
		if _, ok := expected[sub.Name()]; ok {
			expected[sub.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected subcommand %q not found in data command", name)
		}
	}

	if len(subcommands) != 6 {
		t.Errorf("expected 6 subcommands, got %d", len(subcommands))
	}
}

// ---------------------------------------------------------------------------
// find-type
// ---------------------------------------------------------------------------

func TestFindTypeCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"find-type"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no search term provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra args validation error, got: %v", err)
	}
}

func TestFindTypeCmd_TopFlagDefault(t *testing.T) {
	t.Parallel()

	cmd := newFindTypeCmd()
	topFlag := cmd.Flags().Lookup("top")
	if topFlag == nil {
		t.Fatal("expected --top flag to exist")
	}
	if topFlag.DefValue != "10" {
		t.Errorf("expected --top default to be 10, got %s", topFlag.DefValue)
	}
}

// ---------------------------------------------------------------------------
// metadata
// ---------------------------------------------------------------------------

func TestMetadataCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"metadata"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no entity set name provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra args validation error, got: %v", err)
	}
}

func TestMetadataCmd_BooleanFlags(t *testing.T) {
	t.Parallel()

	cmd := newMetadataCmd()
	for _, flagName := range []string{"enums", "keys", "constraints", "relationships"} {
		f := cmd.Flags().Lookup(flagName)
		if f == nil {
			t.Errorf("expected --%s flag to exist on metadata command", flagName)
		}
		if f != nil && f.DefValue != "false" {
			t.Errorf("expected --%s default to be false, got %s", flagName, f.DefValue)
		}
	}
}

// ---------------------------------------------------------------------------
// find
// ---------------------------------------------------------------------------

func TestFindCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"find"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no OData path provided")
	}
	if !strings.Contains(err.Error(), "Entity path") {
		t.Errorf("expected missing entity path error, got: %v", err)
	}
}

func TestFindCmd_QueryFlag(t *testing.T) {
	t.Parallel()

	cmd := newFindCmd()
	f := cmd.Flags().Lookup("query")
	if f == nil {
		t.Fatal("expected --query flag to exist on find command")
	}
	if f.DefValue != "" {
		t.Errorf("expected --query default to be empty, got %q", f.DefValue)
	}
}

// ---------------------------------------------------------------------------
// create
// ---------------------------------------------------------------------------

func TestCreateCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"create", "--data", `{"Name":"Test"}`})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no OData path provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s)") {
		t.Errorf("expected cobra args validation error, got: %v", err)
	}
}

func TestCreateCmd_RequiresDataFlag(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"create", "Customers"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --data not provided")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "required flag") && !strings.Contains(errStr, "data") {
		t.Errorf("expected required flag error for --data, got: %v", err)
	}
}

func TestCreateCmd_InvalidJSON(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"create", "Customers", "--data", "not valid json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON in --data")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// update
// ---------------------------------------------------------------------------

func TestUpdateCmd_RequiresDataFlag(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"update"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --data not provided")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "required flag") && !strings.Contains(errStr, "data") {
		t.Errorf("expected required flag error for --data, got: %v", err)
	}
}

func TestUpdateCmd_InvalidJSON(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"update", "--data", "not valid json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON in --data")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

func TestUpdateCmd_EmptyArray(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"update", "--data", "[]"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty update array")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Errorf("expected 'at least one' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// delete
// ---------------------------------------------------------------------------

func TestDeleteCmd_RequiresPathsFlag(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"delete", "--confirm"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --paths not provided")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "required flag") && !strings.Contains(errStr, "paths") {
		t.Errorf("expected required flag error for --paths, got: %v", err)
	}
}

func TestDeleteCmd_RequiresConfirm(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"delete", "--paths", `["Customers(dataAreaId='USMF',CustomerAccount='US-001')"]`})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --confirm not provided")
	}
	if !strings.Contains(err.Error(), "confirm") {
		t.Errorf("expected error about --confirm, got: %v", err)
	}
}

func TestDeleteCmd_InvalidPathsJSON(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"delete", "--paths", "not valid json", "--confirm"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON in --paths")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

func TestDeleteCmd_EmptyPaths(t *testing.T) {
	t.Parallel()

	cmd := newDataCmd()
	cmd.SetArgs([]string{"delete", "--paths", "[]", "--confirm"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty paths array")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Errorf("expected 'at least one' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// parseRawQueryOptions
// ---------------------------------------------------------------------------

func TestParseRawQueryOptions_Empty(t *testing.T) {
	t.Parallel()

	opts, err := parseRawQueryOptions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Filter != "" || opts.Select != "" || opts.Expand != "" || opts.OrderBy != "" {
		t.Errorf("expected empty options from empty string, got filter=%q select=%q expand=%q orderby=%q",
			opts.Filter, opts.Select, opts.Expand, opts.OrderBy)
	}
	if opts.Top != 0 || opts.Skip != 0 {
		t.Errorf("expected zero top/skip, got top=%d skip=%d", opts.Top, opts.Skip)
	}
	if opts.Count {
		t.Error("expected count=false for empty query")
	}
	if opts.CrossCompany != nil {
		t.Error("expected nil CrossCompany for empty query (so auto-inject triggers)")
	}
}

func TestParseRawQueryOptions_AllFields(t *testing.T) {
	t.Parallel()

	raw := "$filter=Name eq 'Contoso'&$select=CustomerAccount,Name&$expand=Orders&$orderby=Name asc&$top=10&$skip=5&$count=true&cross-company=true"
	opts, err := parseRawQueryOptions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Filter != "Name eq 'Contoso'" {
		t.Errorf("filter: expected \"Name eq 'Contoso'\", got %q", opts.Filter)
	}
	if opts.Select != "CustomerAccount,Name" {
		t.Errorf("select: expected \"CustomerAccount,Name\", got %q", opts.Select)
	}
	if opts.Expand != "Orders" {
		t.Errorf("expand: expected \"Orders\", got %q", opts.Expand)
	}
	if opts.OrderBy != "Name asc" {
		t.Errorf("orderby: expected \"Name asc\", got %q", opts.OrderBy)
	}
	if opts.Top != 10 {
		t.Errorf("top: expected 10, got %d", opts.Top)
	}
	if opts.Skip != 5 {
		t.Errorf("skip: expected 5, got %d", opts.Skip)
	}
	if !opts.Count {
		t.Error("expected count=true")
	}
	if opts.CrossCompany == nil || !*opts.CrossCompany {
		t.Error("expected cross-company=true")
	}
}

func TestParseRawQueryOptions_CrossCompanyFalse(t *testing.T) {
	t.Parallel()

	opts, err := parseRawQueryOptions("cross-company=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.CrossCompany == nil {
		t.Fatal("expected CrossCompany to be set")
	}
	if *opts.CrossCompany {
		t.Error("expected CrossCompany to be false")
	}
}

func TestParseRawQueryOptions_SelectOnly(t *testing.T) {
	t.Parallel()

	opts, err := parseRawQueryOptions("$select=CustomerAccount,Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Select != "CustomerAccount,Name" {
		t.Errorf("expected select \"CustomerAccount,Name\", got %q", opts.Select)
	}
	// cross-company should be nil (not present in raw query)
	if opts.CrossCompany != nil {
		t.Error("expected nil CrossCompany when not in query string")
	}
}
