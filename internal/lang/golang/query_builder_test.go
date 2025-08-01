package golang

import (
	"strings"
	"testing"
)

// TestGoProvider_GetQuery tests query generation for different node types
func TestGoProvider_GetQuery(t *testing.T) {
	p := &goProvider{}

	tests := []struct {
		name          string
		nodeType      string
		nodeName      string
		expectOK      bool
		shouldContain []string
	}{
		{
			name:          "function_query",
			nodeType:      "func",
			nodeName:      "main",
			expectOK:      true,
			shouldContain: []string{"function_declaration", `(#eq? @name "main")`, "@target"},
		},
		{
			name:          "variable_query",
			nodeType:      "var",
			nodeName:      "config",
			expectOK:      true,
			shouldContain: []string{"var_declaration", `(#eq? @name "config")`, "@target"},
		},
		{
			name:          "constant_query",
			nodeType:      "const",
			nodeName:      "MaxSize",
			expectOK:      true,
			shouldContain: []string{"const_declaration", `(#eq? @name "MaxSize")`, "@target"},
		},
		{
			name:          "struct_query",
			nodeType:      "struct",
			nodeName:      "User",
			expectOK:      true,
			shouldContain: []string{"type_declaration", `(#eq? @name "User")`, "struct_type", "@target"},
		},
		{
			name:          "field_query",
			nodeType:      "field",
			nodeName:      "Name",
			expectOK:      true,
			shouldContain: []string{"field_declaration", `(#eq? @name "Name")`, "@target"},
		},
		{
			name:          "call_query",
			nodeType:      "call",
			nodeName:      "fmt.Println",
			expectOK:      true,
			shouldContain: []string{"call_expression", `(#eq? @name "fmt.Println")`, "@target"},
		},
		{
			name:          "assignment_query",
			nodeType:      "assign",
			nodeName:      "result",
			expectOK:      true,
			shouldContain: []string{"assignment_statement", "short_var_declaration", `(#eq? @name "result")`, "@target"},
		},
		{
			name:          "import_query",
			nodeType:      "import",
			nodeName:      "fmt",
			expectOK:      true,
			shouldContain: []string{"import_spec", `(#eq? @path "fmt")`, "@target"},
		},
		{
			name:          "import_query_with_path",
			nodeType:      "import",
			nodeName:      "github.com/pkg/errors",
			expectOK:      true,
			shouldContain: []string{"import_spec", `(#eq? @path "github.com/pkg/errors")`, "@target"},
		},
		{
			name:          "if_statement_query_wildcard",
			nodeType:      "if",
			nodeName:      "*",
			expectOK:      true,
			shouldContain: []string{"if_statement", "@target"},
		},
		{
			name:          "block_query_wildcard",
			nodeType:      "block",
			nodeName:      "*",
			expectOK:      true,
			shouldContain: []string{"block", "@target"},
		},
		{
			name:     "if_statement_query_non_wildcard",
			nodeType: "if",
			nodeName: "condition",
			expectOK: false,
		},
		{
			name:     "block_query_non_wildcard",
			nodeType: "block",
			nodeName: "body",
			expectOK: false,
		},
		{
			name:     "unknown_node_type",
			nodeType: "unknown",
			nodeName: "test",
			expectOK: false,
		},
		{
			name:     "empty_node_type",
			nodeType: "",
			nodeName: "test",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, ok := p.GetQuery(tt.nodeType, tt.nodeName)

			if ok != tt.expectOK {
				t.Fatalf("GetQuery(%q, %q) ok = %v, want %v", tt.nodeType, tt.nodeName, ok, tt.expectOK)
			}

			if !tt.expectOK {
				if query != "" {
					t.Errorf("GetQuery(%q, %q) returned non-empty query for invalid input: %q", tt.nodeType, tt.nodeName, query)
				}
				return
			}

			if query == "" {
				t.Fatalf("GetQuery(%q, %q) returned empty query", tt.nodeType, tt.nodeName)
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(query, expected) {
					t.Errorf("GetQuery(%q, %q) result should contain %q.\nGot: %s", tt.nodeType, tt.nodeName, expected, query)
				}
			}
		})
	}
}

// TestGoProvider_BuildTreeSitterQuery tests query building with error conditions
func TestGoProvider_BuildTreeSitterQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       *Query
		expectError bool
		errorMsg    string
		validate    func(*testing.T, string)
	}{
		{
			name: "simple_function_query",
			query: &Query{
				NodeType:   "func",
				Identifier: "main",
				Not:        false,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Result should contain function_declaration")
				}
				if !strings.Contains(result, `(#eq? @name "main")`) {
					t.Error("Result should contain name predicate")
				}
				if !strings.Contains(result, "@target") {
					t.Error("Result should contain @target")
				}
			},
		},
		{
			name: "negated_query",
			query: &Query{
				NodeType:   "func",
				Identifier: "test",
				Not:        true,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#not-eq?") {
					t.Error("Negated query should contain #not-eq?")
				}
			},
		},
		{
			name: "wildcard_negated_query",
			query: &Query{
				NodeType:   "func",
				Identifier: "Test*",
				Not:        true,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#not-match?") {
					t.Error("Negated wildcard query should contain #not-match?")
				}
			},
		},
		{
			name: "query_with_child",
			query: &Query{
				NodeType:   "func",
				Identifier: "main",
				Not:        false,
				Children: []Query{
					{
						NodeType:   "var",
						Identifier: "config",
						Not:        false,
						Children:   []Query{},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Result should contain function_declaration")
				}
				if !strings.Contains(result, "var_declaration") {
					t.Error("Result should contain var_declaration")
				}
				if strings.Count(result, "@target") != 1 {
					t.Error("Result should contain exactly one @target (child @target should be removed)")
				}
			},
		},
		{
			name: "nested_children",
			query: &Query{
				NodeType:   "func",
				Identifier: "handler",
				Not:        false,
				Children: []Query{
					{
						NodeType:   "if",
						Identifier: "*",
						Not:        false,
						Children: []Query{
							{
								NodeType:   "call",
								Identifier: "log.Error",
								Not:        false,
								Children:   []Query{},
							},
						},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Result should contain function_declaration")
				}
				if !strings.Contains(result, "if_statement") {
					t.Error("Result should contain if_statement")
				}
				if !strings.Contains(result, "call_expression") {
					t.Error("Result should contain call_expression")
				}
			},
		},
		{
			name: "if_with_non_wildcard_identifier",
			query: &Query{
				NodeType:   "if",
				Identifier: "condition",
				Not:        false,
				Children:   []Query{},
			},
			expectError: true,
			errorMsg:    "only * supported for if/block",
		},
		{
			name: "block_with_non_wildcard_identifier",
			query: &Query{
				NodeType:   "block",
				Identifier: "body",
				Not:        false,
				Children:   []Query{},
			},
			expectError: true,
			errorMsg:    "only * supported for if/block",
		},
		{
			name: "if_with_wildcard_identifier",
			query: &Query{
				NodeType:   "if",
				Identifier: "*",
				Not:        false,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "if_statement") {
					t.Error("Result should contain if_statement")
				}
			},
		},
		{
			name: "block_with_wildcard_identifier",
			query: &Query{
				NodeType:   "block",
				Identifier: "*",
				Not:        false,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "block") {
					t.Error("Result should contain block")
				}
			},
		},
		{
			name: "unknown_node_type",
			query: &Query{
				NodeType:   "unknown",
				Identifier: "test",
				Not:        false,
				Children:   []Query{},
			},
			expectError: true,
			errorMsg:    "unknown node type: unknown",
		},
		{
			name: "child_query_error",
			query: &Query{
				NodeType:   "func",
				Identifier: "main",
				Not:        false,
				Children: []Query{
					{
						NodeType:   "unknown",
						Identifier: "test",
						Not:        false,
						Children:   []Query{},
					},
				},
			},
			expectError: true,
			errorMsg:    "unknown node type: unknown",
		},
		{
			name: "wildcard_identifier",
			query: &Query{
				NodeType:   "func",
				Identifier: "*",
				Not:        false,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "function_declaration") {
					t.Error("Result should contain function_declaration")
				}
				// Should not contain any predicate for wildcard
				if strings.Contains(result, "#eq?") || strings.Contains(result, "#match?") {
					t.Error("Wildcard query should not contain predicates")
				}
			},
		},
		{
			name: "complex_wildcard_patterns",
			query: &Query{
				NodeType:   "func",
				Identifier: "Handle*Request",
				Not:        false,
				Children:   []Query{},
			},
			expectError: false,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "#match?") {
					t.Error("Complex wildcard should use #match? predicate")
				}
				if !strings.Contains(result, "^Handle.*Request$") {
					t.Error("Should contain proper regex pattern")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildTreeSitterQuery(tt.query)

			if tt.expectError {
				if err == nil {
					t.Fatalf("BuildTreeSitterQuery() expected error, got none")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildTreeSitterQuery() unexpected error: %v", err)
			}

			if result == "" {
				t.Fatal("BuildTreeSitterQuery() returned empty result")
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestQueryConstraintCombination tests different constraint combinations
func TestQueryConstraintCombination(t *testing.T) {
	tests := []struct {
		name     string
		query    *Query
		validate func(*testing.T, string)
	}{
		{
			name: "predicate_only",
			query: &Query{
				NodeType:   "func",
				Identifier: "test",
				Children:   []Query{},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, `(#eq? @name "test")`) {
					t.Error("Should contain predicate only")
				}
			},
		},
		{
			name: "child_constraint_only",
			query: &Query{
				NodeType:   "func",
				Identifier: "*",
				Children: []Query{
					{
						NodeType:   "var",
						Identifier: "config",
						Children:   []Query{},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "var_declaration") {
					t.Error("Should contain child constraint")
				}
				// Should not contain predicate for wildcard parent
				if strings.Contains(result, `(#eq? @name "*")`) {
					t.Error("Should not contain wildcard predicate")
				}
			},
		},
		{
			name: "predicate_and_child_constraint",
			query: &Query{
				NodeType:   "func",
				Identifier: "main",
				Children: []Query{
					{
						NodeType:   "var",
						Identifier: "config",
						Children:   []Query{},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, `(#eq? @name "main")`) {
					t.Error("Should contain parent predicate")
				}
				if !strings.Contains(result, "var_declaration") {
					t.Error("Should contain child constraint")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildTreeSitterQuery(tt.query)
			if err != nil {
				t.Fatalf("BuildTreeSitterQuery() unexpected error: %v", err)
			}

			tt.validate(t, result)
		})
	}
}
