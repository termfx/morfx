package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestInputValidation tests input validation and error handling
func TestInputValidation(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name        string
		dslQuery    string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name:        "empty_query",
			dslQuery:    "",
			expectError: true,
			errorMsg:    "query cannot be empty",
			description: "Empty query should fail",
		},
		{
			name:        "invalid_format_no_colon",
			dslQuery:    "func",
			expectError: true,
			errorMsg:    "invalid DSL query",
			description: "Query without colon should fail",
		},
		{
			name:        "unknown_node_type",
			dslQuery:    "unknown:test",
			expectError: true,
			errorMsg:    "unknown node type",
			description: "Unknown node type should fail",
		},
		{
			name:        "if_with_non_wildcard",
			dslQuery:    "if:condition",
			expectError: true,
			errorMsg:    "only * supported for if/block",
			description: "If with non-wildcard identifier should fail",
		},
		{
			name:        "block_with_non_wildcard",
			dslQuery:    "block:body",
			expectError: true,
			errorMsg:    "only * supported for if/block",
			description: "Block with non-wildcard identifier should fail",
		},
		{
			name:        "if_with_specific_name",
			dslQuery:    "if:Nombre",
			expectError: true,
			errorMsg:    "only * supported for if/block",
			description: "If with specific name should fail (required negative test)",
		},
		{
			name:        "block_with_specific_name",
			dslQuery:    "block:Nombre",
			expectError: true,
			errorMsg:    "only * supported for if/block",
			description: "Block with specific name should fail (required negative test)",
		},
		{
			name:        "whitespace_only_query",
			dslQuery:    "   ",
			expectError: true,
			errorMsg:    "invalid DSL query",
			description: "Whitespace-only query should fail",
		},
		{
			name:        "invalid_child_query",
			dslQuery:    "func:main > invalid",
			expectError: true,
			errorMsg:    "invalid DSL query",
			description: "Invalid child query should fail",
		},
		{
			name:        "nested_separators",
			dslQuery:    "func:main > > var:config",
			expectError: true,
			errorMsg:    "invalid DSL query",
			description: "Empty parts between separators should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for validation test %q, but got none. %s", tt.dslQuery, tt.description)
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain %q, got %q. %s", tt.errorMsg, err.Error(), tt.description)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for validation test %q: %v. %s", tt.dslQuery, err, tt.description)
				}
			}
		})
	}
}

// TestControlStructureValidation tests specific control structure validation
func TestControlStructureValidation(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		expectError   bool
		description   string
	}{
		{
			name:          "if_wildcard_success",
			dslQuery:      "if:*",
			expectedQuery: "(if_statement) @target",
			expectError:   false,
			description:   "If with wildcard should succeed",
		},
		{
			name:          "block_wildcard_success",
			dslQuery:      "block:*",
			expectedQuery: "(block) @target",
			expectError:   false,
			description:   "Block with wildcard should succeed",
		},
		{
			name:        "if_non_wildcard_error",
			dslQuery:    "if:condition",
			expectError: true,
			description: "If with non-wildcard identifier should fail",
		},
		{
			name:        "block_non_wildcard_error",
			dslQuery:    "block:body",
			expectError: true,
			description: "Block with non-wildcard identifier should fail",
		},
		{
			name:        "if_empty_identifier_error",
			dslQuery:    "if:",
			expectError: true,
			description: "If with empty identifier should fail",
		},
		{
			name:        "block_empty_identifier_error",
			dslQuery:    "block:",
			expectError: true,
			description: "Block with empty identifier should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for control structure validation test %q, but got none. %s", tt.dslQuery, tt.description)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for control structure validation test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Control structure validation test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestWhitespaceHandling tests whitespace handling in queries
func TestWhitespaceHandling(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "leading_trailing_whitespace",
			dslQuery:      "  func:main  ",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) @target",
			description:   "Leading and trailing whitespace should be handled",
		},
		{
			name:          "whitespace_around_colon",
			dslQuery:      "func : main",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) @target",
			description:   "Whitespace around colon should be handled",
		},
		{
			name:          "whitespace_around_separator",
			dslQuery:      "func:main  >  var:config",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")))] @target",
			description:   "Whitespace around separator should be handled",
		},
		{
			name:          "complex_whitespace",
			dslQuery:      "  func : main  >  var : config  ",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")))] @target",
			description:   "Complex whitespace should be handled",
		},
		{
			name:          "empty_identifier_after_colon",
			dslQuery:      "func:",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"\")) @target",
			description:   "Empty identifier after colon should be treated as empty string",
		},
		{
			name:          "whitespace_in_multi_target",
			dslQuery:      "var: x , y , z ",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"x\") (#eq? @type \", y , z\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"x\") (#eq? @type \", y , z\")))] @target",
			description:   "Whitespace in multi-target should be handled (current behavior)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for whitespace handling test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Whitespace handling test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestRegexEscaping tests regex escaping functionality
func TestRegexEscaping(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "special_characters_in_exact_match",
			dslQuery:      "func:test.+",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"test.+\")) @target",
			description:   "Special regex characters should be escaped in exact matches",
		},
		{
			name:          "brackets_in_exact_match",
			dslQuery:      "func:test[0-9]",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"test[0-9]\")) @target",
			description:   "Brackets should be escaped in exact matches",
		},
		{
			name:          "parentheses_in_exact_match",
			dslQuery:      "func:test()",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"test()\")) @target",
			description:   "Parentheses should be escaped in exact matches",
		},
		{
			name:          "wildcard_with_special_chars",
			dslQuery:      "func:Handle*[0-9]+",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Handle.*\\[0-9\\]\\+$\")) @target",
			description:   "Special regex characters should be escaped in wildcard patterns",
		},
		{
			name:          "wildcard_with_dots",
			dslQuery:      "call:fmt.*",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#match? @name \"^fmt\\..*\")) @target",
			description:   "Dots should be escaped in wildcard patterns",
		},
		{
			name:          "wildcard_with_complex_regex_chars",
			dslQuery:      "func:Test*{1,3}",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Test.*\\{1,3\\}$\")) @target",
			description:   "Complex regex characters should be escaped in wildcard patterns (current behavior)",
		},
		{
			name:          "wildcard_with_question_mark",
			dslQuery:      "func:Handle*?",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Handle.*\\?$\")) @target",
			description:   "Question marks should be escaped in wildcard patterns",
		},
		{
			name:          "wildcard_with_pipe",
			dslQuery:      "func:Test*|Debug*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Test\\*\\|Debug.*\")) @target",
			description:   "Pipe characters should be escaped in wildcard patterns (current behavior)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for regex escaping test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Regex escaping test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}
