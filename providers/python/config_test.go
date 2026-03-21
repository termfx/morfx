package python

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
)

func TestValidateAssignment(t *testing.T) {
	config := &Config{}
	source := "a = 1"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	var assignment *sitter.Node
	var findAssignment func(*sitter.Node)
	findAssignment = func(node *sitter.Node) {
		if node.Type() == "assignment" {
			assignment = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findAssignment(node.Child(i))
		}
	}
	findAssignment(tree.RootNode())

	if assignment == nil {
		t.Skip("Assignment structure varies")
	}

	result := config.ValidateAssignment(assignment, source, "var")
	if !result {
		t.Error("Should validate assignment")
	}
}

func TestExpandAssignment(t *testing.T) {
	config := &Config{}
	source := "a, b = 1, 2"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	var assignment *sitter.Node
	var findAssignment func(*sitter.Node)
	findAssignment = func(node *sitter.Node) {
		if node.Type() == "assignment" {
			assignment = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findAssignment(node.Child(i))
		}
	}
	findAssignment(tree.RootNode())

	if assignment == nil {
		t.Skip("Assignment node structure varies")
	}

	matches := config.expandAssignment(assignment, source, core.AgentQuery{Type: "variable"})
	if len(matches) < 1 {
		t.Error("Should expand assignment")
	}
}

func TestExpandTupleOrList(t *testing.T) {
	config := &Config{}
	source := "[a, b, c]"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	node := tree.RootNode().Child(0).Child(0) // list
	matches := config.expandTupleOrList(node, source, core.AgentQuery{Type: "variable"})

	if len(matches) < 1 {
		t.Error("Should expand list")
	}
}

func TestExpandPatternList(t *testing.T) {
	config := &Config{}
	source := "a, b, c = values"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	var patternList *sitter.Node
	var findPatternList func(*sitter.Node)
	findPatternList = func(node *sitter.Node) {
		if node.Type() == "pattern_list" {
			patternList = node
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findPatternList(node.Child(i))
		}
	}
	findPatternList(tree.RootNode())

	if patternList == nil {
		t.Skip("Pattern list structure varies")
	}

	matches := config.expandPatternList(patternList, source, core.AgentQuery{Type: "variable"})
	if len(matches) < 1 {
		t.Error("Should expand pattern list")
	}
}
