package guardrails

import (
	"testing"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

func TestEngine_CrossCompanyRequired(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	// Should block: dataAreaId without cross-company
	results := e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "$filter=dataAreaId eq 'USMF'",
	})
	if !HasBlockers(results) {
		t.Error("expected blocker for dataAreaId without cross-company")
	}

	// Should pass: has cross-company
	results = e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "cross-company=true&$filter=dataAreaId eq 'USMF'",
	})
	blocked := false
	for _, r := range results {
		if r.Rule == "cross-company-required" {
			blocked = true
		}
	}
	if blocked {
		t.Error("should not block when cross-company=true is present")
	}
}

func TestEngine_SelectRecommended(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	results := e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "$top=10",
	})
	warnings := Warnings(results)
	found := false
	for _, w := range warnings {
		if w.Rule == "select-recommended" {
			found = true
		}
	}
	if !found {
		t.Error("expected select-recommended warning")
	}

	// With $select, no warning
	results = e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "$top=10&$select=Name,ID",
	})
	for _, r := range results {
		if r.Rule == "select-recommended" {
			t.Error("should not warn when $select is present")
		}
	}
}

func TestEngine_DeleteRequiresConfirm(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	// Without confirm: should block
	results := e.Evaluate(&CheckContext{
		Command:    "data delete",
		HasConfirm: false,
	})
	if !HasBlockers(results) {
		t.Error("expected blocker for delete without --confirm")
	}

	// With confirm: should pass
	results = e.Evaluate(&CheckContext{
		Command:    "data delete",
		HasConfirm: true,
	})
	for _, r := range results {
		if r.Rule == "delete-requires-confirm" && r.Blocked {
			t.Error("should not block when --confirm is set")
		}
	}
}

func TestEngine_WideQuery(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	// Empty query: should warn
	results := e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "",
	})
	found := false
	for _, r := range results {
		if r.Rule == "wide-query-warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected wide-query-warning for empty query")
	}

	// With $top: no warning
	results = e.Evaluate(&CheckContext{
		Command:     "data find",
		QueryString: "$top=5&$select=Name",
	})
	for _, r := range results {
		if r.Rule == "wide-query-warning" {
			t.Error("should not warn when $top is present")
		}
	}
}

func TestEngine_EnumFormat(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	// Numeric enum value: should warn
	results := e.Evaluate(&CheckContext{
		Command: "data create",
		Body:    map[string]interface{}{"OrderStatus": float64(1)},
	})
	found := false
	for _, r := range results {
		if r.Rule == "enum-format" {
			found = true
		}
	}
	if !found {
		t.Error("expected enum-format warning for numeric enum value")
	}

	// String enum value: no warning
	results = e.Evaluate(&CheckContext{
		Command: "data create",
		Body:    map[string]interface{}{"OrderStatus": "Active"},
	})
	for _, r := range results {
		if r.Rule == "enum-format" {
			t.Error("should not warn for string enum value")
		}
	}
}

func TestEngine_NonMatchingCommands(t *testing.T) {
	t.Parallel()

	e := NewEngine()

	// select-recommended only applies to data find
	results := e.Evaluate(&CheckContext{
		Command:     "data create",
		QueryString: "$top=10",
	})
	for _, r := range results {
		if r.Rule == "select-recommended" {
			t.Error("select-recommended should not trigger for data create")
		}
	}
}

func TestEngine_CustomRule(t *testing.T) {
	t.Parallel()

	e := NewEngine()
	e.AddRule(Rule{
		Name:        "custom-rule",
		Description: "Test custom rule",
		Check: func(ctx *CheckContext) *types.GuardrailResult {
			if ctx.Company == "BLOCKED" {
				return &types.GuardrailResult{
					Rule:     "custom-rule",
					Severity: SeverityError,
					Message:  "company BLOCKED is not allowed",
					Blocked:  true,
				}
			}
			return nil
		},
	})

	results := e.Evaluate(&CheckContext{Company: "BLOCKED"})
	if !HasBlockers(results) {
		t.Error("expected custom rule to block")
	}

	results = e.Evaluate(&CheckContext{Company: "USMF"})
	blocked := false
	for _, r := range results {
		if r.Rule == "custom-rule" {
			blocked = true
		}
	}
	if blocked {
		t.Error("custom rule should not trigger for USMF")
	}
}

func TestHasBlockers(t *testing.T) {
	t.Parallel()

	if HasBlockers(nil) {
		t.Error("nil should not have blockers")
	}
	if HasBlockers([]types.GuardrailResult{}) {
		t.Error("empty should not have blockers")
	}
	if !HasBlockers([]types.GuardrailResult{{Blocked: true}}) {
		t.Error("should detect blocker")
	}
}

func TestWarnings(t *testing.T) {
	t.Parallel()

	results := []types.GuardrailResult{
		{Severity: SeverityError},
		{Severity: SeverityWarn},
		{Severity: SeverityInfo},
		{Severity: SeverityWarn},
	}

	warnings := Warnings(results)
	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(warnings))
	}
}
