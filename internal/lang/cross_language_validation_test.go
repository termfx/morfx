package lang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/lang/golang"
	"github.com/termfx/morfx/internal/lang/javascript"
	"github.com/termfx/morfx/internal/lang/python"
	"github.com/termfx/morfx/internal/lang/typescript"
	"github.com/termfx/morfx/internal/provider"
)

// TestCrossLanguageQueryValidation validates that all providers can handle the same universal queries
func TestCrossLanguageQueryValidation(t *testing.T) {
	providers := map[string]provider.LanguageProvider{
		"go":         golang.NewProvider(),
		"javascript": javascript.NewProvider(),
		"python":     python.NewProvider(),
		"typescript": typescript.NewProvider(),
	}

	testCases := []struct {
		name         string
		query        *core.Query
		expectSuccess []string // Which providers should succeed
	}{
		{
			name: "simple function query",
			query: &core.Query{
				Kind:       core.KindFunction,
				Pattern:    "test",
				Attributes: make(map[string]string),
				Raw:        "function:test",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "function with wildcard",
			query: &core.Query{
				Kind:       core.KindFunction,
				Pattern:    "test*",
				Attributes: make(map[string]string),
				Raw:        "function:test*",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "class query",
			query: &core.Query{
				Kind:       core.KindClass,
				Pattern:    "User",
				Attributes: make(map[string]string),
				Raw:        "class:User",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "variable query",
			query: &core.Query{
				Kind:       core.KindVariable,
				Pattern:    "count",
				Attributes: make(map[string]string),
				Raw:        "variable:count",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "method query",
			query: &core.Query{
				Kind:       core.KindMethod,
				Pattern:    "getName",
				Attributes: make(map[string]string),
				Raw:        "method:getName",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "AND logical query",
			query: &core.Query{
				Kind:     "logical",
				Operator: "AND",
				Children: []core.Query{
					{Kind: core.KindFunction, Pattern: "test", Attributes: make(map[string]string), Raw: "function:test"},
					{Kind: core.KindVariable, Pattern: "x", Attributes: make(map[string]string), Raw: "variable:x"},
				},
				Attributes: make(map[string]string),
				Raw:        "function:test & variable:x",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "OR logical query",
			query: &core.Query{
				Kind:     "logical",
				Operator: "OR",
				Children: []core.Query{
					{Kind: core.KindFunction, Pattern: "init", Attributes: make(map[string]string), Raw: "function:init"},
					{Kind: core.KindMethod, Pattern: "setup", Attributes: make(map[string]string), Raw: "method:setup"},
				},
				Attributes: make(map[string]string),
				Raw:        "function:init | method:setup",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "NOT query",
			query: &core.Query{
				Kind:       core.KindFunction,
				Pattern:    "test",
				Operator:   "NOT",
				Attributes: make(map[string]string),
				Raw:        "function:test",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
		{
			name: "hierarchical query",
			query: &core.Query{
				Kind:     core.KindMethod,
				Pattern:  "getName",
				Operator: "HIERARCHY",
				Children: []core.Query{
					{Kind: core.KindClass, Pattern: "User", Attributes: make(map[string]string), Raw: "class:User"},
				},
				Attributes: make(map[string]string),
				Raw:        "class:User > method:getName",
			},
			expectSuccess: []string{"go", "javascript", "python", "typescript"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := make(map[string]string)
			errors := make(map[string]error)

			// Test each provider
			for langName, prov := range providers {
				result, err := prov.TranslateQuery(tc.query)
				if err != nil {
					errors[langName] = err
				} else {
					results[langName] = result
				}
			}

			// Validate expected successes
			for _, expectedLang := range tc.expectSuccess {
				if err, hasError := errors[expectedLang]; hasError {
					t.Errorf("Expected %s to succeed but got error: %v", expectedLang, err)
					continue
				}

				if result, hasResult := results[expectedLang]; !hasResult || result == "" {
					t.Errorf("Expected %s to return non-empty result", expectedLang)
					continue
				}

				t.Logf("✓ %s provider successfully translated query: %s", expectedLang, tc.query.Raw)
			}

			// Log all results for debugging
			t.Logf("Query: %s", tc.query.Raw)
			for lang, result := range results {
				t.Logf("  %s: %s", lang, abbreviateQuery(result))
			}
			for lang, err := range errors {
				t.Logf("  %s: ERROR - %v", lang, err)
			}
		})
	}
}

// TestUniversalNodeKindConsistency validates that all providers map universal kinds consistently
func TestUniversalNodeKindConsistency(t *testing.T) {
	providers := map[string]provider.LanguageProvider{
		"go":         golang.NewProvider(),
		"javascript": javascript.NewProvider(),
		"python":     python.NewProvider(),
		"typescript": typescript.NewProvider(),
	}

	universalKinds := []core.NodeKind{
		core.KindFunction,
		core.KindClass,
		core.KindMethod,
		core.KindVariable,
		core.KindImport,
		core.KindConstant,
		core.KindField,
		core.KindCall,
		core.KindAssignment,
		core.KindCondition,
		core.KindLoop,
		core.KindBlock,
		core.KindComment,
	}

	for _, kind := range universalKinds {
		t.Run(string(kind), func(t *testing.T) {
			mappingCounts := make(map[string]int)

			for langName, prov := range providers {
				mappings := prov.TranslateKind(kind)
				mappingCounts[langName] = len(mappings)

				if len(mappings) == 0 {
					t.Logf("⚠️  %s has no mappings for %s", langName, kind)
				} else {
					t.Logf("✓ %s has %d mapping(s) for %s", langName, len(mappings), kind)
					for _, mapping := range mappings {
						t.Logf("    Node types: %v", mapping.NodeTypes)
					}
				}
			}

			// Check if all providers have at least some mapping for core kinds
			coreKinds := []core.NodeKind{core.KindFunction, core.KindClass, core.KindVariable}
			for _, coreKind := range coreKinds {
				if kind == coreKind {
					allHaveMappings := true
					for langName, count := range mappingCounts {
						if count == 0 {
							allHaveMappings = false
							t.Errorf("Critical: %s provider has no mappings for core kind %s", langName, kind)
						}
					}
					if allHaveMappings {
						t.Logf("✓ All providers support core kind: %s", kind)
					}
				}
			}
		})
	}
}

// TestLogicalOperatorConsistency ensures all providers handle logical operations consistently
func TestLogicalOperatorConsistency(t *testing.T) {
	providers := map[string]provider.LanguageProvider{
		"go":         golang.NewProvider(),
		"javascript": javascript.NewProvider(),
		"python":     python.NewProvider(),
		"typescript": typescript.NewProvider(),
	}

	logicalQueries := []struct {
		name     string
		operator string
		children []core.Query
	}{
		{
			name:     "simple AND",
			operator: "AND",
			children: []core.Query{
				{Kind: core.KindFunction, Pattern: "test", Attributes: make(map[string]string)},
				{Kind: core.KindVariable, Pattern: "x", Attributes: make(map[string]string)},
			},
		},
		{
			name:     "simple OR",
			operator: "OR",
			children: []core.Query{
				{Kind: core.KindFunction, Pattern: "init", Attributes: make(map[string]string)},
				{Kind: core.KindFunction, Pattern: "setup", Attributes: make(map[string]string)},
			},
		},
	}

	for _, lq := range logicalQueries {
		t.Run(lq.name, func(t *testing.T) {
			query := &core.Query{
				Kind:       "logical",
				Operator:   lq.operator,
				Children:   lq.children,
				Attributes: make(map[string]string),
			}

			successCount := 0
			for langName, prov := range providers {
				result, err := prov.TranslateQuery(query)
				if err != nil {
					t.Errorf("%s failed to handle %s query: %v", langName, lq.operator, err)
				} else if result == "" {
					t.Errorf("%s returned empty result for %s query", langName, lq.operator)
				} else {
					successCount++
					t.Logf("✓ %s handles %s: %s", langName, lq.operator, abbreviateQuery(result))
				}
			}

			if successCount == len(providers) {
				t.Logf("✓ All providers successfully handle %s operations", lq.operator)
			} else {
				t.Errorf("Only %d/%d providers handle %s operations", successCount, len(providers), lq.operator)
			}
		})
	}
}

// abbreviateQuery truncates long query strings for readable logs
func abbreviateQuery(query string) string {
	if len(query) <= 100 {
		return query
	}
	
	// Count lines
	lines := strings.Split(query, "\n")
	if len(lines) == 1 {
		return query[:100] + "..."
	}
	
	// Show first line and count
	return lines[0] + fmt.Sprintf(" ... (+%d more lines)", len(lines)-1)
}