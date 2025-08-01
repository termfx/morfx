package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestWildcardPatterns tests all wildcard pattern functionality
func TestWildcardPatterns(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "prefix_wildcard_Foo_star",
			dslQuery:      "func:Foo*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Foo.*\")) @target",
			description:   "Foo* pattern should match functions starting with 'Foo'",
		},
		{
			name:          "suffix_wildcard_star_Foo",
			dslQuery:      "func:*Foo",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \".*Foo$\")) @target",
			description:   "*Foo pattern should match functions ending with 'Foo'",
		},
		{
			name:          "contains_wildcard_star_Foo_star",
			dslQuery:      "func:*Foo*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \".*Foo.*\")) @target",
			description:   "*Foo* pattern should match functions containing 'Foo'",
		},
		{
			name:          "complex_wildcard_Foo_star_Bar",
			dslQuery:      "func:Foo*Bar",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Foo.*Bar$\")) @target",
			description:   "Foo*Bar pattern should match functions starting with 'Foo' and ending with 'Bar'",
		},
		{
			name:          "wildcard_Handle_star",
			dslQuery:      "func:Handle*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Handle.*\")) @target",
			description:   "Handle* pattern should match functions starting with 'Handle'",
		},
		{
			name:          "wildcard_star_Handler",
			dslQuery:      "func:*Handler",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \".*Handler$\")) @target",
			description:   "*Handler pattern should match functions ending with 'Handler'",
		},
		{
			name:          "wildcard_star_Test_star",
			dslQuery:      "func:*Test*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \".*Test.*\")) @target",
			description:   "*Test* pattern should match functions containing 'Test'",
		},
		{
			name:          "wildcard_Handle_star_Request",
			dslQuery:      "func:Handle*Request",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Handle.*Request$\")) @target",
			description:   "Handle*Request pattern should match functions starting with 'Handle' and ending with 'Request'",
		},
		{
			name:          "universal_wildcard",
			dslQuery:      "func:*",
			expectedQuery: "(function_declaration) @target",
			description:   "Universal wildcard should match any function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for wildcard pattern %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Wildcard pattern test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestComplexWildcardPatterns tests advanced wildcard pattern scenarios
func TestComplexWildcardPatterns(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name        string
		dslQuery    string
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name:     "complex_wildcard_with_underscores",
			dslQuery: "func:Test_*_Handler",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#match?") {
					t.Error("Should use #match? for wildcard")
				}
				if !strings.Contains(result, "^Test_.*_Handler$") {
					t.Error("Should contain proper regex pattern with escaped underscores")
				}
			},
		},
		{
			name:     "wildcard_with_numbers",
			dslQuery: "func:Handler*V2",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "^Handler.*V2$") {
					t.Error("Should handle numbers in wildcard patterns")
				}
			},
		},
		{
			name:     "multiple_wildcards_in_hierarchy",
			dslQuery: "func:Test* > var:*Config > field:*",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#match?") {
					t.Error("Should use #match? for wildcard function")
				}
				if !strings.Contains(result, "^Test.*") {
					t.Error("Should contain Test* pattern")
				}
				if !strings.Contains(result, ".*Config$") {
					t.Error("Should contain *Config pattern")
				}
			},
		},
		{
			name:     "wildcard_with_special_regex_chars",
			dslQuery: "func:Handle*[0-9]+",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#match?") {
					t.Error("Should use #match? for wildcard")
				}
				if !strings.Contains(result, "^Handle.*\\[0-9\\]\\+$") {
					t.Error("Should properly escape regex special characters")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for query %q, but got none", tt.dslQuery)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for query %q: %v", tt.dslQuery, err)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestWildcardPatternsInDifferentNodeTypes tests wildcard patterns across different node types
func TestWildcardPatternsInDifferentNodeTypes(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "var_wildcard_pattern",
			dslQuery:      "var:config*",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-match? @name \"^config.*\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-match? @name \"^config.*\")))] @target",
			description:   "Variable wildcard pattern should work",
		},
		{
			name:          "const_wildcard_pattern",
			dslQuery:      "const:MAX_*",
			expectedQuery: "(const_declaration (const_spec name: (identifier_list (identifier) @name)) (#any-match? @name \"^MAX_.*\")) @target",
			description:   "Constant wildcard pattern should work",
		},
		{
			name:          "struct_wildcard_pattern",
			dslQuery:      "struct:*Config",
			expectedQuery: "(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) (#match? @name \".*Config$\")) @target",
			description:   "Struct wildcard pattern should work",
		},
		{
			name:          "field_wildcard_pattern",
			dslQuery:      "field:*Name",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-match? @name \".*Name$\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-match? @name \".*Name$\"))] @target",
			description:   "Field wildcard pattern should work",
		},
		{
			name:          "call_wildcard_pattern",
			dslQuery:      "call:fmt.*",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#match? @name \"^fmt\\..*\")) @target",
			description:   "Call wildcard pattern should work with escaped dots",
		},
		{
			name:          "assign_wildcard_pattern",
			dslQuery:      "assign:result*",
			expectedQuery: "[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-match? @name \"^result.*\") @target",
			description:   "Assignment wildcard pattern should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for wildcard pattern %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Wildcard pattern test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}
