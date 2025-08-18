package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/garaekz/fileman/internal/types"
)

// UniversalParser provides language-agnostic DSL parsing capabilities.
// It translates universal DSL queries into structured Query objects that
// can be processed by language-specific providers.
type UniversalParser struct {
	// supportedKinds defines the set of universal node kinds supported by the parser
	supportedKinds map[types.NodeKind]bool
	// operators defines the supported query operators
	operators map[string]bool
}

// NewUniversalParser creates a new instance of the universal DSL parser
func NewUniversalParser() *UniversalParser {
	return &UniversalParser{
		supportedKinds: map[types.NodeKind]bool{
			// Full names (primary)
			"function":   true,
			"variable":   true,
			"class":      true,
			"method":     true,
			"field":      true,
			"import":     true,
			"call":       true,
			"assignment": true,
			"condition":  true,
			"if":         true,
			"block":      true,
			"loop":       true,
			"struct":     true,
			"const":      true,
			"constant":   true, // Support both const and constant
			// Legacy short names (for backward compatibility)
			"func":   true, // -> function
			"var":    true, // -> variable
			"assign": true, // -> assignment
		},
		operators: map[string]bool{
			">": true, // hierarchical (parent > child)
			"!": true, // negation
			"&": true, // logical AND
			"|": true, // logical OR
			"(": true, // grouping
			")": true, // grouping
		},
	}
}

// ParseQueryWithProvider parses a DSL query string using language-specific provider
// This allows for language-specific DSL terms (Go: func:, Python: def:, etc.)
func (p *UniversalParser) ParseQueryWithProvider(dsl string, provider types.LanguageProvider) (*types.Query, error) {
	if dsl == "" {
		return nil, fmt.Errorf("empty query string")
	}

	// Normalize whitespace
	dsl = strings.TrimSpace(dsl)
	dsl = regexp.MustCompile(`\s+`).ReplaceAllString(dsl, " ")

	// Check for negation
	negated := false
	if strings.HasPrefix(dsl, "!") {
		negated = true
		dsl = strings.TrimSpace(dsl[1:])
	}

	// Check for hierarchical queries (parent > child)
	if strings.Contains(dsl, ">") {
		return p.parseHierarchicalQueryWithProvider(dsl, negated, provider)
	}

	// Check for logical operators (AND, OR)
	if strings.Contains(dsl, "&") || strings.Contains(dsl, "|") {
		return p.parseLogicalQueryWithProvider(dsl, negated, provider)
	}

	// Parse simple query
	return p.parseSimpleQueryWithProvider(dsl, negated, provider)
}

// ParseQuery parses a universal DSL query string into a structured Query object
// Examples:
//   - "function:main" -> Query{Kind: "function", Pattern: "main"}
//   - "variable:* string" -> Query{Kind: "variable", Pattern: "*", Type: "string"}
//   - "class:User > method:getName" -> hierarchical query
//   - "!function:test*" -> negated query with wildcard
func (p *UniversalParser) ParseQuery(dsl string) (*types.Query, error) {
	if dsl == "" {
		return nil, fmt.Errorf("empty query string")
	}

	// Normalize whitespace
	dsl = strings.TrimSpace(dsl)
	dsl = regexp.MustCompile(`\s+`).ReplaceAllString(dsl, " ")

	// Check for negation
	negated := false
	if strings.HasPrefix(dsl, "!") {
		negated = true
		dsl = strings.TrimSpace(dsl[1:])
	}

	// Check for hierarchical queries (parent > child)
	if strings.Contains(dsl, ">") {
		return p.parseHierarchicalQuery(dsl, negated)
	}

	// Check for logical operators (AND, OR)
	if strings.Contains(dsl, "&") || strings.Contains(dsl, "|") {
		return p.parseLogicalQuery(dsl, negated)
	}

	// Parse simple query
	return p.parseSimpleQuery(dsl, negated)
}

// parseSimpleQuery parses a basic DSL query without operators
// Format: "kind:pattern [type] [constraints...]"
func (p *UniversalParser) parseSimpleQuery(dsl string, negated bool) (*types.Query, error) {
	parts := strings.Fields(dsl)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid query format")
	}

	// Parse "kind:pattern" format
	kindPattern := parts[0]
	if !strings.Contains(kindPattern, ":") {
		return nil, fmt.Errorf("invalid query format: expected 'kind:pattern', got '%s'", kindPattern)
	}

	kindParts := strings.SplitN(kindPattern, ":", 2)
	if len(kindParts) != 2 {
		return nil, fmt.Errorf("invalid query format: expected 'kind:pattern', got '%s'", kindPattern)
	}

	kind := types.NodeKind(kindParts[0])
	pattern := kindParts[1]

	if !p.supportedKinds[kind] {
		return nil, fmt.Errorf("unsupported node kind: %s", kind)
	}

	// Normalize legacy kind names to full names
	kind = p.normalizeKind(kind)

	query := &types.Query{
		Kind:       kind,
		Pattern:    pattern,
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation through operator
	if negated {
		query.Operator = "!"
	}

	// Parse type (second part)
	if len(parts) > 1 {
		query.Attributes["type"] = parts[1]
	}

	// Parse additional constraints as attributes
	for i := 2; i < len(parts); i++ {
		query.Attributes[fmt.Sprintf("constraint_%d", i-1)] = parts[i]
	}

	return query, nil
}

// parseHierarchicalQuery parses queries with parent > child relationships
// Format: "parent_kind parent_name > child_kind child_name"
func (p *UniversalParser) parseHierarchicalQuery(dsl string, negated bool) (*types.Query, error) {
	parts := strings.Split(dsl, ">")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid hierarchical query format")
	}

	// Parse parent query
	parentDSL := strings.TrimSpace(parts[0])
	parent, err := p.parseSimpleQuery(parentDSL, false)
	if err != nil {
		return nil, fmt.Errorf("invalid parent query: %w", err)
	}

	// Parse child query
	childDSL := strings.TrimSpace(parts[1])
	child, err := p.parseSimpleQuery(childDSL, false)
	if err != nil {
		return nil, fmt.Errorf("invalid child query: %w", err)
	}

	// Create hierarchical query with child as main and parent as nested
	query := &types.Query{
		Kind:       child.Kind,
		Pattern:    child.Pattern,
		Attributes: child.Attributes,
		Operator:   ">",
		Children:   []types.Query{*parent},
		Raw:        dsl,
	}

	// Handle negation
	if negated {
		query.Operator = "!>" // Combine negation with hierarchy
	}

	return query, nil
}

// parseLogicalQuery parses queries with logical operators (AND, OR)
// Format: "query1 & query2" or "query1 | query2"
func (p *UniversalParser) parseLogicalQuery(dsl string, negated bool) (*types.Query, error) {
	// For now, implement basic AND/OR support
	// This can be extended for more complex logical expressions

	var operator string
	var parts []string

	if strings.Contains(dsl, "&") {
		operator = "&&"
		parts = strings.Split(dsl, "&")
	} else if strings.Contains(dsl, "|") {
		operator = "||"
		parts = strings.Split(dsl, "|")
	} else {
		return nil, fmt.Errorf("no logical operator found")
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("logical queries must have exactly two operands")
	}

	// Parse left operand
	leftDSL := strings.TrimSpace(parts[0])
	left, err := p.ParseQuery(leftDSL)
	if err != nil {
		return nil, fmt.Errorf("invalid left operand: %w", err)
	}

	// Parse right operand
	rightDSL := strings.TrimSpace(parts[1])
	right, err := p.ParseQuery(rightDSL)
	if err != nil {
		return nil, fmt.Errorf("invalid right operand: %w", err)
	}

	// Create logical query with children
	query := &types.Query{
		Kind:       "logical",
		Operator:   operator,
		Children:   []types.Query{*left, *right},
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation
	if negated {
		query.Operator = "!" + operator
	}

	return query, nil
}

// ValidateQuery validates a DSL query string without parsing it
func (p *UniversalParser) ValidateQuery(dsl string) error {
	_, err := p.ParseQuery(dsl)
	return err
}

// GetSupportedKinds returns the list of supported node kinds
func (p *UniversalParser) GetSupportedKinds() []types.NodeKind {
	kinds := make([]types.NodeKind, 0, len(p.supportedKinds))
	for kind := range p.supportedKinds {
		kinds = append(kinds, kind)
	}
	return kinds
}

// GetSupportedOperators returns the list of supported query operators
func (p *UniversalParser) GetSupportedOperators() []string {
	operators := make([]string, 0, len(p.operators))
	for op := range p.operators {
		operators = append(operators, op)
	}
	return operators
}

// IsWildcard checks if a name contains wildcard patterns
func (p *UniversalParser) IsWildcard(name string) bool {
	return strings.Contains(name, "*") || strings.Contains(name, "?")
}

// NormalizeQuery normalizes a query string by removing extra whitespace
// and standardizing format
func (p *UniversalParser) NormalizeQuery(dsl string) string {
	dsl = strings.TrimSpace(dsl)
	dsl = regexp.MustCompile(`\s+`).ReplaceAllString(dsl, " ")
	return dsl
}

// parseSimpleQueryWithProvider parses a basic DSL query with language provider
func (p *UniversalParser) parseSimpleQueryWithProvider(dsl string, negated bool, provider types.LanguageProvider) (*types.Query, error) {
	parts := strings.Fields(dsl)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid query format")
	}

	// Parse "kind:pattern" format
	kindPattern := parts[0]
	if !strings.Contains(kindPattern, ":") {
		return nil, fmt.Errorf("invalid query format: expected 'kind:pattern', got '%s'", kindPattern)
	}

	kindParts := strings.SplitN(kindPattern, ":", 2)
	if len(kindParts) != 2 {
		return nil, fmt.Errorf("invalid query format: expected 'kind:pattern', got '%s'", kindPattern)
	}

	dslKind := kindParts[0]
	pattern := kindParts[1]

	// Use provider to normalize language-specific DSL to universal kind
	kind := provider.NormalizeDSLKind(dslKind)

	// Check if normalized kind is supported
	if !p.supportedKinds[kind] {
		return nil, fmt.Errorf("unsupported node kind: %s (DSL: %s)", kind, dslKind)
	}

	query := &types.Query{
		Kind:       kind,
		Pattern:    pattern,
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation through operator
	if negated {
		query.Operator = "!"
	}

	// Parse type (second part)
	if len(parts) > 1 {
		query.Attributes["type"] = parts[1]
	}

	// Parse additional constraints as attributes
	for i := 2; i < len(parts); i++ {
		query.Attributes[fmt.Sprintf("constraint_%d", i-1)] = parts[i]
	}

	return query, nil
}

// parseHierarchicalQueryWithProvider parses queries with parent > child relationships using provider
func (p *UniversalParser) parseHierarchicalQueryWithProvider(dsl string, negated bool, provider types.LanguageProvider) (*types.Query, error) {
	parts := strings.Split(dsl, ">")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid hierarchical query format")
	}

	// Parse parent query
	parentDSL := strings.TrimSpace(parts[0])
	parent, err := p.parseSimpleQueryWithProvider(parentDSL, false, provider)
	if err != nil {
		return nil, fmt.Errorf("invalid parent query: %w", err)
	}

	// Parse child query
	childDSL := strings.TrimSpace(parts[1])
	child, err := p.parseSimpleQueryWithProvider(childDSL, false, provider)
	if err != nil {
		return nil, fmt.Errorf("invalid child query: %w", err)
	}

	// Create hierarchical query with child as main and parent as nested
	query := &types.Query{
		Kind:       child.Kind,
		Pattern:    child.Pattern,
		Attributes: child.Attributes,
		Operator:   ">",
		Children:   []types.Query{*parent},
		Raw:        dsl,
	}

	// Handle negation
	if negated {
		query.Operator = "!>" // Combine negation with hierarchy
	}

	return query, nil
}

// parseLogicalQueryWithProvider parses queries with logical operators using provider
func (p *UniversalParser) parseLogicalQueryWithProvider(dsl string, negated bool, provider types.LanguageProvider) (*types.Query, error) {
	var operator string
	var parts []string

	if strings.Contains(dsl, "&") {
		operator = "&&"
		parts = strings.Split(dsl, "&")
	} else if strings.Contains(dsl, "|") {
		operator = "||"
		parts = strings.Split(dsl, "|")
	} else {
		return nil, fmt.Errorf("no logical operator found")
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("logical queries must have exactly two operands")
	}

	// Parse left operand
	leftDSL := strings.TrimSpace(parts[0])
	left, err := p.ParseQueryWithProvider(leftDSL, provider)
	if err != nil {
		return nil, fmt.Errorf("invalid left operand: %w", err)
	}

	// Parse right operand
	rightDSL := strings.TrimSpace(parts[1])
	right, err := p.ParseQueryWithProvider(rightDSL, provider)
	if err != nil {
		return nil, fmt.Errorf("invalid right operand: %w", err)
	}

	// Create logical query with children
	query := &types.Query{
		Kind:       "logical",
		Operator:   operator,
		Children:   []types.Query{*left, *right},
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation
	if negated {
		query.Operator = "!" + operator
	}

	return query, nil
}

// normalizeKind converts legacy short kind names to their full equivalents
func (p *UniversalParser) normalizeKind(kind types.NodeKind) types.NodeKind {
	switch kind {
	case "func":
		return "function"
	case "var":
		return "variable"
	case "assign":
		return "assignment"
	default:
		return kind
	}
}
