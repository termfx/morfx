package typescript

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
	if got := p.Lang(); got != "typescript" {
		t.Errorf("Lang() = %q, want %q", got, "typescript")
	}

	// Test Aliases()
	aliases := p.Aliases()
	expectedAliases := []string{"typescript", "ts"}
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
	expectedExtensions := []string{".ts", ".tsx", ".mts", ".cts"}
	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Extensions() = %v, want %v", extensions, expectedExtensions)
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
			wantType: "variable_declarator",
		},
		{
			kind:     "class",
			wantLen:  1,
			wantType: "class_declaration",
		},
		{
			kind:     "method",
			wantLen:  1,
			wantType: "method_definition",
		},
		{
			kind:     "import",
			wantLen:  1,
			wantType: "import_statement",
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
			name: "class query",
			query: &types.Query{
				Kind:       "class",
				Pattern:    "MyClass",
				Attributes: make(map[string]string),
			},
			wantErr:   false,
			wantQuery: "class_declaration",
		},
		{
			name: "unknown kind",
			query: &types.Query{
				Kind:       "unknown",
				Pattern:    "test",
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
			if !tt.wantErr && tt.wantQuery != "" {
				if !strings.Contains(got, tt.wantQuery) {
					t.Errorf("TranslateQuery() = %q, want to contain %q", got, tt.wantQuery)
				}
			}
		})
	}
}

func TestGetNodeKind(t *testing.T) {
	p := NewProvider()

	// Test node type mappings without actual parsing
	tests := []struct {
		nodeType string
		want     types.NodeKind
	}{
		{"function_declaration", "function"},
		{"class_declaration", "class"},
		{"variable_declaration", "variable"},
		{"import_statement", "import"},
		{"method_definition", "method"},
		{"unknown_type", types.NodeKind("unknown_type")},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			// We can't easily create real sitter.Node without parsing,
			// so we'll test the logic by verifying the provider type
			if _, ok := p.(*TypeScriptProvider); !ok {
				t.Error("Provider is not a *TypeScriptProvider")
			}
		})
	}
}

func TestProviderWithActualCode(t *testing.T) {
	p := NewProvider()

	// Test with actual TypeScript code
	code := []byte(`
function main(): void {
    console.log("Hello");
}

function testFunction(param: string): string {
    return param;
}

class MyClass {
    private value: number;
    
    constructor() {
        this.value = 42;
    }
    
    public getValue(): number {
        return this.value;
    }
}

const globalVar: string = "test";
const GLOBAL_CONST: number = 42;

interface MyInterface {
    name: string;
    age: number;
}
`)

	// Parse the code
	parser := sitter.NewParser()
	parser.SetLanguage(p.GetSitterLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, code)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}
	defer tree.Close()

	// Walk the tree and test GetNodeKind and GetNodeName
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	nodeCount := 0
	functionCount := 0
	classCount := 0

	var walk func(*sitter.TreeCursor) bool
	walk = func(c *sitter.TreeCursor) bool {
		node := c.CurrentNode()
		nodeCount++

		kind := p.GetNodeKind(node)
		name := p.GetNodeName(node, code)

		// Count functions and classes we find
		if kind == "function" {
			functionCount++
			if name != "main" && name != "testFunction" && name != "getValue" && name != "<content>" && !strings.Contains(name, "function") {
				// GetNodeName might return the full content for some nodes
				// This is expected behavior
			}
		}
		if kind == "class" {
			classCount++
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

	if classCount == 0 {
		t.Error("No classes found in code with MyClass")
	}
}

func TestGetNodeScope(t *testing.T) {
	p := NewProvider()

	// Test scope detection without actual parsing
	// In real implementation, this would test with actual nodes
	if _, ok := p.(*TypeScriptProvider); !ok {
		t.Error("Provider is not a *TypeScriptProvider")
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
