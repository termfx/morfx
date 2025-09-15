package javascript

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

// Config implements LanguageConfig for JavaScript
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "javascript"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".js", ".jsx", ".mjs", ".cjs"}
}

// GetLanguage returns tree-sitter language for JavaScript
func (c *Config) GetLanguage() *sitter.Language {
	return javascript.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to JavaScript AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_declaration", "function_expression", "arrow_function", "method_definition"}
	case "class":
		return []string{"class_declaration", "class_expression"}
	case "variable", "var", "const", "let":
		return []string{"variable_declarator", "lexical_declaration"}
	case "import", "export":
		return []string{"import_statement", "export_statement"}
	case "interface": // For TS in JS files
		return []string{"interface_declaration"}
	case "type": // For TS in JS files
		return []string{"type_alias_declaration"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from JavaScript AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "function_declaration":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "class_declaration", "class_expression":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "method_definition":
		if keyNode := node.ChildByFieldName("key"); keyNode != nil {
			return source[keyNode.StartByte():keyNode.EndByte()]
		}
	case "variable_declarator":
		if idNode := node.ChildByFieldName("id"); idNode != nil {
			return source[idNode.StartByte():idNode.EndByte()]
		}
	case "lexical_declaration":
		// Find first variable declarator
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_declarator" {
				if idNode := child.ChildByFieldName("id"); idNode != nil {
					return source[idNode.StartByte():idNode.EndByte()]
				}
			}
		}
	case "import_statement", "export_statement":
		// Get import/export source
		if sourceNode := node.ChildByFieldName("source"); sourceNode != nil {
			path := source[sourceNode.StartByte():sourceNode.EndByte()]
			return strings.Trim(path, `"'`)
		}
	case "arrow_function", "function_expression":
		return "anonymous"
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

// IsExported checks if identifier is exported (in JS, typically uppercase or starts with capital)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	// In JavaScript, we consider functions/classes starting with capital as "exported/public"
	// or check for common export patterns
	return name[0] >= 'A' && name[0] <= 'Z'
}
