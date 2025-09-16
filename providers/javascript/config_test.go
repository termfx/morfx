package javascript

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
)

func TestExpandVariableDeclaration(t *testing.T) {
	config := &Config{}
	source := "let a = 1, b = 2;"
	query := core.AgentQuery{Type: "variable"}

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find variable_declaration
	var varDecl *sitter.Node
	var findVarDecl func(*sitter.Node)
	findVarDecl = func(node *sitter.Node) {
		if node.Type() == "variable_declaration" {
			varDecl = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findVarDecl(node.Child(i))
		}
	}
	findVarDecl(tree.RootNode())

	if varDecl == nil {
		t.Skip("Could not find variable_declaration")
	}

	matches := config.expandVariableDeclaration(varDecl, source, query)

	if len(matches) < 2 {
		t.Errorf("Expected 2 matches for a,b, got %d", len(matches))
	}
}

func TestGetArrowFunctionName(t *testing.T) {
	config := &Config{}
	source := "const test = () => {};"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	var arrowFunction *sitter.Node
	var findArrowFunction func(*sitter.Node)
	findArrowFunction = func(node *sitter.Node) {
		if node.Type() == "arrow_function" {
			arrowFunction = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findArrowFunction(node.Child(i))
		}
	}
	findArrowFunction(tree.RootNode())

	if arrowFunction == nil {
		t.Skip("Could not find arrow_function")
	}

	name := config.getArrowFunctionName(arrowFunction, source)
	if name != "anonymous" {
		t.Errorf("Expected name 'anonymous', got '%s'", name)
	}
}
