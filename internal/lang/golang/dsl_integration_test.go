package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestEndToEndIntegration tests end-to-end DSL to Tree-sitter query generation
func TestEndToEndIntegration(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "simple_function_integration",
			dslQuery:      "func:main",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) @target",
			description:   "Simple function query should work end-to-end",
		},
		{
			name:          "complex_nested_integration",
			dslQuery:      "func:TestHandler > if:* > call:t.Error",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"TestHandler\")) . (if_statement) . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"t.Error\")) @target",
			description:   "Complex nested queries should work end-to-end",
		},
		{
			name:          "negated_complex_integration",
			dslQuery:      "!func:Test* > call:fmt.Println",
			expectedQuery: "(function_declaration name: (identifier) @name (#not-match? @name \"^Test.*\")) . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"fmt.Println\")) @target",
			description:   "Negated complex queries should work end-to-end",
		},
		{
			name:          "multi_level_hierarchy_integration",
			dslQuery:      "func:main > var:config Config > field:Host string",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")))] . [(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Host\") (#eq? @type \"string\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Host\") (#eq? @type \"string\"))] @target",
			description:   "Multi-level hierarchy with types should work end-to-end",
		},
		{
			name:          "wildcard_hierarchy_integration",
			dslQuery:      "func:* > var:* > call:*",
			expectedQuery: "(function_declaration) . [(var_declaration (var_spec type: (type_identifier) @type) ) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) ))] . (call_expression) @target",
			description:   "Wildcard hierarchy should work end-to-end",
		},
		{
			name:          "mixed_patterns_integration",
			dslQuery:      "func:Handle* > assign:result > call:util.*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Handle.*\")) . [(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-eq? @name \"result\") . (call_expression function: [(identifier) (selector_expression)] @name (#match? @name \"^util\\..*\")) @target",
			description:   "Mixed patterns should work end-to-end",
		},
		{
			name:          "import_handling_integration",
			dslQuery:      "import:github.com/pkg/errors",
			expectedQuery: "(import_spec path: (interpreted_string_literal) @path (#eq? @path \"github.com/pkg/errors\")) @target",
			description:   "Import handling should work end-to-end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for integration test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Integration test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestRealWorldScenarios tests real-world usage scenarios
func TestRealWorldScenarios(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "test_function_pattern",
			dslQuery:      "func:Test*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Test.*\")) @target",
			description:   "Test function pattern should work",
		},
		{
			name:          "benchmark_function_pattern",
			dslQuery:      "func:Benchmark*",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \"^Benchmark.*\")) @target",
			description:   "Benchmark function pattern should work",
		},
		{
			name:          "http_handler_pattern",
			dslQuery:      "func:*Handler",
			expectedQuery: "(function_declaration name: (identifier) @name (#match? @name \".*Handler$\")) @target",
			description:   "HTTP handler pattern should work",
		},
		{
			name:          "error_handling_pattern",
			dslQuery:      "if:* > call:log.Error",
			expectedQuery: "(if_statement) . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"log.Error\")) @target",
			description:   "Error handling pattern should work",
		},
		{
			name:          "config_struct_pattern",
			dslQuery:      "struct:*Config > field:* string",
			expectedQuery: "(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) (#match? @name \".*Config$\")) . (field_declaration type: (type_identifier) @type (#eq? @type \"string\")) @target",
			description:   "Config struct pattern should work",
		},
		{
			name:          "database_connection_pattern",
			dslQuery:      "var:db *sql.DB",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"db\") (#match? @type \".*sql\\.DB$\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"db\") (#match? @type \".*sql\\.DB$\")))] @target",
			description:   "Database connection pattern should work",
		},
		{
			name:          "context_usage_pattern",
			dslQuery:      "func:* > var:ctx context.Context",
			expectedQuery: "(function_declaration) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"ctx\") (#eq? @type \"context.Context\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"ctx\") (#eq? @type \"context.Context\")))] @target",
			description:   "Context usage pattern should work",
		},
		{
			name:          "json_marshaling_pattern",
			dslQuery:      "call:json.Marshal",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"json.Marshal\")) @target",
			description:   "JSON marshaling pattern should work",
		},
		{
			name:          "goroutine_pattern",
			dslQuery:      "func:* > call:go *",
			expectedQuery: "(function_declaration) . (call_expression function: [(identifier) (selector_expression)] @name (#match? @name \"^go .*\")) @target",
			description:   "Goroutine pattern should work",
		},
		{
			name:          "interface_implementation_pattern",
			dslQuery:      "struct:* > field:* interface{}",
			expectedQuery: "(type_declaration (type_spec type: (struct_type))) . (field_declaration type: (type_identifier) @type (#eq? @type \"interface{}\")) @target",
			description:   "Interface implementation pattern should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for real-world scenario %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Real-world scenario test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestQueryBuilding tests query building improvements
func TestQueryBuilding(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name        string
		dslQuery    string
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name:     "hierarchy_parent_child",
			dslQuery: "func:main > if:* > call:log.Error",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Should contain function_declaration")
				}
				if !strings.Contains(result, "if_statement") {
					t.Error("Should contain if_statement")
				}
				if !strings.Contains(result, "call_expression") {
					t.Error("Should contain call_expression")
				}
				if strings.Count(result, "@target") != 1 {
					t.Error("Should contain exactly one @target")
				}
			},
		},
		{
			name:     "target_annotation",
			dslQuery: "func:handler > var:config",
			validate: func(t *testing.T, result string) {
				if strings.Count(result, "@target") != 1 {
					t.Error("Should contain exactly one @target")
				}
				// @target should be at the end (on the child)
				if !strings.HasSuffix(strings.TrimSpace(result), "@target") {
					t.Error("@target should be at the end of the query")
				}
			},
		},
		{
			name:        "import_handling_quoted",
			dslQuery:    "import:\"fmt\"",
			expectError: true,
			validate: func(t *testing.T, result string) {
				// This should error - we don't get to validate the result
			},
		},
		{
			name:     "import_handling_path",
			dslQuery: "import:github.com/pkg/errors",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "import_spec") {
					t.Error("Should contain import_spec")
				}
				if !strings.Contains(result, `(#eq? @path "github.com/pkg/errors")`) {
					t.Error("Should contain correct path predicate")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for query building test %q, but got none", tt.dslQuery)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for query building test %q: %v", tt.dslQuery, err)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
