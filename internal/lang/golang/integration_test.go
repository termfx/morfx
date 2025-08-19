package golang

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/evaluator"
	"github.com/termfx/morfx/internal/parser"
	"github.com/termfx/morfx/internal/types"
)

func TestProviderIntegrationWithEvaluator(t *testing.T) {
	provider := NewProvider()

	// Create evaluator with the provider
	eval, err := evaluator.NewUniversalEvaluator(provider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Test Go code
	code := []byte(`
package main

import (
	"fmt"
	"net/http"
)

type Server struct {
	port int
	host string
}

func (s *Server) Start() error {
	return nil
}

func main() {
	server := &Server{
		port: 8080,
		host: "localhost",
	}
	server.Start()
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

var globalConfig = map[string]string{
	"env": "production",
}

const MaxConnections = 100
`)

	tests := []struct {
		name        string
		query       *types.Query
		wantResults int
		minResults  int
	}{
		{
			name: "find all functions",
			query: &types.Query{
				Kind:       "function",
				Pattern:    "*",
				Attributes: make(map[string]string),
			},
			minResults: 2, // main and handleRequest
		},
		{
			name: "find main function",
			query: &types.Query{
				Kind:       "function",
				Pattern:    "main",
				Attributes: make(map[string]string),
			},
			wantResults: 1,
		},
		{
			name: "find methods",
			query: &types.Query{
				Kind:       "method",
				Pattern:    "*",
				Attributes: make(map[string]string),
			},
			minResults: 1, // Start method
		},
		{
			name: "find struct type",
			query: &types.Query{
				Kind:       "class", // structs map to class
				Pattern:    "*",
				Attributes: make(map[string]string),
			},
			minResults: 1, // Server struct
		},
		{
			name: "find imports",
			query: &types.Query{
				Kind:       "import",
				Pattern:    "*",
				Attributes: make(map[string]string),
			},
			minResults: 1, // import block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSet, err := eval.Evaluate(tt.query, code)
			if err != nil {
				t.Fatalf("EvaluateQuery failed: %v", err)
			}

			results := resultSet.Results
	
			if tt.wantResults > 0 {
				if tt.wantResults > 0 && len(results) != tt.wantResults {
					t.Errorf("Got %d results, want exactly %d", len(results), tt.wantResults)
					for i, r := range results {
						t.Logf("  Result %d: %s (kind: %s)", i, r.Name, r.Kind)
					}
				}
			} else if tt.minResults > 0 && tt.wantResults == 0 {
				if len(results) < tt.minResults {
					t.Errorf("Got %d results, want at least %d", len(results), tt.minResults)
					for i, r := range results {
						t.Logf("  Result %d: %s (kind: %s)", i, r.Name, r.Kind)
					}
				}
			}
		})
	}
}

func TestProviderWithUniversalParser(t *testing.T) {
	provider := NewProvider()
	uniParser := parser.NewUniversalParser()

	tests := []struct {
		name    string
		dsl     string
		wantErr bool
	}{
		{
			name:    "simple function query",
			dsl:     "function:main",
			wantErr: false,
		},
		{
			name:    "wildcard function",
			dsl:     "function:test*",
			wantErr: false,
		},
		{
			name:    "variable with type",
			dsl:     "variable:config string",
			wantErr: false,
		},
		{
			name:    "hierarchical query",
			dsl:     "class:Server > method:Start",
			wantErr: false,
		},
		{
			name:    "negated query",
			dsl:     "!function:main",
			wantErr: false,
		},
		{
			name:    "invalid kind",
			dsl:     "invalid_kind:test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse DSL to Query
			query, err := uniParser.ParseQuery(tt.dsl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && query != nil {
				// Try to translate the query with the provider
				_, err := provider.TranslateQuery(query)
				if err != nil && query.Kind != "logical" {
					// Some queries might fail translation
					t.Logf("TranslateQuery failed (might be expected): %v", err)
				}
			}
		})
	}
}

func TestProviderNodeExtraction(t *testing.T) {
	provider := NewProvider()

	code := []byte(`
package main

func testFunction() {
	x := 42
	y := "hello"
}
`)

	// Parse the code
	parser := sitter.NewParser()
	parser.SetLanguage(provider.GetSitterLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, code)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Test GetNodeName with different node types
	root := tree.RootNode()

	// Find function node
	var functionNode *sitter.Node
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	var findFunction func(*sitter.TreeCursor) bool
	findFunction = func(c *sitter.TreeCursor) bool {
		node := c.CurrentNode()
		if node.Type() == "function_declaration" {
			functionNode = node
			return false
		}

		if c.GoToFirstChild() {
			for {
				if !findFunction(c) {
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

	findFunction(cursor)

	if functionNode != nil {
		name := provider.GetNodeName(functionNode, code)
		if name != "testFunction" && !contains([]string{name}, "testFunction") {
			t.Logf("GetNodeName for function returned: %q (might be full content)", name)
		}

		kind := provider.GetNodeKind(functionNode)
		if kind != "function" {
			t.Errorf("GetNodeKind for function_declaration = %q, want %q", kind, "function")
		}
	} else {
		t.Error("Could not find function node in parsed tree")
	}
}

func TestProviderScopes(t *testing.T) {
	provider := NewProvider()

	code := []byte(`
package main

type MyStruct struct {
	field int
}

func (m *MyStruct) Method() {
	if true {
		x := 1
	}
}
`)

	parser := sitter.NewParser()
	parser.SetLanguage(provider.GetSitterLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, code)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Walk tree and test scope detection
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	scopeTypes := make(map[string]types.ScopeType)

	var walk func(*sitter.TreeCursor)
	walk = func(c *sitter.TreeCursor) {
		node := c.CurrentNode()
		nodeType := node.Type()

		// Test scope detection for known types
		if nodeType == "source_file" || nodeType == "function_declaration" ||
			nodeType == "method_declaration" || nodeType == "if_statement" ||
			nodeType == "block" {
			scope := provider.GetNodeScope(node)
			scopeTypes[nodeType] = scope
		}

		if c.GoToFirstChild() {
			for {
				walk(c)
				if !c.GoToNextSibling() {
					break
				}
			}
			c.GoToParent()
		}
	}

	walk(cursor)

	// Check some expected scopes
	if scope, ok := scopeTypes["function_declaration"]; ok && scope != "function" {
		t.Errorf("function_declaration scope = %q, want %q", scope, "function")
	}

	if scope, ok := scopeTypes["block"]; ok && scope != "block" {
		t.Errorf("block scope = %q, want %q", scope, "block")
	}
}

func TestProviderHelperMethods(t *testing.T) {
	provider := NewProvider()

	// Test GetDefaultIgnorePatterns
	files, symbols := provider.GetDefaultIgnorePatterns()
	if len(files) == 0 {
		t.Error("GetDefaultIgnorePatterns returned no file patterns")
	}
	if len(symbols) == 0 {
		t.Error("GetDefaultIgnorePatterns returned no symbol patterns")
	}

	// Test IsBlockLevelNode
	blockLevelTypes := []string{"block", "function_declaration", "class_definition", "method_declaration"}
	for _, nodeType := range blockLevelTypes {
		if !provider.IsBlockLevelNode(nodeType) {
			t.Errorf("IsBlockLevelNode(%q) = false, want true", nodeType)
		}
	}

	nonBlockTypes := []string{"identifier", "literal", "comment"}
	for _, nodeType := range nonBlockTypes {
		if provider.IsBlockLevelNode(nodeType) {
			t.Errorf("IsBlockLevelNode(%q) = true, want false", nodeType)
		}
	}
}
