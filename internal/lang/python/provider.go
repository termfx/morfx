package python

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	python_sitter "github.com/smacker/go-tree-sitter/python"

	"github.com/termfx/morfx/internal/provider"
	"github.com/termfx/morfx/internal/types"
)

// PythonProvider implements the LanguageProvider interface for Python language support
type PythonProvider struct {
	provider.BaseProvider
}

// NewProvider creates a new instance of the Python language provider
func NewProvider() provider.LanguageProvider {
	p := &PythonProvider{}
	p.Initialize()
	return p
}

// Initialize sets up the Python provider with language-specific mappings
func (p *PythonProvider) Initialize() {
	// Define how universal kinds map to Python AST
	mappings := []provider.NodeMapping{
		{
			Kind:        "function",
			NodeTypes:   []string{"function_definition"},
			NameCapture: "@name",
			Template:    `(function_definition name: (identifier) @name %s)`,
		},
		{
			Kind:        "class",
			NodeTypes:   []string{"class_definition"},
			NameCapture: "@name",
			Template:    `(class_definition name: (identifier) @name %s)`,
		},
		{
			Kind:        "method",
			NodeTypes:   []string{"function_definition"},
			NameCapture: "@name",
			Template:    `(class_definition body: (block (function_definition name: (identifier) @name %s)))`,
			Attributes: map[string]string{
				"context": "class_body",
			},
		},
		{
			Kind:        "variable",
			NodeTypes:   []string{"assignment", "annotated_assignment"},
			NameCapture: "@name",
			Template:    `[(assignment left: (identifier) @name %s) (annotated_assignment left: (identifier) @name %s)]`,
		},
		{
			Kind:        "import",
			NodeTypes:   []string{"import_statement", "import_from_statement"},
			NameCapture: "@name",
			Template:    `[(import_statement) (import_from_statement) %s] @name`,
		},
		{
			Kind:        "decorator",
			NodeTypes:   []string{"decorator"},
			NameCapture: "@name",
			Template:    `(decorator (identifier) @name %s)`,
		},
		{
			Kind:        "constant",
			NodeTypes:   []string{"assignment"},
			NameCapture: "@name",
			// Python convention: UPPER_CASE names are constants
			Template: `(assignment left: (identifier) @name (#match? @name "^[A-Z_]+$") %s)`,
		},
		{
			Kind:        "field",
			NodeTypes:   []string{"assignment"},
			NameCapture: "@name",
			// Instance variables in __init__ method
			Template: `(function_definition name: (identifier) @init (#eq? @init "__init__") body: (block (expression_statement (assignment left: (attribute object: (identifier) @self (#eq? @self "self") attribute: (identifier) @name) %s))))`,
		},
		{
			Kind:        "call",
			NodeTypes:   []string{"call"},
			NameCapture: "@name",
			Template:    `(call function: [(identifier) (attribute)] @name %s)`,
		},
		{
			Kind:        "assignment",
			NodeTypes:   []string{"assignment", "augmented_assignment"},
			NameCapture: "@name",
			Template:    `[(assignment left: (_) @name %s) (augmented_assignment left: (_) @name %s)]`,
		},
		{
			Kind:        "condition",
			NodeTypes:   []string{"if_statement", "elif_clause"},
			NameCapture: "@condition",
			Template:    `[(if_statement condition: (_) @condition %s) (elif_clause condition: (_) @condition %s)]`,
		},
		{
			Kind:        "loop",
			NodeTypes:   []string{"for_statement", "while_statement"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (while_statement)] @loop %s`,
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
			NodeTypes:   []string{"type", "type_alias_statement"},
			NameCapture: "@type",
			Template:    `[(type) (type_alias_statement)] @type %s`,
		},
	}

	p.BuildMappings(mappings)
}

// Lang returns the canonical name of the language
func (p *PythonProvider) Lang() string {
	return "python"
}

// Aliases returns alternative names for this language
func (p *PythonProvider) Aliases() []string {
	return []string{"python", "py", "python3", "py3"}
}

// Extensions returns file extensions for this language
func (p *PythonProvider) Extensions() []string {
	return []string{".py", ".pyw", ".pyi"}
}

// GetSitterLanguage returns the Tree-sitter language for Python
func (p *PythonProvider) GetSitterLanguage() *sitter.Language {
	return python_sitter.GetLanguage()
}

// TranslateQuery translates a universal query to Python-specific Tree-sitter query
func (p *PythonProvider) TranslateQuery(q *types.Query) (string, error) {
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
func (p *PythonProvider) buildQueryFromMapping(mapping provider.NodeMapping, q *types.Query) string {
	// Handle pattern matching
	patternConstraint := ""
	if q.Pattern != "" && q.Pattern != "*" {
		// Convert wildcard patterns to regex
		regexPattern := p.convertWildcardToRegex(q.Pattern)
		patternConstraint = fmt.Sprintf(`(#match? %s "%s")`, mapping.NameCapture, regexPattern)
	}

	// Handle type constraints (for annotated assignments in Python)
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
func (p *PythonProvider) convertWildcardToRegex(pattern string) string {
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
func (p *PythonProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	switch node.Type() {
	case "function_definition":
		// Check if it's a method (inside a class)
		parent := node.Parent()
		for parent != nil {
			if parent.Type() == "class_definition" {
				return "method"
			}
			parent = parent.Parent()
		}
		return "function"
	case "class_definition":
		return "class"
	case "assignment", "annotated_assignment":
		return "variable"
	case "augmented_assignment":
		return "assignment"
	case "import_statement", "import_from_statement":
		return "import"
	case "decorator":
		return "decorator"
	case "call":
		return "call"
	case "if_statement", "elif_clause":
		return "condition"
	case "for_statement", "while_statement":
		return "loop"
	case "block":
		return "block"
	case "comment":
		return "comment"
	case "type", "type_alias_statement":
		return "type"
	default:
		return types.NodeKind(node.Type()) // Fallback to node type
	}
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *PythonProvider) GetNodeName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_definition", "class_definition":
		return p.extractIdentifier(node, "name", source)
	case "assignment", "annotated_assignment":
		return p.extractAssignmentTarget(node, source)
	case "augmented_assignment":
		return p.extractAugmentedAssignmentTarget(node, source)
	case "import_statement":
		return p.extractImportName(node, source)
	case "import_from_statement":
		return p.extractFromImportName(node, source)
	case "decorator":
		return p.extractDecoratorName(node, source)
	case "call":
		return p.extractCallName(node, source)
	default:
		// Fallback: try to get text content
		return node.Content(source)
	}
}

// GetNodeScope provides Python-specific scope detection
func (p *PythonProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	switch node.Type() {
	case "module":
		return "file"
	case "class_definition":
		return "class"
	case "function_definition":
		return "function"
	case "block", "if_statement", "for_statement", "while_statement", "with_statement":
		return "block"
	default:
		if node.Parent() != nil {
			return p.GetNodeScope(node.Parent())
		}
		return "file"
	}
}

// Helper methods for name extraction
func (p *PythonProvider) extractIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

func (p *PythonProvider) extractAssignmentTarget(node *sitter.Node, source []byte) string {
	if left := node.ChildByFieldName("left"); left != nil {
		switch left.Type() {
		case "identifier":
			return left.Content(source)
		case "attribute":
			// For self.attribute assignments
			if attr := left.ChildByFieldName("attribute"); attr != nil {
				return attr.Content(source)
			}
		case "pattern_list", "tuple_pattern":
			// For multiple assignment like a, b = values
			if left.ChildCount() > 0 {
				return left.Child(0).Content(source)
			}
		}
		return left.Content(source)
	}
	return ""
}

func (p *PythonProvider) extractAugmentedAssignmentTarget(node *sitter.Node, source []byte) string {
	if left := node.ChildByFieldName("left"); left != nil {
		return left.Content(source)
	}
	return ""
}

func (p *PythonProvider) extractImportName(node *sitter.Node, source []byte) string {
	// Look for dotted_name or aliased_import
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "dotted_name":
			return child.Content(source)
		case "aliased_import":
			if name := child.ChildByFieldName("name"); name != nil {
				return name.Content(source)
			}
		}
	}
	return ""
}

func (p *PythonProvider) extractFromImportName(node *sitter.Node, source []byte) string {
	// Get the module name being imported from
	if module := node.ChildByFieldName("module_name"); module != nil {
		return module.Content(source)
	}
	return ""
}

func (p *PythonProvider) extractDecoratorName(node *sitter.Node, source []byte) string {
	// Decorator can be identifier or attribute (e.g., @property or @app.route)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			return child.Content(source)
		case "attribute":
			if attr := child.ChildByFieldName("attribute"); attr != nil {
				return attr.Content(source)
			}
		case "call":
			// For decorators with arguments like @app.route("/")
			if function := child.ChildByFieldName("function"); function != nil {
				return p.extractCallName(function, source)
			}
		}
	}
	return ""
}

func (p *PythonProvider) extractCallName(node *sitter.Node, source []byte) string {
	if function := node.ChildByFieldName("function"); function != nil {
		switch function.Type() {
		case "identifier":
			return function.Content(source)
		case "attribute":
			if attr := function.ChildByFieldName("attribute"); attr != nil {
				return attr.Content(source)
			}
		}
		return function.Content(source)
	}
	return ""
}
