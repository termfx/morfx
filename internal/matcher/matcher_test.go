package matcher

import (
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"

	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/types"
)

// MockLanguageProvider implements types.LanguageProvider for testing
type MockLanguageProvider struct {
	language *sitter.Language
}

// Basic metadata
func (m *MockLanguageProvider) Lang() string {
	return "javascript"
}

func (m *MockLanguageProvider) Aliases() []string {
	return []string{"js", "node"}
}

func (m *MockLanguageProvider) Extensions() []string {
	return []string{".js", ".mjs"}
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return m.language
}

// DSL Translation
func (m *MockLanguageProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping {
	return []types.NodeMapping{{
		Kind:      kind,
		NodeTypes: []string{"function_declaration"},
		Template:  "(function_declaration) @target",
	}}
}

func (m *MockLanguageProvider) TranslateQuery(q *types.Query) (string, error) {
	return "(function_declaration) @target", nil
}

// Language-specific DSL support
func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	return types.KindFunction
}

func (m *MockLanguageProvider) GetSupportedDSLKinds() []string {
	return []string{"function", "variable", "class"}
}

// Parsing helpers
func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{"visibility": "public"}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	return types.KindFunction
}

func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string {
	return "test_node"
}

// Language-specific optimizations
func (m *MockLanguageProvider) OptimizeQuery(q *types.Query) *types.Query {
	return q
}

func (m *MockLanguageProvider) EstimateQueryCost(q *types.Query) int {
	return 1
}

// Scope detection
func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	return types.ScopeFile
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	return nil
}

// Code structure helpers
func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool {
	return true
}

func (m *MockLanguageProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{"*.test.js"}, []string{"test_*"}
}

// OrganizeImports organizes import statements in source code
func (m *MockLanguageProvider) OrganizeImports(source []byte) ([]byte, error) {
	return source, nil
}

// Format formats the source code according to language conventions
func (m *MockLanguageProvider) Format(source []byte) ([]byte, error) {
	return source, nil
}

// QuickCheck performs basic syntax and semantic validation
func (m *MockLanguageProvider) QuickCheck(source []byte) []types.QuickCheckDiagnostic {
	return []types.QuickCheckDiagnostic{}
}

// MockCompoundQueryProvider extends MockLanguageProvider with compound query support
type MockCompoundQueryProvider struct {
	*MockLanguageProvider
	supportsCompound bool
	evaluationResult any
	evaluationError  error
	validationError  error
}

func (m *MockCompoundQueryProvider) EvaluateQuery(query string, source []byte) (any, error) {
	return m.evaluationResult, m.evaluationError
}

func (m *MockCompoundQueryProvider) SupportsCompoundQueries() bool {
	return m.supportsCompound
}

func (m *MockCompoundQueryProvider) ValidateQuery(query string) error {
	return m.validationError
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		provider    types.LanguageProvider
		wantErr     bool
		errorSubstr string
	}{
		{
			name:     "valid query",
			pattern:  "(function_declaration) @target",
			provider: &MockLanguageProvider{language: javascript.GetLanguage()},
			wantErr:  false,
		},
		{
			name:        "invalid query syntax",
			pattern:     "invalid query syntax",
			provider:    &MockLanguageProvider{language: javascript.GetLanguage()},
			wantErr:     true,
			errorSubstr: "", // tree-sitter will provide specific error
		},
		{
			name:     "empty pattern",
			pattern:  "",
			provider: &MockLanguageProvider{language: javascript.GetLanguage()},
			wantErr:  false, // tree-sitter accepts empty patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &model.Config{
				Pattern:  tt.pattern,
				Provider: tt.provider,
			}

			matcher, err := New(cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error, got nil")
				}
				if tt.errorSubstr != "" && err != nil {
					if !contains(err.Error(), tt.errorSubstr) {
						t.Errorf("New() error = %v, want substring %v", err, tt.errorSubstr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error = %v", err)
				return
			}

			if matcher == nil {
				t.Errorf("New() returned nil matcher")
				return
			}

			if matcher.lang == nil {
				t.Errorf("New() matcher.lang is nil")
			}

			if matcher.query == nil {
				t.Errorf("New() matcher.query is nil")
			}
		})
	}
}

func TestMatcher_Find(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		source      string
		expectedLen int
		wantErr     bool
		errorSubstr string
	}{
		{
			name:        "find function declarations",
			pattern:     "(function_declaration) @target",
			source:      "function test() { return 42; }\nfunction another() { return 'hello'; }",
			expectedLen: 2,
			wantErr:     false,
		},
		{
			name:        "find variable declarations",
			pattern:     "(variable_declaration) @target",
			source:      "var x = 1;\nlet y = 2;\nconst z = 3;",
			expectedLen: 1, // Mock provider returns single result
			wantErr:     false,
		},
		{
			name:        "no matches",
			pattern:     "(class_declaration) @target",
			source:      "function test() { return 42; }",
			expectedLen: 0,
			wantErr:     false,
		},
		{
			name:        "empty source",
			pattern:     "(function_declaration) @target",
			source:      "",
			expectedLen: 0,
			wantErr:     false,
		},
		{
			name:        "invalid source syntax",
			pattern:     "(function_declaration) @target",
			source:      "function test( { invalid syntax",
			expectedLen: 0, // tree-sitter handles invalid syntax gracefully
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &model.Config{
				Pattern:  tt.pattern,
				Provider: &MockLanguageProvider{language: javascript.GetLanguage()},
			}

			matcher, err := New(cfg)
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			nodes, err := matcher.Find([]byte(tt.source))
			if tt.wantErr {
				if err == nil {
					t.Errorf("Find() expected error, got nil")
				}
				if tt.errorSubstr != "" && err != nil {
					if !contains(err.Error(), tt.errorSubstr) {
						t.Errorf("Find() error = %v, want substring %v", err, tt.errorSubstr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Find() unexpected error = %v", err)
				return
			}

			if len(nodes) != tt.expectedLen {
				t.Errorf("Find() got %d nodes, want %d", len(nodes), tt.expectedLen)
			}

			// Verify all returned nodes are not nil
			for i, node := range nodes {
				if node == nil {
					t.Errorf("Find() node[%d] is nil", i)
				}
			}
		})
	}
}

func TestNewCompound(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		provider types.LanguageProvider
		wantErr  bool
	}{
		{
			name:     "valid compound matcher",
			pattern:  "(function_declaration) @target",
			provider: &MockLanguageProvider{language: javascript.GetLanguage()},
			wantErr:  false,
		},
		{
			name:    "compound matcher with compound provider",
			pattern: "(function_declaration) @target",
			provider: &MockCompoundQueryProvider{
				MockLanguageProvider: &MockLanguageProvider{language: javascript.GetLanguage()},
				supportsCompound:     true,
			},
			wantErr: false,
		},
		{
			name:     "invalid pattern",
			pattern:  "invalid syntax",
			provider: &MockLanguageProvider{language: javascript.GetLanguage()},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &model.Config{
				Pattern:  tt.pattern,
				Provider: tt.provider,
			}

			matcher, err := NewCompound(cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCompound() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewCompound() unexpected error = %v", err)
				return
			}

			if matcher == nil {
				t.Errorf("NewCompound() returned nil matcher")
				return
			}

			if matcher.Matcher == nil {
				t.Errorf("NewCompound() matcher.Matcher is nil")
			}

			if matcher.provider == nil {
				t.Errorf("NewCompound() matcher.provider is nil")
			}
		})
	}
}

func TestCompoundMatcher_FindCompound(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		source           string
		provider         types.LanguageProvider
		expectedCompound bool
		wantErr          bool
		errorSubstr      string
	}{
		{
			name:   "compound provider success",
			query:  "function AND public",
			source: "function test() { return 42; }",
			provider: &MockCompoundQueryProvider{
				MockLanguageProvider: &MockLanguageProvider{language: javascript.GetLanguage()},
				supportsCompound:     true,
				evaluationResult:     map[string]interface{}{"matches": 1},
			},
			expectedCompound: true,
			wantErr:          false,
		},
		{
			name:   "compound provider error",
			query:  "invalid compound query",
			source: "function test() { return 42; }",
			provider: &MockCompoundQueryProvider{
				MockLanguageProvider: &MockLanguageProvider{language: javascript.GetLanguage()},
				supportsCompound:     true,
				evaluationError:      fmt.Errorf("evaluation failed"),
			},
			expectedCompound: false,
			wantErr:          true,
			errorSubstr:      "compound query evaluation failed",
		},
		{
			name:             "fallback to simple query",
			query:            "(function_declaration) @target",
			source:           "function test() { return 42; }",
			provider:         &MockLanguageProvider{language: javascript.GetLanguage()},
			expectedCompound: false,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &model.Config{
				Pattern:  "(function_declaration) @target", // Base pattern for matcher creation
				Provider: tt.provider,
			}

			matcher, err := NewCompound(cfg)
			if err != nil {
				t.Fatalf("NewCompound() unexpected error = %v", err)
			}

			result, err := matcher.FindCompound(tt.query, []byte(tt.source))
			if tt.wantErr {
				if err == nil {
					t.Errorf("FindCompound() expected error, got nil")
				}
				if tt.errorSubstr != "" && err != nil {
					if !contains(err.Error(), tt.errorSubstr) {
						t.Errorf("FindCompound() error = %v, want substring %v", err, tt.errorSubstr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("FindCompound() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("FindCompound() returned nil result")
				return
			}

			if result.IsCompound() != tt.expectedCompound {
				t.Errorf("FindCompound() IsCompound() = %v, want %v", result.IsCompound(), tt.expectedCompound)
			}

			if result.Query != tt.query {
				t.Errorf("FindCompound() Query = %v, want %v", result.Query, tt.query)
			}

			if string(result.Source) != tt.source {
				t.Errorf("FindCompound() Source = %v, want %v", string(result.Source), tt.source)
			}
		})
	}
}

func TestCompoundResult_GetNodes(t *testing.T) {
	tests := []struct {
		name          string
		result        *CompoundResult
		expectedNodes int
		expectedNil   bool
	}{
		{
			name: "simple result with nodes",
			result: &CompoundResult{
				Nodes: []*sitter.Node{{}, {}}, // Mock nodes
			},
			expectedNodes: 2,
			expectedNil:   false,
		},
		{
			name: "compound result without nodes",
			result: &CompoundResult{
				ResultSet: map[string]interface{}{"matches": 1},
			},
			expectedNodes: 0,
			expectedNil:   true,
		},
		{
			name:          "empty result",
			result:        &CompoundResult{},
			expectedNodes: 0,
			expectedNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := tt.result.GetNodes()
			if tt.expectedNil {
				if nodes != nil {
					t.Errorf("GetNodes() expected nil, got %v", nodes)
				}
			} else {
				if len(nodes) != tt.expectedNodes {
					t.Errorf("GetNodes() got %d nodes, want %d", len(nodes), tt.expectedNodes)
				}
			}
		})
	}
}

func TestCompoundResult_IsCompound(t *testing.T) {
	tests := []struct {
		name     string
		result   *CompoundResult
		expected bool
	}{
		{
			name: "compound result",
			result: &CompoundResult{
				ResultSet: map[string]interface{}{"matches": 1},
			},
			expected: true,
		},
		{
			name: "simple result",
			result: &CompoundResult{
				Nodes: []*sitter.Node{{}},
			},
			expected: false,
		},
		{
			name:     "empty result",
			result:   &CompoundResult{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsCompound(); got != tt.expected {
				t.Errorf("IsCompound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
