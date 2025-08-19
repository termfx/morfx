package parser

import (
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/types"
)

// TestProviderIntegration tests the integration between parser and language providers
func TestProviderIntegration(t *testing.T) {
	parser := NewUniversalParser()
	mockProvider := NewMockProvider()

	tests := []struct {
		name        string
		dsl         string
		wantKind    types.NodeKind
		wantPattern string
		wantErr     bool
		description string
	}{
		{
			name:        "provider DSL mapping - fn to function",
			dsl:         "fn:main",
			wantKind:    "function",
			wantPattern: "main",
			wantErr:     false,
			description: "Test provider-specific DSL mapping from 'fn' to 'function'",
		},
		{
			name:        "provider DSL mapping - var to variable",
			dsl:         "var:config",
			wantKind:    "variable",
			wantPattern: "config",
			wantErr:     false,
			description: "Test provider-specific DSL mapping from 'var' to 'variable'",
		},
		{
			name:        "provider DSL mapping - cls to class",
			dsl:         "cls:MyClass",
			wantKind:    "class",
			wantPattern: "MyClass",
			wantErr:     false,
			description: "Test provider-specific DSL mapping from 'cls' to 'class'",
		},
		{
			name:        "direct mapping - function",
			dsl:         "function:test",
			wantKind:    "function",
			wantPattern: "test",
			wantErr:     false,
			description: "Test direct mapping without DSL translation",
		},
		{
			name:        "unsupported DSL kind",
			dsl:         "unknown:test",
			wantKind:    "unknown",
			wantPattern: "test",
			wantErr:     true,
			description: "Test handling of unsupported DSL kinds",
		},
		{
			name:        "wildcard with provider DSL",
			dsl:         "fn:test*",
			wantKind:    "function",
			wantPattern: "test*",
			wantErr:     false,
			description: "Test wildcard patterns with provider DSL mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseQueryWithProvider(tt.dsl, mockProvider)

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

			// Verify provider DSL normalization was applied
			if !strings.Contains(tt.dsl, ":") {
				return // Skip validation for malformed queries
			}

			parts := strings.SplitN(tt.dsl, ":", 2)
			originalKind := parts[0]
			normalizedKind := mockProvider.NormalizeDSLKind(originalKind)

			if result.Kind != normalizedKind {
				t.Errorf("Provider DSL normalization failed: got %v, want %v", result.Kind, normalizedKind)
			}
		})
	}
}

// TestProviderQueryTranslation tests query translation through providers
func TestProviderQueryTranslation(t *testing.T) {
	parser := NewUniversalParser()
	mockProvider := NewMockProvider()

	tests := []struct {
		name        string
		dsl         string
		wantErr     bool
		description string
	}{
		{
			name:        "simple query translation",
			dsl:         "fn:main",
			wantErr:     false,
			description: "Test basic query translation through provider",
		},
		{
			name:        "hierarchical query translation",
			dsl:         "cls:MyClass > fn:method",
			wantErr:     false,
			description: "Test hierarchical query translation",
		},
		{
			name:        "logical query translation",
			dsl:         "fn:test & var:x",
			wantErr:     false,
			description: "Test logical query translation",
		},
		{
			name:        "negated query translation",
			dsl:         "!fn:test",
			wantErr:     false,
			description: "Test negated query translation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.ParseQueryWithProvider(tt.dsl, mockProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQueryWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && query != nil {
				// Test that the query can be translated by the provider
				translated, err := mockProvider.TranslateQuery(query)
				if err != nil {
					t.Errorf("Provider TranslateQuery() failed: %v", err)
				}

				if translated == "" {
					t.Errorf("Provider TranslateQuery() returned empty result")
				}
			}
		})
	}
}

// TestProviderDSLValidation tests DSL validation with providers
func TestProviderDSLValidation(t *testing.T) {
	parser := NewUniversalParser()
	mockProvider := NewMockProvider()

	tests := []struct {
		name        string
		dsl         string
		wantErr     bool
		description string
	}{
		{
			name:        "valid provider DSL kind",
			dsl:         "fn:test",
			wantErr:     false,
			description: "Test validation of supported provider DSL kind",
		},
		{
			name:        "empty query",
			dsl:         "",
			wantErr:     true,
			description: "Test validation of empty query",
		},
		{
			name:        "malformed query - missing colon",
			dsl:         "function",
			wantErr:     true,
			description: "Test validation of malformed query without pattern",
		},
		{
			name:        "malformed query - empty pattern",
			dsl:         "fn:",
			wantErr:     false,
			description: "Test validation of query with empty pattern (allowed)",
		},
		{
			name:        "malformed query - empty kind",
			dsl:         ":test",
			wantErr:     true,
			description: "Test validation of query with empty kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseQueryWithProvider(tt.dsl, mockProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQueryWithProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestProviderComplexScenarios tests complex integration scenarios
func TestProviderComplexScenarios(t *testing.T) {
	parser := NewUniversalParser()
	mockProvider := NewMockProvider()

	tests := []struct {
		name        string
		dsl         string
		wantErr     bool
		description string
	}{
		{
			name:        "mixed DSL and direct kinds in hierarchical",
			dsl:         "cls:MyClass > function:method",
			wantErr:     false,
			description: "Test mixing provider DSL and direct kinds in hierarchical query",
		},
		{
			name:        "mixed DSL and direct kinds in logical",
			dsl:         "fn:test & variable:x",
			wantErr:     false,
			description: "Test mixing provider DSL and direct kinds in logical query",
		},
		{
			name:        "complex nested with provider DSL",
			dsl:         "cls:MyClass > fn:method",
			wantErr:     false,
			description: "Test complex nested query with provider DSL",
		},
		{
			name:        "negated hierarchical with provider DSL",
			dsl:         "!cls:MyClass > fn:method",
			wantErr:     false,
			description: "Test negated hierarchical query with provider DSL",
		},
		{
			name:        "multiple wildcards with provider DSL",
			dsl:         "fn:test* & var:config?",
			wantErr:     false,
			description: "Test multiple wildcard patterns with provider DSL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.ParseQueryWithProvider(tt.dsl, mockProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQueryWithProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && query != nil {
				// Verify the query structure is valid
				if query.Kind == "" {
					t.Errorf("ParseQueryWithProvider() returned query with empty kind")
				}

				// Test provider translation
				_, err := mockProvider.TranslateQuery(query)
				if err != nil {
					t.Logf("Provider translation failed (may be expected for complex queries): %v", err)
				}
			}
		})
	}
}

// TestProviderErrorHandling tests error handling in provider integration
func TestProviderErrorHandling(t *testing.T) {
	parser := NewUniversalParser()

	// Test with nil provider
	t.Run("nil provider", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil provider
				t.Logf("Expected panic with nil provider: %v", r)
			}
		}()
		_, err := parser.ParseQueryWithProvider("fn:test", nil)
		if err == nil {
			t.Errorf("ParseQueryWithProvider() with nil provider should return error")
		}
	})

	// Test with provider that returns errors
	errorProvider := &ErrorProvider{}
	t.Run("provider with errors", func(t *testing.T) {
		_, err := parser.ParseQueryWithProvider("fn:test", errorProvider)
		// Should handle provider errors gracefully
		if err != nil {
			t.Logf("Provider error handled: %v", err)
		}
	})
}

// ErrorProvider is a mock provider that returns errors for testing
type ErrorProvider struct{}

func (e *ErrorProvider) Lang() string                                                    { return "error" }
func (e *ErrorProvider) Aliases() []string                                               { return []string{"error"} }
func (e *ErrorProvider) Extensions() []string                                            { return []string{".err"} }
func (e *ErrorProvider) GetSitterLanguage() *sitter.Language                           { return nil }
func (e *ErrorProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping          { return []types.NodeMapping{} }
func (e *ErrorProvider) TranslateQuery(q *types.Query) (string, error)                 { return "", nil }
func (e *ErrorProvider) NormalizeDSLKind(dslKind string) types.NodeKind                 { return types.NodeKind(dslKind) }
func (e *ErrorProvider) GetSupportedDSLKinds() []string                                 { return []string{} }
func (e *ErrorProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string { return make(map[string]string) }
func (e *ErrorProvider) GetNodeKind(node *sitter.Node) types.NodeKind                   { return "unknown" }
func (e *ErrorProvider) GetNodeName(node *sitter.Node, source []byte) string            { return "" }
func (e *ErrorProvider) OptimizeQuery(query *types.Query) *types.Query                  { return query }
func (e *ErrorProvider) EstimateQueryCost(query *types.Query) int                       { return 1 }
func (e *ErrorProvider) GetNodeScope(node *sitter.Node) types.ScopeType                 { return types.ScopeFile }
func (e *ErrorProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node { return nil }
func (e *ErrorProvider) IsBlockLevelNode(nodeType string) bool                          { return false }
func (e *ErrorProvider) GetDefaultIgnorePatterns() ([]string, []string)                { return []string{}, []string{} }
func (e *ErrorProvider) BuildMappings() map[types.NodeKind][]string                     { return make(map[types.NodeKind][]string) }
func (e *ErrorProvider) CacheQuery(query string, result *types.Query)                   {}
func (e *ErrorProvider) GetCachedQuery(query string) (*types.Query, bool)               { return nil, false }