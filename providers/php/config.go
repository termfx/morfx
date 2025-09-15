package php

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
)

// Config implements LanguageConfig for PHP
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "php"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".php", ".phtml", ".php4", ".php5", ".phps"}
}

// GetLanguage returns tree-sitter language for PHP
func (c *Config) GetLanguage() *sitter.Language {
	return php.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to PHP AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_definition", "method_declaration"}
	case "class":
		return []string{"class_declaration"}
	case "interface":
		return []string{"interface_declaration"}
	case "trait":
		return []string{"trait_declaration"}
	case "variable", "var":
		return []string{"simple_parameter", "property_declaration", "variable_name"}
	case "constant", "const":
		return []string{"const_declaration"}
	case "namespace":
		return []string{"namespace_definition"}
	case "use", "import":
		return []string{"namespace_use_declaration"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from PHP AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "function_definition", "class_declaration", "interface_declaration", "trait_declaration":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "method_declaration":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "property_declaration":
		// Find first property name
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_name" {
				name := source[child.StartByte():child.EndByte()]
				return strings.TrimPrefix(name, "$") // Remove $ prefix
			}
		}
	case "variable_name":
		name := source[node.StartByte():node.EndByte()]
		return strings.TrimPrefix(name, "$") // Remove $ prefix
	case "namespace_definition":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "namespace_use_declaration":
		// Find the used namespace
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "qualified_name" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	}

	// Fallback: try to find first identifier or name field
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "name" {
			return source[child.StartByte():child.EndByte()]
		}
	}

	return ""
}

// IsExported checks if identifier is exported (in PHP, typically public methods/properties)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	// In PHP, consider non-underscore prefixed names as public/exported
	// Private/protected typically start with underscore
	return !strings.HasPrefix(name, "_")
}
