package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/termfx/morfx/internal/core"
)

// UniversalParser provides completely language-agnostic DSL parsing capabilities.
// It translates universal DSL queries into structured core.Query objects that
// can be processed by any language provider without language-specific knowledge.
//
// Design Principles:
// - Accept ALL common programming terms from different languages
// - Map everything to universal core.NodeKind constants
// - Support single-character operators as primary (for efficiency)
// - Provide aliases for familiarity (&&, ||, and, or, not)
// - Produce only pure core.Query structs with zero language coupling
type UniversalParser struct {
	// kindAliases maps all common programming terms to universal kinds
	kindAliases map[string]core.NodeKind
	// operatorAliases maps all operator variations to normalized forms
	operatorAliases map[string]string
	// supportedKinds defines the set of universal node kinds supported
	supportedKinds map[core.NodeKind]bool
}

// NewUniversalParser creates a new instance of the completely language-agnostic parser.
// It initializes comprehensive mappings for all common programming language constructs.
func NewUniversalParser() *UniversalParser {
	p := &UniversalParser{
		kindAliases:     make(map[string]core.NodeKind),
		operatorAliases: make(map[string]string),
		supportedKinds:  make(map[core.NodeKind]bool),
	}

	p.initializeKindAliases()
	p.initializeOperatorAliases()
	p.initializeSupportedKinds()

	return p
}

// initializeKindAliases sets up comprehensive mappings from all common programming
// terms across different languages to universal core.NodeKind constants.
func (p *UniversalParser) initializeKindAliases() {
	// Function mappings - covers all major languages
	p.kindAliases["function"] = core.KindFunction  // JavaScript, TypeScript, Python
	p.kindAliases["func"] = core.KindFunction      // Go, Swift
	p.kindAliases["def"] = core.KindFunction       // Python, Ruby
	p.kindAliases["fn"] = core.KindFunction        // Rust, Scala
	p.kindAliases["sub"] = core.KindFunction       // Perl, VB
	p.kindAliases["procedure"] = core.KindFunction // Pascal, Ada
	p.kindAliases["method"] = core.KindMethod      // All OOP languages

	// Variable mappings - covers different declaration styles
	p.kindAliases["variable"] = core.KindVariable  // Generic term
	p.kindAliases["var"] = core.KindVariable       // JavaScript, Go, Pascal
	p.kindAliases["let"] = core.KindVariable       // JavaScript, TypeScript, Swift
	p.kindAliases["const"] = core.KindConstant     // JavaScript, TypeScript, C++
	p.kindAliases["constant"] = core.KindConstant  // Generic term
	p.kindAliases["final"] = core.KindConstant     // Java, Dart
	p.kindAliases["readonly"] = core.KindConstant  // C#, TypeScript
	p.kindAliases["immutable"] = core.KindConstant // D, Scala

	// Class and type mappings - covers OOP and structural typing
	p.kindAliases["class"] = core.KindClass         // Most OOP languages
	p.kindAliases["cls"] = core.KindClass           // Abbreviated form
	p.kindAliases["struct"] = core.KindClass        // Go, C, C++, Rust
	p.kindAliases["type"] = core.KindType           // Go, TypeScript, Haskell
	p.kindAliases["interface"] = core.KindInterface // Go, Java, C#, TypeScript
	p.kindAliases["protocol"] = core.KindInterface  // Swift, Objective-C
	p.kindAliases["trait"] = core.KindInterface     // Rust, Scala
	p.kindAliases["enum"] = core.KindEnum           // Most languages
	p.kindAliases["enumeration"] = core.KindEnum    // Verbose form

	// Import mappings - covers different module systems
	p.kindAliases["import"] = core.KindImport  // Python, Java, JavaScript
	p.kindAliases["require"] = core.KindImport // Node.js, Ruby
	p.kindAliases["include"] = core.KindImport // C/C++, PHP, Ruby
	p.kindAliases["use"] = core.KindImport     // Rust, PHP, C#
	p.kindAliases["using"] = core.KindImport   // C#, C++
	p.kindAliases["from"] = core.KindImport    // Python import variations

	// Field and property mappings
	p.kindAliases["field"] = core.KindField     // Generic term
	p.kindAliases["property"] = core.KindField  // C#, Python, JavaScript
	p.kindAliases["attribute"] = core.KindField // Python, XML contexts
	p.kindAliases["member"] = core.KindField    // C++, C#
	p.kindAliases["slot"] = core.KindField      // Lisp, some OOP languages

	// Function call mappings
	p.kindAliases["call"] = core.KindCall    // Generic term
	p.kindAliases["invoke"] = core.KindCall  // Java, C#
	p.kindAliases["apply"] = core.KindCall   // Functional languages
	p.kindAliases["execute"] = core.KindCall // Generic term

	// Assignment mappings
	p.kindAliases["assignment"] = core.KindAssignment // Generic term
	p.kindAliases["assign"] = core.KindAssignment     // Shortened form
	p.kindAliases["set"] = core.KindAssignment        // Setter context

	// Control flow mappings
	p.kindAliases["condition"] = core.KindCondition // Generic term
	p.kindAliases["if"] = core.KindCondition        // Most languages
	p.kindAliases["switch"] = core.KindCondition    // C-family, JavaScript
	p.kindAliases["case"] = core.KindCondition      // Switch case
	p.kindAliases["when"] = core.KindCondition      // Ruby, Kotlin
	p.kindAliases["match"] = core.KindCondition     // Rust, Scala pattern matching

	// Loop mappings
	p.kindAliases["loop"] = core.KindLoop    // Generic term
	p.kindAliases["for"] = core.KindLoop     // Most languages
	p.kindAliases["while"] = core.KindLoop   // Most languages
	p.kindAliases["do"] = core.KindLoop      // do-while loops
	p.kindAliases["foreach"] = core.KindLoop // C#, PHP
	p.kindAliases["repeat"] = core.KindLoop  // Pascal, some languages

	// Block and scope mappings
	p.kindAliases["block"] = core.KindBlock // Generic term
	p.kindAliases["scope"] = core.KindBlock // Conceptual grouping
	p.kindAliases["begin"] = core.KindBlock // Pascal, some languages
	p.kindAliases["end"] = core.KindBlock   // Block terminators

	// Comment mappings
	p.kindAliases["comment"] = core.KindComment       // Generic term
	p.kindAliases["doc"] = core.KindComment           // Documentation
	p.kindAliases["documentation"] = core.KindComment // Full form

	// Decorator/annotation mappings
	p.kindAliases["decorator"] = core.KindDecorator  // Python
	p.kindAliases["annotation"] = core.KindDecorator // Java, C#

	// Exception handling mappings
	p.kindAliases["try"] = core.KindTryCatch     // Most languages
	p.kindAliases["catch"] = core.KindTryCatch   // Most languages
	p.kindAliases["except"] = core.KindTryCatch  // Python
	p.kindAliases["rescue"] = core.KindTryCatch  // Ruby
	p.kindAliases["finally"] = core.KindTryCatch // Cleanup blocks

	// Return and throw mappings
	p.kindAliases["return"] = core.KindReturn // Most languages
	p.kindAliases["yield"] = core.KindReturn  // Generator context
	p.kindAliases["throw"] = core.KindThrow   // JavaScript, Java, C#
	p.kindAliases["raise"] = core.KindThrow   // Python, Ruby
	p.kindAliases["panic"] = core.KindThrow   // Go, Rust

	// Parameter mappings
	p.kindAliases["parameter"] = core.KindParameter // Generic term
	p.kindAliases["param"] = core.KindParameter     // Shortened form
	p.kindAliases["argument"] = core.KindParameter  // Function arguments
	p.kindAliases["arg"] = core.KindParameter       // Shortened form
}

// initializeOperatorAliases sets up comprehensive mappings for all operator variations.
// Primary operators are single characters for CLI efficiency, with familiar aliases.
func (p *UniversalParser) initializeOperatorAliases() {
	// AND operator variations - & is primary for efficiency
	p.operatorAliases["&"] = "AND"   // Primary - single character for CLI efficiency
	p.operatorAliases["&&"] = "AND"  // C-family languages, JavaScript
	p.operatorAliases["and"] = "AND" // Python, Ruby, English-like languages
	p.operatorAliases["AND"] = "AND" // Already normalized

	// OR operator variations - | is primary for efficiency
	p.operatorAliases["|"] = "OR"  // Primary - single character for CLI efficiency
	p.operatorAliases["||"] = "OR" // C-family languages, JavaScript
	p.operatorAliases["or"] = "OR" // Python, Ruby, English-like languages
	p.operatorAliases["OR"] = "OR" // Already normalized

	// NOT operator variations - ! is primary for efficiency
	p.operatorAliases["!"] = "NOT"   // Primary - single character for CLI efficiency
	p.operatorAliases["not"] = "NOT" // Python, Ruby, English-like languages
	p.operatorAliases["NOT"] = "NOT" // Already normalized

	// HIERARCHY operator - > is primary and only form
	p.operatorAliases[">"] = "HIERARCHY"         // Parent > child relationships
	p.operatorAliases["HIERARCHY"] = "HIERARCHY" // Already normalized
}

// initializeSupportedKinds populates the set of supported universal node kinds.
func (p *UniversalParser) initializeSupportedKinds() {
	supportedKindsList := []core.NodeKind{
		core.KindFunction, core.KindVariable, core.KindClass, core.KindMethod,
		core.KindImport, core.KindConstant, core.KindField, core.KindCall,
		core.KindAssignment, core.KindCondition, core.KindLoop, core.KindBlock,
		core.KindComment, core.KindDecorator, core.KindType, core.KindInterface,
		core.KindEnum, core.KindParameter, core.KindReturn, core.KindThrow,
		core.KindTryCatch,
	}

	for _, kind := range supportedKindsList {
		p.supportedKinds[kind] = true
	}
}

// ParseQuery parses a universal DSL query string into a structured core.Query object.
// This method is completely language-agnostic and works with ALL programming languages.
//
// Examples of supported syntax:
//   - "function:main" -> Query{Kind: KindFunction, Pattern: "main"}
//   - "def:test*" -> Query{Kind: KindFunction, Pattern: "test*"} (Python style)
//   - "func:Test* & !struct:mock" -> Complex query with AND and NOT (Go style)
//   - "func:Test* && !struct:mock" -> Same as above with double operators
//   - "class:User > method:getName" -> Hierarchical query
//   - "variable:* string" -> Query with type constraint
func (p *UniversalParser) ParseQuery(dsl string) (*core.Query, error) {
	if dsl == "" {
		return nil, fmt.Errorf("empty query string")
	}

	// Normalize whitespace
	dsl = strings.TrimSpace(dsl)
	dsl = regexp.MustCompile(`\s+`).ReplaceAllString(dsl, " ")

	// Check for negation at the beginning
	negated := false
	if strings.HasPrefix(dsl, "!") || strings.HasPrefix(strings.ToLower(dsl), "not ") {
		if strings.HasPrefix(dsl, "!") {
			negated = true
			dsl = strings.TrimSpace(dsl[1:])
		} else if strings.HasPrefix(strings.ToLower(dsl), "not ") {
			negated = true
			dsl = strings.TrimSpace(dsl[4:])
		}
	}

	// Check for hierarchical queries (parent > child)
	if strings.Contains(dsl, ">") {
		return p.parseHierarchicalQuery(dsl, negated)
	}

	// Check for logical operators (AND, OR) - support all variations
	if p.containsLogicalOperator(dsl) {
		return p.parseLogicalQuery(dsl, negated)
	}

	// Parse simple query
	return p.parseSimpleQuery(dsl, negated)
}

// containsLogicalOperator checks if the DSL contains any logical operator variation
func (p *UniversalParser) containsLogicalOperator(dsl string) bool {
	lowerDSL := strings.ToLower(dsl)
	return strings.Contains(dsl, "&") ||
		strings.Contains(dsl, "|") ||
		strings.Contains(lowerDSL, " and ") ||
		strings.Contains(lowerDSL, " or ")
}

// parseSimpleQuery parses a basic DSL query without operators.
// Format: "kind:pattern [type] [constraints...]"
func (p *UniversalParser) parseSimpleQuery(dsl string, negated bool) (*core.Query, error) {
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

	kindAlias := strings.ToLower(kindParts[0])
	pattern := kindParts[1]

	// Map alias to universal kind
	kind, exists := p.kindAliases[kindAlias]
	if !exists {
		return nil, fmt.Errorf("unsupported node kind: %s (supported: %v)", kindAlias, p.getSupportedAliases())
	}

	// Verify the mapped kind is supported
	if !p.supportedKinds[kind] {
		return nil, fmt.Errorf("mapped kind not supported: %s", kind)
	}

	query := &core.Query{
		Kind:       kind,
		Pattern:    pattern,
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation through operator
	if negated {
		query.Operator = "NOT"
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

// parseHierarchicalQuery parses queries with parent > child relationships.
// Format: "parent_kind:parent_name > child_kind:child_name"
func (p *UniversalParser) parseHierarchicalQuery(dsl string, negated bool) (*core.Query, error) {
	parts := strings.Split(dsl, ">")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid hierarchical query format: expected 'parent > child', got '%s'", dsl)
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
	query := &core.Query{
		Kind:       child.Kind,
		Pattern:    child.Pattern,
		Attributes: child.Attributes,
		Operator:   "HIERARCHY",
		Children:   []core.Query{*parent},
		Raw:        dsl,
	}

	// Handle negation by combining with hierarchy
	if negated {
		query.Operator = "NOT"
		// Store the hierarchical relationship in children with specific operator
		hierarchyQuery := &core.Query{
			Kind:       child.Kind,
			Pattern:    child.Pattern,
			Attributes: child.Attributes,
			Operator:   "HIERARCHY",
			Children:   []core.Query{*parent},
			Raw:        dsl,
		}
		query.Children = []core.Query{*hierarchyQuery}
	}

	return query, nil
}

// parseLogicalQuery parses queries with logical operators (AND, OR).
// Supports all operator variations: &, &&, and, |, ||, or
func (p *UniversalParser) parseLogicalQuery(dsl string, negated bool) (*core.Query, error) {
	var operator string
	var parts []string

	// Determine operator and split - try all variations
	lowerDSL := strings.ToLower(dsl)
	if strings.Contains(lowerDSL, " and ") {
		operator = "AND"
		parts = p.splitByOperator(dsl, []string{" and ", " AND "})
	} else if strings.Contains(lowerDSL, " or ") {
		operator = "OR"
		parts = p.splitByOperator(dsl, []string{" or ", " OR "})
	} else if strings.Contains(dsl, "&&") {
		operator = "AND"
		parts = strings.Split(dsl, "&&")
	} else if strings.Contains(dsl, "||") {
		operator = "OR"
		parts = strings.Split(dsl, "||")
	} else if strings.Contains(dsl, "&") {
		operator = "AND"
		parts = strings.Split(dsl, "&")
	} else if strings.Contains(dsl, "|") {
		operator = "OR"
		parts = strings.Split(dsl, "|")
	} else {
		return nil, fmt.Errorf("no logical operator found in query: %s", dsl)
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("logical queries must have exactly two operands, got %d in: %s", len(parts), dsl)
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
	query := &core.Query{
		Kind:       "logical", // Special kind for logical operations
		Operator:   operator,
		Children:   []core.Query{*left, *right},
		Attributes: make(map[string]string),
		Raw:        dsl,
	}

	// Handle negation
	if negated {
		query.Operator = "NOT"
		// Store the original logical operation in children
		logicalQuery := &core.Query{
			Kind:       "logical",
			Operator:   operator,
			Children:   []core.Query{*left, *right},
			Attributes: make(map[string]string),
			Raw:        dsl,
		}
		query.Children = []core.Query{*logicalQuery}
	}

	return query, nil
}

// splitByOperator splits a string by any of the given operators (case-insensitive)
func (p *UniversalParser) splitByOperator(text string, operators []string) []string {
	for _, op := range operators {
		if strings.Contains(strings.ToLower(text), strings.ToLower(op)) {
			// Find the actual position in the original text
			lowerText := strings.ToLower(text)
			lowerOp := strings.ToLower(op)
			pos := strings.Index(lowerText, lowerOp)
			if pos >= 0 {
				return []string{
					text[:pos],
					text[pos+len(op):],
				}
			}
		}
	}
	return []string{text}
}

// ValidateQuery validates a DSL query string without parsing it
func (p *UniversalParser) ValidateQuery(dsl string) error {
	_, err := p.ParseQuery(dsl)
	return err
}

// GetSupportedKinds returns the list of supported universal node kinds
func (p *UniversalParser) GetSupportedKinds() []core.NodeKind {
	kinds := make([]core.NodeKind, 0, len(p.supportedKinds))
	for kind := range p.supportedKinds {
		kinds = append(kinds, kind)
	}
	return kinds
}

// GetSupportedAliases returns all supported kind aliases
func (p *UniversalParser) GetSupportedAliases() []string {
	aliases := make([]string, 0, len(p.kindAliases))
	for alias := range p.kindAliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// getSupportedAliases returns the first 10 supported aliases for error messages
func (p *UniversalParser) getSupportedAliases() []string {
	aliases := p.GetSupportedAliases()
	if len(aliases) > 10 {
		return aliases[:10]
	}
	return aliases
}

// GetSupportedOperators returns all supported operator variations
func (p *UniversalParser) GetSupportedOperators() []string {
	operators := make([]string, 0, len(p.operatorAliases))
	for op := range p.operatorAliases {
		operators = append(operators, op)
	}
	return operators
}

// NormalizeOperator converts any operator alias to its normalized form
func (p *UniversalParser) NormalizeOperator(operator string) string {
	if normalized, exists := p.operatorAliases[operator]; exists {
		return normalized
	}
	return operator
}

// IsWildcard checks if a pattern contains wildcard characters
func (p *UniversalParser) IsWildcard(pattern string) bool {
	return strings.Contains(pattern, "*") || strings.Contains(pattern, "?")
}

// NormalizeQuery normalizes a query string by removing extra whitespace
func (p *UniversalParser) NormalizeQuery(dsl string) string {
	dsl = strings.TrimSpace(dsl)
	dsl = regexp.MustCompile(`\s+`).ReplaceAllString(dsl, " ")
	return dsl
}
