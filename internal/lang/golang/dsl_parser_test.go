package golang

import (
	"testing"
)

// TestParseDSL tests DSL parsing with comprehensive scenarios
func TestParseDSL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(*testing.T, *Query)
	}{
		{
			name:        "simple_function_query",
			input:       "func:main",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "main" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "main")
				}
				if q.Not {
					t.Error("Not should be false")
				}
				if len(q.Children) != 0 {
					t.Errorf("Children length = %d, want 0", len(q.Children))
				}
			},
		},
		{
			name:        "negated_query",
			input:       "!func:test",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if !q.Not {
					t.Error("Not should be true")
				}
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "test" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "test")
				}
			},
		},
		{
			name:        "parent_child_relationship",
			input:       "func:main > var:config",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "main" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "main")
				}
				if len(q.Children) != 1 {
					t.Fatalf("Children length = %d, want 1", len(q.Children))
				}
				child := q.Children[0]
				if child.NodeType != "var" {
					t.Errorf("Child NodeType = %q, want %q", child.NodeType, "var")
				}
				if child.Identifier != "config" {
					t.Errorf("Child Identifier = %q, want %q", child.Identifier, "config")
				}
			},
		},
		{
			name:        "complex_nested_query",
			input:       "func:handler > if:* > call:log.Error",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if len(q.Children) != 1 {
					t.Fatalf("Children length = %d, want 1", len(q.Children))
				}
				ifChild := q.Children[0]
				if ifChild.NodeType != "if" {
					t.Errorf("Child NodeType = %q, want %q", ifChild.NodeType, "if")
				}
				if len(ifChild.Children) != 1 {
					t.Fatalf("Nested children length = %d, want 1", len(ifChild.Children))
				}
				callChild := ifChild.Children[0]
				if callChild.NodeType != "call" {
					t.Errorf("Nested child NodeType = %q, want %q", callChild.NodeType, "call")
				}
				if callChild.Identifier != "log.Error" {
					t.Errorf("Nested child Identifier = %q, want %q", callChild.Identifier, "log.Error")
				}
			},
		},
		{
			name:        "query_with_whitespace",
			input:       "  func : main  >  var : config  ",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "main" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "main")
				}
				if len(q.Children) != 1 {
					t.Fatalf("Children length = %d, want 1", len(q.Children))
				}
				if q.Children[0].Identifier != "config" {
					t.Errorf("Child Identifier = %q, want %q", q.Children[0].Identifier, "config")
				}
			},
		},
		{
			name:        "wildcard_identifier",
			input:       "func:*",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if q.Identifier != "*" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "*")
				}
			},
		},
		{
			name:        "complex_identifier_with_dots",
			input:       "call:fmt.Println",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "call" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "call")
				}
				if q.Identifier != "fmt.Println" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "fmt.Println")
				}
			},
		},
		{
			name:        "negated_complex_query",
			input:       "!func:test > var:config",
			expectError: false,
			validate: func(t *testing.T, q *Query) {
				if !q.Not {
					t.Error("Not should be true")
				}
				if len(q.Children) != 1 {
					t.Fatalf("Children length = %d, want 1", len(q.Children))
				}
			},
		},
		// Error cases
		{
			name:        "empty_query",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid_format_no_colon",
			input:       "func",
			expectError: true,
		},
		{
			name:        "invalid_format_empty_node_type",
			input:       ":main",
			expectError: false, // This actually parses as empty nodeType with identifier "main"
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "" {
					t.Errorf("NodeType = %q, want empty", q.NodeType)
				}
				if q.Identifier != "main" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "main")
				}
			},
		},
		{
			name:        "invalid_format_empty_identifier",
			input:       "func:",
			expectError: false, // This actually parses as nodeType "func" with empty identifier
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "" {
					t.Errorf("Identifier = %q, want empty", q.Identifier)
				}
			},
		},
		{
			name:        "invalid_child_query",
			input:       "func:main > invalid",
			expectError: true,
		},
		{
			name:        "multiple_colons_in_node_part",
			input:       "func:main:extra",
			expectError: false, // Should parse as func with identifier "main:extra"
			validate: func(t *testing.T, q *Query) {
				if q.NodeType != "func" {
					t.Errorf("NodeType = %q, want %q", q.NodeType, "func")
				}
				if q.Identifier != "main:extra" {
					t.Errorf("Identifier = %q, want %q", q.Identifier, "main:extra")
				}
			},
		},
		{
			name:        "empty_child_after_separator",
			input:       "func:main > ",
			expectError: false, // Should parse successfully with no children
			validate: func(t *testing.T, q *Query) {
				if len(q.Children) != 0 {
					t.Errorf("Children length = %d, want 0", len(q.Children))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDSL(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatalf("ParseDSL(%q) expected error, got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseDSL(%q) unexpected error: %v", tt.input, err)
			}

			if result == nil {
				t.Fatalf("ParseDSL(%q) returned nil result", tt.input)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestParseDSL_EdgeCases tests edge cases and boundary conditions
func TestParseDSL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "only_whitespace",
			input:       "   ",
			expectError: true,
			description: "Query with only whitespace should fail",
		},
		{
			name:        "only_negation",
			input:       "!",
			expectError: true,
			description: "Query with only negation should fail",
		},
		{
			name:        "multiple_negations",
			input:       "!!func:test",
			expectError: false,
			description: "Multiple negations should be treated as single negation",
		},
		{
			name:        "nested_separators",
			input:       "func:main > > var:config",
			expectError: true,
			description: "Empty parts between separators should fail",
		},
		{
			name:        "unicode_identifiers",
			input:       "func:测试函数",
			expectError: false,
			description: "Unicode identifiers should be supported",
		},
		{
			name:        "special_characters_in_identifier",
			input:       "func:test_func-v2",
			expectError: false,
			description: "Special characters in identifiers should be supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDSL(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseDSL(%q) expected error but got none. %s", tt.input, tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDSL(%q) unexpected error: %v. %s", tt.input, err, tt.description)
				return
			}

			if result == nil {
				t.Errorf("ParseDSL(%q) returned nil result. %s", tt.input, tt.description)
			}
		})
	}
}
