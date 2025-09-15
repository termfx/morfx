package golang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
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
		return []string{"type_spec"} // Need additional check for struct type
	case "interface":
		return []string{"type_spec"} // Need additional check for interface type
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
		// Variables can have multiple names - get first one
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
