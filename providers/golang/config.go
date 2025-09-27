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
	if nodes, ok := c.aliasMap()[queryType]; ok {
		return nodes
	}
	return []string{queryType}
}

func (c *Config) aliasMap() map[string][]string {
	return map[string][]string{
		"function":  {"function_declaration", "method_declaration"},
		"func":      {"function_declaration", "method_declaration"},
		"fn":        {"function_declaration", "method_declaration"},
		"struct":    {"type_spec"},
		"interface": {"type_spec"},
		"iface":     {"type_spec"},
		"variable":  {"var_declaration", "short_var_declaration"},
		"var":       {"var_declaration", "short_var_declaration"},
		"constant":  {"const_declaration"},
		"const":     {"const_declaration"},
		"import":    {"import_declaration"},
		"type":      {"type_declaration", "type_spec"},
		"method":    {"method_declaration"},
		"field":     {"field_declaration"},
		"comment":   {"comment"},
		"comments":  {"comment"},
	}
}

// SupportedQueryTypes returns colloquial query types/aliases for Go
func (c *Config) SupportedQueryTypes() []string {
	m := c.aliasMap()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SmartAppend provides heuristics for inserting Go code snippets in sensible locations.
func (c *Config) SmartAppend(source string, target *sitter.Node, content string) (string, bool) {
	if target == nil {
		return "", false
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", false
	}

	switch target.Type() {
	case "source_file":
		return c.smartAppendSourceFile(source, target, trimmed), true
	default:
		return "", false
	}
}

func (c *Config) smartAppendSourceFile(source string, root *sitter.Node, content string) string {
	hint := classifyGoAppend(content)
	switch hint {
	case goAppendImport:
		if modified, ok := c.insertAfterLastImport(source, root, content); ok {
			return modified
		}
	case goAppendFunction:
		if modified, ok := c.insertAfterLastOf(source, root, content, []string{"function_declaration", "method_declaration"}, true); ok {
			return modified
		}
	case goAppendType, goAppendConst, goAppendVar:
		types := []string{"type_declaration"}
		if hint == goAppendConst {
			types = append(types, "const_declaration")
		}
		if hint == goAppendVar {
			types = append(types, "var_declaration")
		}
		if modified, ok := c.insertAfterLastOf(source, root, content, types, false); ok {
			return modified
		}
	}

	// Fallback: append at end with graceful spacing.
	end := int(root.EndByte())
	return insertTopLevelBlock(source, end, content, true)
}

func (c *Config) insertAfterLastImport(source string, root *sitter.Node, content string) (string, bool) {
	var lastImport *sitter.Node
	for i := int(root.NamedChildCount()) - 1; i >= 0; i-- {
		child := root.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == "import_declaration" {
			lastImport = child
			break
		}
	}

	if lastImport == nil {
		// Place after package clause if present.
		if pkg := root.NamedChild(0); pkg != nil && pkg.Type() == "package_clause" {
			offset := int(pkg.EndByte())
			return insertTopLevelBlock(source, offset, "import "+strings.TrimSpace(content), true), true
		}
		return "", false
	}

	offset := int(lastImport.EndByte())
	insert := content
	if !strings.HasPrefix(strings.TrimSpace(insert), "import") {
		insert = "import " + strings.TrimSpace(insert)
	}
	return insertTopLevelBlock(source, offset, insert, false), true
}

func (c *Config) insertAfterLastOf(source string, root *sitter.Node, content string, types []string, ensureBlank bool) (string, bool) {
	set := make(map[string]struct{}, len(types))
	for _, t := range types {
		set[t] = struct{}{}
	}

	var last *sitter.Node
	for i := int(root.NamedChildCount()) - 1; i >= 0; i-- {
		child := root.NamedChild(i)
		if child == nil {
			continue
		}
		if _, ok := set[child.Type()]; ok {
			last = child
			break
		}
	}

	if last != nil {
		offset := int(last.EndByte())
		return insertTopLevelBlock(source, offset, content, ensureBlank), true
	}

	// If no matching declaration, fall back after imports if present.
	if modified, ok := c.insertAfterLastImport(source, root, content); ok {
		return modified, true
	}

	return "", false
}

const (
	goAppendImport   = "import"
	goAppendFunction = "function"
	goAppendType     = "type"
	goAppendVar      = "var"
	goAppendConst    = "const"
)

func classifyGoAppend(content string) string {
	lower := strings.ToLower(strings.TrimSpace(content))
	if strings.HasPrefix(lower, "import ") || (strings.HasPrefix(lower, `"`) && !strings.Contains(lower, "\n")) {
		return goAppendImport
	}
	if strings.HasPrefix(lower, "func ") || strings.HasPrefix(lower, "func(") {
		return goAppendFunction
	}
	if strings.HasPrefix(lower, "type ") {
		return goAppendType
	}
	if strings.HasPrefix(lower, "const ") {
		return goAppendConst
	}
	if strings.HasPrefix(lower, "var ") {
		return goAppendVar
	}
	return "general"
}

func insertTopLevelBlock(source string, offset int, content string, ensureBlank bool) string {
	before := source[:offset]
	after := source[offset:]

	trimmed := strings.TrimRight(content, "\n")

	var leading string
	trimmedBefore := strings.TrimRight(before, " \t")
	switch {
	case strings.HasSuffix(trimmedBefore, "\n\n"):
		leading = ""
	case strings.HasSuffix(trimmedBefore, "\n"):
		if ensureBlank {
			leading = "\n"
		} else {
			leading = ""
		}
	default:
		leading = "\n\n"
	}

	insertion := leading + trimmed
	if !strings.HasSuffix(insertion, "\n") {
		insertion += "\n"
	}

	if len(after) > 0 && after[0] != '\n' {
		insertion += "\n"
	}

	return before + insertion + after
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
	case "comment":
		return c.extractCommentContent(source[node.StartByte():node.EndByte()])
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

func (c *Config) extractCommentContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
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
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
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
	case "import_declaration":
		return c.expandImportDeclaration(node, source, query)
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
		if child.Type() == "var_spec" || child.Type() == "const_spec" {
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

func (c *Config) expandImportDeclaration(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "import_spec" {
			continue
		}
		var name string
		if nameNode := child.ChildByFieldName("name"); nameNode != nil {
			name = source[nameNode.StartByte():nameNode.EndByte()]
		}
		if name == "" {
			if pathNode := child.ChildByFieldName("path"); pathNode != nil {
				// Trim quotes
				raw := source[pathNode.StartByte():pathNode.EndByte()]
				if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') {
					name = strings.Trim(raw, "\"'")
				} else {
					name = raw
				}
			}
		}
		matches = append(matches, core.CodeMatch{
			Node:      child,
			Name:      name,
			Type:      query.Type,
			NodeType:  child.Type(),
			StartByte: child.StartByte(),
			EndByte:   child.EndByte(),
			Line:      child.StartPoint().Row,
			Column:    child.StartPoint().Column,
		})
	}

	if len(matches) == 0 {
		// Fallback single match
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

	return matches
}
