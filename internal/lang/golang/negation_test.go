package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestNegationHandling tests all negation functionality
func TestNegationHandling(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "negated_simple_function",
			dslQuery:      "!func:test",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-eq? @name \"test\")) @target",
			description:   "Negated function should use #not-eq?",
		},
		{
			name:          "negated_wildcard_function",
			dslQuery:      "!func:Test*",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) @target",
			description:   "Negated wildcard function should use #not-match?",
		},
		{
			name:          "negated_call_expression",
			dslQuery:      "!call:fmt.Println",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#not-eq? @name \"fmt.Println\")) @target",
			description:   "Negated call should use #not-eq?",
		},
		{
			name:          "negated_variable",
			dslQuery:      "!var:config",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-not-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-not-eq? @name \"config\")))] @target",
			description:   "Negated variable should use #any-not-eq?",
		},
		{
			name:          "negated_constant",
			dslQuery:      "!const:MaxSize",
			expectedQuery: "(const_declaration (const_spec name: (identifier_list (identifier) @name)) (#any-not-eq? @name \"MaxSize\")) @target",
			description:   "Negated constant should use #any-not-eq?",
		},
		{
			name:          "negated_struct",
			dslQuery:      "!struct:User",
			expectedQuery: "(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) (#not-eq? @name \"User\")) @target",
			description:   "Negated struct should use #not-eq?",
		},
		{
			name:          "negated_field",
			dslQuery:      "!field:Name",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-not-eq? @name \"Name\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-not-eq? @name \"Name\"))] @target",
			description:   "Negated field should use #any-not-eq?",
		},
		{
			name:          "negated_assignment",
			dslQuery:      "!assign:result",
			expectedQuery: "[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-not-eq? @name \"result\") @target",
			description:   "Negated assignment should use #any-not-eq?",
		},
		{
			name:          "negated_import",
			dslQuery:      "!import:fmt",
			expectedQuery: "(import_spec path: (interpreted_string_literal) @path (#not-eq? @path \"fmt\")) @target",
			description:   "Negated import should use #not-eq?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for negation test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Negation test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestNegationWithWildcards tests negation combined with wildcard patterns
func TestNegationWithWildcards(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "negated_prefix_wildcard",
			dslQuery:      "!func:Test*",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) @target",
			description:   "Negated prefix wildcard should use #not-match?",
		},
		{
			name:          "negated_suffix_wildcard",
			dslQuery:      "!func:*Handler",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \".*Handler$\")) @target",
			description:   "Negated suffix wildcard should use #not-match?",
		},
		{
			name:          "negated_contains_wildcard",
			dslQuery:      "!func:*Test*",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \".*Test.*\")) @target",
			description:   "Negated contains wildcard should use #not-match?",
		},
		{
			name:          "negated_complex_wildcard",
			dslQuery:      "!func:Handle*Request",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Handle.*Request$\")) @target",
			description:   "Negated complex wildcard should use #not-match?",
		},
		{
			name:          "negated_call_wildcard",
			dslQuery:      "!call:fmt.*",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#not-match? @name \"^fmt\\..*\")) @target",
			description:   "Negated call wildcard should use #not-match? with escaped dots",
		},
		{
			name:          "negated_var_wildcard",
			dslQuery:      "!var:config*",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-not-match? @name \"^config.*\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-not-match? @name \"^config.*\")))] @target",
			description:   "Negated variable wildcard should use #any-not-match?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for negated wildcard test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Negated wildcard test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestNegationInComplexQueries tests negation in complex hierarchical queries
func TestNegationInComplexQueries(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "negated_parent_with_child",
			dslQuery:      "!func:Test* > call:t.Error",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"t.Error\")) @target",
			description:   "Negated parent with child should work correctly",
		},
		{
			name:          "parent_with_negated_child",
			dslQuery:      "func:TestHandler > !call:fmt.Println",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"TestHandler\")) . (call_expression function: [(identifier) (selector_expression)] @name (#not-eq? @name \"fmt.Println\")) @target",
			description:   "Parent with negated child should work correctly",
		},
		{
			name:          "negated_parent_negated_child",
			dslQuery:      "!func:Test* > !call:fmt.*",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) . (call_expression function: [(identifier) (selector_expression)] @name (#not-match? @name \"^fmt\\..*\")) @target",
			description:   "Both negated parent and child should work correctly",
		},
		{
			name:          "negated_complex_hierarchy",
			dslQuery:      "!func:Test* > if:* > !call:log.*",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) . (if_statement) . (call_expression function: [(identifier) (selector_expression)] @name (#not-match? @name \"^log\\..*\")) @target",
			description:   "Complex negated hierarchy should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for complex negation test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Complex negation test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}
