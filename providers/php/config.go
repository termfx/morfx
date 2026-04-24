package php

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"

	"github.com/oxhq/morfx/core"
	base "github.com/oxhq/morfx/providers/base"
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

// SupportedQueryTypes returns colloquial query types/aliases for PHP
func (c *Config) SupportedQueryTypes() []string {
	m := c.aliasMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// MapQueryTypeToNodeTypes maps query types to PHP AST node types
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
		"function":      {"function_definition", "method_declaration", "anonymous_function_creation_expression", "arrow_function"},
		"func":          {"function_definition", "method_declaration", "anonymous_function_creation_expression", "arrow_function"},
		"fn":            {"function_definition", "method_declaration", "anonymous_function_creation_expression", "arrow_function"},
		"closure":       {"anonymous_function_creation_expression"},
		"method":        {"method_declaration"},
		"constructor":   {"method_declaration"},
		"ctor":          {"method_declaration"},
		"class":         {"class_declaration"},
		"interface":     {"interface_declaration"},
		"trait":         {"trait_declaration"},
		"variable":      {"assignment_expression", "simple_parameter", "property_declaration", "variable_name"},
		"var":           {"assignment_expression", "simple_parameter", "property_declaration", "variable_name"},
		"assignment":    {"assignment_expression"},
		"assign":        {"assignment_expression"},
		"property":      {"property_declaration"},
		"field":         {"property_declaration"},
		"constant":      {"const_declaration", "class_constant_declaration"},
		"const":         {"const_declaration", "class_constant_declaration"},
		"namespace":     {"namespace_definition"},
		"use":           {"namespace_use_declaration"},
		"import":        {"namespace_use_declaration"},
		"include":       {"include_expression"},
		"include_once":  {"include_once_expression"},
		"require":       {"require_expression"},
		"require_once":  {"require_once_expression"},
		"call":          {"function_call_expression", "member_call_expression", "scoped_call_expression"},
		"return":        {"return_statement"},
		"condition":     {"if_statement", "switch_statement"},
		"if":            {"if_statement"},
		"block":         {"compound_statement", "declaration_list"},
		"loop":          {"for_statement", "foreach_statement", "while_statement", "do_statement"},
		"for":           {"for_statement", "foreach_statement"},
		"foreach":       {"foreach_statement"},
		"enum":          {"enum_declaration"},
		"array":         {"array_creation_expression"},
		"array_element": {"array_element_initializer"},
		"array_item":    {"array_element_initializer"},
		"element":       {"array_element_initializer"},
		"dict":          {"array_creation_expression"},
		"hash":          {"array_creation_expression"},
		"map":           {"array_creation_expression"},
		"object":        {"array_creation_expression"},
		"list":          {"list_literal"},
		"destructure":   {"list_literal"},
		"comment":       {"comment"},
		"comments":      {"comment"},
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
	case "arrow_function", "anonymous_function_creation_expression":
		// If assigned to a variable, use the variable name
		if parent := node.Parent(); parent != nil && parent.Type() == "assignment_expression" {
			if left := parent.ChildByFieldName("left"); left != nil {
				if left.Type() == "variable_name" {
					name := source[left.StartByte():left.EndByte()]
					return strings.TrimPrefix(name, "$")
				}
			}
		}
		return "anonymous"
	case "property_declaration":
		// Find first property name
		for _, child := range c.propertyVariableNodes(node) {
			name := source[child.StartByte():child.EndByte()]
			return strings.TrimPrefix(name, "$") // Remove $ prefix
		}
	case "variable_name":
		name := source[node.StartByte():node.EndByte()]
		return strings.TrimPrefix(name, "$") // Remove $ prefix
	case "assignment_expression":
		// Extract variable name from left side
		if left := c.variableTargetNode(node); left != nil {
			name := source[left.StartByte():left.EndByte()]
			return strings.TrimPrefix(name, "$")
		}
	case "simple_parameter":
		if nameNode := c.variableTargetNode(node); nameNode != nil {
			name := source[nameNode.StartByte():nameNode.EndByte()]
			return strings.TrimPrefix(name, "$")
		}
	case "namespace_definition":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "namespace_use_declaration":
		// For single-clause uses, attempt to extract the namespace or alias
		// Multi-clause/group uses are expanded in ExpandMatches
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "qualified_name" {
				return source[child.StartByte():child.EndByte()]
			}
			if child.Type() == "namespace_use_clause" {
				if alias := child.ChildByFieldName("alias"); alias != nil {
					return source[alias.StartByte():alias.EndByte()]
				}
				if nameNode := child.ChildByFieldName("name"); nameNode != nil {
					return source[nameNode.StartByte():nameNode.EndByte()]
				}
			}
		}
	case "const_declaration", "class_constant_declaration":
		// Try to find the declared constant name
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "name" {
				return source[child.StartByte():child.EndByte()]
			}
			// Some grammars wrap const entries; try to find nested name fields
			if inner := child.ChildByFieldName("name"); inner != nil {
				return source[inner.StartByte():inner.EndByte()]
			}
		}
	case "array_element_initializer":
		// Try to use the key or value to derive a name-like label
		if key := node.ChildByFieldName("key"); key != nil {
			return strings.TrimSpace(source[key.StartByte():key.EndByte()])
		}
		if val := node.ChildByFieldName("value"); val != nil {
			return strings.TrimSpace(source[val.StartByte():val.EndByte()])
		}
	case "function_call_expression", "member_call_expression", "scoped_call_expression":
		if function := node.ChildByFieldName("function"); function != nil {
			return strings.TrimPrefix(source[function.StartByte():function.EndByte()], "$")
		}
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return strings.TrimPrefix(source[nameNode.StartByte():nameNode.EndByte()], "$")
		}
	case "return_statement":
		return "return"
	case "comment":
		return c.commentSummary(source[node.StartByte():node.EndByte()])
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

func (c *Config) commentSummary(raw string) string {
	trimmed := strings.TrimSpace(raw)
	// Strip common PHP comment prefixes
	trimmed = strings.TrimPrefix(trimmed, "///")
	trimmed = strings.TrimPrefix(trimmed, "//")
	trimmed = strings.TrimPrefix(trimmed, "#")
	trimmed = strings.TrimPrefix(trimmed, "/*")
	trimmed = strings.TrimPrefix(trimmed, "/**")
	trimmed = strings.TrimSuffix(trimmed, "*/")
	trimmed = strings.TrimSpace(trimmed)
	if idx := strings.Index(trimmed, "\n"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
	return trimmed
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

// ValidateVisibility checks PHP visibility modifiers for better export detection
func (c *Config) ValidateVisibility(node *sitter.Node, source string) bool {
	parent := node.Parent()
	for parent != nil {
		if parent.Type() == "property_declaration" || parent.Type() == "method_declaration" {
			// Check for explicit visibility modifiers
			for i := 0; i < int(parent.ChildCount()); i++ {
				child := parent.Child(i)
				childText := source[child.StartByte():child.EndByte()]
				if childText == "private" || childText == "protected" {
					return false // Explicitly not exported
				}
				if childText == "public" {
					return true // Explicitly exported
				}
			}
		}
		parent = parent.Parent()
	}
	// Fallback to underscore rule
	name := c.ExtractNodeName(node, source)
	return !strings.HasPrefix(name, "_")
}

// ExpandMatches handles multiple property declarations in PHP
func (c *Config) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	makeMatch := func(target *sitter.Node, name string) []base.Target {
		return []base.Target{base.NewTarget(target, query.Type, name)}
	}

	switch node.Type() {
	case "property_declaration":
		return c.expandPropertyDeclaration(node, source, query)
	case "const_declaration", "class_constant_declaration":
		return c.expandConstDeclaration(node, source, query)
	case "namespace_use_declaration":
		return c.expandNamespaceUse(node, source, query)
	case "assignment_expression":
		if left := c.variableTargetNode(node); left != nil {
			name := strings.TrimPrefix(source[left.StartByte():left.EndByte()], "$")
			return makeMatch(left, name)
		}
		if left := node.ChildByFieldName("left"); left != nil {
			if left.Type() == "list_literal" {
				return c.expandListLiteral(left, source, query)
			}
		}
		name := c.ExtractNodeName(node, source)
		return makeMatch(node, name)
	case "simple_parameter":
		if nameNode := c.variableTargetNode(node); nameNode != nil {
			name := strings.TrimPrefix(source[nameNode.StartByte():nameNode.EndByte()], "$")
			return makeMatch(nameNode, name)
		}
		name := c.ExtractNodeName(node, source)
		return makeMatch(node, name)
	case "variable_name":
		name := strings.TrimPrefix(source[node.StartByte():node.EndByte()], "$")
		return makeMatch(node, name)
	default:
		name := c.ExtractNodeName(node, source)
		return makeMatch(node, name)
	}
}

func (c *Config) ValidateQueryNode(node *sitter.Node, source, queryType string) bool {
	switch queryType {
	case "constructor", "ctor":
		return node.Type() == "method_declaration" && strings.EqualFold(c.ExtractNodeName(node, source), "__construct")
	default:
		return true
	}
}

func (c *Config) expandPropertyDeclaration(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	for _, child := range c.propertyVariableNodes(node) {
		name := strings.TrimPrefix(source[child.StartByte():child.EndByte()], "$")
		matches = append(matches, base.NewTarget(child, query.Type, name))
	}

	return matches
}

func (c *Config) propertyVariableNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}

	var nodes []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(current *sitter.Node) {
		if current == nil {
			return
		}
		if current.Type() == "variable_name" {
			nodes = append(nodes, current)
			return
		}
		for i := 0; i < int(current.ChildCount()); i++ {
			walk(current.Child(i))
		}
	}
	walk(node)
	return nodes
}

func (c *Config) variableTargetNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}

	switch node.Type() {
	case "assignment_expression":
		if left := node.ChildByFieldName("left"); left != nil && left.Type() == "variable_name" {
			return left
		}
	case "simple_parameter":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil && nameNode.Type() == "variable_name" {
			return nameNode
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_name" {
				return child
			}
		}
	}

	return nil
}

func (c *Config) expandListLiteral(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_name" {
			name := strings.TrimPrefix(source[child.StartByte():child.EndByte()], "$")
			matches = append(matches, base.NewTarget(child, query.Type, name))
		}
	}
	return matches
}

func (c *Config) expandConstDeclaration(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	// Constants can be declared as: const A = 1, B = 2;
	// Collect each "name" token in the declaration
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "name" {
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, base.NewTarget(child, query.Type, name))
		} else if inner := child.ChildByFieldName("name"); inner != nil {
			name := source[inner.StartByte():inner.EndByte()]
			matches = append(matches, base.NewTarget(inner, query.Type, name))
		}
	}

	return matches
}

func (c *Config) expandNamespaceUse(node *sitter.Node, source string, query core.AgentQuery) []base.Target {
	var matches []base.Target

	// Handle grouped and multiple clauses: use A\B, C\D as Alias;
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "namespace_use_clause":
			// Prefer alias, otherwise name
			if alias := child.ChildByFieldName("alias"); alias != nil {
				name := source[alias.StartByte():alias.EndByte()]
				matches = append(matches, base.NewTarget(alias, query.Type, name))
				continue
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, base.NewTarget(nameNode, query.Type, name))
				continue
			}
		case "qualified_name":
			// Simple single-clause use
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, base.NewTarget(child, query.Type, name))
		}
	}

	// Fallback: single match for the whole declaration
	if len(matches) == 0 {
		name := c.ExtractNodeName(node, source)
		return []base.Target{base.NewTarget(node, query.Type, name)}
	}

	return matches
}
