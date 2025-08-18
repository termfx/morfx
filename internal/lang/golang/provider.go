package golang

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	golang_sitter "github.com/smacker/go-tree-sitter/golang"

	"github.com/garaekz/fileman/internal/provider"
	"github.com/garaekz/fileman/internal/types"
)

// GoProvider implements the LanguageProvider interface for Go language support
type GoProvider struct {
	provider.BaseProvider
	// Go-specific DSL vocabulary mapping
	dslVocabulary map[string]types.NodeKind
}

// NewProvider creates a new instance of the Go language provider
func NewProvider() provider.LanguageProvider {
	p := &GoProvider{}
	p.Initialize()
	return p
}

// Initialize sets up the Go provider with language-specific mappings
func (p *GoProvider) Initialize() {
	// Define Go-specific DSL vocabulary
	p.dslVocabulary = map[string]types.NodeKind{
		// Go-specific DSL terms
		"func":   "function",
		"var":    "variable",
		"const":  "constant",
		"type":   "class", // Go structs/types map to universal class concept
		"import": "import",
		"call":   "call",
		"assign": "assignment",
		"if":     "condition",
		"for":    "loop",
		"method": "method",
		"field":  "field",
		"block":  "block",
		"struct": "class", // Go struct maps to universal class concept
		// Also support universal terms for compatibility
		"function":   "function",
		"variable":   "variable",
		"constant":   "constant",
		"class":      "class",
		"assignment": "assignment",
		"condition":  "condition",
	}

	// Define how universal kinds map to Go AST
	mappings := []provider.NodeMapping{
		{
			Kind:        "function",
			NodeTypes:   []string{"function_declaration", "method_declaration"},
			NameCapture: "@name",
			Template:    `(function_declaration name: (identifier) @name %s)`,
		},
		{
			Kind:        "variable",
			NodeTypes:   []string{"var_declaration", "short_var_declaration"},
			NameCapture: "@name",
			TypeCapture: "@type",
			Template:    `(var_declaration (var_spec name: (identifier) @name %s))`,
		},
		{
			Kind:        "class",
			NodeTypes:   []string{"type_declaration"}, // Go uses structs
			NameCapture: "@name",
			Template:    `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type) %s))`,
		},
		{
			Kind:        "method",
			NodeTypes:   []string{"method_declaration"},
			NameCapture: "@name",
			Template:    `(method_declaration name: (field_identifier) @name %s)`,
		},
		{
			Kind:        "import",
			NodeTypes:   []string{"import_declaration"},
			NameCapture: "@name",
			Template:    `(import_declaration (import_spec_list (import_spec path: (interpreted_string_literal) @name %s)))`,
		},
		{
			Kind:        "constant",
			NodeTypes:   []string{"const_declaration"},
			NameCapture: "@name",
			Template:    `(const_declaration (const_spec name: (identifier) @name %s))`,
		},
		{
			Kind:        "field",
			NodeTypes:   []string{"field_declaration"},
			NameCapture: "@name",
			TypeCapture: "@type",
			Template:    `(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type %s)`,
		},
		{
			Kind:        "call",
			NodeTypes:   []string{"call_expression"},
			NameCapture: "@name",
			Template:    `(call_expression function: [(identifier) (selector_expression)] @name %s)`,
		},
		{
			Kind:        "assignment",
			NodeTypes:   []string{"assignment_statement"},
			NameCapture: "@name",
			Template:    `(assignment_statement left: (expression_list (identifier) @name) %s)`,
		},
		{
			Kind:        "condition",
			NodeTypes:   []string{"if_statement"},
			NameCapture: "@condition",
			Template:    `(if_statement condition: (_) @condition %s)`,
		},
		{
			Kind:        "loop",
			NodeTypes:   []string{"for_statement", "range_clause"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (range_clause)] @loop %s`,
		},
		{
			Kind:        "block",
			NodeTypes:   []string{"block"},
			NameCapture: "@block",
			Template:    `(block) @block %s`,
		},
		{
			Kind:        "comment",
			NodeTypes:   []string{"comment"},
			NameCapture: "@comment",
			Template:    `(comment) @comment %s`,
		},
		{
			Kind:        "type",
			NodeTypes:   []string{"type_identifier", "pointer_type", "slice_type", "array_type"},
			NameCapture: "@type",
			Template:    `[(type_identifier) (pointer_type) (slice_type) (array_type)] @type %s`,
		},
	}

	p.BuildMappings(mappings)
}

// Lang returns the canonical name of the language
func (p *GoProvider) Lang() string {
	return "go"
}

// Aliases returns alternative names for this language
func (p *GoProvider) Aliases() []string {
	return []string{"go", "golang"}
}

// Extensions returns file extensions for this language
func (p *GoProvider) Extensions() []string {
	return []string{".go"}
}

// GetSitterLanguage returns the Tree-sitter language for Go
func (p *GoProvider) GetSitterLanguage() *sitter.Language {
	return golang_sitter.GetLanguage()
}

// NormalizeDSLKind translates Go-specific DSL terms to universal kinds
func (p *GoProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	if universalKind, exists := p.dslVocabulary[dslKind]; exists {
		return universalKind
	}
	// Fallback to original if not found
	return types.NodeKind(dslKind)
}

// GetSupportedDSLKinds returns the Go-specific DSL vocabulary
func (p *GoProvider) GetSupportedDSLKinds() []string {
	kinds := make([]string, 0, len(p.dslVocabulary))
	for kind := range p.dslVocabulary {
		kinds = append(kinds, kind)
	}
	return kinds
}

// TranslateQuery translates a universal query to Go-specific Tree-sitter query
func (p *GoProvider) TranslateQuery(q *types.Query) (string, error) {
	// Normalize Go-specific DSL terms to universal kinds
	kind := p.NormalizeDSLKind(string(q.Kind))

	mappings := p.TranslateKind(kind)
	if len(mappings) == 0 {
		return "", fmt.Errorf("unsupported node kind: %s", q.Kind)
	}

	// Build query based on pattern and attributes
	var queries []string
	for _, mapping := range mappings {
		query := p.buildQueryFromMapping(mapping, q)
		if query != "" {
			queries = append(queries, query)
		}
	}

	if len(queries) == 0 {
		return "", fmt.Errorf("no valid queries generated for kind: %s", q.Kind)
	}

	return strings.Join(queries, "\n"), nil
}

// buildQueryFromMapping constructs a Tree-sitter query from a mapping and query
func (p *GoProvider) buildQueryFromMapping(mapping provider.NodeMapping, q *types.Query) string {
	// Handle pattern matching
	patternConstraint := ""
	if q.Pattern != "" && q.Pattern != "*" {
		// Convert wildcard patterns to regex
		regexPattern := p.convertWildcardToRegex(q.Pattern)
		patternConstraint = fmt.Sprintf(`(#match? %s "%s")`, mapping.NameCapture, regexPattern)
	}

	// Handle type constraints
	typeConstraint := ""
	if typeAttr, hasType := q.Attributes["type"]; hasType && mapping.TypeCapture != "" {
		typeRegex := p.convertWildcardToRegex(typeAttr)
		typeConstraint = fmt.Sprintf(`(#match? %s "%s")`, mapping.TypeCapture, typeRegex)
	}

	// Combine constraints
	constraints := []string{}
	if patternConstraint != "" {
		constraints = append(constraints, patternConstraint)
	}
	if typeConstraint != "" {
		constraints = append(constraints, typeConstraint)
	}

	constraintStr := strings.Join(constraints, " ")
	return fmt.Sprintf(mapping.Template, constraintStr)
}

// convertWildcardToRegex converts wildcard patterns to regex
func (p *GoProvider) convertWildcardToRegex(pattern string) string {
	// Escape regex special characters except * and ?
	escaped := strings.ReplaceAll(pattern, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "+", "\\+")
	escaped = strings.ReplaceAll(escaped, "^", "\\^")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")

	// Convert wildcards to regex
	escaped = strings.ReplaceAll(escaped, "*", ".*")
	escaped = strings.ReplaceAll(escaped, "?", ".")

	// Anchor the pattern
	return "^" + escaped + "$"
}

// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *GoProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	switch node.Type() {
	case "function_declaration":
		return "function"
	case "method_declaration":
		return "method"
	case "var_declaration", "short_var_declaration":
		return "variable"
	case "type_declaration":
		return "class" // Go structs map to universal class concept
	case "import_declaration":
		return "import"
	case "const_declaration":
		return "constant"
	case "field_declaration":
		return "field"
	case "call_expression":
		return "call"
	case "assignment_statement":
		return "assignment"
	case "if_statement":
		return "condition"
	case "for_statement", "range_clause":
		return "loop"
	case "block":
		return "block"
	case "comment":
		return "comment"
	case "type_identifier", "pointer_type", "slice_type", "array_type":
		return "type"
	default:
		return types.NodeKind(node.Type()) // Fallback to node type
	}
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *GoProvider) GetNodeName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_declaration":
		return p.extractIdentifier(node, "name", source)
	case "method_declaration":
		return p.extractFieldIdentifier(node, "name", source)
	case "var_declaration":
		return p.extractFromVarSpec(node, source)
	case "short_var_declaration":
		return p.extractFromExpressionList(node, "left", source)
	case "type_declaration":
		return p.extractFromTypeSpec(node, source)
	case "import_declaration":
		return p.extractImportPath(node, source)
	case "const_declaration":
		return p.extractFromConstSpec(node, source)
	case "field_declaration":
		return p.extractFieldName(node, source)
	case "call_expression":
		return p.extractCallName(node, source)
	case "assignment_statement":
		return p.extractFromExpressionList(node, "left", source)
	default:
		// Fallback: try to get text content
		return node.Content(source)
	}
}

// Helper methods for name extraction
func (p *GoProvider) extractIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

func (p *GoProvider) extractFieldIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

func (p *GoProvider) extractFromVarSpec(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "var_spec" {
			if nameChild := child.ChildByFieldName("name"); nameChild != nil {
				if nameChild.Type() == "identifier_list" && nameChild.ChildCount() > 0 {
					return nameChild.Child(0).Content(source)
				}
				return nameChild.Content(source)
			}
		}
	}
	return ""
}

func (p *GoProvider) extractFromExpressionList(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		if child.Type() == "expression_list" && child.ChildCount() > 0 {
			return child.Child(0).Content(source)
		}
		return child.Content(source)
	}
	return ""
}

func (p *GoProvider) extractFromTypeSpec(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_spec" {
			if nameChild := child.ChildByFieldName("name"); nameChild != nil {
				return nameChild.Content(source)
			}
		}
	}
	return ""
}

func (p *GoProvider) extractImportPath(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "import_spec" {
			if pathChild := child.ChildByFieldName("path"); pathChild != nil {
				// Remove quotes from string literal
				path := pathChild.Content(source)
				if len(path) >= 2 && path[0] == '"' && path[len(path)-1] == '"' {
					return path[1 : len(path)-1]
				}
				return path
			}
		}
	}
	return ""
}

func (p *GoProvider) extractFromConstSpec(node *sitter.Node, source []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "const_spec" {
			if nameChild := child.ChildByFieldName("name"); nameChild != nil {
				return nameChild.Content(source)
			}
		}
	}
	return ""
}

func (p *GoProvider) extractFieldName(node *sitter.Node, source []byte) string {
	if nameChild := node.ChildByFieldName("name"); nameChild != nil {
		if nameChild.Type() == "field_identifier_list" && nameChild.ChildCount() > 0 {
			return nameChild.Child(0).Content(source)
		}
		return nameChild.Content(source)
	}
	return ""
}

func (p *GoProvider) extractCallName(node *sitter.Node, source []byte) string {
	if funcChild := node.ChildByFieldName("function"); funcChild != nil {
		switch funcChild.Type() {
		case "identifier":
			return funcChild.Content(source)
		case "selector_expression":
			if fieldChild := funcChild.ChildByFieldName("field"); fieldChild != nil {
				return fieldChild.Content(source)
			}
		}
		return funcChild.Content(source)
	}
	return ""
}
