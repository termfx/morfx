package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestBasicDSLTranslation tests basic DSL translation functionality
func TestBasicDSLTranslation(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		expectError   bool
		description   string
	}{
		{
			name:          "simple_function_query",
			dslQuery:      "func:main",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) @target",
			expectError:   false,
			description:   "Basic function query should work",
		},
		{
			name:          "simple_nested_query",
			dslQuery:      "func:main > var:config",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")))] @target",
			expectError:   false,
			description:   "Basic nested query should work",
		},
		{
			name:          "wildcard_identifier",
			dslQuery:      "func:*",
			expectedQuery: "(function_declaration) @target",
			expectError:   false,
			description:   "Wildcard identifier should work",
		},
		{
			name:          "basic_negated_query",
			dslQuery:      "!func:main",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-eq? @name \"main\")) @target",
			expectError:   false,
			description:   "Basic negated query should work",
		},
		{
			name:          "import_unquoted",
			dslQuery:      "import:fmt",
			expectedQuery: "(import_spec path: (interpreted_string_literal) @path (#eq? @path \"fmt\")) @target",
			expectError:   false,
			description:   "Unquoted import should work",
		},
		{
			name:          "import_quoted",
			dslQuery:      "import:\"fmt\"",
			expectedQuery: "",
			expectError:   true,
			description:   "Quoted import should fail in Phase 2",
		},
		{
			name:          "if_statement_wildcard",
			dslQuery:      "if:*",
			expectedQuery: "(if_statement) @target",
			expectError:   false,
			description:   "If statement with wildcard should work",
		},
		{
			name:          "block_wildcard",
			dslQuery:      "block:*",
			expectedQuery: "(block) @target",
			expectError:   false,
			description:   "Block with wildcard should work",
		},
		{
			name:          "call_with_selector",
			dslQuery:      "call:util.SHA1FileHex",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"util.SHA1FileHex\")) @target",
			expectError:   false,
			description:   "Call with selector expression should work",
		},
		{
			name:        "error_unknown_node_type",
			dslQuery:    "unknown:test",
			expectError: true,
			description: "Unknown node type should fail",
		},
		{
			name:        "error_invalid_dsl_format",
			dslQuery:    "func",
			expectError: true,
			description: "Invalid DSL format should fail",
		},
		{
			name:        "error_if_with_non_wildcard",
			dslQuery:    "if:condition",
			expectError: true,
			description: "If with non-wildcard should fail",
		},
		{
			name:        "error_block_with_non_wildcard",
			dslQuery:    "block:condition",
			expectError: true,
			description: "Block with non-wildcard should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for query %q, but got none", tt.dslQuery)
				}
				// Optionally, check for specific error message content
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for query %q: %v", tt.dslQuery, err)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Mismatch for query %q:\nExpected: %s\nActual:   %s", tt.dslQuery, expectedQuery, actualQuery)
			}
		})
	}
}
