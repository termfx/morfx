package golang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/core"
)

// Config implements LanguageConfig for Go
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "go"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".go", ".mod"}
}

// GetLanguage returns tree-sitter language for Go
func (c *Config) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to Go AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_declaration", "method_declaration"}
	case "struct":
		return []string{"type_spec"}
	case "interface":
		return []string{"type_spec"}
	case "variable", "var":
		return []string{"var_declaration", "short_var_declaration"}
	case "constant", "const":
		return []string{"const_declaration"}
	case "import":
		return []string{"import_declaration"}
	case "type":
		return []string{"type_declaration", "type_spec"}
	case "method":
		return []string{"method_declaration"}
	case "field":
		return []string{"field_declaration"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from Go AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	// Try standard name field first
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}

	// Special handling for specific node types
	switch node.Type() {
	case "type_spec":
		// Check if it's actually a struct or interface
		typeNode := node.ChildByFieldName("type")
		if typeNode != nil && (typeNode.Type() == "struct_type" || typeNode.Type() == "interface_type") {
			if nameNode := node.ChildByFieldName("name"); nameNode != nil {
				return source[nameNode.StartByte():nameNode.EndByte()]
			}
		}
	case "import_declaration":
		// Get import path
		if pathNode := node.ChildByFieldName("path"); pathNode != nil {
			path := source[pathNode.StartByte():pathNode.EndByte()]
			return strings.Trim(path, `"`)
		}
	case "var_declaration", "const_declaration", "short_var_declaration":
		// Variables can have multiple names - get first identifier for now
		// TODO: Provider should create separate matches for each variable
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	}

	// Fallback: try to find first identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return source[child.StartByte():child.EndByte()]
		}
	}

	return ""
}

// IsExported checks if identifier is exported (starts with capital letter in Go)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// ValidateTypeSpec checks if type_spec matches the specific query type
func (c *Config) ValidateTypeSpec(node *sitter.Node, source, queryType string) bool {
	if node.Type() != "type_spec" {
		return true // Not a type_spec, use default validation
	}

	// Find the type definition part
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return false
	}

	switch queryType {
	case "struct":
		return typeNode.Type() == "struct_type"
	case "interface":
		return typeNode.Type() == "interface_type"
	default:
		return true // For other queries like "type", accept any type_spec
	}
}

// ExpandMatches handles multi-variable declarations in Go
func (c *Config) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	switch node.Type() {
	case "var_declaration", "const_declaration":
		return c.expandVarDeclaration(node, source, query)
	case "short_var_declaration":
		return c.expandShortVarDeclaration(node, source, query)
	default:
		// Default single match
		name := c.ExtractNodeName(node, source)
		return []core.CodeMatch{{
			Node:      node,
			Name:      name,
			Type:      query.Type,
			NodeType:  node.Type(),
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
			Line:      node.StartPoint().Row,
			Column:    node.StartPoint().Column,
		}}
	}
}

func (c *Config) expandVarDeclaration(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	// Find var_spec nodes within var_declaration
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "var_spec" {
			matches = append(matches, c.expandVarSpec(child, source, query)...)
		}
	}

	return matches
}

func (c *Config) expandVarSpec(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	// var_spec contains identifiers directly (not in identifier_list)
	// Structure: var_spec -> identifier, ',', identifier, type_identifier
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, core.CodeMatch{
				Node:      child,
				Name:      name,
				Type:      query.Type,
				NodeType:  "identifier",
				StartByte: child.StartByte(),
				EndByte:   child.EndByte(),
				Line:      child.StartPoint().Row,
				Column:    child.StartPoint().Column,
			})
		}
	}

	return matches
}

func (c *Config) expandShortVarDeclaration(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	// short_var_declaration -> expression_list -> identifiers
	// Structure: short_var_declaration -> expression_list, ':=', expression_list
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "expression_list" {
			// First expression_list contains the identifiers
			for j := 0; j < int(child.ChildCount()); j++ {
				identifier := child.Child(j)
				if identifier.Type() == "identifier" {
					name := source[identifier.StartByte():identifier.EndByte()]
					matches = append(matches, core.CodeMatch{
						Node:      identifier,
						Name:      name,
						Type:      query.Type,
						NodeType:  "identifier",
						StartByte: identifier.StartByte(),
						EndByte:   identifier.EndByte(),
						Line:      identifier.StartPoint().Row,
						Column:    identifier.StartPoint().Column,
					})
				}
			}
			// Only process first expression_list (left side of :=)
			break
		}
	}

	return matches
}
