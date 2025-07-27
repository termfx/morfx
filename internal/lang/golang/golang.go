package golang

import (
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	this "github.com/smacker/go-tree-sitter/golang"

	"github.com/garaekz/fileman/internal/provider"
)

// goProvider implements the LanguageProvider interface for Go language.
type goProvider struct{}

// New creates a new instance of the Go language provider.
func New() provider.LanguageProvider {
	return &goProvider{}
}

type Query struct {
	Not        bool
	NodeType   string
	Identifier string
	Children   []Query
}

// templates contains the mappings from the DSL to the Tree-sitter queries.
var templates = map[string]string{
	// DSL node types
	"func":   `(function_declaration name: (identifier) @name %s) @target`,
	"const":  `(const_declaration (const_spec name: (identifier) @name %s)) @target`,
	"var":    `(var_declaration (var_spec name: (identifier) @name %s)) @target`,
	"struct": `(type_declaration (type_spec name: (type_identifier) @name %s type: (struct_type))) @target`,
	"field":  `(field_declaration name: (field_identifier) @name %s) @target`,
	"call":   `(call_expression function: (identifier) @name %s) @target`,
	"assign": `(assignment_statement left: (expression_list (identifier) @name) %s) @target`,
	"if":     `(if_statement %s) @target`,
	"import": `(import_spec (interpreted_string_literal) @path %s) @target`,
	"block":  `(block %s) @target`,
}

// goBlockLevelNodes defines the block-level nodes for Go language.
var goBlockLevelNodes = map[string]struct{}{
	"func": {}, "const": {}, "var": {}, "struct": {}, "field": {}, "call": {}, "assign": {}, "if": {}, "import": {}, "block": {},
}

// Lang returns the canonical name of the language handled by this provider.
func (p *goProvider) Lang() string {
	return "go"
}

// Aliases returns the names by which this provider is known.
func (p *goProvider) Aliases() []string {
	return []string{"go", "golang"}
}

// GetDefaultIgnorePatterns returns the patterns for ignoring test files and symbols.
func (p *goProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{"*_test.go"}, []string{"Test*", "Benchmark*", "Example*"}
}

// IsBlockLevelNode checks if a DSL node type is considered a block.
func (p *goProvider) IsBlockLevelNode(nodeType string) bool {
	_, ok := goBlockLevelNodes[nodeType]
	return ok
}

func (p *goProvider) GetSitterLanguage() *sitter.Language {
	return this.GetLanguage()
}

// GetQuery returns the Tree-sitter query formatted for a node type and name.
func (p *goProvider) GetQuery(nodeType, nodeName string) (string, bool) {
	template, ok := templates[nodeType]
	if !ok {
		return "", false
	}

	// Handle import nodes specially
	if nodeType == "import" {
		quotedName := strconv.Quote(nodeName)
		predicate := fmt.Sprintf(`(#eq? @path %s)`, strconv.Quote(quotedName))
		return fmt.Sprintf(template, predicate), true
	}

	// Handle all other node types
	predicate := fmt.Sprintf(`(#eq? @name "%s")`, nodeName)
	return fmt.Sprintf(template, predicate), true
}

// parseDSL parses a DSL query string into a Query struct.
func parseDSL(query string) (*Query, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Handle negation
	not := false
	if strings.HasPrefix(query, "!") {
		not = true
		query = strings.TrimPrefix(query, "!")
	}

	// Split by > for parent/child relationships
	parts := strings.Split(query, ">")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid query format")
	}

	// Parse the first part (current node)
	firstPart := strings.TrimSpace(parts[0])
	nodeParts := strings.SplitN(firstPart, ":", 2)
	if len(nodeParts) != 2 {
		return nil, fmt.Errorf("query must be in the format 'nodeType:identifier'")
	}

	nodeType := strings.TrimSpace(nodeParts[0])
	identifier := strings.TrimSpace(nodeParts[1])

	result := &Query{
		Not:        not,
		NodeType:   nodeType,
		Identifier: identifier,
		Children:   []Query{},
	}

	// Parse children if there are more parts
	if len(parts) > 1 {
		childQuery := strings.Join(parts[1:], ">")
		childQuery = strings.TrimSpace(childQuery)
		if childQuery != "" {
			child, err := parseDSL(childQuery)
			if err != nil {
				return nil, fmt.Errorf("error parsing child query: %w", err)
			}
			result.Children = append(result.Children, *child)
		}
	}

	return result, nil
}

// MatchesWildcard checks if a string matches a wildcard pattern.
// This function can be used for runtime wildcard matching outside of Tree-sitter queries.
func MatchesWildcard(pattern, text string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == text
	}

	// Handle different wildcard patterns
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *Foo* - contains
		middle := pattern[1 : len(pattern)-1]
		return strings.Contains(text, middle)
	} else if strings.HasPrefix(pattern, "*") {
		// *Foo - ends with
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	} else if strings.HasSuffix(pattern, "*") {
		// Foo* - starts with
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	} else {
		// Foo*Bar - starts with Foo and ends with Bar
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(text, parts[0]) && strings.HasSuffix(text, parts[1])
		}
	}

	return false
}

// buildTreeSitterPredicate builds the Tree-sitter predicate for identifier matching.
func buildTreeSitterPredicate(identifier string) string {
	if identifier == "*" {
		return "" // No predicate needed for wildcard
	}

	if !strings.Contains(identifier, "*") {
		return fmt.Sprintf(`(#eq? @name "%s")`, identifier)
	}

	// For wildcards, we need to use regex predicates
	if strings.HasPrefix(identifier, "*") && strings.HasSuffix(identifier, "*") {
		// *Foo* - contains
		middle := identifier[1 : len(identifier)-1]
		return fmt.Sprintf(`(#match? @name ".*%s.*")`, middle)
	} else if strings.HasPrefix(identifier, "*") {
		// *Foo - ends with
		suffix := identifier[1:]
		return fmt.Sprintf(`(#match? @name ".*%s$")`, suffix)
	} else if strings.HasSuffix(identifier, "*") {
		// Foo* - starts with
		prefix := identifier[:len(identifier)-1]
		return fmt.Sprintf(`(#match? @name "^%s.*")`, prefix)
	} else {
		// Foo*Bar - starts with Foo and ends with Bar
		parts := strings.Split(identifier, "*")
		if len(parts) == 2 {
			return fmt.Sprintf(`(#match? @name "^%s.*%s$")`, parts[0], parts[1])
		}
	}

	return fmt.Sprintf(`(#eq? @name "%s")`, identifier)
}

// buildTreeSitterQuery builds a Tree-sitter query from a parsed Query struct.
func (p *goProvider) buildTreeSitterQuery(q *Query) (string, error) {
	template, ok := templates[q.NodeType]
	if !ok {
		return "", fmt.Errorf("unknown node type: %s", q.NodeType)
	}

	predicate := buildTreeSitterPredicate(q.Identifier)

	// Handle children - for parent/child relationships, we need to nest the queries
	var childConstraint string
	if len(q.Children) > 0 {
		child := q.Children[0] // Handle first child
		childTSQuery, err := p.buildTreeSitterQuery(&child)
		if err != nil {
			return "", err
		}
		// Remove the @target from child query since it's nested
		childTSQuery = strings.Replace(childTSQuery, " @target", "", 1)
		childConstraint = fmt.Sprintf(" (%s)", childTSQuery)
	}

	// Combine predicate and child constraint
	var constraint string
	if predicate != "" && childConstraint != "" {
		constraint = predicate + childConstraint
	} else if predicate != "" {
		constraint = predicate
	} else if childConstraint != "" {
		constraint = childConstraint
	}

	query := fmt.Sprintf(template, constraint)

	// Handle negation using Tree-sitter's #is-not? predicate
	if q.Not {
		// For negation, we use the #is-not? predicate which is supported by Tree-sitter
		if strings.Contains(query, "#eq?") {
			query = strings.Replace(query, "#eq?", "#is-not?", 1)
		} else if strings.Contains(query, "#match?") {
			query = strings.Replace(query, "#match?", "#is-not?", 1)
		}
	}

	return query, nil
}

// TranslateDSL translates a DSL query into a Tree-sitter query.
func (p *goProvider) TranslateDSL(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Parse the DSL query
	parsedQuery, err := parseDSL(query)
	if err != nil {
		return "", fmt.Errorf("invalid DSL query: %w", err)
	}

	// Build the Tree-sitter query
	tsQuery, err := p.buildTreeSitterQuery(parsedQuery)
	if err != nil {
		return "", err
	}

	return tsQuery, nil
}

// MatchesWildcard is exported to allow other parts of the system to use wildcard matching.
// This is useful for file pattern matching, symbol filtering, etc.
func (p *goProvider) MatchesWildcard(pattern, text string) bool {
	return MatchesWildcard(pattern, text)
}

// HasNegationPredicates checks if a Tree-sitter query contains negation predicates.
func (p *goProvider) HasNegationPredicates(query string) bool {
	return strings.Contains(query, "#is-not?")
}
