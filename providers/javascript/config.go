package javascript

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"

	"github.com/oxhq/morfx/core"
	base "github.com/oxhq/morfx/providers/base"
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
	if nodes, ok := c.aliasMap()[queryType]; ok {
		return nodes
	}
	return []string{queryType}
}

func (c *Config) NormalizeQueryType(queryType string) string {
	switch strings.TrimSpace(queryType) {
	case "func", "fn":
		return "function"
	case "var":
		return "variable"
	default:
		return strings.TrimSpace(queryType)
	}
}

func (c *Config) aliasMap() map[string][]string {
	return map[string][]string{
		"function":    {"function_declaration", "function_expression", "arrow_function", "method_definition"},
		"func":        {"function_declaration", "function_expression", "arrow_function", "method_definition"},
		"fn":          {"function_declaration", "function_expression", "arrow_function", "method_definition"},
		"method":      {"method_definition"},
		"constructor": {"method_definition"},
		"ctor":        {"method_definition"},
		"class":       {"class_declaration", "class_expression"},
		"property":    {"field_definition"},
		"prop":        {"field_definition"},
		"field":       {"field_definition"},
		"variable":    {"variable_declaration", "lexical_declaration", "variable_declarator"},
		"var":         {"variable_declaration", "lexical_declaration", "variable_declarator"},
		"const":       {"variable_declaration", "lexical_declaration", "variable_declarator"},
		"let":         {"variable_declaration", "lexical_declaration", "variable_declarator"},
		"lambda":      {"arrow_function"},
		"arrow":       {"arrow_function"},
		"array":       {"array", "array_pattern"},
		"object":      {"object", "object_pattern"},
		"import":      {"import_statement"},
		"export":      {"export_statement"},
		"call":        {"call_expression", "new_expression"},
		"assignment":  {"assignment_expression", "augmented_assignment_expression"},
		"assign":      {"assignment_expression", "augmented_assignment_expression"},
		"return":      {"return_statement"},
		"condition":   {"if_statement", "switch_statement"},
		"if":          {"if_statement"},
		"block":       {"statement_block"},
		"loop":        {"for_statement", "for_in_statement", "while_statement", "do_statement"},
		"for":         {"for_statement", "for_in_statement"},
		"interface":   {"interface_declaration"},
		"type":        {"type_alias_declaration"},
		"decorator":   {"decorator"},
		"comment":     {"comment"},
		"comments":    {"comment"},
	}
}

// SupportedQueryTypes returns colloquial query types/aliases for JavaScript
func (c *Config) SupportedQueryTypes() []string {
	m := c.aliasMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
		if nameNode := c.nameFieldNode(node, "name", "key"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "field_definition":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "property_identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "variable_declarator":
		if nameNode := c.nameFieldNode(node, "name", "id"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "variable_declaration", "lexical_declaration":
		// Find first variable declarator
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_declarator" {
				if nameNode := c.nameFieldNode(child, "name", "id"); nameNode != nil {
					return source[nameNode.StartByte():nameNode.EndByte()]
				}
			}
		}
	case "import_statement", "export_statement":
		// Get import/export source
		if sourceNode := node.ChildByFieldName("source"); sourceNode != nil {
			path := source[sourceNode.StartByte():sourceNode.EndByte()]
			return strings.Trim(path, `"'`)
		}
	case "call_expression":
		if function := node.ChildByFieldName("function"); function != nil {
			return source[function.StartByte():function.EndByte()]
		}
	case "new_expression":
		if constructor := node.ChildByFieldName("constructor"); constructor != nil {
			return source[constructor.StartByte():constructor.EndByte()]
		}
	case "assignment_expression", "augmented_assignment_expression":
		if left := node.ChildByFieldName("left"); left != nil {
			return source[left.StartByte():left.EndByte()]
		}
	case "return_statement":
		return "return"
	case "arrow_function", "function_expression":
		// Try to infer name from assignment or context
		return c.getArrowFunctionName(node, source)
	case "comment":
		return c.commentSummary(source[node.StartByte():node.EndByte()])
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

func (c *Config) commentSummary(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "///")
	trimmed = strings.TrimPrefix(trimmed, "//")
	trimmed = strings.TrimPrefix(trimmed, "/*")
	trimmed = strings.TrimPrefix(trimmed, "/**")
	trimmed = strings.TrimSuffix(trimmed, "*/")
	trimmed = strings.TrimPrefix(trimmed, "#")
	trimmed = strings.TrimSpace(trimmed)
	if idx := strings.Index(trimmed, "\n"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
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

// ExpandMatches handles destructuring and multi-variable declarations in JavaScript
func (c *Config) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	switch node.Type() {
	case "variable_declaration", "lexical_declaration":
		return c.expandVariableDeclaration(node, source, query)
	case "variable_declarator":
		return c.expandVariableDeclarator(node, source, query)
	case "arrow_function":
		// Fix arrow function names
		name := c.getArrowFunctionName(node, source)
		return []base.Target{base.NewTarget(node, query.Type, name)}
	case "import_statement":
		return c.expandImportStatement(node, source, query)
	case "export_statement":
		return c.expandExportStatement(node, source, query)
	default:
		// Default single match
		name := c.ExtractNodeName(node, source)
		return []base.Target{base.NewTarget(node, query.Type, name)}
	}
}

func (c *Config) ValidateQueryNode(node *sitter.Node, source, queryType string) bool {
	switch queryType {
	case "constructor", "ctor":
		return node.Type() == "method_definition" && c.ExtractNodeName(node, source) == "constructor"
	default:
		return true
	}
}

func (c *Config) expandVariableDeclaration(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declarator" {
			matches = append(matches, c.expandVariableDeclarator(child, source, query)...)
		}
	}

	return matches
}

func (c *Config) expandVariableDeclarator(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	idNode := c.nameFieldNode(node, "name", "id")
	if idNode == nil {
		return matches
	}

	switch idNode.Type() {
	case "identifier":
		name := source[idNode.StartByte():idNode.EndByte()]
		matches = append(matches, base.NewTarget(idNode, query.Type, name))
	case "array_pattern":
		matches = append(matches, c.expandArrayPattern(idNode, source, query)...)
	case "object_pattern":
		matches = append(matches, c.expandObjectPattern(idNode, source, query)...)
	}

	return matches
}

func (c *Config) expandImportStatement(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target
	// Try to capture each imported binding or namespace
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "import_specifier":
			// import { a as b } from 'x'
			if alias := child.ChildByFieldName("alias"); alias != nil {
				name := source[alias.StartByte():alias.EndByte()]
				matches = append(matches, base.NewTarget(child, query.Type, name))
				continue
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, base.NewTarget(child, query.Type, name))
				continue
			}
		case "namespace_import":
			// import * as ns from 'x'
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, base.NewTarget(child, query.Type, name))
				continue
			}
		}
	}
	if len(matches) == 0 {
		name := c.ExtractNodeName(node, source)
		return []base.Target{base.NewTarget(node, query.Type, name)}
	}
	return matches
}

func (c *Config) expandExportStatement(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target
	// Handle named exports: export { a as b }
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "export_specifier" {
			if alias := child.ChildByFieldName("alias"); alias != nil {
				name := source[alias.StartByte():alias.EndByte()]
				matches = append(matches, base.NewTarget(child, query.Type, name))
				continue
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, base.NewTarget(child, query.Type, name))
				continue
			}
		}
	}
	if len(matches) == 0 {
		name := c.ExtractNodeName(node, source)
		return []base.Target{base.NewTarget(node, query.Type, name)}
	}
	return matches
}

func (c *Config) expandArrayPattern(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, base.NewTarget(child, query.Type, name))
		}
	}

	return matches
}

func (c *Config) expandObjectPattern(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "shorthand_property_identifier":
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, base.NewTarget(child, query.Type, name))
		case "pair":
			if valueNode := child.ChildByFieldName("value"); valueNode != nil && valueNode.Type() == "identifier" {
				name := source[valueNode.StartByte():valueNode.EndByte()]
				matches = append(matches, base.NewTarget(valueNode, query.Type, name))
			}
		}
	}

	return matches
}

func (c *Config) getArrowFunctionName(node *sitter.Node, source string) string {
	parent := node.Parent()
	if parent != nil && parent.Type() == "variable_declarator" {
		if idNode := c.nameFieldNode(parent, "name", "id"); idNode != nil && idNode.Type() == "identifier" {
			return source[idNode.StartByte():idNode.EndByte()]
		}
	}

	if parent != nil && parent.Type() == "assignment_expression" {
		if leftNode := parent.ChildByFieldName("left"); leftNode != nil {
			if leftNode.Type() == "member_expression" {
				if propNode := leftNode.ChildByFieldName("property"); propNode != nil {
					return source[propNode.StartByte():propNode.EndByte()]
				}
			} else if leftNode.Type() == "identifier" {
				return source[leftNode.StartByte():leftNode.EndByte()]
			}
		}
	}

	return "anonymous"
}

func (c *Config) nameFieldNode(node *sitter.Node, fields ...string) *sitter.Node {
	for _, field := range fields {
		if fieldNode := node.ChildByFieldName(field); fieldNode != nil {
			return fieldNode
		}
	}
	return nil
}
