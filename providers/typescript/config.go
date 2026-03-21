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
	if nodes, ok := c.aliasMap()[queryType]; ok {
		return nodes
	}
	return []string{queryType}
}

func (c *Config) aliasMap() map[string][]string {
	return map[string][]string{
		"function":    {"function_declaration", "function_expression", "arrow_function", "method_definition", "method_signature", "public_field_definition"},
		"func":        {"function_declaration", "function_expression", "arrow_function", "method_definition", "method_signature", "public_field_definition"},
		"fn":          {"function_declaration", "function_expression", "arrow_function", "method_definition", "method_signature", "public_field_definition"},
		"class":       {"class_declaration", "class_expression"},
		"interface":   {"interface_declaration"},
		"iface":       {"interface_declaration"},
		"type":        {"type_alias_declaration"},
		"enum":        {"enum_declaration"},
		"enum_member": {"enum_member"},
		"member":      {"enum_member"},
		"method":      {"method_definition", "method_signature"},
		"getter":      {"method_definition", "method_signature"},
		"setter":      {"method_definition", "method_signature"},
		"accessor":    {"method_definition", "method_signature"},
		"constructor": {"method_definition"},
		"ctor":        {"method_definition"},
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
		"module":      {"module_declaration"},
		"namespace":   {"namespace_declaration"},
		"property":    {"public_field_definition", "private_field_definition", "field_definition", "property_signature"},
		"prop":        {"public_field_definition", "private_field_definition", "field_definition", "property_signature"},
		"field":       {"public_field_definition", "private_field_definition", "field_definition", "property_signature"},
		"signature":   {"method_signature", "function_signature", "construct_signature", "index_signature", "call_signature"},
		"decorator":   {"decorator"},
		"comment":     {"comment"},
		"comments":    {"comment"},
	}
}

// SupportedQueryTypes returns colloquial query types/aliases for TypeScript
func (c *Config) SupportedQueryTypes() []string {
	m := c.aliasMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SmartAppend keeps appended members inside TypeScript block scopes such as
// classes and interfaces instead of falling back to top-level insertion.
func (c *Config) SmartAppend(source string, target *sitter.Node, content string) (string, bool) {
	if target == nil {
		return "", false
	}

	trimmed := strings.Trim(content, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return "", false
	}

	switch target.Type() {
	case "source_file", "program":
		return c.smartAppendSourceFile(source, target, trimmed), true
	case "class_declaration", "class_expression", "interface_declaration":
		return c.appendInsideBlock(source, target, trimmed), true
	default:
		return "", false
	}
}

func (c *Config) smartAppendSourceFile(source string, root *sitter.Node, content string) string {
	if scope, ok := c.inferImplicitAppendScope(root, content); ok {
		return c.appendInsideBlock(source, scope, content)
	}

	end := int(root.EndByte())
	if end < 0 || end > len(source) {
		end = len(source)
	}

	before := source[:end]
	after := source[end:]

	leading := "\n\n"
	switch {
	case strings.HasSuffix(before, "\n\n"):
		leading = ""
	case strings.HasSuffix(before, "\n"):
		leading = "\n"
	}

	trimmed := strings.Trim(content, "\n")
	if trimmed == "" {
		return source
	}

	insertion := leading + trimmed
	if !strings.HasSuffix(insertion, "\n") {
		insertion += "\n"
	}

	return before + insertion + after
}

func (c *Config) inferImplicitAppendScope(root *sitter.Node, content string) (*sitter.Node, bool) {
	if looksLikeTypeScriptTopLevelAppend(content) {
		return nil, false
	}

	var scope *sitter.Node
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		candidate := c.appendScopeCandidate(child)
		if candidate == nil {
			continue
		}

		if scope != nil {
			return nil, false
		}
		scope = candidate
	}

	return scope, scope != nil
}

func (c *Config) appendScopeCandidate(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}

	switch node.Type() {
	case "class_declaration", "interface_declaration":
		return node
	case "export_statement":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			if candidate := c.appendScopeCandidate(node.NamedChild(i)); candidate != nil {
				return candidate
			}
		}
	}

	return nil
}

func looksLikeTypeScriptTopLevelAppend(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	topLevelPrefixes := []string{
		"import ",
		"export ",
		"class ",
		"interface ",
		"type ",
		"enum ",
		"function ",
		"async function ",
		"const ",
		"let ",
		"var ",
		"namespace ",
		"module ",
		"declare ",
	}

	for _, prefix := range topLevelPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

func (c *Config) appendInsideBlock(source string, target *sitter.Node, content string) string {
	start := int(target.StartByte())
	end := int(target.EndByte())
	if start < 0 || end > len(source) || start >= end {
		return source
	}

	insertPos := c.findBlockInsertPos(source, start, end)
	if insertPos <= start || insertPos > len(source) {
		return source
	}

	classIndent := lineIndentationAt(source, start)
	memberIndent := c.detectMemberIndent(source[start:insertPos], classIndent)
	normalized := normalizeIndentedBlock(content, memberIndent)
	before := source[:insertPos]

	openBrace := strings.Index(source[start:insertPos], "{")
	hasMembers := false
	if openBrace >= 0 {
		body := source[start+openBrace+1 : insertPos]
		hasMembers = strings.TrimSpace(body) != ""
	}

	leading := "\n"
	switch {
	case strings.HasSuffix(before, "\n\n"):
		leading = ""
	case strings.HasSuffix(before, "\n"):
		if hasMembers {
			leading = "\n"
		} else {
			leading = ""
		}
	}

	trailing := "\n" + classIndent
	return before + leading + normalized + trailing + source[insertPos:]
}

func (c *Config) findBlockInsertPos(source string, start, end int) int {
	for i := end - 1; i >= start; i-- {
		if source[i] == '}' {
			return i
		}
	}
	return end
}

func (c *Config) detectMemberIndent(blockSource, classIndent string) string {
	lines := strings.Split(blockSource, "\n")
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := leadingWhitespace(line)
		if len(indent) > len(classIndent) {
			return indent
		}
	}

	if strings.Contains(blockSource, "\t") {
		return classIndent + "\t"
	}

	return classIndent + "  "
}

func normalizeIndentedBlock(content, indent string) string {
	trimmed := strings.Trim(content, "\n")
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		width := len(leadingWhitespace(line))
		if minIndent == -1 || width < minIndent {
			minIndent = width
		}
	}

	if minIndent < 0 {
		minIndent = 0
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
			continue
		}

		if minIndent > 0 && len(line) >= minIndent {
			line = line[minIndent:]
		}
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
}

func lineIndentationAt(source string, offset int) string {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}

	lineStart := strings.LastIndex(source[:offset], "\n") + 1
	return leadingWhitespace(source[lineStart:offset])
}

func leadingWhitespace(line string) string {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
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
	case "public_field_definition", "private_field_definition", "field_definition":
		// Class field definition: fieldName = value
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "property_identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "property_signature":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "property_identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "enum_member":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
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
	case "variable_declarator":
		return c.expandVariableDeclarator(node, source, query)
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
	case "import_statement":
		return c.expandImportStatement(node, source, query)
	case "export_statement":
		return c.expandExportStatement(node, source, query)
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

func (c *Config) expandImportStatement(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "import_specifier":
			if alias := child.ChildByFieldName("alias"); alias != nil {
				name := source[alias.StartByte():alias.EndByte()]
				matches = append(matches, core.CodeMatch{Node: child, Name: name, Type: query.Type, NodeType: child.Type(), StartByte: child.StartByte(), EndByte: child.EndByte(), Line: child.StartPoint().Row, Column: child.StartPoint().Column})
				continue
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, core.CodeMatch{Node: child, Name: name, Type: query.Type, NodeType: child.Type(), StartByte: child.StartByte(), EndByte: child.EndByte(), Line: child.StartPoint().Row, Column: child.StartPoint().Column})
				continue
			}
		case "namespace_import":
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, core.CodeMatch{Node: child, Name: name, Type: query.Type, NodeType: child.Type(), StartByte: child.StartByte(), EndByte: child.EndByte(), Line: child.StartPoint().Row, Column: child.StartPoint().Column})
				continue
			}
		}
	}
	if len(matches) == 0 {
		name := c.ExtractNodeName(node, source)
		return []core.CodeMatch{{Node: node, Name: name, Type: query.Type, NodeType: node.Type(), StartByte: node.StartByte(), EndByte: node.EndByte(), Line: node.StartPoint().Row, Column: node.StartPoint().Column}}
	}
	return matches
}

func (c *Config) expandExportStatement(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "export_specifier" {
			if alias := child.ChildByFieldName("alias"); alias != nil {
				name := source[alias.StartByte():alias.EndByte()]
				matches = append(matches, core.CodeMatch{Node: child, Name: name, Type: query.Type, NodeType: child.Type(), StartByte: child.StartByte(), EndByte: child.EndByte(), Line: child.StartPoint().Row, Column: child.StartPoint().Column})
				continue
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				name := source[nameNode.StartByte():nameNode.EndByte()]
				matches = append(matches, core.CodeMatch{Node: child, Name: name, Type: query.Type, NodeType: child.Type(), StartByte: child.StartByte(), EndByte: child.EndByte(), Line: child.StartPoint().Row, Column: child.StartPoint().Column})
				continue
			}
		}
	}
	if len(matches) == 0 {
		name := c.ExtractNodeName(node, source)
		return []core.CodeMatch{{Node: node, Name: name, Type: query.Type, NodeType: node.Type(), StartByte: node.StartByte(), EndByte: node.EndByte(), Line: node.StartPoint().Row, Column: node.StartPoint().Column}}
	}
	return matches
}
