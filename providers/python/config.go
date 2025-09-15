package python

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// Config implements LanguageConfig for Python
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "python"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".py", ".pyw", ".pyi"}
}

// GetLanguage returns tree-sitter language for Python
func (c *Config) GetLanguage() *sitter.Language {
	return python.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to Python AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_definition", "async_function_definition"}
	case "class":
		return []string{"class_definition"}
	case "variable", "var":
		return []string{"assignment", "global_statement", "nonlocal_statement"}
	case "import":
		return []string{"import_statement", "import_from_statement"}
	case "decorator":
		return []string{"decorator"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from Python AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "function_definition", "async_function_definition", "class_definition":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "assignment":
		// Find left side of assignment
		if leftNode := node.ChildByFieldName("left"); leftNode != nil {
			if leftNode.Type() == "identifier" {
				return source[leftNode.StartByte():leftNode.EndByte()]
			}
		}
	case "import_statement":
		// Get first imported module
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "dotted_name" || child.Type() == "identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "import_from_statement":
		// Get module being imported from
		if moduleNode := node.ChildByFieldName("module_name"); moduleNode != nil {
			return source[moduleNode.StartByte():moduleNode.EndByte()]
		}
	case "decorator":
		// Get decorator name
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" || child.Type() == "attribute" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	}

	// Fallback: try to find first identifier child
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return source[child.StartByte():child.EndByte()]
		}
	}

	return ""
}

// IsExported checks if identifier is exported (in Python, typically non-underscore prefixed)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	// In Python, single underscore prefix indicates "internal use"
	// Double underscore indicates "private" (name mangling)
	return !strings.HasPrefix(name, "_")
}
