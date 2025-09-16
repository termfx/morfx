package golang

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
)

func TestExpandVarDeclaration(t *testing.T) {
	config := &Config{}
	source := "package main\nvar a, b int"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find var_declaration: source_file -> var_declaration (skip package_clause and newline)
	var varDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "var_declaration" {
			varDecl = child
			break
		}
	}
	if varDecl == nil || varDecl.Type() != "var_declaration" {
		t.Fatalf("Expected var_declaration, got %s", func() string {
			if varDecl == nil {
				return "<nil>"
			}
			return varDecl.Type()
		}())
	}

	query := core.AgentQuery{Type: "variable"}
	matches := config.expandVarDeclaration(varDecl, source, query)

	if len(matches) < 2 {
		t.Errorf("Expected 2 matches for 'a, b', got %d", len(matches))
	}
}

func TestExpandVarSpec(t *testing.T) {
	config := &Config{}
	source := "package main\nvar a, b, c int"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Navigate: source_file -> var_declaration -> var_spec
	var varDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "var_declaration" {
			varDecl = child
			break
		}
	}
	if varDecl == nil {
		t.Fatal("Could not find var_declaration node")
	}

	var varSpec *sitter.Node
	for i := 0; i < int(varDecl.ChildCount()); i++ {
		child := varDecl.Child(i)
		if child.Type() == "var_spec" {
			varSpec = child
			break
		}
	}

	if varSpec == nil {
		t.Fatal("Could not find var_spec node")
	}

	query := core.AgentQuery{Type: "variable"}
	matches := config.expandVarSpec(varSpec, source, query)

	// var_spec contains identifiers directly, not identifier_list
	expectedCount := 3 // a, b, c
	if len(matches) != expectedCount {
		t.Errorf("Expected %d matches, got %d", expectedCount, len(matches))
	}
}

func TestExpandShortVarDeclaration(t *testing.T) {
	config := &Config{}
	source := "package main\na, b := 1, 2"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find short_var_declaration
	var shortVarDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "short_var_declaration" {
			shortVarDecl = child
			break
		}
	}
	if shortVarDecl == nil || shortVarDecl.Type() != "short_var_declaration" {
		t.Fatalf("Expected short_var_declaration, got %s", func() string {
			if shortVarDecl == nil {
				return "<nil>"
			}
			return shortVarDecl.Type()
		}())
	}

	query := core.AgentQuery{Type: "variable"}
	matches := config.expandShortVarDeclaration(shortVarDecl, source, query)

	// short_var_declaration has expression_list -> identifier nodes
	expectedCount := 2 // a, b
	if len(matches) != expectedCount {
		t.Errorf("Expected %d matches, got %d", expectedCount, len(matches))
	}
}

func TestExpandMatchesWithVarDeclaration(t *testing.T) {
	config := &Config{}
	source := "package main\nvar x, y int"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find var_declaration
	var varDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "var_declaration" {
			varDecl = child
			break
		}
	}
	query := core.AgentQuery{Type: "variable"}

	matches := config.ExpandMatches(varDecl, source, query)

	if len(matches) < 1 {
		t.Error("ExpandMatches should find variable matches")
	}
}

func TestExpandMatchesWithShortVar(t *testing.T) {
	config := &Config{}
	source := "package main\nx, y := 1, 2"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find short_var_declaration
	var shortVarDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "short_var_declaration" {
			shortVarDecl = child
			break
		}
	}
	if shortVarDecl == nil {
		t.Fatal("Could not find short_var_declaration node")
	}

	query := core.AgentQuery{Type: "variable"}
	matches := config.ExpandMatches(shortVarDecl, source, query)

	expectedCount := 2
	if len(matches) != expectedCount {
		t.Errorf("Expected %d matches, got %d", expectedCount, len(matches))
	}
}

func TestExpandMatchesDefaultCase(t *testing.T) {
	config := &Config{}
	source := "package main\nfunc test() {}"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Find function_declaration
	var funcDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "function_declaration" {
			funcDecl = child
			break
		}
	}
	query := core.AgentQuery{Type: "function"}

	matches := config.ExpandMatches(funcDecl, source, query)

	if len(matches) != 1 {
		t.Errorf("Expected 1 match for function, got %d", len(matches))
	}

	if len(matches) > 0 && matches[0].Name != "test" {
		t.Errorf("Expected function name 'test', got '%s'", matches[0].Name)
	}
}

func TestValidateTypeSpecStruct(t *testing.T) {
	config := &Config{}
	source := "package main\ntype User struct { Name string }"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Navigate to type_spec within type_declaration
	var typeDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "type_declaration" {
			typeDecl = child
			break
		}
	}
	if typeDecl == nil {
		t.Fatal("Could not find type_declaration node")
	}

	var typeSpec *sitter.Node
	for i := 0; i < int(typeDecl.ChildCount()); i++ {
		child := typeDecl.Child(i)
		if child.Type() == "type_spec" {
			typeSpec = child
			break
		}
	}

	if typeSpec == nil {
		t.Fatal("Could not find type_spec node")
	}

	if !config.ValidateTypeSpec(typeSpec, source, "struct") {
		t.Error("Should validate struct type_spec")
	}

	if config.ValidateTypeSpec(typeSpec, source, "interface") {
		t.Error("Should not validate struct as interface")
	}
}

func TestValidateTypeSpecInterface(t *testing.T) {
	config := &Config{}
	source := "package main\ntype Reader interface { Read() }"

	parser := sitter.NewParser()
	parser.SetLanguage(config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	defer tree.Close()

	// Navigate to type_spec
	var typeDecl *sitter.Node
	for i := 0; i < int(tree.RootNode().ChildCount()); i++ {
		child := tree.RootNode().Child(i)
		if child.Type() == "type_declaration" {
			typeDecl = child
			break
		}
	}
	if typeDecl == nil {
		t.Fatal("Could not find type_declaration node")
	}
	var typeSpec *sitter.Node
	for i := 0; i < int(typeDecl.ChildCount()); i++ {
		child := typeDecl.Child(i)
		if child.Type() == "type_spec" {
			typeSpec = child
			break
		}
	}

	if typeSpec == nil {
		t.Fatal("Could not find type_spec node")
	}

	if !config.ValidateTypeSpec(typeSpec, source, "interface") {
		t.Error("Should validate interface type_spec")
	}

	if config.ValidateTypeSpec(typeSpec, source, "struct") {
		t.Error("Should not validate interface as struct")
	}
}
