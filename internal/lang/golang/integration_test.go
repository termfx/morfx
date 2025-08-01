package golang

import (
	"strings"
	"testing"
)

// TestGoProvider_TranslateDSL tests comprehensive DSL translation scenarios
func TestGoProvider_TranslateDSL(t *testing.T) {
	p := &goProvider{}

	tests := []struct {
		name        string
		dslQuery    string
		expectError bool
		errorMsg    string
		validate    func(*testing.T, string)
	}{
		{
			name:     "simple_function_query",
			dslQuery: "func:main",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Should contain function_declaration")
				}
				if !strings.Contains(result, `(#eq? @name "main")`) {
					t.Error("Should contain name predicate")
				}
			},
		},
		{
			name:     "call_with_selector_expression",
			dslQuery: "call:fmt.Println",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "call_expression") {
					t.Error("Should contain call_expression")
				}
				if !strings.Contains(result, "selector_expression") {
					t.Error("Should support selector_expression")
				}
			},
		},
		{
			name:     "assignment_statement",
			dslQuery: "assign:result",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "assignment_statement") {
					t.Error("Should contain assignment_statement")
				}
				if !strings.Contains(result, "short_var_declaration") {
					t.Error("Should contain short_var_declaration")
				}
			},
		},
		{
			name:     "import_statement",
			dslQuery: "import:fmt",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "import_spec") {
					t.Error("Should contain import_spec")
				}
				if !strings.Contains(result, `(#eq? @path "fmt")`) {
					t.Error("Should contain path predicate")
				}
				// Should NOT contain @name predicate for imports
				if strings.Contains(result, `(#eq? @name "fmt")`) {
					t.Error("Import should use @path, not @name")
				}
			},
		},
		{
			name:     "wildcard_function",
			dslQuery: "func:Handle*",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#match?") {
					t.Error("Wildcard should use #match? predicate")
				}
				if !strings.Contains(result, "^Handle.*") {
					t.Error("Should contain proper regex pattern")
				}
			},
		},
		{
			name:     "negated_function",
			dslQuery: "!func:test",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#not-eq?") {
					t.Error("Negated query should use #not-eq?")
				}
			},
		},
		{
			name:     "negated_wildcard",
			dslQuery: "!func:Test*",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#not-match?") {
					t.Error("Negated wildcard should use #not-match?")
				}
			},
		},
		{
			name:     "parent_child_relationship",
			dslQuery: "func:main > var:config",
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Should contain function_declaration")
				}
				if !strings.Contains(result, "var_declaration") {
					t.Error("Should contain var_declaration")
				}
				if strings.Count(result, "@target") != 1 {
					t.Error("Should contain exactly one @target")
				}
			},
		},
		{
			name:     "complex_nested_query",
			dslQuery: "func:handler > if:* > call:log.Error",
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
			},
		},
		// Error cases
		{
			name:        "empty_query",
			dslQuery:    "",
			expectError: true,
			errorMsg:    "query cannot be empty",
		},
		{
			name:        "invalid_format",
			dslQuery:    "func",
			expectError: true,
			errorMsg:    "invalid DSL query",
		},
		{
			name:        "unknown_node_type",
			dslQuery:    "unknown:test",
			expectError: true,
			errorMsg:    "unknown node type",
		},
		{
			name:        "if_with_non_wildcard",
			dslQuery:    "if:condition",
			expectError: true,
			errorMsg:    "only * supported for if/block",
		},
		{
			name:        "block_with_non_wildcard",
			dslQuery:    "block:body",
			expectError: true,
			errorMsg:    "only * supported for if/block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.TranslateDSL(tt.dslQuery)

			if tt.expectError {
				if err == nil {
					t.Fatalf("TranslateDSL(%q) expected error, got none", tt.dslQuery)
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("TranslateDSL(%q) unexpected error: %v", tt.dslQuery, err)
			}

			if result == "" {
				t.Fatalf("TranslateDSL(%q) returned empty result", tt.dslQuery)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}

			t.Logf("DSL Query: %s\nTree-sitter Query: %s", tt.dslQuery, result)
		})
	}
}

// TestDSLAcceptanceScenarios tests real-world DSL usage scenarios
func TestDSLAcceptanceScenarios(t *testing.T) {
	p := &goProvider{}

	// Positive tests (should match)
	positiveTests := []struct {
		name        string
		dslQuery    string
		description string
	}{
		{
			name:        "call_selector_expression",
			dslQuery:    "call:util.SHA1FileHex",
			description: "Function call with package selector",
		},
		{
			name:        "assign_with_call_child",
			dslQuery:    "assign:fileHash > call:util.SHA1FileHex",
			description: "Assignment containing a function call",
		},
		{
			name:        "func_with_var_child",
			dslQuery:    "func:* > var:* core.ModelConfig",
			description: "Any function containing any variable with type",
		},
		{
			name:        "struct_with_field_child",
			dslQuery:    "struct:* > field:Secret string",
			description: "Any struct with Secret field of string type",
		},
		{
			name:        "import_fmt",
			dslQuery:    "import:fmt",
			description: "Import of fmt package",
		},
		{
			name:        "test_function_pattern",
			dslQuery:    "func:Test*",
			description: "Test functions pattern",
		},
		{
			name:        "handler_function_pattern",
			dslQuery:    "func:*Handler",
			description: "Handler functions pattern",
		},
	}

	for _, tt := range positiveTests {
		t.Run("positive_"+tt.name, func(t *testing.T) {
			result, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Expected no error for DSL query '%s', but got: %v", tt.dslQuery, err)
			}
			if result == "" {
				t.Fatalf("Expected non-empty result for DSL query '%s'", tt.dslQuery)
			}
			t.Logf("%s: %s -> %s", tt.description, tt.dslQuery, result)
		})
	}

	// Negative tests (should error)
	negativeTests := []struct {
		name        string
		dslQuery    string
		description string
	}{
		{
			name:        "if_with_name",
			dslQuery:    "if:Nombre",
			description: "If statement with specific name (should only accept *)",
		},
		{
			name:        "block_with_name",
			dslQuery:    "block:Nombre",
			description: "Block with specific name (should only accept *)",
		},
	}

	for _, tt := range negativeTests {
		t.Run("negative_"+tt.name, func(t *testing.T) {
			_, err := p.TranslateDSL(tt.dslQuery)
			if err == nil {
				t.Fatalf("Expected error for DSL query '%s', but got none", tt.dslQuery)
			}
			t.Logf("%s: %s -> Error: %v", tt.description, tt.dslQuery, err)
		})
	}
}
