package golang

import (
	"fmt"
	"strings"
)

// templates defines Tree-sitter query patterns for different Go language constructs.
// Each template corresponds to a DSL node type and includes capture points for identifiers,
// types, and other relevant information. Wildcard variants are provided for broad matching.
//
// Template structure:
//   - %s placeholders are filled with predicate strings during query building
//   - @name captures identify the primary identifier (function name, variable name, etc.)
//   - @type captures identify type information where applicable
//   - @path captures identify import paths
//   - Wildcard templates omit name captures for broad matching
//
// Supported node types:
//   - func: Function declarations with name capture
//   - const: Constant declarations with name capture
//   - var: Variable declarations with name and type captures (handles both single and list forms)
//   - struct: Struct type declarations with name capture
//   - field: Struct field declarations with name and type captures
//   - call: Function call expressions with name capture (supports both direct and selector calls)
//   - assign: Assignment statements (both regular and short variable declarations)
//   - if: If statements (wildcard only)
//   - import: Import specifications with path capture
//   - block: Block statements (wildcard only)
var templates = map[string]string{
	"func":            `(function_declaration name: (identifier) @name %s)`,
	"func_wildcard":   `(function_declaration)`,
	"const":           `(const_declaration (const_spec name: (identifier_list (identifier) @name)) %s)`,
	"const_wildcard":  `(const_declaration)`,
	"var":             `[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) %s) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) %s))]`,
	"var_wildcard":    `[(var_declaration (var_spec type: (type_identifier) @type) %s) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) %s))]`,
	"struct":          `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) %s)`,
	"struct_wildcard": `(type_declaration (type_spec type: (struct_type)))`,
	"field":           `[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type %s) (field_declaration name: (field_identifier) @name type: (type_identifier) @type %s)]`,
	"field_wildcard":  `(field_declaration type: (type_identifier) @type %s)`,
	"call":            `(call_expression function: [(identifier) (selector_expression)] @name %s)`,
	"call_wildcard":   `(call_expression)`,
	"assign":          `[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] %s`,
	"assign_wildcard": `[(assignment_statement) (short_var_declaration)]`,
	"if":              `(if_statement)`,
	"import":          `(import_spec path: (interpreted_string_literal) @path %s)`,
	"import_wildcard": `(import_spec)`,
	"block":           `(block)`,
}

// BuildTreeSitterQuery constructs a complete Tree-sitter query from a parsed DSL Query.
// This is the main entry point for converting DSL queries into executable Tree-sitter syntax.
//
// The function handles:
//   - Template selection based on node type and wildcard usage
//   - Predicate building for identifiers and types
//   - Hierarchical query construction for parent-child relationships
//   - Target annotation for result capture
//
// Parameters:
//   - q: The parsed DSL Query structure
//
// Returns:
//   - A complete Tree-sitter query string ready for execution
//   - An error if the query cannot be built (unknown node type, validation failures)
//
// Example transformations:
//   - DSL: "func:TestFunc" → Tree-sitter: `(function_declaration name: (identifier) @name (#eq? @name "TestFunc")) @target`
//   - DSL: "var:count int" → Tree-sitter: `[(var_declaration ...) ...] (#eq? @name "count") (#eq? @type "int") @target`
func BuildTreeSitterQuery(q *Query) (string, error) {
	return buildNodeQuery(q, true)
}

// buildNodeQuery recursively builds Tree-sitter query components from DSL Query structures.
// This internal function handles the core logic of query construction, including template
// selection, predicate insertion, and hierarchical query building.
//
// Key responsibilities:
//   - Template selection (regular vs wildcard variants)
//   - Predicate construction and insertion
//   - Child query processing and composition
//   - Target annotation management (only at root level)
//   - Query string cleaning and formatting
//
// Parameters:
//   - q: The Query structure to process
//   - isRoot: Whether this is the root query (affects target annotation)
//
// Returns:
//   - The constructed query string for this node and its children
//   - An error if construction fails
//
// The function implements several important patterns:
//   - Wildcard template selection when identifier is "*"
//   - Special handling for control structures (if/block)
//   - Import path vs name capture distinction
//   - Placeholder filling for predicate insertion
//   - Hierarchical composition with "." operator
func buildNodeQuery(q *Query, isRoot bool) (string, error) {
	// Step 1: Determine the appropriate template key
	// Start with the base node type, then check for wildcard variants
	key := q.NodeType
	if q.Identifier == "*" {
		// Check if a specific wildcard template exists for broader matching
		if _, ok := templates[key+"_wildcard"]; ok {
			key += "_wildcard"
		}
	}

	// Step 2: Retrieve the template pattern
	template, ok := templates[key]
	if !ok {
		return "", fmt.Errorf("unknown node type: %s", q.NodeType)
	}

	// Step 3: Validate control structure constraints
	// Control structures (if/block) only support wildcard matching
	if (q.NodeType == "if" || q.NodeType == "block") && q.Identifier != "*" {
		return "", fmt.Errorf("only * supported for if/block")
	}

	// Step 4: Build predicates for filtering
	var predicates []string
	if q.Identifier != "*" {
		// Determine the appropriate capture name based on node type
		capture := "@name"
		if q.NodeType == "import" {
			capture = "@path" // Import nodes use path capture instead of name
		}
		// Build the predicate using the predicate builder (handles wildcards, multi-target, negation)
		predicates = append(predicates, BuildPredicate(capture, q.Identifier, q.Not, q.NodeType))
	}

	// Add type predicate if specified (for variables and fields)
	if q.Type != "" {
		predicates = append(predicates, BuildPredicate("@type", q.Type, q.Not, q.NodeType))
	}

	// Step 5: Combine predicates into a single string
	predicateStr := strings.Join(predicates, " ")

	// Step 6: Insert predicates into the template
	query := template
	if strings.Contains(query, "%s") {
		// Template has placeholders - fill all of them with the same predicate string
		// This handles templates like var that have multiple %s placeholders
		numPlaceholders := strings.Count(query, "%s")
		args := make([]any, numPlaceholders)
		for i := range args {
			args[i] = predicateStr
		}
		query = fmt.Sprintf(query, args...)
	} else if predicateStr != "" {
		// Template has no placeholders - insert predicates before the closing parenthesis
		closingParen := strings.LastIndex(query, ")")
		if closingParen != -1 {
			query = query[:closingParen] + " " + predicateStr + query[closingParen:]
		} else {
			// Fallback: append to the end if no closing parenthesis found
			query += " " + predicateStr
		}
	}

	// Step 7: Handle hierarchical queries (parent > child relationships)
	if len(q.Children) > 0 {
		// Recursively build child query (children are never root nodes)
		childQuery, err := buildNodeQuery(&q.Children[0], false)
		if err != nil {
			return "", err
		}
		// Remove ALL @target occurrences from child fragment before embedding (Phase 2 requirement)
		for strings.Contains(childQuery, "@target") {
			childQuery = strings.Replace(childQuery, "@target", "", 1)
		}
		childQuery = strings.TrimSpace(childQuery)
		// Compose parent and child with Tree-sitter's "." operator for immediate child relationship
		query = fmt.Sprintf("%s . %s", query, childQuery)
	}

	// Step 8: Handle target annotation for root queries only
	if isRoot {
		// Only the root pattern gets @target capture (Phase 2 requirement)
		query += " @target"
	}

	// Step 9: Clean and return the final query string
	return CleanQueryString(query), nil
}
