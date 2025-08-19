package provider

import (
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"

	"github.com/termfx/morfx/internal/core"
)

// MockLanguageProvider implements the LanguageProvider interface for testing
type MockLanguageProvider struct {
	BaseProvider
	lang       string
	aliases    []string
	extensions []string
	language   *sitter.Language
	// Control fields for testing
	shouldFailTranslation bool
	translationError      error
}

func NewMockLanguageProvider(lang string) *MockLanguageProvider {
	provider := &MockLanguageProvider{
		lang:       lang,
		aliases:    []string{lang + "_alias"},
		extensions: []string{"." + lang},
		language:   javascript.GetLanguage(), // Use JavaScript for consistent testing
	}

	// Initialize base provider mappings
	mappings := []NodeMapping{
		{
			Kind:        core.KindFunction,
			NodeTypes:   []string{"function_declaration"},
			NameCapture: "@name",
			Template:    "(function_declaration name: (identifier) @name)",
			Priority:    100,
		},
		{
			Kind:        core.KindVariable,
			NodeTypes:   []string{"variable_declaration"},
			NameCapture: "@name",
			Template:    "(variable_declaration (variable_declarator name: (identifier) @name))",
			Priority:    50,
		},
	}
	provider.BuildMappings(mappings)

	return provider
}

func (m *MockLanguageProvider) Lang() string {
	return m.lang
}

func (m *MockLanguageProvider) Aliases() []string {
	return m.aliases
}

func (m *MockLanguageProvider) Extensions() []string {
	return m.extensions
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return m.language
}

func (m *MockLanguageProvider) TranslateQuery(q *core.Query) (string, error) {
	if m.shouldFailTranslation {
		return "", m.translationError
	}

	mappings := m.TranslateKind(q.Kind)
	if len(mappings) == 0 {
		return "", fmt.Errorf("no mappings found for kind %s", q.Kind)
	}

	return m.BuildQueryFromMapping(mappings[0], q), nil
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	switch node.Type() {
	case "function_declaration":
		return core.KindFunction
	case "variable_declaration":
		return core.KindVariable
	case "identifier":
		return core.KindParameter
	default:
		return "unknown"
	}
}

func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string {
	return node.Content(source)
}

func TestBaseProvider_BuildMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings []NodeMapping
		want     map[core.NodeKind][]NodeMapping
	}{
		{
			name: "single mapping",
			mappings: []NodeMapping{
				{
					Kind:      core.KindFunction,
					NodeTypes: []string{"function_declaration"},
					Template:  "(function_declaration) @target",
				},
			},
			want: map[core.NodeKind][]NodeMapping{
				core.KindFunction: {
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"function_declaration"},
						Template:  "(function_declaration) @target",
					},
				},
			},
		},
		{
			name: "multiple mappings same kind",
			mappings: []NodeMapping{
				{
					Kind:      core.KindFunction,
					NodeTypes: []string{"function_declaration"},
					Template:  "(function_declaration) @target",
				},
				{
					Kind:      core.KindFunction,
					NodeTypes: []string{"method_declaration"},
					Template:  "(method_declaration) @target",
				},
			},
			want: map[core.NodeKind][]NodeMapping{
				core.KindFunction: {
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"function_declaration"},
						Template:  "(function_declaration) @target",
					},
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"method_declaration"},
						Template:  "(method_declaration) @target",
					},
				},
			},
		},
		{
			name: "multiple mappings different kinds",
			mappings: []NodeMapping{
				{
					Kind:      core.KindFunction,
					NodeTypes: []string{"function_declaration"},
					Template:  "(function_declaration) @target",
				},
				{
					Kind:      core.KindVariable,
					NodeTypes: []string{"variable_declaration"},
					Template:  "(variable_declaration) @target",
				},
			},
			want: map[core.NodeKind][]NodeMapping{
				core.KindFunction: {
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"function_declaration"},
						Template:  "(function_declaration) @target",
					},
				},
				core.KindVariable: {
					{
						Kind:      core.KindVariable,
						NodeTypes: []string{"variable_declaration"},
						Template:  "(variable_declaration) @target",
					},
				},
			},
		},
		{
			name:     "empty mappings",
			mappings: []NodeMapping{},
			want:     map[core.NodeKind][]NodeMapping{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{}
			b.BuildMappings(tt.mappings)

			if !reflect.DeepEqual(b.mappings, tt.want) {
				t.Errorf("BuildMappings() = %v, want %v", b.mappings, tt.want)
			}
		})
	}
}

func TestBaseProvider_CacheOperations(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		query string
	}{
		{
			name:  "cache single query",
			key:   "test_key",
			query: "(function_declaration) @target",
		},
		{
			name:  "cache with empty key",
			key:   "",
			query: "(variable_declaration) @target",
		},
		{
			name:  "cache with empty query",
			key:   "empty_query",
			query: "",
		},
		{
			name:  "cache with special characters",
			key:   "test_key:special!@#",
			query: "(function_declaration name: (identifier) @name)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{}

			// Test caching
			b.CacheQuery(tt.key, tt.query)

			if b.cache == nil {
				t.Error("CacheQuery() cache should be initialized")
				return
			}

			if got, exists := b.cache[tt.key]; !exists || got != tt.query {
				t.Errorf("CacheQuery() cached value = %v, want %v", got, tt.query)
			}

			// Test retrieval
			gotQuery, gotExists := b.GetCachedQuery(tt.key)
			if gotQuery != tt.query {
				t.Errorf("GetCachedQuery() query = %v, want %v", gotQuery, tt.query)
			}
			if !gotExists {
				t.Errorf("GetCachedQuery() exists = %v, want %v", gotExists, true)
			}
		})
	}
}

func TestBaseProvider_GetCachedQuery(t *testing.T) {
	tests := []struct {
		name       string
		setupCache map[string]string
		key        string
		wantQuery  string
		wantExists bool
	}{
		{
			name: "get existing query",
			setupCache: map[string]string{
				"test_key": "(function_declaration) @target",
			},
			key:        "test_key",
			wantQuery:  "(function_declaration) @target",
			wantExists: true,
		},
		{
			name: "get non-existing query",
			setupCache: map[string]string{
				"test_key": "(function_declaration) @target",
			},
			key:        "missing_key",
			wantQuery:  "",
			wantExists: false,
		},
		{
			name:       "get from empty cache",
			setupCache: nil,
			key:        "any_key",
			wantQuery:  "",
			wantExists: false,
		},
		{
			name: "get empty string query",
			setupCache: map[string]string{
				"empty": "",
			},
			key:        "empty",
			wantQuery:  "",
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{cache: tt.setupCache}
			gotQuery, gotExists := b.GetCachedQuery(tt.key)

			if gotQuery != tt.wantQuery {
				t.Errorf("GetCachedQuery() query = %v, want %v", gotQuery, tt.wantQuery)
			}
			if gotExists != tt.wantExists {
				t.Errorf("GetCachedQuery() exists = %v, want %v", gotExists, tt.wantExists)
			}
		})
	}
}

func TestBaseProvider_TranslateKind(t *testing.T) {
	tests := []struct {
		name     string
		mappings map[core.NodeKind][]NodeMapping
		kind     core.NodeKind
		want     []NodeMapping
	}{
		{
			name: "existing kind",
			mappings: map[core.NodeKind][]NodeMapping{
				core.KindFunction: {
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"function_declaration"},
						Template:  "(function_declaration) @target",
					},
				},
			},
			kind: core.KindFunction,
			want: []NodeMapping{
				{
					Kind:      core.KindFunction,
					NodeTypes: []string{"function_declaration"},
					Template:  "(function_declaration) @target",
				},
			},
		},
		{
			name: "non-existing kind",
			mappings: map[core.NodeKind][]NodeMapping{
				core.KindFunction: {
					{
						Kind:      core.KindFunction,
						NodeTypes: []string{"function_declaration"},
						Template:  "(function_declaration) @target",
					},
				},
			},
			kind: core.KindVariable,
			want: nil,
		},
		{
			name:     "nil mappings",
			mappings: nil,
			kind:     core.KindFunction,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{mappings: tt.mappings}
			got := b.TranslateKind(tt.kind)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TranslateKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseProvider_ParseAttributes(t *testing.T) {
	// Create a simple JavaScript AST for testing
	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	source := []byte("function test() { return 42; }")
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()
	functionNode := root.Child(0) // function_declaration node

	tests := []struct {
		name   string
		node   *sitter.Node
		source []byte
	}{
		{
			name:   "function node attributes",
			node:   functionNode,
			source: source,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{}
			got := b.ParseAttributes(tt.node, tt.source)

			// Check that basic attributes are present
			if got["type"] == "" {
				t.Error("Expected type attribute to be set")
			}
			if got["start_line"] == "" {
				t.Error("Expected start_line attribute to be set")
			}
			if got["end_line"] == "" {
				t.Error("Expected end_line attribute to be set")
			}
			if got["start_col"] == "" {
				t.Error("Expected start_col attribute to be set")
			}
			if got["end_col"] == "" {
				t.Error("Expected end_col attribute to be set")
			}
		})
	}
}

func TestBaseProvider_ScopeDetection(t *testing.T) {
	// Create mock nodes with different types
	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	tests := []struct {
		name      string
		source    string
		nodeIndex int
		wantScope core.ScopeType
	}{
		{
			name:      "function declaration",
			source:    "function test() { return 42; }",
			nodeIndex: 0,
			wantScope: core.ScopeFunction,
		},
		{
			name:      "program root",
			source:    "let x = 1;",
			nodeIndex: -1, // Use root node
			wantScope: core.ScopeFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.source))
			defer tree.Close()

			root := tree.RootNode()
			var targetNode *sitter.Node
			if tt.nodeIndex == -1 {
				targetNode = root
			} else if tt.nodeIndex < int(root.ChildCount()) {
				targetNode = root.Child(tt.nodeIndex)
			} else {
				targetNode = root
			}

			b := &BaseProvider{}
			got := b.GetNodeScope(targetNode)

			if got != tt.wantScope {
				t.Errorf("GetNodeScope() = %v, want %v for node type %s", got, tt.wantScope, targetNode.Type())
			}
		})
	}
}

func TestBaseProvider_FindEnclosingScope(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	source := []byte("function outer() { function inner() { let x = 1; } }")
	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()

	// Navigate to nodes more carefully to avoid nil pointers
	if root.ChildCount() == 0 {
		t.Skip("No child nodes found in parsed tree")
		return
	}

	outerFunc := root.Child(0)
	if outerFunc == nil {
		t.Skip("Outer function not found")
		return
	}

	tests := []struct {
		name      string
		node      *sitter.Node
		scope     core.ScopeType
		wantFound bool
	}{
		{
			name:      "find function scope from root",
			node:      root,
			scope:     core.ScopeFunction,
			wantFound: false, // Root has no parent with function scope
		},
		{
			name:      "find file scope from function",
			node:      outerFunc,
			scope:     core.ScopeFile,
			wantFound: true, // Function's parent should be file scope
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseProvider{}
			got := b.FindEnclosingScope(tt.node, tt.scope)

			if tt.wantFound && got == nil {
				t.Errorf("FindEnclosingScope() expected to find scope %v, got nil", tt.scope)
			}
			if !tt.wantFound && got != nil {
				t.Errorf("FindEnclosingScope() expected nil, got %v", got)
			}
		})
	}
}

func TestBaseProvider_HelperMethods(t *testing.T) {
	b := &BaseProvider{}

	t.Run("IsBlockLevelNode", func(t *testing.T) {
		tests := []struct {
			nodeType string
			want     bool
		}{
			{"block", true},
			{"statement_block", true},
			{"function_declaration", true},
			{"class_definition", true},
			{"method_declaration", true},
			{"if_statement", true},
			{"for_statement", true},
			{"while_statement", true},
			{"variable_declaration", false},
			{"identifier", false},
			{"", false},
		}

		for _, tt := range tests {
			got := b.IsBlockLevelNode(tt.nodeType)
			if got != tt.want {
				t.Errorf("IsBlockLevelNode(%s) = %v, want %v", tt.nodeType, got, tt.want)
			}
		}
	})

	t.Run("GetDefaultIgnorePatterns", func(t *testing.T) {
		files, symbols := b.GetDefaultIgnorePatterns()

		if len(files) == 0 {
			t.Error("Expected non-empty file ignore patterns")
		}
		if len(symbols) == 0 {
			t.Error("Expected non-empty symbol ignore patterns")
		}

		// Check for some expected patterns
		expectedFiles := []string{"*_test.*", "vendor/*", "node_modules/*"}
		for _, expected := range expectedFiles {
			found := slices.Contains(files, expected)
			if !found {
				t.Errorf("Expected file pattern %s not found", expected)
			}
		}
	})

	t.Run("NormalizeDSLKind", func(t *testing.T) {
		tests := []struct {
			input string
			want  core.NodeKind
		}{
			{"function", core.NodeKind("function")},
			{"variable", core.NodeKind("variable")},
			{"custom", core.NodeKind("custom")},
			{"", core.NodeKind("")},
		}

		for _, tt := range tests {
			got := b.NormalizeDSLKind(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeDSLKind(%s) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})
}

func TestBaseProvider_QueryOptimization(t *testing.T) {
	b := &BaseProvider{}

	t.Run("OptimizeQuery", func(t *testing.T) {
		tests := []struct {
			name  string
			query *core.Query
			want  *core.Query
		}{
			{
				name: "simple query optimization",
				query: &core.Query{
					Kind:    core.KindFunction,
					Pattern: "test*",
				},
				want: &core.Query{
					Kind:    core.KindFunction,
					Pattern: "test*",
				},
			},
			{
				name:  "nil query",
				query: nil,
				want:  nil,
			},
		}

		for _, tt := range tests {
			got := b.OptimizeQuery(tt.query)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OptimizeQuery() = %v, want %v", got, tt.want)
			}
		}
	})

	t.Run("EstimateQueryCost", func(t *testing.T) {
		tests := []struct {
			name  string
			query *core.Query
			want  int
		}{
			{
				name: "simple query",
				query: &core.Query{
					Kind:    core.KindFunction,
					Pattern: "test*",
				},
				want: 1,
			},
			{
				name: "query with children",
				query: &core.Query{
					Kind:    core.KindFunction,
					Pattern: "test*",
					Children: []core.Query{
						{Kind: core.KindVariable, Pattern: "x"},
						{Kind: core.KindClass, Pattern: "Test"},
					},
				},
				want: 3, // 1 + 1 + 1
			},
			{
				name: "nested children",
				query: &core.Query{
					Kind:    core.KindFunction,
					Pattern: "test*",
					Children: []core.Query{
						{
							Kind:    core.KindVariable,
							Pattern: "x",
							Children: []core.Query{
								{Kind: core.KindClass, Pattern: "Test"},
							},
						},
					},
				},
				want: 3, // 1 + (1 + 1)
			},
		}

		for _, tt := range tests {
			got := b.EstimateQueryCost(tt.query)
			if got != tt.want {
				t.Errorf("EstimateQueryCost() = %v, want %v", got, tt.want)
			}
		}
	})
}

func TestBaseProvider_WildcardRegexConversion(t *testing.T) {
	b := &BaseProvider{}

	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "single asterisk",
			pattern:  "*",
			expected: "^.*$",
		},
		{
			name:     "single question mark",
			pattern:  "?",
			expected: "^.$",
		},
		{
			name:     "pattern with asterisk",
			pattern:  "test*",
			expected: "^test.*$",
		},
		{
			name:     "pattern with question mark",
			pattern:  "test?",
			expected: "^test.$",
		},
		{
			name:     "pattern with both wildcards",
			pattern:  "te*st?",
			expected: "^te.*st.$",
		},
		{
			name:     "pattern with regex special chars",
			pattern:  "test.function+",
			expected: "^test\\.function\\+$",
		},
		{
			name:     "empty pattern",
			pattern:  "",
			expected: "^$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.ConvertWildcardToRegex(tt.pattern)
			if got != tt.expected {
				t.Errorf("ConvertWildcardToRegex(%s) = %s, want %s", tt.pattern, got, tt.expected)
			}
		})
	}
}

func TestBaseProvider_QueryFromMapping(t *testing.T) {
	b := &BaseProvider{}

	tests := []struct {
		name    string
		mapping NodeMapping
		query   *core.Query
		want    string
	}{
		{
			name: "simple template",
			mapping: NodeMapping{
				Kind:        core.KindFunction,
				Template:    "(function_declaration) @target",
				NameCapture: "@name",
			},
			query: &core.Query{
				Kind:    core.KindFunction,
				Pattern: "*",
			},
			want: "(function_declaration) @target",
		},
		{
			name: "template with pattern constraint",
			mapping: NodeMapping{
				Kind:        core.KindFunction,
				Template:    "(function_declaration %s) @target",
				NameCapture: "@name",
			},
			query: &core.Query{
				Kind:    core.KindFunction,
				Pattern: "test*",
			},
			want: "(function_declaration (#match? @name \"^test.*$\")) @target",
		},
		{
			name: "template with type constraint",
			mapping: NodeMapping{
				Kind:        core.KindVariable,
				Template:    "(variable_declaration %s) @target",
				NameCapture: "@name",
				TypeCapture: "@type",
			},
			query: &core.Query{
				Kind:       core.KindVariable,
				Pattern:    "*",
				Attributes: map[string]string{"type": "string"},
			},
			want: "(variable_declaration (#match? @type \"^string$\")) @target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.BuildQueryFromMapping(tt.mapping, tt.query)
			if got != tt.want {
				t.Errorf("BuildQueryFromMapping() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMockLanguageProvider_Interface(t *testing.T) {
	provider := NewMockLanguageProvider("test")

	// Test that it implements the interface
	var _ LanguageProvider = provider

	// Test basic methods
	if provider.Lang() != "test" {
		t.Errorf("Lang() = %s, want test", provider.Lang())
	}

	aliases := provider.Aliases()
	if len(aliases) == 0 {
		t.Error("Expected non-empty aliases")
	}

	extensions := provider.Extensions()
	if len(extensions) == 0 {
		t.Error("Expected non-empty extensions")
	}

	if provider.GetSitterLanguage() == nil {
		t.Error("Expected non-nil sitter language")
	}
}

func TestMockLanguageProvider_TranslateQuery(t *testing.T) {
	provider := NewMockLanguageProvider("test")

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
			},
			wantErr: false,
		},
		{
			name: "variable query",
			query: &core.Query{
				Kind:    core.KindVariable,
				Pattern: "x",
			},
			wantErr: false,
		},
		{
			name: "unknown kind",
			query: &core.Query{
				Kind:    "unknown",
				Pattern: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.TranslateQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("TranslateQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockLanguageProvider_FailureScenarios(t *testing.T) {
	provider := NewMockLanguageProvider("test")
	provider.shouldFailTranslation = true
	provider.translationError = fmt.Errorf("translation failed")

	query := &core.Query{
		Kind:    core.KindFunction,
		Pattern: "test*",
	}

	_, err := provider.TranslateQuery(query)
	if err == nil {
		t.Error("Expected translation to fail")
	}

	if err.Error() != "translation failed" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestProviderThreadSafety(t *testing.T) {
	provider := NewMockLanguageProvider("concurrent_test")

	// Test concurrent access to cache operations
	t.Run("concurrent cache operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				key := fmt.Sprintf("key_%d", id)
				query := fmt.Sprintf("(function_declaration_%d) @target", id)

				// Cache a query
				provider.CacheQuery(key, query)

				// Retrieve the query
				retrievedQuery, exists := provider.GetCachedQuery(key)
				if !exists {
					t.Errorf("Query for key %s should exist", key)
					return
				}
				if retrievedQuery != query {
					t.Errorf("Retrieved query %s != original query %s", retrievedQuery, query)
				}
			}(i)
		}

		wg.Wait()
	})

	// Test concurrent access to mapping operations
	t.Run("concurrent mapping access", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Access mappings for different kinds
				_ = provider.TranslateKind(core.KindFunction)
				_ = provider.TranslateKind(core.KindVariable)
				_ = provider.TranslateKind(core.KindClass)
			}()
		}

		wg.Wait()
	})

	// Test concurrent query translation
	t.Run("concurrent query translation", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50

		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				query := &core.Query{
					Kind:    core.KindFunction,
					Pattern: fmt.Sprintf("test_%d", id),
				}

				_, err := provider.TranslateQuery(query)
				if err != nil {
					t.Errorf("TranslateQuery failed: %v", err)
				}
			}(i)
		}

		wg.Wait()
	})
}

// Benchmark tests
func BenchmarkBaseProvider_CacheOperations(b *testing.B) {
	provider := &BaseProvider{}

	b.Run("cache_set", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			key := fmt.Sprintf("key_%d", i)
			query := fmt.Sprintf("(function_declaration_%d) @target", i)
			provider.CacheQuery(key, query)
		}
	})

	b.Run("cache_get", func(b *testing.B) {
		// Pre-populate cache
		for i := range 1000 {
			key := fmt.Sprintf("key_%d", i)
			query := fmt.Sprintf("(function_declaration_%d) @target", i)
			provider.CacheQuery(key, query)
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			key := fmt.Sprintf("key_%d", i%1000)
			_, _ = provider.GetCachedQuery(key)
		}
	})
}

func BenchmarkBaseProvider_TranslateKind(b *testing.B) {
	provider := &BaseProvider{}
	mappings := []NodeMapping{
		{
			Kind:      core.KindFunction,
			NodeTypes: []string{"function_declaration"},
			Template:  "(function_declaration) @target",
		},
		{
			Kind:      core.KindVariable,
			NodeTypes: []string{"variable_declaration"},
			Template:  "(variable_declaration) @target",
		},
	}
	provider.BuildMappings(mappings)

	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			_ = provider.TranslateKind(core.KindFunction)
		} else {
			_ = provider.TranslateKind(core.KindVariable)
		}
	}
}

func BenchmarkMockProvider_TranslateQuery(b *testing.B) {
	provider := NewMockLanguageProvider("benchmark")
	query := &core.Query{
		Kind:    core.KindFunction,
		Pattern: "test*",
	}

	for b.Loop() {
		_, err := provider.TranslateQuery(query)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBaseProvider_WildcardRegex(b *testing.B) {
	provider := &BaseProvider{}
	patterns := []string{"*", "test*", "test?", "*.go", "func_*_test"}

	for i := 0; b.Loop(); i++ {
		pattern := patterns[i%len(patterns)]
		_ = provider.ConvertWildcardToRegex(pattern)
	}
}
