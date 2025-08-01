package golang

import (
	"strings"
	"testing"
)

// TestPhase2StrictValidation tests Phase 2 strict validation requirements
func TestPhase2StrictValidation(t *testing.T) {
	tests := []struct {
		name        string
		dslQuery    string
		shouldError bool
		errorText   string
	}{
		// Import validation tests
		{
			name:        "valid_unquoted_import",
			dslQuery:    "import:fmt",
			shouldError: false,
		},
		{
			name:        "invalid_quoted_import",
			dslQuery:    `import:"fmt"`,
			shouldError: true,
			errorText:   "import expects unquoted path (e.g., import:fmt)",
		},
		{
			name:        "invalid_partial_quoted_import",
			dslQuery:    `import:github.com/"package"`,
			shouldError: true,
			errorText:   "import expects unquoted path (e.g., import:fmt)",
		},
		// if/block validation tests
		{
			name:        "valid_if_wildcard",
			dslQuery:    "if:*",
			shouldError: false,
		},
		{
			name:        "invalid_if_identifier",
			dslQuery:    "if:condition",
			shouldError: true,
			errorText:   "only * supported for if/block",
		},
		{
			name:        "valid_block_wildcard",
			dslQuery:    "block:*",
			shouldError: false,
		},
		{
			name:        "invalid_block_identifier",
			dslQuery:    "block:main",
			shouldError: true,
			errorText:   "only * supported for if/block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDSL(tt.dslQuery)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for query '%s', but got none", tt.dslQuery)
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain '%s', but got: %s", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for query '%s': %v", tt.dslQuery, err)
				}
			}
		})
	}
}

// TestPhase2PredicateEmission tests that correct predicates are emitted
func TestPhase2PredicateEmission(t *testing.T) {
	provider := &goProvider{}

	tests := []struct {
		name        string
		dslQuery    string
		contains    []string
		notContains []string
	}{
		{
			name:        "import_path_predicate",
			dslQuery:    "import:fmt",
			contains:    []string{`(#eq? @path "fmt")`, "@target"},
			notContains: []string{`"\"fmt\""`}, // Should not double-quote
		},
		{
			name:     "wildcard_anchors_prefix",
			dslQuery: "func:Test*",
			contains: []string{`(#match? @name "^Test.*")`, "@target"}, // func is single-name
		},
		{
			name:     "wildcard_anchors_suffix",
			dslQuery: "func:*Handler",
			contains: []string{`(#match? @name ".*Handler$")`, "@target"}, // func is single-name
		},
		{
			name:     "wildcard_anchors_contains",
			dslQuery: "func:*Test*",
			contains: []string{`(#match? @name ".*Test.*")`, "@target"}, // func is single-name
		},
		{
			name:     "wildcard_anchors_gap",
			dslQuery: "func:Get*Data",
			contains: []string{`(#match? @name "^Get.*Data$")`, "@target"}, // func is single-name
		},
		{
			name:     "multi_name_wildcard_var",
			dslQuery: "var:test*",
			contains: []string{`(#any-match? @name "^test.*")`, "@target"}, // var is multi-name
		},
		{
			name:     "negation_predicate",
			dslQuery: "!func:Test",
			contains: []string{`(#not-eq? @name "Test")`, "@target"}, // func is single-name
		},
		{
			name:     "negation_wildcard_predicate",
			dslQuery: "!func:Test*",
			contains: []string{`(#not-match? @name "^Test.*")`, "@target"}, // func is single-name
		},
		{
			name:     "multi_name_negation_var",
			dslQuery: "!var:test",
			contains: []string{`(#any-not-eq? @name "test")`, "@target"}, // var is multi-name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for query '%s': %v", tt.dslQuery, err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Query result for '%s' should contain '%s'.\nGot: %s", tt.dslQuery, expected, result)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("Query result for '%s' should NOT contain '%s'.\nGot: %s", tt.dslQuery, notExpected, result)
				}
			}

			t.Logf("DSL Query: %s\nTree-sitter Query: %s", tt.dslQuery, result)
		})
	}
}

// TestPhase2TargetCapturePlacement tests that only root queries have @target
func TestPhase2TargetCapturePlacement(t *testing.T) {
	provider := &goProvider{}

	// Test hierarchical query
	result, err := provider.TranslateDSL("assign:fileHash > call:util.SHA1FileHex")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Count @target occurrences
	targetCount := strings.Count(result, "@target")
	if targetCount != 1 {
		t.Errorf("Expected exactly 1 @target capture, got %d in: %s", targetCount, result)
	}

	// Ensure @target is at the end (root level)
	if !strings.HasSuffix(strings.TrimSpace(result), "@target") {
		t.Errorf("@target should be at the end of root query: %s", result)
	}
}

// TestPhase2HasNegationPredicates tests negation predicate detection
func TestPhase2HasNegationPredicates(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "no_negation",
			query:    `(function_declaration name: (identifier) @name (#eq? @name "test")) @target`,
			expected: false,
		},
		{
			name:     "has_not_eq",
			query:    `(function_declaration name: (identifier) @name (#not-eq? @name "test")) @target`,
			expected: true,
		},
		{
			name:     "has_not_match",
			query:    `(function_declaration name: (identifier) @name (#not-match? @name "^test.*")) @target`,
			expected: true,
		},
		{
			name:     "has_both_negations",
			query:    `(function_declaration name: (identifier) @name (#not-eq? @name "test") (#not-match? @name "pattern")) @target`,
			expected: true,
		},
		{
			name:     "has_any_not_eq",
			query:    `(var_declaration name: (identifier_list (identifier) @name) (#any-not-eq? @name "test")) @target`,
			expected: true,
		},
	}

	provider := &goProvider{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.HasNegationPredicates(tt.query)
			if result != tt.expected {
				t.Errorf("HasNegationPredicates(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}
