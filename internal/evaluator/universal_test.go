package evaluator

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang_sitter "github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// MockLanguageProvider implements the provider.LanguageProvider interface for testing
type MockLanguageProvider struct {
	translateFunc    func(*core.Query) (string, error)
	language         *sitter.Language
	getNodeKindFunc  func(*sitter.Node) core.NodeKind
	getNodeNameFunc  func(*sitter.Node, []byte) string
	parseAttrsFunc   func(*sitter.Node, []byte) map[string]string
	getNodeScopeFunc func(*sitter.Node) core.ScopeType
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

func (m *MockLanguageProvider) TranslateKind(kind core.NodeKind) []provider.NodeMapping {
	// Return proper Go Tree-sitter node types
	switch kind {
	case "function":
		return []provider.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"function_declaration", "method_declaration"},
				Template:  "(function_declaration)",
			},
		}
	case "variable":
		return []provider.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"var_declaration", "short_var_declaration"},
				Template:  "(var_declaration)",
			},
		}
	case "class":
		return []provider.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"type_declaration"},
				Template:  "(type_declaration)",
			},
		}
	default:
		return []provider.NodeMapping{
			{
				Kind:      kind,
				NodeTypes: []string{"function_declaration"},
				Template:  "(function_declaration)",
			},
		}
	}
}

func (m *MockLanguageProvider) TranslateQuery(q *core.Query) (string, error) {
	if m.translateFunc != nil {
		return m.translateFunc(q)
	}
	return "(function_declaration)", nil
}

func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	if m.parseAttrsFunc != nil {
		return m.parseAttrsFunc(node, source)
	}
	return map[string]string{}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	if m.getNodeKindFunc != nil {
		return m.getNodeKindFunc(node)
	}
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
	if m.getNodeNameFunc != nil {
		return m.getNodeNameFunc(node, source)
	}
	// Return the actual node content for realistic testing
	return node.Content(source)
}

func (m *MockLanguageProvider) OptimizeQuery(q *core.Query) *core.Query {
	return q
}

func (m *MockLanguageProvider) EstimateQueryCost(q *core.Query) int {
	return 1
}

func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
	if m.getNodeScopeFunc != nil {
		return m.getNodeScopeFunc(node)
	}
	return "file"
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node {
	return node
}

func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool {
	return false
}

func (m *MockLanguageProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{}, []string{}
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
	// Simple normalization for testing
	switch dslKind {
	case "func":
		return "function"
	case "var":
		return "variable"
	default:
		return core.NodeKind(dslKind)
	}
}

// Additional required methods for provider.LanguageProvider interface
func (m *MockLanguageProvider) OrganizeImports(source []byte) ([]byte, error) {
	return source, nil
}

func (m *MockLanguageProvider) Format(source []byte) ([]byte, error) {
	return source, nil
}

func (m *MockLanguageProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
	return []core.QuickCheckDiagnostic{}
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

func TestEvaluate(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	query := &core.Query{
		Kind:    core.KindFunction,
		Pattern: "test*",
		Raw:     "function test*",
	}

	code := []byte(`
func testFunction() {
	// test function
}
`)

	result, err := evaluator.Evaluate(query, code)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestEvaluateWithDifferentQueries(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
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

	code := []byte(`
func testFunction() {
	var varTest = 42
}
`)

	tests := []struct {
		name    string
		query   *core.Query
		wantErr bool
	}{
		{
			name: "function query",
			query: &core.Query{
				Kind:    core.KindFunction,
				Pattern: "test*",
				Raw:     "function test*",
			},
			wantErr: false,
		},
		{
			name: "variable query",
			query: &core.Query{
				Kind:    core.KindVariable,
				Pattern: "var*",
				Raw:     "variable var*",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.query, code)

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

func TestQueryAttributesAndScopes(t *testing.T) {
	coreQuery := &core.Query{
		Kind:       core.KindFunction,
		Pattern:    "test*",
		Attributes: map[string]string{"visibility": "public"},
		Operator:   "",
		Children:   []core.Query{},
		Scope:      core.ScopeFile,
		Raw:        "function[visibility=public] test*",
	}

	// Test that query structure is preserved correctly
	if string(coreQuery.Kind) != "function" {
		t.Errorf("Expected Kind 'function', got %s", coreQuery.Kind)
	}

	if coreQuery.Pattern != "test*" {
		t.Errorf("Expected Pattern 'test*', got %s", coreQuery.Pattern)
	}

	if len(coreQuery.Attributes) != 1 {
		t.Errorf("Expected 1 attribute, got %d", len(coreQuery.Attributes))
	}

	if coreQuery.Attributes["visibility"] != "public" {
		t.Errorf("Expected visibility 'public', got %s", coreQuery.Attributes["visibility"])
	}
}

func TestResultStructure(t *testing.T) {
	// Test that we can create core.Result correctly
	location := core.Location{
		File:      "test.go",
		StartLine: 1,
		EndLine:   1,
		StartCol:  0,
		EndCol:    10,
	}

	metadata := map[string]any{
		"visibility": "public",
		"type":       "string",
	}

	result := &core.Result{
		Kind:       core.KindFunction,
		Name:       "testFunc",
		Location:   location,
		Content:    "func testFunc() {}",
		Metadata:   metadata,
		ParentKind: core.KindClass,
		ParentName: "TestClass",
		Scope:      core.ScopeFile,
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Kind != core.KindFunction {
		t.Errorf("Expected Kind 'function', got %s", result.Kind)
	}

	if result.Name != "testFunc" {
		t.Errorf("Expected Name 'testFunc', got %s", result.Name)
	}

	if result.Location.File != "test.go" {
		t.Errorf("Expected File 'test.go', got %s", result.Location.File)
	}
}

func TestErrorHandling(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "", fmt.Errorf("translation error")
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	code := []byte(`func test() {}`)

	_, err = evaluator.Evaluate(query, code)
	if err == nil {
		t.Error("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "translation error") {
		t.Errorf("Expected translation error, got: %v", err)
	}
}

func TestNilInputHandling(t *testing.T) {
	mockProvider := &MockLanguageProvider{}
	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	tests := []struct {
		name  string
		query *core.Query
		code  []byte
	}{
		{
			name:  "nil query",
			query: nil,
			code:  []byte("func test() {}"),
		},
		{
			name:  "empty code",
			query: &core.Query{Kind: core.KindFunction, Raw: "function"},
			code:  []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluator.Evaluate(tt.query, tt.code)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func BenchmarkEvaluate(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	code := []byte(`
func testFunction() {
	// test function
}
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEvaluateWithComplexQuery(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration (identifier) @name)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind:       core.KindFunction,
		Pattern:    "test*",
		Attributes: map[string]string{"visibility": "public"},
		Raw:        "function[visibility=public] test*",
	}

	code := []byte(`
func testFunction() {
	var testVar = 42
}
func anotherFunction() {
	return true
}
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// Test comprehensive evaluator functionality
func TestUniversalEvaluator_CreationAndValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider provider.LanguageProvider
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid provider",
			provider: &MockLanguageProvider{},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			wantErr:  true,
			errMsg:   "language provider cannot be nil",
		},
		{
			name:     "provider with nil sitter language",
			provider: &MockLanguageProvider{language: nil},
			wantErr:  true,
			errMsg:   "does not provide a valid Tree-sitter language",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator, err := NewUniversalEvaluator(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if evaluator == nil {
				t.Error("Expected non-nil evaluator")
			}
		})
	}
}

// Test evaluator with different query types and complexities
func TestUniversalEvaluator_QueryTypes(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			switch q.Kind {
			case core.KindFunction:
				return "(function_declaration (identifier) @name)", nil
			case core.KindVariable:
				return "(var_declaration (var_spec (identifier) @name))", nil
			case "logical":
				// Return a query that matches both functions and variables
				return "[(function_declaration (identifier) @name) (var_declaration (var_spec (identifier) @name))]", nil
			default:
				return "(function_declaration)", nil
			}
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Test code with various constructs
	code := []byte(`
		package main
		
		import "fmt"
		
		var globalVar = "test"
		
		func testFunction() {
			var localVar = 42
			fmt.Println("Hello, World!")
		}
		
		func anotherFunction(param string) string {
			return param + " modified"
		}
	`)

	tests := []struct {
		name     string
		query    *core.Query
		wantErr  bool
		validate func(*testing.T, *core.ResultSet)
	}{
		{
			name: "simple function query",
			query: &core.Query{
				Kind:    core.KindFunction,
				Pattern: "*",
				Raw:     "function:*",
			},
			wantErr: false,
			validate: func(t *testing.T, rs *core.ResultSet) {
				if rs == nil {
					t.Error("Expected non-nil result set")
					return
				}
				if rs.TotalMatches == 0 {
					t.Error("Expected at least one match")
				}
			},
		},
		{
			name: "function with pattern",
			query: &core.Query{
				Kind:    core.KindFunction,
				Pattern: "test*",
				Raw:     "function:test*",
			},
			wantErr: false,
			validate: func(t *testing.T, rs *core.ResultSet) {
				if rs == nil {
					t.Error("Expected non-nil result set")
					return
				}
				// Should match testFunction
				if rs.TotalMatches == 0 {
					t.Error("Expected matches for pattern test*")
				}
			},
		},
		{
			name: "variable query",
			query: &core.Query{
				Kind:    core.KindVariable,
				Pattern: "*",
				Raw:     "variable:*",
			},
			wantErr: false,
			validate: func(t *testing.T, rs *core.ResultSet) {
				if rs == nil {
					t.Error("Expected non-nil result set")
					return
				}
				// Should match globalVar and localVar
			},
		},
		{
			name: "logical AND query",
			query: &core.Query{
				Kind:     "logical",
				Operator: "AND",
				Children: []core.Query{
					{Kind: core.KindFunction, Pattern: "*", Raw: "function:*"},
					{Kind: core.KindVariable, Pattern: "*", Raw: "variable:*"},
				},
				Raw: "function:* && variable:*",
			},
			wantErr: false,
			validate: func(t *testing.T, rs *core.ResultSet) {
				if rs == nil {
					t.Error("Expected non-nil result set")
				}
			},
		},
		{
			name: "query with attributes",
			query: &core.Query{
				Kind:       core.KindFunction,
				Pattern:    "*",
				Attributes: map[string]string{"type": "public"},
				Raw:        "function:* public",
			},
			wantErr: false,
			validate: func(t *testing.T, rs *core.ResultSet) {
				if rs == nil {
					t.Error("Expected non-nil result set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.query, code)

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

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Test error conditions and edge cases
func TestUniversalEvaluator_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		provider    *MockLanguageProvider
		query       *core.Query
		source      []byte
		expectedErr string
	}{
		{
			name:        "nil query",
			provider:    &MockLanguageProvider{},
			query:       nil,
			source:      []byte("func test() {}"),
			expectedErr: "query cannot be nil",
		},
		{
			name:        "empty source",
			provider:    &MockLanguageProvider{},
			query:       &core.Query{Kind: core.KindFunction, Raw: "function"},
			source:      []byte{},
			expectedErr: "source code cannot be empty",
		},
		{
			name: "provider translation failure",
			provider: &MockLanguageProvider{
				translateFunc: func(q *core.Query) (string, error) {
					return "", fmt.Errorf("translation failed")
				},
			},
			query:       &core.Query{Kind: core.KindFunction, Raw: "function"},
			source:      []byte("func test() {}"),
			expectedErr: "provider failed to translate query",
		},
		{
			name: "invalid tree-sitter query",
			provider: &MockLanguageProvider{
				translateFunc: func(q *core.Query) (string, error) {
					return "(invalid_query_syntax", nil // Missing closing paren
				},
			},
			query:       &core.Query{Kind: core.KindFunction, Raw: "function"},
			source:      []byte("func test() {}"),
			expectedErr: "failed to create Tree-sitter query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator, err := NewUniversalEvaluator(tt.provider)
			if err != nil {
				// Some test cases expect evaluator creation to fail
				if tt.expectedErr != "" && strings.Contains(err.Error(), tt.expectedErr) {
					return
				}
				t.Fatalf("Unexpected error creating evaluator: %v", err)
			}

			_, err = evaluator.Evaluate(tt.query, tt.source)
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectedErr, err)
			}
		})
	}
}

// Test result creation and metadata
func TestUniversalEvaluator_ResultCreation(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration name: (identifier) @name)", nil
		},
	}

	// Override methods for more detailed testing
	mockProvider.getNodeKindFunc = func(node *sitter.Node) core.NodeKind {
		return core.KindFunction
	}

	mockProvider.getNodeNameFunc = func(node *sitter.Node, source []byte) string {
		return "testFunction"
	}

	mockProvider.parseAttrsFunc = func(node *sitter.Node, source []byte) map[string]string {
		return map[string]string{
			"visibility": "public",
			"type":       "function",
		}
	}

	mockProvider.getNodeScopeFunc = func(node *sitter.Node) core.ScopeType {
		return core.ScopeFile
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind:    core.KindFunction,
		Pattern: "test*",
		Raw:     "function:test*",
	}

	code := []byte(`
		func testFunction() {
			// test function
		}
	`)

	result, err := evaluator.Evaluate(query, code)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Test first result
	r := result.Results[0]
	if r.Kind != core.KindFunction {
		t.Errorf("Expected Kind %s, got %s", core.KindFunction, r.Kind)
	}

	if r.Name != "testFunction" {
		t.Errorf("Expected Name 'testFunction', got '%s'", r.Name)
	}

	if r.Scope != core.ScopeFile {
		t.Errorf("Expected Scope %s, got %s", core.ScopeFile, r.Scope)
	}

	// Check metadata
	if r.Metadata == nil {
		t.Error("Expected non-nil metadata")
	} else {
		if r.Metadata["visibility"] != "public" {
			t.Errorf("Expected visibility 'public', got '%v'", r.Metadata["visibility"])
		}
		if r.Metadata["query_kind"] != string(core.KindFunction) {
			t.Errorf("Expected query_kind metadata to be set")
		}
	}

	// Check location information
	if r.Location.StartLine <= 0 {
		t.Error("Expected positive start line")
	}
	if r.Location.EndLine < r.Location.StartLine {
		t.Error("End line should be >= start line")
	}
}

// Test concurrent access
func TestUniversalEvaluator_ConcurrentAccess(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	code := []byte(`
		func testFunction() {
			// test function
		}
	`)

	// Test concurrent evaluation
	var wg sync.WaitGroup
	numGoroutines := 50
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			_, err := evaluator.Evaluate(query, code)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

// Test evaluator methods
func TestUniversalEvaluator_Methods(t *testing.T) {
	mockProvider := &MockLanguageProvider{}
	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	t.Run("GetProvider", func(t *testing.T) {
		provider := evaluator.GetProvider()
		if provider == nil {
			t.Error("Expected non-nil provider")
		}
		if provider.Lang() != mockProvider.Lang() {
			t.Error("Provider mismatch")
		}
	})

	t.Run("GetLanguage", func(t *testing.T) {
		lang := evaluator.GetLanguage()
		if lang != mockProvider.Lang() {
			t.Errorf("Expected language '%s', got '%s'", mockProvider.Lang(), lang)
		}
	})
}

// Test memory usage and performance characteristics
func TestUniversalEvaluator_MemoryUsage(t *testing.T) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	// Large source code to test memory handling
	var codeBuilder strings.Builder
	codeBuilder.WriteString("package main\n")
	for i := 0; i < 1000; i++ {
		codeBuilder.WriteString(fmt.Sprintf("func function%d() { /* function %d */ }\n", i, i))
	}
	code := []byte(codeBuilder.String())

	result, err := evaluator.Evaluate(query, code)
	if err != nil {
		t.Errorf("Evaluation failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Verify results are reasonable
	if result.TotalMatches != len(result.Results) {
		t.Errorf("TotalMatches (%d) != len(Results) (%d)", result.TotalMatches, len(result.Results))
	}
}

// Enhanced benchmarks with different scenarios
func BenchmarkEvaluator_SmallFile(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	code := []byte(`
		func testFunction() {
			// test function
		}
	`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEvaluator_LargeFile(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind: core.KindFunction,
		Raw:  "function",
	}

	// Generate large Go file
	var codeBuilder strings.Builder
	codeBuilder.WriteString("package main\n")
	for i := 0; i < 500; i++ {
		codeBuilder.WriteString(fmt.Sprintf(`
			func function%d() {
				var x = %d
				if x > 0 {
					return
				}
			}
		`, i, i))
	}
	code := []byte(codeBuilder.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEvaluator_ComplexQuery(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			if q.Kind == "logical" {
				return "[(function_declaration) (var_declaration)]", nil
			}
			return "(function_declaration)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind:     "logical",
		Operator: "AND",
		Children: []core.Query{
			{Kind: core.KindFunction, Pattern: "*", Raw: "function:*"},
			{Kind: core.KindVariable, Pattern: "*", Raw: "variable:*"},
		},
		Raw: "function:* && variable:*",
	}

	code := []byte(`
		package main
		
		var globalVar = "test"
		
		func testFunction() {
			var localVar = 42
			anotherFunction(localVar)
		}
		
		func anotherFunction(param int) {
			var result = param * 2
			fmt.Println(result)
		}
	`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkEvaluator_ResultCreation(b *testing.B) {
	mockProvider := &MockLanguageProvider{
		translateFunc: func(q *core.Query) (string, error) {
			return "(function_declaration (identifier) @name)", nil
		},
	}

	evaluator, err := NewUniversalEvaluator(mockProvider)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}

	query := &core.Query{
		Kind:    core.KindFunction,
		Pattern: "*",
		Raw:     "function:*",
	}

	code := []byte(`
		func testFunction1() {}
		func testFunction2() {}
		func testFunction3() {}
	`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := evaluator.Evaluate(query, code)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
		// Access results to ensure they're created
		_ = len(result.Results)
	}
}
