package typescript

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/termfx/morfx/core"
)

// Config implements LanguageConfig for TypeScript
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "typescript"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".ts", ".tsx", ".d.ts"}
}

// GetLanguage returns tree-sitter language for TypeScript
func (c *Config) GetLanguage() *sitter.Language {
	return typescript.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to TypeScript AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{
			"function_declaration",
			"function_expression",
			"arrow_function",
			"method_definition",
			"method_signature",
			"public_field_definition",
		}
	case "class":
		return []string{"class_declaration", "class_expression"}
	case "interface":
		return []string{"interface_declaration"}
	case "type":
		return []string{"type_alias_declaration"}
	case "enum":
		return []string{"enum_declaration"}
	case "variable", "var", "const", "let":
		return []string{"variable_declarator", "lexical_declaration"}
	case "import", "export":
		return []string{"import_statement", "export_statement"}
	case "module", "namespace":
		return []string{"module_declaration", "namespace_declaration"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from TypeScript AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "function_declaration", "class_declaration", "class_expression",
		"interface_declaration", "type_alias_declaration", "enum_declaration",
		"module_declaration", "namespace_declaration":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "method_definition", "method_signature":
		// Try 'key' field first (common in many languages)
		if keyNode := node.ChildByFieldName("key"); keyNode != nil {
			return source[keyNode.StartByte():keyNode.EndByte()]
		}
		// Try property_identifier as direct child (TypeScript specific)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "property_identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "public_field_definition":
		// Class field definition: fieldName = value
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "property_identifier" {
				return source[child.StartByte():child.EndByte()]
			}
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
		// For arrow functions, check if they're assigned to a variable
		parent := node.Parent()
		if parent != nil && parent.Type() == "variable_declarator" {
			if idNode := parent.ChildByFieldName("id"); idNode != nil {
				return source[idNode.StartByte():idNode.EndByte()]
			}
		}
		// Check if it's a property value in an object
		if parent != nil && parent.Type() == "pair" {
			if keyNode := parent.ChildByFieldName("key"); keyNode != nil {
				return source[keyNode.StartByte():keyNode.EndByte()]
			}
		}
		// Check if it's a method in a class (property assignment)
		if parent != nil && parent.Type() == "method_definition" {
			if keyNode := parent.ChildByFieldName("key"); keyNode != nil {
				return source[keyNode.StartByte():keyNode.EndByte()]
			}
		}
		// Check if it's a class field with arrow function (findAll = async () => {})
		if parent != nil && parent.Type() == "assignment_expression" {
			if leftNode := parent.ChildByFieldName("left"); leftNode != nil {
				if leftNode.Type() == "member_expression" {
					if propertyNode := leftNode.ChildByFieldName("property"); propertyNode != nil {
						return source[propertyNode.StartByte():propertyNode.EndByte()]
					}
				} else {
					return source[leftNode.StartByte():leftNode.EndByte()]
				}
			}
		}
		// Check if it's a public_field_definition (class field: findAll = () => {})
		if parent != nil && parent.Type() == "public_field_definition" {
			// Get the first child which should be the property_identifier
			for i := 0; i < int(parent.ChildCount()); i++ {
				child := parent.Child(i)
				if child.Type() == "property_identifier" {
					return source[child.StartByte():child.EndByte()]
				}
			}
		}
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

// IsExported checks if identifier is exported (TS uses various export patterns)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	// In TypeScript, consider PascalCase as exported/public APIs
	return name[0] >= 'A' && name[0] <= 'Z'
}

// ExpandMatches handles destructuring and multi-variable declarations in TypeScript
func (c *Config) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	switch node.Type() {
	case "variable_declaration", "lexical_declaration":
		return c.expandVariableDeclaration(node, source, query)
	case "arrow_function":
		name := c.getArrowFunctionName(node, source)
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
	default:
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

func (c *Config) expandVariableDeclaration(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declarator" {
			matches = append(matches, c.expandVariableDeclarator(child, source, query)...)
		}
	}

	return matches
}

func (c *Config) expandVariableDeclarator(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	idNode := node.ChildByFieldName("id")
	if idNode == nil {
		return matches
	}

	switch idNode.Type() {
	case "identifier":
		name := source[idNode.StartByte():idNode.EndByte()]
		matches = append(matches, core.CodeMatch{
			Node:      idNode,
			Name:      name,
			Type:      query.Type,
			NodeType:  "identifier",
			StartByte: idNode.StartByte(),
			EndByte:   idNode.EndByte(),
			Line:      idNode.StartPoint().Row,
			Column:    idNode.StartPoint().Column,
		})
	case "array_pattern":
		matches = append(matches, c.expandArrayPattern(idNode, source, query)...)
	case "object_pattern":
		matches = append(matches, c.expandObjectPattern(idNode, source, query)...)
	}

	return matches
}

func (c *Config) expandArrayPattern(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

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

func (c *Config) expandObjectPattern(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "shorthand_property_identifier":
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
		case "pair":
			if valueNode := child.ChildByFieldName("value"); valueNode != nil && valueNode.Type() == "identifier" {
				name := source[valueNode.StartByte():valueNode.EndByte()]
				matches = append(matches, core.CodeMatch{
					Node:      valueNode,
					Name:      name,
					Type:      query.Type,
					NodeType:  "identifier",
					StartByte: valueNode.StartByte(),
					EndByte:   valueNode.EndByte(),
					Line:      valueNode.StartPoint().Row,
					Column:    valueNode.StartPoint().Column,
				})
			}
		}
	}

	return matches
}

func (c *Config) getArrowFunctionName(node *sitter.Node, source string) string {
	parent := node.Parent()
	if parent != nil && parent.Type() == "variable_declarator" {
		if idNode := parent.ChildByFieldName("id"); idNode != nil {
			return source[idNode.StartByte():idNode.EndByte()]
		}
	}
	return "anonymous"
}
