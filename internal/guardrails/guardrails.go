// Package guardrails implements centralized validation rules that protect
// against common mistakes when AI agents interact with D365. Rules run
// before command execution and can block, warn, or pass.
package guardrails

import (
	"fmt"
	"strings"

	"github.com/seangalliher/d365-erp-cli/pkg/types"
)

// Severity levels for guardrail results.
const (
	SeverityError = "error"
	SeverityWarn  = "warn"
	SeverityInfo  = "info"
)

// Rule is a single guardrail check function.
type Rule struct {
	Name        string
	Description string
	Check       func(ctx *CheckContext) *types.GuardrailResult
}

// CheckContext provides context for guardrail evaluation.
type CheckContext struct {
	Command     string
	EntityPath  string
	QueryString string
	Company     string
	HasConfirm  bool
	Body        map[string]interface{}
}

// Engine runs guardrail checks against a set of registered rules.
type Engine struct {
	rules []Rule
}

// NewEngine creates a guardrail engine with the default rule set.
func NewEngine() *Engine {
	e := &Engine{}
	e.registerDefaults()
	return e
}

// Evaluate runs all rules and returns results. Returns only non-passing results.
func (e *Engine) Evaluate(ctx *CheckContext) []types.GuardrailResult {
	var results []types.GuardrailResult
	for _, rule := range e.rules {
		result := rule.Check(ctx)
		if result != nil {
			results = append(results, *result)
		}
	}
	return results
}

// HasBlockers returns true if any result is a blocking error.
func HasBlockers(results []types.GuardrailResult) bool {
	for _, r := range results {
		if r.Blocked {
			return true
		}
	}
	return false
}

// Warnings returns only warning-level results.
func Warnings(results []types.GuardrailResult) []types.GuardrailResult {
	var warnings []types.GuardrailResult
	for _, r := range results {
		if r.Severity == SeverityWarn {
			warnings = append(warnings, r)
		}
	}
	return warnings
}

func (e *Engine) registerDefaults() {
	e.rules = []Rule{
		{
			Name:        "cross-company-required",
			Description: "Ensure cross-company=true is set when filtering by dataAreaId",
			Check:       checkCrossCompany,
		},
		{
			Name:        "select-recommended",
			Description: "Warn when $select is not used on queries",
			Check:       checkSelectMissing,
		},
		{
			Name:        "delete-requires-confirm",
			Description: "Block deletes without --confirm flag",
			Check:       checkDeleteConfirm,
		},
		{
			Name:        "wide-query-warning",
			Description: "Warn when querying without $top or $filter",
			Check:       checkWideQuery,
		},
		{
			Name:        "enum-format",
			Description: "Validate enum values use symbol names, not integers",
			Check:       checkEnumFormat,
		},
	}
}

// AddRule adds a custom guardrail rule.
func (e *Engine) AddRule(r Rule) {
	e.rules = append(e.rules, r)
}

// --- Default rule implementations ---

func checkCrossCompany(ctx *CheckContext) *types.GuardrailResult {
	q := strings.ToLower(ctx.QueryString)
	if strings.Contains(q, "dataareaid") && !strings.Contains(q, "cross-company=true") {
		return &types.GuardrailResult{
			Rule:       "cross-company-required",
			Severity:   SeverityError,
			Message:    "Filtering by dataAreaId requires cross-company=true in the query",
			Suggestion: "Add cross-company=true to the query string",
			Blocked:    true,
		}
	}
	return nil
}

func checkSelectMissing(ctx *CheckContext) *types.GuardrailResult {
	if ctx.Command != "data find" && ctx.Command != "data.find" {
		return nil
	}
	q := strings.ToLower(ctx.QueryString)
	if ctx.QueryString != "" && !strings.Contains(q, "$select") {
		return &types.GuardrailResult{
			Rule:       "select-recommended",
			Severity:   SeverityWarn,
			Message:    "Query does not include $select — all fields will be returned",
			Suggestion: "Add $select=Field1,Field2 to return only needed fields for better performance",
			Blocked:    false,
		}
	}
	return nil
}

func checkDeleteConfirm(ctx *CheckContext) *types.GuardrailResult {
	if ctx.Command != "data delete" && ctx.Command != "data.delete" {
		return nil
	}
	if !ctx.HasConfirm {
		return &types.GuardrailResult{
			Rule:       "delete-requires-confirm",
			Severity:   SeverityError,
			Message:    "Delete operations require explicit confirmation",
			Suggestion: "Add --confirm flag to proceed with deletion",
			Blocked:    true,
		}
	}
	return nil
}

func checkWideQuery(ctx *CheckContext) *types.GuardrailResult {
	if ctx.Command != "data find" && ctx.Command != "data.find" {
		return nil
	}
	q := strings.ToLower(ctx.QueryString)
	if ctx.QueryString == "" || (!strings.Contains(q, "$top") && !strings.Contains(q, "$filter")) {
		return &types.GuardrailResult{
			Rule:       "wide-query-warning",
			Severity:   SeverityWarn,
			Message:    "Query has no $top or $filter — may return a large result set",
			Suggestion: "Add $top=N to limit results or $filter to narrow the query",
			Blocked:    false,
		}
	}
	return nil
}

func checkEnumFormat(ctx *CheckContext) *types.GuardrailResult {
	if ctx.Body == nil {
		return nil
	}
	// Check for common enum anti-pattern: passing integer values for enum fields
	// that end in "Status", "Type", "Category", "Mode"
	enumSuffixes := []string{"Status", "Type", "Category", "Mode", "Direction", "State"}
	for key, val := range ctx.Body {
		for _, suffix := range enumSuffixes {
			if strings.HasSuffix(key, suffix) {
				switch val.(type) {
				case float64, int, int64:
					return &types.GuardrailResult{
						Rule:     "enum-format",
						Severity: SeverityWarn,
						Message:  fmt.Sprintf("Field %q appears to be an enum but has a numeric value — use the symbol name instead", key),
						Suggestion: fmt.Sprintf("Pass the enum symbol name (e.g., 'Active') instead of a number for field %q. "+
							"Use 'd365 data metadata <entity> --enums' to see valid enum values.", key),
						Blocked: false,
					}
				}
			}
		}
	}
	return nil
}
