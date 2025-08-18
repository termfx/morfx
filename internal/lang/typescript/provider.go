package typescript

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	typescript_sitter "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/garaekz/fileman/internal/provider"
	"github.com/garaekz/fileman/internal/types"
)

// TypeScriptProvider implements the LanguageProvider interface for TypeScript language support
type TypeScriptProvider struct {
	provider.BaseProvider
}

// NewProvider creates a new instance of the TypeScript language provider
func NewProvider() provider.LanguageProvider {
	p := &TypeScriptProvider{}
	p.Initialize()
	return p
}

// Initialize sets up the TypeScript provider with language-specific mappings
func (p *TypeScriptProvider) Initialize() {
	// Define how universal kinds map to TypeScript AST
	mappings := []provider.NodeMapping{
		{
			Kind:        "function",
			NodeTypes:   []string{"function_declaration", "function", "arrow_function", "generator_function_declaration"},
			NameCapture: "@name",
			Template:    `[(function_declaration name: (identifier) @name %s) (function name: (identifier) @name %s) (generator_function_declaration name: (identifier) @name %s)]`,
		},
		{
			Kind:        "class",
			NodeTypes:   []string{"class_declaration", "class"},
			NameCapture: "@name",
			Template:    `[(class_declaration name: (type_identifier) @name %s) (class name: (type_identifier) @name %s)]`,
		},
		{
			Kind:        "method",
			NodeTypes:   []string{"method_definition", "method_signature"},
			NameCapture: "@name",
			Template:    `[(method_definition name: (property_identifier) @name %s) (method_signature name: (property_identifier) @name %s)]`,
		},
		{
			Kind:        "variable",
			NodeTypes:   []string{"variable_declarator", "lexical_declaration"},
			NameCapture: "@name",
			Template:    `[(variable_declarator name: (identifier) @name %s) (lexical_declaration (variable_declarator name: (identifier) @name %s))]`,
		},
		{
			Kind:        "import",
			NodeTypes:   []string{"import_statement"},
			NameCapture: "@source",
			Template:    `(import_statement source: (string) @source %s)`,
		},
		{
			Kind:        "constant",
			NodeTypes:   []string{"lexical_declaration", "enum_declaration"},
			NameCapture: "@name",
			// const declarations and enums
			Template: `[(lexical_declaration kind: "const" (variable_declarator name: (identifier) @name %s)) (enum_declaration name: (identifier) @name %s)]`,
		},
		{
			Kind:        "field",
			NodeTypes:   []string{"field_definition", "public_field_definition", "property_signature"},
			NameCapture: "@name",
			Template:    `[(field_definition property: (property_identifier) @name %s) (public_field_definition name: (property_identifier) @name %s) (property_signature name: (property_identifier) @name %s)]`,
		},
		{
			Kind:        "call",
			NodeTypes:   []string{"call_expression"},
			NameCapture: "@name",
			Template:    `(call_expression function: [(identifier) (member_expression)] @name %s)`,
		},
		{
			Kind:        "assignment",
			NodeTypes:   []string{"assignment_expression", "augmented_assignment_expression"},
			NameCapture: "@name",
			Template:    `[(assignment_expression left: (_) @name %s) (augmented_assignment_expression left: (_) @name %s)]`,
		},
		{
			Kind:        "condition",
			NodeTypes:   []string{"if_statement", "ternary_expression", "switch_statement"},
			NameCapture: "@condition",
			Template:    `[(if_statement condition: (_) @condition %s) (ternary_expression condition: (_) @condition %s) (switch_statement value: (_) @condition %s)]`,
		},
		{
			Kind:        "loop",
			NodeTypes:   []string{"for_statement", "for_in_statement", "while_statement", "do_statement"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (for_in_statement) (while_statement) (do_statement)] @loop %s`,
		},
		{
			Kind:        "block",
			NodeTypes:   []string{"statement_block"},
			NameCapture: "@block",
			Template:    `(statement_block) @block %s`,
		},
		{
			Kind:        "comment",
			NodeTypes:   []string{"comment"},
			NameCapture: "@comment",
			Template:    `(comment) @comment %s`,
		},
		{
			Kind:        "decorator",
			NodeTypes:   []string{"decorator"},
			NameCapture: "@name",
			Template:    `(decorator (identifier) @name %s)`,
		},
		{
			Kind:        "type",
			NodeTypes:   []string{"type_alias_declaration", "interface_declaration", "type_annotation", "type_identifier"},
			NameCapture: "@name",
			Template:    `[(type_alias_declaration name: (type_identifier) @name %s) (interface_declaration name: (type_identifier) @name %s) (type_annotation) @name (type_identifier) @name %s]`,
		},
	}

	p.BuildMappings(mappings)
}

// Lang returns the canonical name of the language
func (p *TypeScriptProvider) Lang() string {
	return "typescript"
}

// Aliases returns alternative names for this language
func (p *TypeScriptProvider) Aliases() []string {
	return []string{"typescript", "ts"}
}

// Extensions returns file extensions for this language
func (p *TypeScriptProvider) Extensions() []string {
	return []string{".ts", ".tsx", ".mts", ".cts"}
}

// GetSitterLanguage returns the Tree-sitter language for TypeScript
func (p *TypeScriptProvider) GetSitterLanguage() *sitter.Language {
	return typescript_sitter.GetLanguage()
}

// TranslateQuery translates a universal query to TypeScript-specific Tree-sitter query
func (p *TypeScriptProvider) TranslateQuery(q *types.Query) (string, error) {
	mappings := p.TranslateKind(q.Kind)
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
func (p *TypeScriptProvider) buildQueryFromMapping(mapping provider.NodeMapping, q *types.Query) string {
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

	// Handle visibility constraints (TypeScript has public/private/protected)
	visibilityConstraint := ""
	if visibility, hasVisibility := q.Attributes["visibility"]; hasVisibility {
		visibilityConstraint = fmt.Sprintf(`(#match? @visibility "%s")`, visibility)
	}

	// Combine constraints
	constraints := []string{}
	if patternConstraint != "" {
		constraints = append(constraints, patternConstraint)
	}
	if typeConstraint != "" {
		constraints = append(constraints, typeConstraint)
	}
	if visibilityConstraint != "" {
		constraints = append(constraints, visibilityConstraint)
	}

	constraintStr := strings.Join(constraints, " ")

	// Special handling for templates with multiple %s placeholders
	if strings.Count(mapping.Template, "%s") > 1 {
		// For templates with multiple placeholders, use the same constraint for all
		placeholders := make([]any, strings.Count(mapping.Template, "%s"))
		for i := range placeholders {
			placeholders[i] = constraintStr
		}
		return fmt.Sprintf(mapping.Template, placeholders...)
	}

	return fmt.Sprintf(mapping.Template, constraintStr)
}

// convertWildcardToRegex converts wildcard patterns to regex
func (p *TypeScriptProvider) convertWildcardToRegex(pattern string) string {
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
func (p *TypeScriptProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	switch node.Type() {
	case "function_declaration", "function", "arrow_function", "generator_function_declaration":
		return "function"
	case "class_declaration", "class":
		return "class"
	case "method_definition", "method_signature":
		return "method"
	case "variable_declarator":
		return "variable"
	case "lexical_declaration":
		// Check if it's a const
		if p.isConstDeclaration(node) {
			return "constant"
		}
		return "variable"
	case "enum_declaration":
		return "constant"
	case "import_statement", "import_specifier":
		return "import"
	case "field_definition", "public_field_definition", "property_signature":
		return "field"
	case "call_expression":
		return "call"
	case "assignment_expression", "augmented_assignment_expression":
		return "assignment"
	case "if_statement", "ternary_expression", "switch_statement":
		return "condition"
	case "for_statement", "for_in_statement", "while_statement", "do_statement":
		return "loop"
	case "statement_block":
		return "block"
	case "comment":
		return "comment"
	case "decorator":
		return "decorator"
	case "type_alias_declaration", "interface_declaration", "type_annotation", "type_identifier":
		return "type"
	default:
		return types.NodeKind(node.Type()) // Fallback to node type
	}
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *TypeScriptProvider) GetNodeName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_declaration", "generator_function_declaration", "class_declaration":
		return p.extractTypeIdentifier(node, "name", source)
	case "function", "class":
		// Anonymous functions/classes
		if name := node.ChildByFieldName("name"); name != nil {
			return name.Content(source)
		}
		return "<anonymous>"
	case "arrow_function":
		// Arrow functions are usually anonymous
		return "<arrow>"
	case "method_definition", "method_signature":
		return p.extractPropertyIdentifier(node, "name", source)
	case "variable_declarator":
		return p.extractVariableName(node, source)
	case "import_statement":
		return p.extractImportSource(node, source)
	case "field_definition", "public_field_definition", "property_signature":
		return p.extractPropertyIdentifier(node, "name", source)
	case "call_expression":
		return p.extractCallName(node, source)
	case "assignment_expression", "augmented_assignment_expression":
		return p.extractAssignmentTarget(node, source)
	case "type_alias_declaration", "interface_declaration":
		return p.extractTypeIdentifier(node, "name", source)
	case "enum_declaration":
		return p.extractIdentifier(node, "name", source)
	default:
		// Fallback: try to get text content
		return node.Content(source)
	}
}

// GetNodeScope provides TypeScript-specific scope detection
func (p *TypeScriptProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	switch node.Type() {
	case "program":
		return "file"
	case "class_declaration", "class":
		return "class"
	case "interface_declaration":
		return "class" // Interfaces are similar to classes in scope
	case "function_declaration", "function", "arrow_function", "method_definition":
		return "function"
	case "statement_block", "if_statement", "for_statement", "while_statement":
		return "block"
	case "namespace_statement", "module":
		return "namespace"
	default:
		if node.Parent() != nil {
			return p.GetNodeScope(node.Parent())
		}
		return "file"
	}
}

// Helper methods for name extraction
func (p *TypeScriptProvider) extractIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) extractTypeIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		if child.Type() == "type_identifier" || child.Type() == "identifier" {
			return child.Content(source)
		}
		return child.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) extractPropertyIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		if child.Type() == "property_identifier" || child.Type() == "identifier" {
			return child.Content(source)
		}
		// Handle computed property names
		if child.Type() == "computed_property_name" {
			return "[computed]"
		}
		return child.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) extractVariableName(node *sitter.Node, source []byte) string {
	if name := node.ChildByFieldName("name"); name != nil {
		switch name.Type() {
		case "identifier":
			return name.Content(source)
		case "object_pattern", "array_pattern":
			// Destructuring
			return "<destructured>"
		}
		return name.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) extractImportSource(node *sitter.Node, source []byte) string {
	if sourceNode := node.ChildByFieldName("source"); sourceNode != nil {
		// Remove quotes from string
		source := sourceNode.Content(source)
		if len(source) >= 2 && (source[0] == '"' || source[0] == '\'') {
			return source[1 : len(source)-1]
		}
		return source
	}
	return ""
}

func (p *TypeScriptProvider) extractCallName(node *sitter.Node, source []byte) string {
	if function := node.ChildByFieldName("function"); function != nil {
		switch function.Type() {
		case "identifier":
			return function.Content(source)
		case "member_expression":
			if property := function.ChildByFieldName("property"); property != nil {
				return property.Content(source)
			}
		}
		return function.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) extractAssignmentTarget(node *sitter.Node, source []byte) string {
	if left := node.ChildByFieldName("left"); left != nil {
		switch left.Type() {
		case "identifier":
			return left.Content(source)
		case "member_expression":
			if property := left.ChildByFieldName("property"); property != nil {
				return property.Content(source)
			}
		case "object_pattern", "array_pattern":
			return "<destructured>"
		}
		return left.Content(source)
	}
	return ""
}

func (p *TypeScriptProvider) isConstDeclaration(node *sitter.Node) bool {
	// Check if this is a const declaration
	if node.Type() == "lexical_declaration" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "const" {
				return true
			}
		}
	}
	return false
}
