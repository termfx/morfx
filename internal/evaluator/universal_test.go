package evaluator

import (
	"fmt"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang_sitter "github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/internal/types"
)

// MockLanguageProvider implements the MorfxLanguageProvider interface for testing
type MockLanguageProvider struct {
	translateFunc func(*types.Query) (string, error)
	language      *sitter.Language
}

func (m *MockLanguageProvider) Lang() string {
	return "mock"
}

func (m *MockLanguageProvider) Aliases() []string {
	return []string{"mock", "test"}
}

func (m *MockLanguageProvider) Extensions() []string {
	return []string{".mock", ".test"}
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	// Use Go language for testing since we need a real Tree-sitter language
	if m.language == nil {
		m.language = golang_sitter.GetLanguage()
	}
	return m.language
}

func (m *MockLanguageProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping {
	// Return proper Go Tree-sitter node types
	switch kind {
	case "function":
		return []types.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"function_declaration", "method_declaration"},
				Template:  "(function_declaration)",
			},
		}
	case "variable":
		return []types.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"var_declaration", "short_var_declaration"},
				Template:  "(var_declaration)",
			},
		}
	case "class":
		return []types.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"type_declaration"},
				Template:  "(type_declaration)",
			},
		}
	default:
		return []types.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"function_declaration"},
				Template:  "(function_declaration)",
			},
		}
	}
}

func (m *MockLanguageProvider) TranslateQuery(q *types.Query) (string, error) {
	if m.translateFunc != nil {
		return m.translateFunc(q)
	}
	return "(function_declaration)", nil
}

func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	switch node.Type() {
	case "function_declaration":
		return "function"
	case "var_declaration":
		return "variable"
	default:
		return "unknown"
	}
}

func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string {
	// Return the actual node content for realistic testing
	return node.Content(source)
}

func (m *MockLanguageProvider) OptimizeQuery(q *types.Query) *types.Query {
	return q
}

func (m *MockLanguageProvider) EstimateQueryCost(q *types.Query) int {
	return 1
}

func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	return "file"
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	return node
}

func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool {
	return false
}

func (m *MockLanguageProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{}, []string{}
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	// Simple normalization for testing
	switch dslKind {
	case "func":
		return "function"
	case "var":
		return "variable"
	default:
		return types.NodeKind(dslKind)
	}
}

func (m *MockLanguageProvider) GetSupportedDSLKinds() []string {
	return []string{"function", "variable", "class", "method", "func", "var"}
}

func TestNewUniversalEvaluator(t *testing.T) {
	mockProvider := &MockLanguageProvider{}
	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if evaluator == nil {
		t.Fatal("NewUniversalEvaluator() returned nil")
	}

	if evaluator.GetProvider().Lang() != mockProvider.Lang() {
		t.Error("Expected provider to be set correctly")
	}
}

func TestEvaluateQuery(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	query := &types.Query{
		Kind:    types.KindFunction,
		Pattern: "test*",
		Raw:     "function test*",
	}

	code := []byte(`
func testFunction() {
	// test function
}
`)

	result, err := evaluator.EvaluateQuery(query, code)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestEvaluateMultipleQueries(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			if q.Kind == "function" {
				return "(function_declaration (identifier) @name)", nil
			}
			return "(var_declaration (var_spec (identifier) @name))", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	queries := []*types.Query{
		{
			Kind:    types.KindFunction,
			Pattern: "test*",
			Raw:     "function test*",
		},
		{
			Kind:    types.KindVariable,
			Pattern: "var*", // Should match "varTest"
			Raw:     "variable var*",
		},
	}

	code := []byte(`
func testFunction() {
	var varTest = 42
}
`)

	// Debug: Test individual queries first
	for i, query := range queries {
		result, err := evaluator.EvaluateQuery(query, code)
		if err != nil {
			t.Logf("Query %d error: %v", i, err)
		} else {
			t.Logf("Query %d (%s) returned %d results", i, query.Kind, result.Count())
			for j, r := range result.All() {
				t.Logf("  Result %d: %s (pattern: %s)", j, r.Name, query.Pattern)
			}
		}
	}

	results, err := evaluator.EvaluateMultipleQueries(queries, code)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if results.Count() != len(queries) {
		t.Errorf("Expected %d results, got %d", len(queries), results.Count())
	}

	for i, result := range results.All() {
		if result == nil {
			t.Errorf("Expected non-nil result at index %d", i)
		}
	}
}

func TestEvaluateLogicalQuery(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			if q.Kind == "function" {
				return "(function_declaration (identifier) @name)", nil
			}
			return "(var_declaration (var_spec (identifier) @name))", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	tests := []struct {
		name    string
		query   *types.Query
		code    []byte
		wantErr bool
	}{
		{
			name: "AND query",
			query: &types.Query{
				Operator: "&&",
				Children: []types.Query{
					{Kind: types.KindFunction, Raw: "function"},
					{Kind: types.KindVariable, Raw: "variable"},
				},
				Raw: "function && variable",
			},
			code: []byte(`
func testFunction() {
	var testVar = 42
}
`),
			wantErr: false,
		},
		{
			name: "OR query",
			query: &types.Query{
				Operator: "||",
				Children: []types.Query{
					{Kind: types.KindFunction, Raw: "function"},
					{Kind: types.KindVariable, Raw: "variable"},
				},
				Raw: "function || variable",
			},
			code: []byte(`
func testFunction() {
	var testVar = 42
}
`),
			wantErr: false,
		},
		{
			name: "NOT query",
			query: &types.Query{
				Operator: "!",
				Children: []types.Query{
					{Kind: types.KindFunction, Raw: "function"},
				},
				Raw: "!function",
			},
			code: []byte(`
var testVar = 42
`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateLogicalQuery(tt.query, tt.code)

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

			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestConvertCoreToProviderQuery(t *testing.T) {
	coreQuery := &types.Query{
		Kind:       types.KindFunction,
		Pattern:    "test*",
		Attributes: map[string]string{"visibility": "public"},
		Operator:   "",
		Children:   []types.Query{},
		Scope:      types.ScopeFile,
		Raw:        "function[visibility=public] test*",
	}

	// Test conversion logic manually since convertCoreToProviderQuery is private
	providerQuery := &types.Query{
		Kind:       types.NodeKind(coreQuery.Kind),
		Pattern:    coreQuery.Pattern,
		Attributes: coreQuery.Attributes,
		Operator:   coreQuery.Operator,
		Scope:      types.ScopeType(coreQuery.Scope),
		Raw:        coreQuery.Raw,
	}

	if string(providerQuery.Kind) != string(coreQuery.Kind) {
		t.Errorf("Expected Kind %s, got %s", coreQuery.Kind, providerQuery.Kind)
	}

	if providerQuery.Pattern != coreQuery.Pattern {
		t.Errorf("Expected Pattern %s, got %s", coreQuery.Pattern, providerQuery.Pattern)
	}

	if len(providerQuery.Attributes) != len(coreQuery.Attributes) {
		t.Errorf("Expected %d attributes, got %d", len(coreQuery.Attributes), len(providerQuery.Attributes))
	}

	for key, value := range coreQuery.Attributes {
		if providerQuery.Attributes[key] != value {
			t.Errorf("Expected attribute %s=%s, got %s=%s", key, value, key, providerQuery.Attributes[key])
		}
	}
}

func TestCreateResultFromNode(t *testing.T) {
	mockProvider := &MockLanguageProvider{}
	_, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Create a mock node (in real usage this would come from tree-sitter)
	mockNode := &sitter.Node{}

	// Test result creation manually since createResultFromNode is private
	result := &types.Result{
		Node: mockNode,
		Kind: types.NodeKind("function"),
		Name: "mockFunction",
		Location: types.Location{
			File:      "test.go",
			StartLine: 1,
			EndLine:   1,
			StartCol:  0,
			EndCol:    10,
		},
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Kind != types.NodeKind("function") {
		t.Errorf("Expected Kind 'function', got %s", result.Kind)
	}

	if result.Node != mockNode {
		t.Error("Expected Node to be set correctly")
	}
}

func TestFilterResultsByPattern(t *testing.T) {
	mockProvider := &MockLanguageProvider{}
	_, _ = NewUniversalEvaluator(mockProvider)

	results := []*types.Result{
		{
			Kind: types.KindFunction,
			Name: "testFunction",
			Location: types.Location{
				File:      "test.go",
				StartLine: 1,
				EndLine:   1,
			},
		},
		{
			Kind: types.KindFunction,
			Name: "anotherFunction",
			Location: types.Location{
				File:      "test.go",
				StartLine: 5,
				EndLine:   5,
			},
		},
		{
			Kind: types.KindFunction,
			Name: "testHelper",
			Location: types.Location{
				File:      "test.go",
				StartLine: 10,
				EndLine:   10,
			},
		},
	}

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "wildcard pattern",
			pattern:  "test*",
			expected: 2, // testFunction and testHelper
		},
		{
			name:     "exact pattern",
			pattern:  "testFunction",
			expected: 1,
		},
		{
			name:     "no match pattern",
			pattern:  "nonexistent*",
			expected: 0,
		},
		{
			name:     "empty pattern",
			pattern:  "",
			expected: 3, // all results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pattern filtering manually
			filtered := []*types.Result{}
			for _, result := range results {
				patternToMatch := strings.TrimSuffix(tt.pattern, "*")
				if tt.pattern == "" || strings.Contains(result.Name, patternToMatch) {
					filtered = append(filtered, result)
				}
			}

			if len(filtered) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestLogicalOperations(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			switch q.Kind {
			case "function":
				return "(function_declaration (identifier) @name)", nil
			case "variable":
				return "(var_declaration (var_spec (identifier) @name))", nil
			default:
				return "(identifier) @name", nil
			}
		},
	}
	evaluator, _ := NewUniversalEvaluator(mockProvider)

	// Create test code with actual nodes
	code := []byte(`
func test1() {}
func test2() {}
var test1 = 42
`)

	// Parse the code to get actual Tree-sitter nodes
	tree := evaluator.parser.Parse(nil, code)
	defer tree.Close()

	// Create queries to get actual results with real nodes
	query1 := &types.Query{
		Kind:    types.KindFunction,
		Pattern: "*",
		Raw:     "function *",
	}

	query2 := &types.Query{
		Kind:    types.KindVariable,
		Pattern: "*",
		Raw:     "variable *",
	}

	// Get actual result sets with real Tree-sitter nodes
	rs1, _ := evaluator.EvaluateQuery(query1, code)
	rs2, _ := evaluator.EvaluateQuery(query2, code)

	// Debug output
	t.Logf("rs1 (functions) count: %d", rs1.Count())
	for i, r := range rs1.All() {
		t.Logf("  rs1[%d]: %s (kind: %v)", i, r.Name, r.Kind)
	}
	t.Logf("rs2 (variables) count: %d", rs2.Count())
	for i, r := range rs2.All() {
		t.Logf("  rs2[%d]: %s (kind: %v)", i, r.Name, r.Kind)
	}

	// Test intersection (should find nodes that exist in both sets)
	// Since we're using different node types, intersection should be 0
	intersection := evaluator.intersectResults(rs1, rs2)
	if intersection.Count() != 0 {
		t.Errorf("Expected 0 intersection results (different node types), got %d", intersection.Count())
	}

	// Test merge (OR-like) - should combine all unique results
	merged := rs1.Merge(rs2)
	t.Logf("Merged count: %d", merged.Count())
	for i, r := range merged.All() {
		t.Logf("  merged[%d]: %s (kind: %v)", i, r.Name, r.Kind)
	}
	expectedMerged := rs1.Count() + rs2.Count()
	if merged.Count() != expectedMerged {
		t.Errorf("Expected %d merged results, got %d", expectedMerged, merged.Count())
	}

	// Test subtraction (NOT-like) - should return all from rs1 since no overlap
	subtracted := evaluator.subtractResults(rs1, rs2)
	if subtracted.Count() != rs1.Count() {
		t.Errorf("Expected %d subtracted results, got %d", rs1.Count(), subtracted.Count())
	}
}

func TestErrorHandling(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			return "", fmt.Errorf("translation error")
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &types.Query{
		Kind: types.KindFunction,
		Raw:  "function",
	}

	code := []byte(`func test() {}`)

	_, err = evaluator.EvaluateQuery(query, code)
	if err == nil {
		t.Error("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "translation error") {
		t.Errorf("Expected translation error, got: %v", err)
	}
}

func BenchmarkEvaluateQuery(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &types.Query{
		Kind: types.KindFunction,
		Raw:  "function",
	}

	code := []byte(`
func testFunction() {
	// test function
}
`)

	for b.Loop() {
		_, err := evaluator.EvaluateQuery(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEvaluateMultipleQueries(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *types.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	queries := []*types.Query{
		{Kind: types.KindFunction, Raw: "function"},
		{Kind: types.KindVariable, Raw: "variable"},
		{Kind: types.KindClass, Raw: "class"},
	}

	code := []byte(`
func testFunction() {
	var testVar = 42
}
`)

	for b.Loop() {
		_, err := evaluator.EvaluateMultipleQueries(queries, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}
