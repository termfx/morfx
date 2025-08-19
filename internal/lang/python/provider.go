package python

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	python_sitter "github.com/smacker/go-tree-sitter/python"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
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
			Kind:        core.KindFunction,
			NodeTypes:   []string{"function_definition"},
			NameCapture: "@name",
			Template:    `(function_definition name: (identifier) @name %s)`,
		},
		{
			Kind:        core.KindClass,
			NodeTypes:   []string{"class_definition"},
			NameCapture: "@name",
			Template:    `(class_definition name: (identifier) @name %s)`,
		},
		{
			Kind:        core.KindMethod,
			NodeTypes:   []string{"function_definition"},
			NameCapture: "@name",
			Template:    `(class_definition body: (block (function_definition name: (identifier) @name %s)))`,
			Attributes: map[string]string{
				"context": "class_body",
			},
		},
		{
			Kind:        core.KindVariable,
			NodeTypes:   []string{"assignment", "annotated_assignment"},
			NameCapture: "@name",
			Template:    `[(assignment left: (identifier) @name %s) (annotated_assignment left: (identifier) @name %s)]`,
		},
		{
			Kind:        core.KindImport,
			NodeTypes:   []string{"import_statement", "import_from_statement"},
			NameCapture: "@name",
			Template:    `[(import_statement) (import_from_statement) %s] @name`,
		},
		{
			Kind:        core.KindDecorator,
			NodeTypes:   []string{"decorator"},
			NameCapture: "@name",
			Template:    `(decorator (identifier) @name %s)`,
		},
		{
			Kind:        core.KindConstant,
			NodeTypes:   []string{"assignment"},
			NameCapture: "@name",
			// Python convention: UPPER_CASE names are constants
			Template: `(assignment left: (identifier) @name (#match? @name "^[A-Z_]+$") %s)`,
		},
		{
			Kind:        core.KindField,
			NodeTypes:   []string{"assignment"},
			NameCapture: "@name",
			// Instance variables in __init__ method
			Template: `(function_definition name: (identifier) @init (#eq? @init "__init__") body: (block (expression_statement (assignment left: (attribute object: (identifier) @self (#eq? @self "self") attribute: (identifier) @name))))) %s`,
		},
		{
			Kind:        core.KindCall,
			NodeTypes:   []string{"call"},
			NameCapture: "@name",
			Template:    `(call function: [(identifier) (attribute)] @name %s)`,
		},
		{
			Kind:        core.KindAssignment,
			NodeTypes:   []string{"assignment", "augmented_assignment"},
			NameCapture: "@name",
			Template:    `[(assignment left: (_) @name %s) (augmented_assignment left: (_) @name %s)]`,
		},
		{
			Kind:        core.KindCondition,
			NodeTypes:   []string{"if_statement", "elif_clause"},
			NameCapture: "@condition",
			Template:    `[(if_statement condition: (_) @condition %s) (elif_clause condition: (_) @condition %s)]`,
		},
		{
			Kind:        core.KindLoop,
			NodeTypes:   []string{"for_statement", "while_statement"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (while_statement)] @loop %s`,
		},
		{
			Kind:        core.KindBlock,
			NodeTypes:   []string{"block"},
			NameCapture: "@block",
			Template:    `(block) @block %s`,
		},
		{
			Kind:        core.KindComment,
			NodeTypes:   []string{"comment"},
			NameCapture: "@comment",
			Template:    `(comment) @comment %s`,
		},
		{
			Kind:        core.KindType,
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
func (p *PythonProvider) TranslateQuery(q *core.Query) (string, error) {
	// Handle logical operations
	if q.Operator == "AND" || q.Operator == "OR" {
		return p.translateLogicalQuery(q)
	}
	if q.Operator == "NOT" {
		return p.translateNegatedQuery(q)
	}
	if q.Operator == "HIERARCHY" {
		return p.translateHierarchicalQuery(q)
	}

	mappings := p.TranslateKind(q.Kind)
	if len(mappings) == 0 {
		return "", fmt.Errorf("unsupported node kind: %s", q.Kind)
	}

	// Build query based on pattern and attributes
	var queries []string
	for _, mapping := range mappings {
		query := p.BuildQueryFromMapping(mapping, q)
		if query != "" {
			queries = append(queries, query)
		}
	}

	if len(queries) == 0 {
		return "", fmt.Errorf("no valid queries generated for kind: %s", q.Kind)
	}

	return strings.Join(queries, "\n"), nil
}

// translateLogicalQuery handles AND/OR operations by combining multiple queries
func (p *PythonProvider) translateLogicalQuery(q *core.Query) (string, error) {
	if len(q.Children) < 2 {
		return "", fmt.Errorf("logical query must have at least 2 children")
	}

	var subQueries []string
	for _, child := range q.Children {
		childQuery, err := p.TranslateQuery(&child)
		if err != nil {
			return "", fmt.Errorf("error translating child query: %w", err)
		}
		if childQuery != "" {
			subQueries = append(subQueries, childQuery)
		}
	}

	if len(subQueries) == 0 {
		return "", fmt.Errorf("no valid sub-queries generated")
	}

	// For tree-sitter queries, we combine them with newlines
	// The evaluation logic will handle AND/OR semantics at the result level
	return strings.Join(subQueries, "\n"), nil
}

// translateNegatedQuery handles NOT operations
func (p *PythonProvider) translateNegatedQuery(q *core.Query) (string, error) {
	// For NOT operations, we still generate the positive query
	// The negation is handled at the evaluation level
	// Create a copy without the NOT operator
	childQuery := *q
	childQuery.Operator = ""
	
	return p.TranslateQuery(&childQuery)
}

// translateHierarchicalQuery handles parent > child relationships
func (p *PythonProvider) translateHierarchicalQuery(q *core.Query) (string, error) {
	if len(q.Children) != 1 {
		return "", fmt.Errorf("hierarchical query must have exactly 1 parent")
	}

	parent := &q.Children[0]
	
	// Get parent mappings
	parentMappings := p.TranslateKind(parent.Kind)
	if len(parentMappings) == 0 {
		return "", fmt.Errorf("unsupported parent node kind: %s", parent.Kind)
	}
	
	// Get child mappings
	childMappings := p.TranslateKind(q.Kind)
	if len(childMappings) == 0 {
		return "", fmt.Errorf("unsupported child node kind: %s", q.Kind)
	}

	var queries []string
	for _, parentMapping := range parentMappings {
		for _, childMapping := range childMappings {
			// Build hierarchical query
			query := p.buildHierarchicalQuery(parentMapping, parent, childMapping, q)
			if query != "" {
				queries = append(queries, query)
			}
		}
	}

	if len(queries) == 0 {
		return "", fmt.Errorf("no valid hierarchical queries generated")
	}

	return strings.Join(queries, "\n"), nil
}

// buildHierarchicalQuery constructs a tree-sitter query for parent-child relationships
func (p *PythonProvider) buildHierarchicalQuery(parentMapping provider.NodeMapping, parent *core.Query, childMapping provider.NodeMapping, child *core.Query) string {
	// Build pattern constraint for parent
	parentConstraint := ""
	if parent.Pattern != "" && parent.Pattern != "*" {
		regexPattern := p.ConvertWildcardToRegex(parent.Pattern)
		parentConstraint = fmt.Sprintf(`(#match? %s "%s")`, parentMapping.NameCapture, regexPattern)
	}

	// Build pattern constraint for child
	childConstraint := ""
	if child.Pattern != "" && child.Pattern != "*" {
		regexPattern := p.ConvertWildcardToRegex(child.Pattern)
		childConstraint = fmt.Sprintf(`(#match? %s "%s")`, childMapping.NameCapture, regexPattern)
	}

	// Combine constraints
	constraints := []string{}
	if parentConstraint != "" {
		constraints = append(constraints, parentConstraint)
	}
	if childConstraint != "" {
		constraints = append(constraints, childConstraint)
	}
	
	constraintStr := strings.Join(constraints, " ")

	// Build the hierarchical query pattern for Python
	for _, parentType := range parentMapping.NodeTypes {
		for _, childType := range childMapping.NodeTypes {
			if parentType == "class_definition" && childType == "function_definition" {
				return fmt.Sprintf(`(%s body: (block (%s %s %s)))`,
					parentType, childType, childMapping.NameCapture, constraintStr)
			}
			if parentType == "function_definition" && childType == "assignment" {
				return fmt.Sprintf(`(%s body: (block (expression_statement (%s %s %s))))`,
					parentType, childType, childMapping.NameCapture, constraintStr)
			}
		}
	}

	// Generic hierarchical pattern
	return fmt.Sprintf(`(%s (%s %s %s))`,
		parentMapping.NodeTypes[0], childMapping.NodeTypes[0], childMapping.NameCapture, constraintStr)
}

// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *PythonProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	switch node.Type() {
	case "function_definition":
		// Check if it's a method (inside a class)
		parent := node.Parent()
		for parent != nil {
			if parent.Type() == "class_definition" {
				return core.KindMethod
			}
			parent = parent.Parent()
		}
		return core.KindFunction
	case "class_definition":
		return core.KindClass
	case "assignment", "annotated_assignment":
		return core.KindVariable
	case "augmented_assignment":
		return core.KindAssignment
	case "import_statement", "import_from_statement":
		return core.KindImport
	case "decorator":
		return core.KindDecorator
	case "call":
		return core.KindCall
	case "if_statement", "elif_clause":
		return core.KindCondition
	case "for_statement", "while_statement":
		return core.KindLoop
	case "block":
		return core.KindBlock
	case "comment":
		return core.KindComment
	case "type", "type_alias_statement":
		return core.KindType
	default:
		return core.NodeKind(node.Type()) // Fallback to node type
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
func (p *PythonProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
	switch node.Type() {
	case "module":
		return core.ScopeFile
	case "class_definition":
		return core.ScopeClass
	case "function_definition":
		return core.ScopeFunction
	case "block", "if_statement", "for_statement", "while_statement", "with_statement":
		return core.ScopeBlock
	default:
		if node.Parent() != nil {
			return p.GetNodeScope(node.Parent())
		}
		return core.ScopeFile
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

// QuickCheck performs basic syntax and semantic checks on Python code
func (p *PythonProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
	var diagnostics []core.QuickCheckDiagnostic

	parser := sitter.NewParser()
	parser.SetLanguage(p.GetSitterLanguage())
	tree := parser.Parse(nil, source)
	defer tree.Close()

	// Check for syntax errors
	p.checkForErrors(tree.RootNode(), source, &diagnostics)

	// Check for basic semantic issues
	p.checkBasicSemantics(tree.RootNode(), source, &diagnostics)

	return diagnostics
}

// checkForErrors recursively checks for ERROR nodes in the syntax tree
func (p *PythonProvider) checkForErrors(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
	if node.Type() == "ERROR" {
		*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
			Severity: "error",
			Message:  "Syntax error",
			Line:     int(node.StartPoint().Row) + 1,
			Column:   int(node.StartPoint().Column) + 1,
			Code:     "syntax-error",
		})
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.checkForErrors(node.Child(i), source, diagnostics)
	}
}

// checkBasicSemantics performs basic semantic checks
func (p *PythonProvider) checkBasicSemantics(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
	// Check for function definitions without bodies
	if node.Type() == "function_definition" {
		if body := node.ChildByFieldName("body"); body == nil {
			*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
				Severity: "error",
				Message:  "Function definition missing body",
				Line:     int(node.StartPoint().Row) + 1,
				Column:   int(node.StartPoint().Column) + 1,
				Code:     "missing-function-body",
			})
		}
	}

	// Check for class definitions without bodies
	if node.Type() == "class_definition" {
		if body := node.ChildByFieldName("body"); body == nil {
			*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
				Severity: "error",
				Message:  "Class definition missing body",
				Line:     int(node.StartPoint().Row) + 1,
				Column:   int(node.StartPoint().Column) + 1,
				Code:     "missing-class-body",
			})
		}
	}

	// Recursively check child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		p.checkBasicSemantics(node.Child(i), source, diagnostics)
	}
}
