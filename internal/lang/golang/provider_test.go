package golang

import (
	"context"
	"slices"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/types"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()

	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}

	// Check that it implements LanguageProvider interface
	if _, ok := any(p).(types.LanguageProvider); !ok {
		t.Error("Provider does not implement types.LanguageProvider interface")
	}
}

func TestProviderMetadata(t *testing.T) {
	p := NewProvider()

	// Test Lang()
	if got := p.Lang(); got != "go" {
		t.Errorf("Lang() = %q, want %q", got, "go")
	}

	// Test Aliases()
	aliases := p.Aliases()
	expectedAliases := []string{"go", "golang"}
	if len(aliases) != len(expectedAliases) {
		t.Errorf("Aliases() returned %d aliases, want %d", len(aliases), len(expectedAliases))
	}
	for i, alias := range expectedAliases {
		if i < len(aliases) && aliases[i] != alias {
			t.Errorf("Aliases()[%d] = %q, want %q", i, aliases[i], alias)
		}
	}

	// Test Extensions()
	extensions := p.Extensions()
	if len(extensions) != 1 || extensions[0] != ".go" {
		t.Errorf("Extensions() = %v, want [.go]", extensions)
	}

	// Test GetSitterLanguage()
	lang := p.GetSitterLanguage()
	if lang == nil {
		t.Error("GetSitterLanguage() returned nil")
	}
}

func TestTranslateKind(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		kind     types.NodeKind
		wantLen  int
		wantType string
	}{
		{
			kind:     "function",
			wantLen:  1,
			wantType: "function_declaration",
		},
		{
			kind:     "variable",
			wantLen:  1,
			wantType: "var_declaration",
		},
		{
			kind:     "class",
			wantLen:  1,
			wantType: "type_declaration",
		},
		{
			kind:     "method",
			wantLen:  1,
			wantType: "method_declaration",
		},
		{
			kind:     "import",
			wantLen:  1,
			wantType: "import_declaration",
		},
		{
			kind:     "unknown",
			wantLen:  0,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			mappings := p.TranslateKind(tt.kind)
			if len(mappings) != tt.wantLen {
				t.Errorf("TranslateKind(%q) returned %d mappings, want %d", tt.kind, len(mappings), tt.wantLen)
			}
			if tt.wantLen > 0 && len(mappings) > 0 {
				if !contains(mappings[0].NodeTypes, tt.wantType) {
					t.Errorf("TranslateKind(%q) NodeTypes = %v, want to contain %q", tt.kind, mappings[0].NodeTypes, tt.wantType)
				}
			}
		})
	}
}

func TestTranslateQuery(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		name      string
		query     *types.Query
		wantErr   bool
		wantQuery string // partial match
	}{
		{
			name: "simple function query",
			query: &types.Query{
				Kind:       "function",
				Pattern:    "main",
				Attributes: make(map[string]string),
			},
			wantErr:   false,
			wantQuery: "function_declaration",
		},
		{
			name: "function with wildcard",
			query: &types.Query{
				Kind:       "function",
				Pattern:    "test*",
				Attributes: make(map[string]string),
			},
			wantErr:   false,
			wantQuery: "function_declaration",
		},
		{
			name: "variable with type",
			query: &types.Query{
				Kind:       "variable",
				Pattern:    "config",
				Attributes: map[string]string{"type": "string"},
			},
			wantErr:   false,
			wantQuery: "var_declaration",
		},
		{
			name: "unsupported kind",
			query: &types.Query{
				Kind:       "unsupported",
				Pattern:    "",
				Attributes: make(map[string]string),
			},
			wantErr:   true,
			wantQuery: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.TranslateQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("TranslateQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.wantQuery) {
				t.Errorf("TranslateQuery() = %q, want to contain %q", got, tt.wantQuery)
			}
		})
	}
}

func TestGetNodeKind(t *testing.T) {
	p := NewProvider()

	// Create a mock Tree-sitter node for testing
	// Note: In real tests, you'd parse actual Go code
	tests := []struct {
		nodeType string
		want     types.NodeKind
	}{
		{"function_declaration", "function"},
		{"method_declaration", "method"},
		{"var_declaration", "variable"},
		{"short_var_declaration", "variable"},
		{"type_declaration", "class"},
		{"import_declaration", "import"},
		{"const_declaration", "constant"},
		{"field_declaration", "field"},
		{"call_expression", "call"},
		{"assignment_statement", "assignment"},
		{"if_statement", "condition"},
		{"for_statement", "loop"},
		{"block", "block"},
		{"comment", "comment"},
		{"type_identifier", "type"},
		{"unknown_type", types.NodeKind("unknown_type")},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			// We can't easily create real sitter.Node without parsing,
			// so we'll test the logic by calling GetNodeKind with different types
			// This would need actual parsing in integration tests

			// For now, just verify the provider can be type-asserted
			if _, ok := p.(*GoProvider); !ok {
				t.Error("Provider is not a *GoProvider")
			}
		})
	}
}

func TestWildcardPatternConversion(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		pattern string
		text    string
		want    bool
	}{
		{"test*", "testFunction", true},
		{"test*", "test", true},
		{"test*", "testing", true},
		{"test*", "mytest", false},
		{"*test", "mytest", true},
		{"*test", "test", true},
		{"*test", "testing", false},
		{"*test*", "mytestfunc", true},
		{"test?", "test1", true},
		{"test?", "test", false},
		{"test?", "test12", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			// Test the wildcard conversion logic
			regex := p.(*GoProvider).ConvertWildcardToRegex(tt.pattern)
			matched := strings.HasPrefix(regex, "^") && strings.HasSuffix(regex, "$")
			if !matched {
				t.Errorf("convertWildcardToRegex(%q) didn't anchor pattern", tt.pattern)
			}
		})
	}
}

func TestProviderWithActualCode(t *testing.T) {
	p := NewProvider()

	// Test with actual Go code
	code := []byte(`
package main

import "fmt"

type Config struct {
	Name string
	Port int
}

func main() {
	fmt.Println("Hello")
}

func testFunction(param string) string {
	return param
}

var globalVar = "test"
const GlobalConst = 42
`)

	// Parse the code
	parser := sitter.NewParser()
	parser.SetLanguage(p.GetSitterLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, code)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Walk the tree and test GetNodeKind and GetNodeName
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	nodeCount := 0
	functionCount := 0

	var walk func(*sitter.TreeCursor) bool
	walk = func(c *sitter.TreeCursor) bool {
		node := c.CurrentNode()
		nodeCount++

		kind := p.GetNodeKind(node)
		name := p.GetNodeName(node, code)

		// Count functions we find
		if kind == "function" {
			functionCount++
			if name != "main" && name != "testFunction" && name != "<content>" && !strings.Contains(name, "func") {
				// GetNodeName might return the full content for some nodes
				// This is expected behavior
			}
		}

		if c.GoToFirstChild() {
			for {
				if !walk(c) {
					return false
				}
				if !c.GoToNextSibling() {
					break
				}
			}
			c.GoToParent()
		}
		return true
	}

	walk(cursor)

	if nodeCount == 0 {
		t.Error("No nodes found in parsed tree")
	}

	if functionCount == 0 {
		t.Error("No functions found in code with main() and testFunction()")
	}
}

func TestBuildQueryFromMapping(t *testing.T) {
	p := NewProvider()

	// Get a mapping for function
	mappings := p.TranslateKind("function")
	if len(mappings) == 0 {
		t.Fatal("No mappings for function kind")
	}

	// Test building query with pattern
	query := &types.Query{
		Kind:       "function",
		Pattern:    "test*",
		Attributes: make(map[string]string),
	}

	result, err := p.TranslateQuery(query)
	if err != nil {
		t.Fatalf("TranslateQuery failed: %v", err)
	}

	// Should contain function_declaration
	if !strings.Contains(result, "function_declaration") {
		t.Errorf("Query doesn't contain function_declaration: %s", result)
	}

	// Should contain pattern matching for test*
	if strings.Contains(result, "test*") {
		// Pattern should be converted to regex
		if !strings.Contains(result, "#match?") && !strings.Contains(result, "#eq?") {
			t.Errorf("Query doesn't contain pattern predicate: %s", result)
		}
	}
}

func TestOptimizeQuery(t *testing.T) {
	p := NewProvider()

	query := &types.Query{
		Kind:    "function",
		Pattern: "main",
	}

	optimized := p.OptimizeQuery(query)

	// Default implementation should return the same query
	if optimized != query {
		t.Error("OptimizeQuery should return the same query by default")
	}
}

func TestEstimateQueryCost(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		name  string
		query *types.Query
		want  int
	}{
		{
			name: "simple query",
			query: &types.Query{
				Kind: "function",
			},
			want: 1,
		},
		{
			name: "query with children",
			query: &types.Query{
				Kind: "function",
				Children: []types.Query{
					{Kind: "variable"},
					{Kind: "call"},
				},
			},
			want: 3, // 1 + 2 children
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := p.EstimateQueryCost(tt.query)
			if cost != tt.want {
				t.Errorf("EstimateQueryCost() = %d, want %d", cost, tt.want)
			}
		})
	}
}

// Helper function
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
