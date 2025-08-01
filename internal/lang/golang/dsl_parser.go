package golang

import (
	"fmt"
	"strings"
)

// Query represents a parsed DSL query.
type Query struct {
	Not        bool
	NodeType   string
	Identifier string
	Type       string
	Children   []Query
}

// HasNegationPredicates recursively checks if the query or any of its children
// contains negation predicates. This is used for query optimization and validation,
// as negated queries may require special handling in Tree-sitter.
//
// The method performs a depth-first traversal of the query tree to detect
// any negation flags, which is important for:
//   - Query performance optimization
//   - Validation of complex negated hierarchical queries
//   - Determining appropriate Tree-sitter predicate strategies
//
// Returns:
//   - true if this query or any descendant has Not=true
//   - false if no negation predicates are found in the entire query tree
func (q *Query) HasNegationPredicates() bool {
	if q.Not {
		return true
	}
	for _, child := range q.Children {
		if child.HasNegationPredicates() {
			return true
		}
	}
	return false
}

// ParseDSL parses a DSL query string into a structured Query representation.
// This is the main entry point for DSL parsing, handling the complete syntax
// including negation, hierarchical relationships, multi-target identifiers,
// and type constraints.
//
// Supported DSL syntax:
//   - Basic: "nodeType:identifier"
//   - Negation: "!nodeType:identifier"
//   - With type: "var:name type" or "field:name type"
//   - Multi-target: "func:name1,name2,name3"
//   - Wildcards: "func:Test*" or "func:*Handler"
//   - Hierarchical: "struct:User > field:Name"
//   - Import paths: "import:\"github.com/pkg\""
//
// The parser handles several complex scenarios:
//   - Multi-target identifiers with optional type constraints
//   - Import path quote removal and normalization
//   - Control structure validation (if/block only support wildcards)
//   - Recursive parsing for hierarchical relationships
//
// Parameters:
//   - query: The DSL query string to parse
//
// Returns:
//   - A structured Query representation
//   - An error if the query syntax is invalid or unsupported
//
// Examples:
//   - ParseDSL("func:TestFunc") → Query for function named "TestFunc"
//   - ParseDSL("!var:count int") → Negated query for int variable "count"
//   - ParseDSL("struct:User > field:*") → Hierarchical query for any field in User struct
func ParseDSL(query string) (*Query, error) {
	// Step 1: Basic validation
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Step 2: Handle negation prefix
	// Extract negation flag and remove the '!' prefix for further processing
	not := false
	if strings.HasPrefix(query, "!") {
		not = true
		query = strings.TrimPrefix(query, "!")
	}

	// Step 3: Split hierarchical relationships
	// Use '>' as the separator for parent-child relationships (e.g., "struct:User > field:Name")
	parts := strings.Split(query, ">")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid query format")
	}

	// Step 4: Parse the current node (first part of the hierarchy)
	firstPart := strings.TrimSpace(parts[0])
	nodeParts := strings.SplitN(firstPart, ":", 2)
	if len(nodeParts) != 2 {
		return nil, fmt.Errorf("query must be in the format 'nodeType:identifier'")
	}

	nodeType := strings.TrimSpace(nodeParts[0])
	identifier := strings.TrimSpace(nodeParts[1])
	var varType string

	// Step 5: Validate control structure constraints early
	// Control structures (if/block) only support wildcard matching
	if (nodeType == "if" || nodeType == "block") && identifier != "*" {
		return nil, fmt.Errorf("only * supported for if/block")
	}

	// Step 6: Handle type extraction for variables and fields
	// Parse type information from "identifier type" format
	// Multi-name matching derives from Go AST identifier_list, not DSL commas
	if nodeType == "var" || nodeType == "field" {
		// Single identifier scenario: parse "name type" format
		parts := strings.Fields(identifier)
		if len(parts) == 2 {
			// Simple case: "name type"
			identifier = parts[0]
			varType = parts[1]
		} else if len(parts) > 2 {
			// Complex type: "name complex type name"
			identifier = parts[0]
			varType = strings.Join(parts[1:], " ")
		}
		// If len(parts) == 1, no type specified, identifier stays as is
	}

	// Step 7: Handle import path validation (Phase 2 strict validation)
	// Import tokens must be unquoted in the DSL
	if nodeType == "import" {
		if strings.Contains(identifier, "\"") {
			return nil, fmt.Errorf("import expects unquoted path (e.g., import:fmt)")
		}
	}

	// Step 8: Construct the Query result for the current node
	result := &Query{
		Not:        not,
		NodeType:   nodeType,
		Identifier: identifier,
		Type:       varType,
		Children:   []Query{},
	}

	// Step 9: Recursively parse child queries for hierarchical relationships
	if len(parts) > 1 {
		// Reconstruct the child query string from remaining parts
		childQuery := strings.Join(parts[1:], ">")
		childQuery = strings.TrimSpace(childQuery)
		if childQuery != "" {
			// Recursive call to parse the child query
			child, err := ParseDSL(childQuery)
			if err != nil {
				return nil, fmt.Errorf("error parsing child query: %w", err)
			}
			result.Children = append(result.Children, *child)
		}
	}

	return result, nil
}
