package parser

import (
	"slices"
	"testing"

	"github.com/termfx/morfx/internal/types"
)

func TestNewUniversalParser(t *testing.T) {
	parser := NewUniversalParser()

	if parser == nil {
		t.Fatal("NewUniversalParser() returned nil")
	}

	// Check that supported kinds are initialized
	supportedKinds := parser.GetSupportedKinds()
	if len(supportedKinds) == 0 {
		t.Error("Expected supported kinds to be initialized")
	}

	// Check that supported operators are initialized
	supportedOps := parser.GetSupportedOperators()
	if len(supportedOps) == 0 {
		t.Error("Expected supported operators to be initialized")
	}
}

func TestParseSimpleQuery(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name     string
		input    string
		expected types.Query
		wantErr  bool
	}{
		{
			name:  "simple function query",
			input: "function:*",
			expected: types.Query{
				Kind:       types.KindFunction,
				Pattern:    "*",
				Attributes: make(map[string]string),
				Raw:        "function:*",
			},
			wantErr: false,
		},

		{
			name:  "function with pattern",
			input: "function:test*",
			expected: types.Query{
				Kind:       types.KindFunction,
				Pattern:    "test*",
				Attributes: make(map[string]string),
				Raw:        "function:test*",
			},
			wantErr: false,
		},
		{
			name:  "function with type",
			input: "function:main public",
			expected: types.Query{
				Kind:       types.KindFunction,
				Pattern:    "main",
				Attributes: map[string]string{"type": "public"},
				Raw:        "function:main public",
			},
			wantErr: false,
		},
		{
			name:  "function with pattern and type",
			input: "function:test* public",
			expected: types.Query{
				Kind:       types.KindFunction,
				Pattern:    "test*",
				Attributes: map[string]string{"type": "public"},
				Raw:        "function:test* public",
			},
			wantErr: false,
		},
		{
			name:    "invalid kind",
			input:   "invalidkind:test",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:  "NOT query",
			input: "!function:*",
			expected: types.Query{
				Kind:       types.KindFunction,
				Pattern:    "*",
				Operator:   "!",
				Attributes: make(map[string]string),
				Raw:        "function:*",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseQuery(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Kind != tt.expected.Kind {
				t.Errorf("Expected Kind %s, got %s", tt.expected.Kind, result.Kind)
			}

			if result.Pattern != tt.expected.Pattern {
				t.Errorf("Expected Pattern '%s', got '%s'", tt.expected.Pattern, result.Pattern)
			}

			if result.Raw != tt.expected.Raw {
				t.Errorf("Expected Raw '%s', got '%s'", tt.expected.Raw, result.Raw)
			}

			// Check attributes if expected
			if tt.expected.Attributes != nil {
				if result.Attributes == nil {
					t.Error("Expected attributes but got nil")
					return
				}
				for key, expectedValue := range tt.expected.Attributes {
					if actualValue, exists := result.Attributes[key]; !exists || actualValue != expectedValue {
						t.Errorf("Expected attribute %s=%s, got %s=%s", key, expectedValue, key, actualValue)
					}
				}
			}
		})
	}
}

func TestParseLogicalQuery(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name     string
		input    string
		expected types.Query
		wantErr  bool
	}{
		{
			name:  "AND query",
			input: "function:* & variable:*",
			expected: types.Query{
				Kind:       "logical",
				Operator:   "&&",
				Attributes: make(map[string]string),
				Children: []types.Query{
					{Kind: types.KindFunction, Pattern: "*", Raw: "function:*", Attributes: make(map[string]string)},
					{Kind: types.KindVariable, Pattern: "*", Raw: "variable:*", Attributes: make(map[string]string)},
				},
				Raw: "function:* & variable:*",
			},
			wantErr: false,
		},
		{
			name:  "OR query",
			input: "function:* | variable:*",
			expected: types.Query{
				Kind:       "logical",
				Operator:   "||",
				Attributes: make(map[string]string),
				Children: []types.Query{
					{Kind: types.KindFunction, Pattern: "*", Raw: "function:*", Attributes: make(map[string]string)},
					{Kind: types.KindVariable, Pattern: "*", Raw: "variable:*", Attributes: make(map[string]string)},
				},
				Raw: "function:* | variable:*",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseQuery(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Operator != tt.expected.Operator {
				t.Errorf("Expected Operator '%s', got '%s'", tt.expected.Operator, result.Operator)
			}

			if len(result.Children) != len(tt.expected.Children) {
				t.Errorf("Expected %d children, got %d", len(tt.expected.Children), len(result.Children))
				return
			}

			for i, expectedChild := range tt.expected.Children {
				if result.Children[i].Kind != expectedChild.Kind {
					t.Errorf("Expected child %d Kind %s, got %s", i, expectedChild.Kind, result.Children[i].Kind)
				}
			}
		})
	}
}

func TestParseHierarchicalQuery(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name     string
		input    string
		expected types.Query
		wantErr  bool
	}{
		{
			name:  "parent > child query",
			input: "class:* > method:*",
			expected: types.Query{
				Kind:       types.KindMethod,
				Pattern:    "*",
				Operator:   ">",
				Attributes: make(map[string]string),
				Children: []types.Query{
					{Kind: types.KindClass, Pattern: "*", Raw: "class:*", Attributes: make(map[string]string)},
				},
				Raw: "class:* > method:*",
			},
			wantErr: false,
		},
		{
			name:  "nested hierarchical query",
			input: "class:* > variable:*",
			expected: types.Query{
				Kind:       types.KindVariable,
				Pattern:    "*",
				Operator:   ">",
				Attributes: make(map[string]string),
				Children: []types.Query{
					{Kind: types.KindClass, Pattern: "*", Raw: "class:*", Attributes: make(map[string]string)},
				},
				Raw: "class:* > variable:*",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseQuery(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Operator != tt.expected.Operator {
				t.Errorf("Expected Operator '%s', got '%s'", tt.expected.Operator, result.Operator)
			}

			if len(result.Children) != len(tt.expected.Children) {
				t.Errorf("Expected %d children, got %d", len(tt.expected.Children), len(result.Children))
				return
			}

			for i, expectedChild := range tt.expected.Children {
				if result.Children[i].Kind != expectedChild.Kind {
					t.Errorf("Expected child %d Kind %s, got %s", i, expectedChild.Kind, result.Children[i].Kind)
				}
			}
		})
	}
}

func TestValidateQuery(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid simple query",
			query:   "function:*",
			wantErr: false,
		},
		{
			name:    "valid logical query",
			query:   "function:* & variable:*",
			wantErr: false,
		},
		{
			name:    "valid hierarchical query",
			query:   "class:* > method:*",
			wantErr: false,
		},
		{
			name:    "empty query string",
			query:   "",
			wantErr: true,
		},
		{
			name:    "invalid hierarchical format",
			query:   "class:* > method:* > variable:*",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateQuery(tt.query)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetSupportedKinds(t *testing.T) {
	parser := NewUniversalParser()
	kinds := parser.GetSupportedKinds()

	if len(kinds) == 0 {
		t.Error("Expected non-empty supported kinds")
	}

	// Check for some expected kinds
	expectedKinds := []types.NodeKind{
		types.KindFunction,
		types.KindVariable,
		types.KindClass,
		types.KindMethod,
	}

	for _, expected := range expectedKinds {
		found := slices.Contains(kinds, expected)
		if !found {
			t.Errorf("Expected kind %s not found in supported kinds", expected)
		}
	}
}

func TestGetSupportedOperators(t *testing.T) {
	parser := NewUniversalParser()
	operators := parser.GetSupportedOperators()

	if len(operators) == 0 {
		t.Error("Expected non-empty supported operators")
	}

	// Should have some operators
	if len(operators) == 0 {
		t.Error("Expected some operators, got none")
	}

	// Check that we get a list of strings
	for _, op := range operators {
		if op == "" {
			t.Error("Found empty operator string")
		}
	}
}

func TestParseComplexQuery(t *testing.T) {
	parser := NewUniversalParser()

	// Test complex query with logical operators
	query, err := parser.ParseQuery("function:* & class:*")
	if err != nil {
		t.Errorf("Unexpected error parsing complex query: %v", err)
	}

	if query.Operator != "&&" {
		t.Errorf("Expected operator '&&', got %s", query.Operator)
	}
}

func BenchmarkParseSimpleQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function test*"

	for b.Loop() {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkParseLogicalQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function & variable"

	for b.Loop() {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkValidateQuery(b *testing.B) {
	parser := NewUniversalParser()

	for b.Loop() {
		err := parser.ValidateQuery("function test*")
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}
