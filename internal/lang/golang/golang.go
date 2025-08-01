package golang

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	this "github.com/smacker/go-tree-sitter/golang"

	"github.com/garaekz/fileman/internal/provider"
)

// goProvider implements the LanguageProvider interface for Go language support.
// It provides comprehensive DSL-to-Tree-sitter translation capabilities,
// wildcard pattern matching, and Go-specific language constructs handling.
//
// The provider supports all major Go language constructs including:
//   - Function declarations and calls
//   - Variable and constant declarations
//   - Struct definitions and field access
//   - Import statements
//   - Control structures (if statements, blocks)
//   - Assignment operations
//
// Key features:
//   - Multi-target identifier support (comma-separated lists)
//   - Wildcard pattern matching with various patterns
//   - Hierarchical query support (parent > child relationships)
//   - Negation support for all query types
//   - Type constraint support for variables and fields
type goProvider struct{}

// New creates a new instance of the Go language provider.
// This is the factory function used by the language manager to instantiate
// Go language support capabilities.
//
// Returns:
//   - A LanguageProvider implementation configured for Go language processing
func New() provider.LanguageProvider {
	return &goProvider{}
}

// goBlockLevelNodes defines the set of DSL node types that are considered
// block-level constructs in Go. These nodes represent major structural
// elements that can be targeted by DSL queries.
//
// Supported block-level nodes:
//   - func: Function declarations
//   - const: Constant declarations
//   - var: Variable declarations
//   - struct: Struct type definitions
//   - field: Struct field declarations
//   - call: Function call expressions
//   - assign: Assignment statements
//   - if: If statements
//   - import: Import specifications
//   - block: Block statements
//   - source_file: Top-level source file node
var goBlockLevelNodes = map[string]struct{}{
	"func": {}, "const": {}, "var": {}, "struct": {}, "field": {}, "call": {}, "assign": {}, "if": {}, "import": {}, "block": {}, "source_file": {},
}

// Lang returns the canonical name of the language handled by this provider.
// This identifier is used throughout the system for language-specific operations.
func (p *goProvider) Lang() string {
	return "go"
}

// Aliases returns the alternative names by which this provider can be referenced.
// This allows users to specify either "go" or "golang" when selecting the language.
func (p *goProvider) Aliases() []string {
	return []string{"go", "golang"}
}

// GetDefaultIgnorePatterns returns the default patterns for filtering out
// test-related files and symbols during code analysis operations.
//
// File patterns:
//   - "*_test.go": Standard Go test files
//
// Symbol patterns:
//   - "Test*": Test functions
//   - "Benchmark*": Benchmark functions
//   - "Example*": Example functions
//
// These patterns help focus analysis on production code while excluding
// test infrastructure that might not be relevant for refactoring operations.
func (p *goProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{"*_test.go"}, []string{"Test*", "Benchmark*", "Example*"}
}

// IsBlockLevelNode determines whether a given DSL node type represents
// a block-level construct in Go. Block-level nodes are major structural
// elements that can serve as meaningful targets for code analysis and refactoring.
//
// Parameters:
//   - nodeType: The DSL node type to check
//
// Returns:
//   - true if the node type is considered block-level
//   - false for inline or expression-level constructs
func (p *goProvider) IsBlockLevelNode(nodeType string) bool {
	_, ok := goBlockLevelNodes[nodeType]
	return ok
}

// GetSitterLanguage returns the Tree-sitter Language instance for Go.
// This provides the parser with the necessary grammar definitions for
// parsing Go source code into syntax trees.
func (p *goProvider) GetSitterLanguage() *sitter.Language {
	return this.GetLanguage()
}

// GetQuery constructs a Tree-sitter query for the specified node type and name.
// This is a legacy method that provides basic query construction capabilities,
// primarily used for simple, non-hierarchical queries.
//
// The method handles:
//   - Template selection based on node type
//   - Wildcard template variants
//   - Basic predicate construction
//   - Special cases for imports and control structures
//
// Note: For complex queries with hierarchical relationships, multi-target support,
// and advanced features, use TranslateDSL instead.
//
// Parameters:
//   - nodeType: The type of Go construct to match
//   - nodeName: The identifier to match (supports "*" for wildcards)
//
// Returns:
//   - The constructed Tree-sitter query string
//   - true if the query was successfully constructed, false otherwise
func (p *goProvider) GetQuery(nodeType, nodeName string) (string, bool) {
	// Step 1: Retrieve the base template for the node type
	template, ok := templates[nodeType]
	if !ok {
		return "", false
	}

	var query string

	// Step 2: Handle special cases with node-type-specific logic
	switch nodeType {
	case "import":
		// Import nodes use @path capture instead of @name
		// nodeName is already unquoted by the DSL parser
		predicate := fmt.Sprintf(`(#eq? @path "%s")`, nodeName)
		query = fmt.Sprintf(template, predicate)
	case "call":
		// Function calls support both wildcard and specific name matching
		if nodeName == "*" {
			query = templates["call_wildcard"]
		} else {
			predicate := fmt.Sprintf(`(#eq? @name "%s")`, nodeName)
			query = fmt.Sprintf(template, predicate)
		}
	case "if", "block":
		// Control structures only support wildcard matching
		if nodeName != "*" {
			return "", false // Only * supported for if/block
		}
		query = template
	default:
		// Step 3: Handle general node types with wildcard support
		if nodeName == "*" {
			// Try to use a specific wildcard template if available
			if wildcardTemplate, exists := templates[nodeType+"_wildcard"]; exists {
				query = wildcardTemplate
			} else {
				// Fall back to the regular template without predicates
				query = template
			}
		} else {
			// Build a specific name predicate for exact matching
			predicate := fmt.Sprintf(`(#eq? @name "%s")`, nodeName)
			if strings.Contains(template, "%s") {
				// Template has placeholder - use sprintf to insert predicate
				query = fmt.Sprintf(template, predicate)
			} else {
				// Template has no placeholder - append predicate manually
				query = template + " " + predicate
			}
		}
	}

	// Step 4: Add target annotation for result capture
	// All queries need @target annotation for the matcher to identify results
	query += " @target"
	return query, true
}

// TranslateDSL translates a complete DSL query string into an executable Tree-sitter query.
// This is the primary method for DSL processing, supporting the full range of DSL features
// including hierarchical relationships, multi-target identifiers, wildcard patterns,
// negation, and type constraints.
//
// Supported DSL features:
//   - Basic queries: "func:TestFunc"
//   - Negation: "!func:TestFunc"
//   - Wildcards: "func:Test*", "func:*Handler"
//   - Multi-target: "func:func1,func2,func3"
//   - Type constraints: "var:count int", "field:name string"
//   - Hierarchical: "struct:User > field:Name"
//   - Import paths: "import:\"github.com/pkg\""
//
// The translation process involves:
//  1. DSL parsing into structured Query representation
//  2. Query validation and constraint checking
//  3. Tree-sitter query construction with appropriate predicates
//  4. Query optimization and formatting
//
// Parameters:
//   - query: The DSL query string to translate
//
// Returns:
//   - A complete Tree-sitter query ready for execution
//   - An error if the DSL syntax is invalid or translation fails
//
// Examples:
//   - "func:TestFunc" → `(function_declaration name: (identifier) @name (#eq? @name "TestFunc")) @target`
//   - "!var:count int" → `[(var_declaration ...)] (#not-eq? @name "count") (#not-eq? @type "int") @target`
func (p *goProvider) TranslateDSL(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Parse the DSL query
	parsedQuery, err := ParseDSL(query)
	if err != nil {
		return "", fmt.Errorf("invalid DSL query: %w", err)
	}

	// Build the Tree-sitter query
	tsQuery, err := BuildTreeSitterQuery(parsedQuery)
	if err != nil {
		return "", err
	}

	return tsQuery, nil
}

// MatchesWildcard provides wildcard pattern matching capabilities to other parts
// of the system. This method exposes the internal wildcard matching logic for
// use in file pattern matching, symbol filtering, and other text matching scenarios.
//
// The method supports the same wildcard patterns as the DSL:
//   - "*" matches any string
//   - "prefix*" matches strings starting with "prefix"
//   - "*suffix" matches strings ending with "suffix"
//   - "*contains*" matches strings containing "contains"
//   - "prefix*suffix" matches strings with specific prefix and suffix
//
// Common use cases:
//   - File pattern matching in ignore lists
//   - Symbol filtering based on naming patterns
//   - Dynamic query result filtering
//   - Configuration-based pattern matching
//
// Parameters:
//   - pattern: The wildcard pattern to match against
//   - text: The text to test for pattern matching
//
// Returns:
//   - true if the text matches the wildcard pattern
//   - false otherwise
func (p *goProvider) MatchesWildcard(pattern, text string) bool {
	return MatchesWildcard(pattern, text)
}

// HasNegationPredicates checks if a Tree-sitter query contains negation predicates.
// This method implements Phase 2 portable negation detection, supporting only
// #not-eq? and #not-match? predicates for maximum Tree-sitter compatibility.
//
// Used for:
//   - Query validation and optimization
//   - Determining query execution strategies
//   - Performance analysis of complex negated queries
//
// Parameters:
//   - query: The Tree-sitter query string to analyze
//
// Returns:
//   - true if the query contains #not-eq? or #not-match? predicates
//   - false if no negation predicates are detected
func (p *goProvider) HasNegationPredicates(query string) bool {
	return HasNegationPredicates(query)
}
