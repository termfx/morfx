package javascript

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	javascript_sitter "github.com/smacker/go-tree-sitter/javascript"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// JavaScriptProvider implements the LanguageProvider interface for JavaScript language support
type JavaScriptProvider struct {
	provider.BaseProvider
}

// NewProvider creates a new instance of the JavaScript language provider
func NewProvider() provider.LanguageProvider {
	p := &JavaScriptProvider{}
	p.Initialize()
	return p
}

// Initialize sets up the JavaScript provider with language-specific mappings
func (p *JavaScriptProvider) Initialize() {
	// Define how universal kinds map to JavaScript AST
	mappings := []provider.NodeMapping{
		{
			Kind:        core.KindFunction,
			NodeTypes:   []string{"function_declaration", "function", "arrow_function", "generator_function_declaration"},
			NameCapture: "@name",
			Template:    `[(function_declaration name: (identifier) @name) (function name: (identifier) @name) (generator_function_declaration name: (identifier) @name)] %s`,
		},
		{
			Kind:        core.KindClass,
			NodeTypes:   []string{"class_declaration", "class"},
			NameCapture: "@name",
			Template:    `[(class_declaration name: (identifier) @name) (class name: (identifier) @name)] %s`,
		},
		{
			Kind:        core.KindMethod,
			NodeTypes:   []string{"method_definition"},
			NameCapture: "@name",
			Template:    `(method_definition name: (property_identifier) @name) %s`,
		},
		{
			Kind:        core.KindVariable,
			NodeTypes:   []string{"variable_declarator", "lexical_declaration"},
			NameCapture: "@name",
			Template:    `[(variable_declarator name: (identifier) @name) (lexical_declaration (variable_declarator name: (identifier) @name))] %s`,
		},
		{
			Kind:        core.KindImport,
			NodeTypes:   []string{"import_statement"},
			NameCapture: "@source",
			Template:    `(import_statement source: (string) @source) %s`,
		},
		{
			Kind:        core.KindConstant,
			NodeTypes:   []string{"lexical_declaration"},
			NameCapture: "@name",
			Template:    `(lexical_declaration (variable_declarator name: (identifier) @name)) %s`,
			Attributes: map[string]string{
				"kind": "const",
			},
		},
		{
			Kind:        core.KindField,
			NodeTypes:   []string{"field_definition", "public_field_definition"},
			NameCapture: "@name",
			Template:    `[(field_definition name: (property_identifier) @name) (public_field_definition name: (property_identifier) @name)] %s`,
		},
		{
			Kind:        core.KindCall,
			NodeTypes:   []string{"call_expression"},
			NameCapture: "@name",
			Template:    `(call_expression function: [(identifier) (member_expression)] @name) %s`,
		},
		{
			Kind:        core.KindAssignment,
			NodeTypes:   []string{"assignment_expression", "augmented_assignment_expression"},
			NameCapture: "@name",
			Template:    `[(assignment_expression left: (_) @name) (augmented_assignment_expression left: (_) @name)] %s`,
		},
		{
			Kind:        core.KindCondition,
			NodeTypes:   []string{"if_statement", "ternary_expression", "switch_statement"},
			NameCapture: "@condition",
			Template:    `[(if_statement condition: (_) @condition) (ternary_expression condition: (_) @condition) (switch_statement value: (_) @condition)] %s`,
		},
		{
			Kind:        core.KindLoop,
			NodeTypes:   []string{"for_statement", "for_in_statement", "while_statement", "do_statement"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (for_in_statement) (while_statement) (do_statement)] @loop %s`,
		},
		{
			Kind:        core.KindBlock,
			NodeTypes:   []string{"statement_block"},
			NameCapture: "@block",
			Template:    `(statement_block) @block %s`,
		},
		{
			Kind:        core.KindComment,
			NodeTypes:   []string{"comment"},
			NameCapture: "@comment",
			Template:    `(comment) @comment %s`,
		},
		{
			Kind:        core.KindDecorator,
			NodeTypes:   []string{"decorator"},
			NameCapture: "@name",
			Template:    `(decorator (identifier) @name) %s`,
		},
		{
			Kind:        core.KindType,
			NodeTypes:   []string{"type_annotation", "type_identifier"},
			NameCapture: "@type",
			Template:    `[(type_annotation) (type_identifier)] @type %s`,
		},
	}

	p.BuildMappings(mappings)
}

// Lang returns the canonical name of the language
func (p *JavaScriptProvider) Lang() string {
	return "javascript"
}

// Aliases returns alternative names for this language
func (p *JavaScriptProvider) Aliases() []string {
	return []string{"javascript", "js", "node", "nodejs"}
}

// Extensions returns file extensions for this language
func (p *JavaScriptProvider) Extensions() []string {
	return []string{".js", ".mjs", ".cjs", ".jsx"}
}

// GetSitterLanguage returns the Tree-sitter language for JavaScript
func (p *JavaScriptProvider) GetSitterLanguage() *sitter.Language {
	return javascript_sitter.GetLanguage()
}

// TranslateQuery translates a universal query to JavaScript-specific Tree-sitter query
func (p *JavaScriptProvider) TranslateQuery(q *core.Query) (string, error) {
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
func (p *JavaScriptProvider) translateLogicalQuery(q *core.Query) (string, error) {
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
func (p *JavaScriptProvider) translateNegatedQuery(q *core.Query) (string, error) {
	// For NOT operations, we still generate the positive query
	// The negation is handled at the evaluation level
	// Create a copy without the NOT operator
	childQuery := *q
	childQuery.Operator = ""
	
	return p.TranslateQuery(&childQuery)
}

// translateHierarchicalQuery handles parent > child relationships
func (p *JavaScriptProvider) translateHierarchicalQuery(q *core.Query) (string, error) {
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
func (p *JavaScriptProvider) buildHierarchicalQuery(parentMapping provider.NodeMapping, parent *core.Query, childMapping provider.NodeMapping, child *core.Query) string {
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

	// Build the hierarchical query pattern
	// This is a simplified approach - in practice, you'd need more sophisticated nesting
	for _, parentType := range parentMapping.NodeTypes {
		for _, childType := range childMapping.NodeTypes {
			if parentType == "class_declaration" && (childType == "method_definition" || childType == "field_definition" || childType == "public_field_definition") {
				return fmt.Sprintf(`(%s body: (class_body (%s %s %s)))`,
					parentType, childType, childMapping.NameCapture, constraintStr)
			}
			if parentType == "function_declaration" && childType == "variable_declarator" {
				return fmt.Sprintf(`(%s body: (statement_block (lexical_declaration (variable_declarator %s %s))))`,
					parentType, childMapping.NameCapture, constraintStr)
			}
		}
	}

	// Generic hierarchical pattern
	return fmt.Sprintf(`(%s (%s %s %s))`,
		parentMapping.NodeTypes[0], childMapping.NodeTypes[0], childMapping.NameCapture, constraintStr)
}

// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *JavaScriptProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	switch node.Type() {
	case "function_declaration", "function", "arrow_function", "generator_function_declaration":
		return core.KindFunction
	case "class_declaration", "class":
		return core.KindClass
	case "method_definition":
		return core.KindMethod
	case "variable_declarator":
		return core.KindVariable
	case "lexical_declaration":
		// Check if it's a const
		if p.isConstDeclaration(node) {
			return core.KindConstant
		}
		return core.KindVariable
	case "import_statement", "import_specifier":
		return core.KindImport
	case "field_definition", "public_field_definition":
		return core.KindField
	case "call_expression":
		return core.KindCall
	case "assignment_expression", "augmented_assignment_expression":
		return core.KindAssignment
	case "if_statement", "ternary_expression", "switch_statement":
		return core.KindCondition
	case "for_statement", "for_in_statement", "while_statement", "do_statement":
		return core.KindLoop
	case "statement_block":
		return core.KindBlock
	case "comment":
		return core.KindComment
	case "decorator":
		return core.KindDecorator
	case "type_annotation", "type_identifier":
		return core.KindType
	default:
		return core.NodeKind(node.Type()) // Fallback to node type
	}
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *JavaScriptProvider) GetNodeName(node *sitter.Node, source []byte) string {
	switch node.Type() {
	case "function_declaration", "generator_function_declaration", "class_declaration":
		return p.extractIdentifier(node, "name", source)
	case "function", "class":
		// Anonymous functions/classes
		if name := node.ChildByFieldName("name"); name != nil {
			return name.Content(source)
		}
		return "<anonymous>"
	case "arrow_function":
		// Arrow functions are usually anonymous
		return "<arrow>"
	case "method_definition":
		return p.extractPropertyIdentifier(node, "name", source)
	case "variable_declarator":
		return p.extractVariableName(node, source)
	case "import_statement":
		return p.extractImportSource(node, source)
	case "field_definition", "public_field_definition":
		return p.extractPropertyIdentifier(node, "property", source)
	case "call_expression":
		return p.extractCallName(node, source)
	case "assignment_expression", "augmented_assignment_expression":
		return p.extractAssignmentTarget(node, source)
	default:
		// Fallback: try to get text content
		return node.Content(source)
	}
}

// GetNodeScope provides JavaScript-specific scope detection
func (p *JavaScriptProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
	switch node.Type() {
	case "program":
		return core.ScopeFile
	case "class_declaration", "class":
		return core.ScopeClass
	case "function_declaration", "function", "arrow_function", "method_definition":
		return core.ScopeFunction
	case "statement_block", "if_statement", "for_statement", "while_statement":
		return core.ScopeBlock
	default:
		if node.Parent() != nil {
			return p.GetNodeScope(node.Parent())
		}
		return core.ScopeFile
	}
}

// Helper methods for name extraction
func (p *JavaScriptProvider) extractIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

func (p *JavaScriptProvider) extractPropertyIdentifier(node *sitter.Node, fieldName string, source []byte) string {
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

func (p *JavaScriptProvider) extractVariableName(node *sitter.Node, source []byte) string {
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

func (p *JavaScriptProvider) extractImportSource(node *sitter.Node, source []byte) string {
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

func (p *JavaScriptProvider) extractCallName(node *sitter.Node, source []byte) string {
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

func (p *JavaScriptProvider) extractAssignmentTarget(node *sitter.Node, source []byte) string {
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

func (p *JavaScriptProvider) isConstDeclaration(node *sitter.Node) bool {
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

// QuickCheck performs basic syntax and semantic checks on JavaScript code
func (p *JavaScriptProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
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
func (p *JavaScriptProvider) checkForErrors(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
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
func (p *JavaScriptProvider) checkBasicSemantics(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
	// Check for function declarations without bodies
	if node.Type() == "function_declaration" {
		if body := node.ChildByFieldName("body"); body == nil {
			*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
				Severity: "error",
				Message:  "Function declaration missing body",
				Line:     int(node.StartPoint().Row) + 1,
				Column:   int(node.StartPoint().Column) + 1,
				Code:     "missing-function-body",
			})
		}
	}

	// Check for class declarations without bodies
	if node.Type() == "class_declaration" {
		if body := node.ChildByFieldName("body"); body == nil {
			*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
				Severity: "error",
				Message:  "Class declaration missing body",
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
