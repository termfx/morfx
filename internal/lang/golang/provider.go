package golang

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	golang_sitter "github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// GoProvider implements the LanguageProvider interface for Go language support
type GoProvider struct {
	provider.BaseProvider
	// Go-specific DSL vocabulary mapping
	dslVocabulary map[string]core.NodeKind
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
	p.dslVocabulary = map[string]core.NodeKind{
		// Go-specific DSL terms
		"func":   core.KindFunction,
		"var":    core.KindVariable,
		"const":  core.KindConstant,
		"type":   core.KindClass, // Go structs/types map to universal class concept
		"import": core.KindImport,
		"call":   core.KindCall,
		"assign": core.KindAssignment,
		"if":     core.KindCondition,
		"for":    core.KindLoop,
		"method": core.KindMethod,
		"field":  core.KindField,
		"block":  core.KindBlock,
		"struct": core.KindClass, // Go struct maps to universal class concept
		// Also support universal terms for compatibility
		"function":   core.KindFunction,
		"variable":   core.KindVariable,
		"constant":   core.KindConstant,
		"class":      core.KindClass,
		"assignment": core.KindAssignment,
		"condition":  core.KindCondition,
	}

	// Define how universal kinds map to Go AST
	mappings := []provider.NodeMapping{
		{
			Kind:        core.KindFunction,
			NodeTypes:   []string{"function_declaration", "method_declaration"},
			NameCapture: "@name",
			Template:    `(function_declaration name: (identifier) @name %s)`,
		},
		{
			Kind:        core.KindVariable,
			NodeTypes:   []string{"var_declaration", "short_var_declaration"},
			NameCapture: "@name",
			TypeCapture: "@type",
			Template:    `(var_declaration (var_spec name: (identifier) @name %s))`,
		},
		{
			Kind:        core.KindClass,
			NodeTypes:   []string{"type_declaration"}, // Go uses structs
			NameCapture: "@name",
			Template:    `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type) %s))`,
		},
		{
			Kind:        core.KindMethod,
			NodeTypes:   []string{"method_declaration"},
			NameCapture: "@name",
			Template:    `(method_declaration name: (field_identifier) @name %s)`,
		},
		{
			Kind:        core.KindImport,
			NodeTypes:   []string{"import_declaration"},
			NameCapture: "@name",
			Template:    `(import_declaration (import_spec_list (import_spec path: (interpreted_string_literal) @name %s)))`,
		},
		{
			Kind:        core.KindConstant,
			NodeTypes:   []string{"const_declaration"},
			NameCapture: "@name",
			Template:    `(const_declaration (const_spec name: (identifier) @name %s))`,
		},
		{
			Kind:        core.KindField,
			NodeTypes:   []string{"field_declaration"},
			NameCapture: "@name",
			TypeCapture: "@type",
			Template:    `(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type %s)`,
		},
		{
			Kind:        core.KindCall,
			NodeTypes:   []string{"call_expression"},
			NameCapture: "@name",
			Template:    `(call_expression function: [(identifier) (selector_expression)] @name %s)`,
		},
		{
			Kind:        core.KindAssignment,
			NodeTypes:   []string{"assignment_statement"},
			NameCapture: "@name",
			Template:    `(assignment_statement left: (expression_list (identifier) @name) %s)`,
		},
		{
			Kind:        core.KindCondition,
			NodeTypes:   []string{"if_statement"},
			NameCapture: "@condition",
			Template:    `(if_statement condition: (_) @condition %s)`,
		},
		{
			Kind:        core.KindLoop,
			NodeTypes:   []string{"for_statement", "range_clause"},
			NameCapture: "@loop",
			Template:    `[(for_statement) (range_clause)] @loop %s`,
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
func (p *GoProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
	if universalKind, exists := p.dslVocabulary[dslKind]; exists {
		return universalKind
	}
	// Fallback to original if not found
	return core.NodeKind(dslKind)
}

// TranslateQuery translates a universal query to Go-specific Tree-sitter query
func (p *GoProvider) TranslateQuery(q *core.Query) (string, error) {
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

	// Normalize Go-specific DSL terms to universal kinds
	kind := p.NormalizeDSLKind(string(q.Kind))

	mappings := p.TranslateKind(kind)
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
func (p *GoProvider) translateLogicalQuery(q *core.Query) (string, error) {
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
func (p *GoProvider) translateNegatedQuery(q *core.Query) (string, error) {
	// For NOT operations, we still generate the positive query
	// The negation is handled at the evaluation level
	// Create a copy without the NOT operator
	childQuery := *q
	childQuery.Operator = ""
	
	return p.TranslateQuery(&childQuery)
}

// translateHierarchicalQuery handles parent > child relationships
func (p *GoProvider) translateHierarchicalQuery(q *core.Query) (string, error) {
	if len(q.Children) != 1 {
		return "", fmt.Errorf("hierarchical query must have exactly 1 parent")
	}

	parent := &q.Children[0]
	
	// Get parent mappings
	parentMappings := p.TranslateKind(p.NormalizeDSLKind(string(parent.Kind)))
	if len(parentMappings) == 0 {
		return "", fmt.Errorf("unsupported parent node kind: %s", parent.Kind)
	}
	
	// Get child mappings
	childMappings := p.TranslateKind(p.NormalizeDSLKind(string(q.Kind)))
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
func (p *GoProvider) buildHierarchicalQuery(parentMapping provider.NodeMapping, parent *core.Query, childMapping provider.NodeMapping, child *core.Query) string {
	// For hierarchical queries in Go, we need to generate the child query but ensure proper nesting
	// The approach is to generate a regular child query since tree-sitter handles the structural matching
	
	// Build child query with proper constraints
	childQuery := p.BuildQueryFromMapping(childMapping, child)
	if childQuery != "" {
		return childQuery
	}

	// Fallback to basic template
	constraintParts := []string{}
	
	// Add pattern constraint for child
	if child.Pattern != "" && child.Pattern != "*" {
		regexPattern := p.ConvertWildcardToRegex(child.Pattern)
		constraintParts = append(constraintParts, fmt.Sprintf(`(#match? %s "%s")`, childMapping.NameCapture, regexPattern))
	}
	
	// Add type constraint if available
	if typeAttr, hasType := child.Attributes["type"]; hasType && childMapping.TypeCapture != "" {
		typeRegex := p.ConvertWildcardToRegex(typeAttr)
		constraintParts = append(constraintParts, fmt.Sprintf(`(#match? %s "%s")`, childMapping.TypeCapture, typeRegex))
	}
	
	constraintStr := strings.Join(constraintParts, " ")
	
	// Use the child mapping template
	if childMapping.Template != "" {
		return fmt.Sprintf(childMapping.Template, constraintStr)
	}
	
	// Final fallback
	return fmt.Sprintf(`(%s %s %s)`, childMapping.NodeTypes[0], childMapping.NameCapture, constraintStr)
}

// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *GoProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	switch node.Type() {
	case "function_declaration":
		return core.KindFunction
	case "method_declaration":
		return core.KindMethod
	case "var_declaration", "short_var_declaration":
		return core.KindVariable
	case "type_declaration":
		return core.KindClass // Go structs map to universal class concept
	case "import_declaration":
		return core.KindImport
	case "const_declaration":
		return core.KindConstant
	case "field_declaration":
		return core.KindField
	case "call_expression":
		return core.KindCall
	case "assignment_statement":
		return core.KindAssignment
	case "if_statement":
		return core.KindCondition
	case "for_statement", "range_clause":
		return core.KindLoop
	case "block":
		return core.KindBlock
	case "comment":
		return core.KindComment
	case "type_identifier", "pointer_type", "slice_type", "array_type":
		return core.KindType
	default:
		return core.NodeKind(node.Type()) // Fallback to node type
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

// QuickCheck performs quick syntax and semantic checks
func (p *GoProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
	var diagnostics []core.QuickCheckDiagnostic

	// Parse the source code
	parser := sitter.NewParser()
	parser.SetLanguage(golang_sitter.GetLanguage())
	tree, err := parser.ParseCtx(nil, nil, source)
	if err != nil {
		diagnostics = append(diagnostics, core.QuickCheckDiagnostic{
			Severity: "error",
			Message:  fmt.Sprintf("Parse error: %v", err),
			Line:     1,
			Column:   1,
		})
		return diagnostics
	}
	defer tree.Close()

	// Check for syntax errors
	p.checkForErrors(tree.RootNode(), source, &diagnostics)

	// Basic semantic checks
	p.checkBasicSemantics(tree.RootNode(), source, &diagnostics)

	return diagnostics
}

// checkForErrors recursively finds ERROR nodes and reports them
func (p *GoProvider) checkForErrors(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
	if node.Type() == "ERROR" {
		startPoint := node.StartPoint()
		*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
			Severity: "error",
			Message:  "Syntax error",
			Line:     int(startPoint.Row) + 1,
			Column:   int(startPoint.Column) + 1,
		})
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.checkForErrors(node.Child(i), source, diagnostics)
	}
}

// checkBasicSemantics performs basic semantic validation
func (p *GoProvider) checkBasicSemantics(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
	// Check for common Go issues
	switch node.Type() {
	case "function_declaration":
		// Check if function has a body (not just a declaration)
		if node.ChildByFieldName("body") == nil {
			startPoint := node.StartPoint()
			*diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
				Severity: "warning",
				Message:  "Function declaration without body",
				Line:     int(startPoint.Row) + 1,
				Column:   int(startPoint.Column) + 1,
			})
		}
	case "import_declaration":
		// Check for unused imports (basic check)
		// This would need more sophisticated analysis in a full implementation
	}

	// Recursively check children
	for i := 0; i < int(node.ChildCount()); i++ {
		p.checkBasicSemantics(node.Child(i), source, diagnostics)
	}
}
