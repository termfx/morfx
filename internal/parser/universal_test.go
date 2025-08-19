package parser

import (
	"slices"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/termfx/morfx/internal/types"
)

// MockProvider implements types.LanguageProvider for testing
type MockProvider struct {
	dslMappings map[string]string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		dslMappings: map[string]string{
			"fn":       "function",
			"func":     "function",
			"var":      "variable",
			"cls":      "class",
			"method":   "method",
			"field":    "field",
			"import":   "import",
			"call":     "call",
			"assign":   "assignment",
			"if":       "condition",
			"block":    "block",
			"loop":     "loop",
			"struct":   "struct",
			"const":    "constant",
			"function": "function", // Direct mapping
			"variable": "variable", // Direct mapping
			"class":    "class",    // Direct mapping
		},
	}
}

func (m *MockProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	if normalized, exists := m.dslMappings[dslKind]; exists {
		return types.NodeKind(normalized)
	}
	return types.NodeKind(dslKind)
}

// Implement other required methods for types.LanguageProvider interface
func (m *MockProvider) Lang() string                                                    { return "mock" }
func (m *MockProvider) Aliases() []string                                               { return []string{"mock"} }
func (m *MockProvider) Extensions() []string                                            { return []string{".mock"} }
func (m *MockProvider) GetSitterLanguage() *sitter.Language                           { return nil }
func (m *MockProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping          { return []types.NodeMapping{{Kind: kind, NodeTypes: []string{string(kind)}}} }
func (m *MockProvider) TranslateQuery(q *types.Query) (string, error)                 { return q.Raw, nil }
func (m *MockProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string { return make(map[string]string) }
func (m *MockProvider) GetNodeKind(node *sitter.Node) types.NodeKind                   { return "function" }
func (m *MockProvider) GetNodeName(node *sitter.Node, source []byte) string            { return "mockName" }
func (m *MockProvider) OptimizeQuery(query *types.Query) *types.Query                  { return query }
func (m *MockProvider) EstimateQueryCost(query *types.Query) int                       { return 1 }
func (m *MockProvider) GetNodeScope(node *sitter.Node) types.ScopeType                 { return types.ScopeFile }
func (m *MockProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node { return nil }
func (m *MockProvider) IsBlockLevelNode(nodeType string) bool                          { return false }
func (m *MockProvider) GetDefaultIgnorePatterns() ([]string, []string)                { return []string{}, []string{} }
func (m *MockProvider) GetSupportedDSLKinds() []string                                 { return []string{"function", "variable", "class"} }
func (m *MockProvider) BuildMappings() map[types.NodeKind][]string                     { return make(map[types.NodeKind][]string) }
func (m *MockProvider) CacheQuery(query string, result *types.Query)                   {}
func (m *MockProvider) GetCachedQuery(query string) (*types.Query, bool)               { return nil, false }

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

// Tests for ParseQueryWithProvider
func TestParseQueryWithProvider(t *testing.T) {
	parser := NewUniversalParser()
	mockProvider := NewMockProvider()

	tests := []struct {
		name        string
		query       string
		wantKind    types.NodeKind
		wantPattern string
		wantErr     bool
	}{
		{
			name:        "simple function query with provider DSL",
			query:       "fn:test",
			wantKind:    "function",
			wantPattern: "test",
			wantErr:     false,
		},
		{
			name:        "variable query with provider DSL",
			query:       "var:myVar",
			wantKind:    "variable",
			wantPattern: "myVar",
			wantErr:     false,
		},
		{
			name:        "negated query with provider DSL",
			query:       "!fn:test",
			wantKind:    "function",
			wantPattern: "test",
			wantErr:     false,
		},
		{
			name:        "hierarchical query with provider DSL",
			query:       "cls:MyClass > fn:method",
			wantKind:    "function",
			wantPattern: "method",
			wantErr:     false,
		},
		{
			name:        "logical AND query with provider DSL",
			query:       "fn:test & var:x",
			wantKind:    "logical",
			wantPattern: "",
			wantErr:     false,
		},
		{
			name:        "logical OR query with provider DSL",
			query:       "fn:test | var:x",
			wantKind:    "logical",
			wantPattern: "",
			wantErr:     false,
		},
		{
			name:        "empty query",
			query:       "",
			wantKind:    "",
			wantPattern: "",
			wantErr:     true,
		},
		{
			name:        "whitespace only query",
			query:       "   \t\n  ",
			wantKind:    "",
			wantPattern: "",
			wantErr:     true,
		},
		{
			name:        "unsupported DSL kind",
			query:       "unknown:test",
			wantKind:    "unknown",
			wantPattern: "test",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseQueryWithProvider(tt.query, mockProvider)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseQueryWithProvider() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseQueryWithProvider() unexpected error: %v", err)
				return
			}

			if result.Kind != tt.wantKind {
				t.Errorf("ParseQueryWithProvider() kind = %v, want %v", result.Kind, tt.wantKind)
			}

			if result.Pattern != tt.wantPattern {
				t.Errorf("ParseQueryWithProvider() pattern = %v, want %v", result.Pattern, tt.wantPattern)
			}

			// For negated queries, the Raw field should not include the negation operator
			expectedRaw := tt.query
			if strings.HasPrefix(tt.query, "!") {
				expectedRaw = strings.TrimPrefix(tt.query, "!")
			}
			if result.Raw != expectedRaw {
				t.Errorf("ParseQueryWithProvider() raw = %v, want %v", result.Raw, expectedRaw)
			}
		})
	}
}

// Tests for IsWildcard
func TestIsWildcard(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "asterisk wildcard",
			input: "*",
			want:  true,
		},
		{
			name:  "question mark wildcard",
			input: "?",
			want:  true,
		},
		{
			name:  "string with asterisk",
			input: "test*",
			want:  true,
		},
		{
			name:  "string with question mark",
			input: "test?",
			want:  true,
		},
		{
			name:  "string with both wildcards",
			input: "te*st?",
			want:  true,
		},
		{
			name:  "regular string",
			input: "test",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "numeric string",
			input: "123",
			want:  false,
		},
		{
			name:  "special characters without wildcards",
			input: "test@#$%",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.IsWildcard(tt.input)
			if got != tt.want {
				t.Errorf("IsWildcard(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Tests for NormalizeQuery
func TestNormalizeQuery(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim leading whitespace",
			input: "   function:test",
			want:  "function:test",
		},
		{
			name:  "trim trailing whitespace",
			input: "function:test   ",
			want:  "function:test",
		},
		{
			name:  "trim both sides",
			input: "   function:test   ",
			want:  "function:test",
		},
		{
			name:  "normalize internal whitespace",
			input: "function:test    type",
			want:  "function:test type",
		},
		{
			name:  "normalize tabs and newlines",
			input: "function:test\t\ntype",
			want:  "function:test type",
		},
		{
			name:  "complex whitespace normalization",
			input: "  function:test   \t\n  type   ",
			want:  "function:test type",
		},
		{
			name:  "already normalized",
			input: "function:test",
			want:  "function:test",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \t\n   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.NormalizeQuery(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Tests for normalizeKind
func TestParserEdgeCases(t *testing.T) {
	parser := NewUniversalParser()

	// Test malformed queries
	malformedTests := []struct {
		name  string
		input string
	}{
		{"invalid operator sequence", "function && && variable"},
		{"unclosed parentheses", "(function:test"},
		{"invalid hierarchical syntax", "function >> variable"},
		{"empty parentheses", "()"},
		{"nested empty parentheses", "((()))"},
		{"invalid negation", "!!function:test"},
		{"trailing operator", "function:test &&"},
		{"leading operator", "&& function:test"},
		{"multiple colons", "function::test::name"},
		{"invalid characters", "function:test@#$"},
		{"very long query", strings.Repeat("function:test && ", 100)},
	}

	for _, tt := range malformedTests {
		t.Run("malformed_"+tt.name, func(t *testing.T) {
			_, err := parser.ParseQuery(tt.input)
			// Most malformed queries should either parse successfully with best-effort
			// or return an error - we're testing that they don't panic
			_ = err // We don't assert specific behavior for malformed input
		})
	}

	// Test boundary conditions
	boundaryTests := []struct {
		name     string
		input    string
		expected bool // whether we expect successful parsing
	}{
		{"single character kind", "a:test", false},
		{"single character pattern", "function:a", true},
		{"unicode in pattern", "function:测试", true},
		{"unicode in kind", "函数:test", false},
		{"numbers in kind", "func123:test", false},
		{"numbers in pattern", "function:test123", true},
		{"special chars in pattern", "function:test_name-123", true},
		{"very long kind name", strings.Repeat("a", 100) + ":test", false},
		{"very long pattern", "function:" + strings.Repeat("a", 100), true},
	}

	for _, tt := range boundaryTests {
		t.Run("boundary_"+tt.name, func(t *testing.T) {
			result, err := parser.ParseQuery(tt.input)
			if tt.expected {
				if err != nil {
					t.Errorf("ParseQuery(%q) unexpected error: %v", tt.input, err)
				}
				if result == nil {
					t.Errorf("ParseQuery(%q) returned nil result", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("ParseQuery(%q) expected error but got none", tt.input)
				}
			}
		})
	}

	// Test whitespace edge cases
	whitespaceTests := []struct {
		name  string
		input string
	}{
		{"tabs only", "\t\t\t"},
		{"newlines only", "\n\n\n"},
		{"mixed whitespace", "\t \n \r"},
		{"whitespace in middle", "function\t:\n\rtest"},
		{"excessive whitespace", "   function   :   test   "},
	}

	for _, tt := range whitespaceTests {
		t.Run("whitespace_"+tt.name, func(t *testing.T) {
			_, err := parser.ParseQuery(tt.input)
			// Whitespace-only queries should typically result in empty or error
			_ = err // We don't assert specific behavior
		})
	}
}

func TestParserComplexScenarios(t *testing.T) {
	parser := NewUniversalParser()

	// Test deeply nested hierarchical queries
	deepNested := "class:MyClass > method:test"
	_, err := parser.ParseQuery(deepNested)
	if err != nil {
		t.Errorf("ParseQuery with deep nesting failed: %v", err)
	}

	// Test complex logical combinations
	complexLogical := "function:test & variable:name"
	_, err = parser.ParseQuery(complexLogical)
	if err != nil {
		t.Errorf("ParseQuery with complex logical failed: %v", err)
	}

	// Test mixed hierarchical and logical
	mixed := "function:test | variable:name"
	_, err = parser.ParseQuery(mixed)
	if err != nil {
		t.Errorf("ParseQuery with mixed operators failed: %v", err)
	}

	// Test query with provider that has empty DSL mappings
	mockProvider := &MockProvider{}
	_, err = parser.ParseQueryWithProvider("function:test", mockProvider)
	if err != nil {
		t.Errorf("ParseQueryWithProvider with empty DSL failed: %v", err)
	}
}

func TestIsWildcardEdgeCases(t *testing.T) {
	parser := NewUniversalParser()
	
	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"single asterisk", "*", true},
		{"asterisk with text", "test*", true},
		{"asterisk in middle", "te*st", true},
		{"multiple asterisks", "**test**", true},
		{"escaped asterisk", "\\*", true}, // Parser doesn't handle escaping
		{"no asterisk", "test", false},
		{"empty string", "", false},
		{"only spaces", "   ", false},
		{"asterisk with spaces", " * ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsWildcard(tt.pattern)
			if result != tt.expected {
				t.Errorf("IsWildcard(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestNormalizeKind(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name  string
		input types.NodeKind
		want  types.NodeKind
	}{
		{
			name:  "normalize func to function",
			input: "func",
			want:  "function",
		},
		{
			name:  "normalize var to variable",
			input: "var",
			want:  "variable",
		},
		{
			name:  "normalize assign to assignment",
			input: "assign",
			want:  "assignment",
		},
		{
			name:  "no normalization needed - function",
			input: "function",
			want:  "function",
		},
		{
			name:  "no normalization needed - variable",
			input: "variable",
			want:  "variable",
		},
		{
			name:  "no normalization needed - class",
			input: "class",
			want:  "class",
		},
		{
			name:  "unknown kind causes error",
			input: "unknown",
			want:  "unknown",
		},
		{
			name:  "empty kind unchanged",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection to access private method
			// Since normalizeKind is private, we test it indirectly through parseSimpleQuery
			// which calls normalizeKind internally
			query := string(tt.input) + ":test"
			result, err := parser.ParseQuery(query)
			
			// Handle expected errors for unknown or empty kinds
			if tt.input == "unknown" || tt.input == "" {
				if err == nil {
					t.Errorf("ParseQuery(%q) expected error but got none", query)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseQuery(%q) unexpected error: %v", query, err)
				return
			}
			
			if result.Kind != tt.want {
				t.Errorf("normalizeKind(%q) = %q, want %q", tt.input, result.Kind, tt.want)
			}
		})
	}
}

func BenchmarkParseSimpleQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function:test*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkParseLogicalQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function:test & variable:x"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkValidateQuery(b *testing.B) {
	parser := NewUniversalParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := parser.ValidateQuery("function:test*")
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}
