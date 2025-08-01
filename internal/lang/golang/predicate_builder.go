package golang

import (
	"fmt"
	"strings"
)

// MatchesWildcard checks if a string matches a wildcard pattern using Go's string operations.
// This function provides runtime wildcard matching capabilities outside of Tree-sitter queries,
// useful for file pattern matching, symbol filtering, and other text matching scenarios.
//
// Supported wildcard patterns:
//   - "*" matches any string (including empty)
//   - "prefix*" matches strings starting with "prefix"
//   - "*suffix" matches strings ending with "suffix"
//   - "*contains*" matches strings containing "contains"
//   - "prefix*suffix" matches strings starting with "prefix" and ending with "suffix"
//
// Examples:
//   - MatchesWildcard("*", "anything") returns true
//   - MatchesWildcard("Test*", "TestFunc") returns true
//   - MatchesWildcard("*Handler", "HTTPHandler") returns true
//   - MatchesWildcard("*User*", "GetUserData") returns true
//   - MatchesWildcard("Get*Data", "GetUserData") returns true
//   - MatchesWildcard("exact", "exact") returns true
//   - MatchesWildcard("Test*", "Handler") returns false
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

// BuildPredicate constructs Tree-sitter predicates for identifier matching.
// This function handles the core logic of predicate construction, supporting exact matches,
// wildcard patterns, and negation. It automatically selects appropriate predicate types
// based on the capture type and pattern characteristics.
//
// Phase 2 Implementation:
//   - Uses #any-* predicates for nodes that can have multiple identifiers from AST
//   - Single-node captures use regular #eq?/#match? predicates
//   - Negation uses portable #not-eq?/#not-match? predicates only
//
// Predicate Selection Logic:
//   - For @name captures on multi-name nodes (var, const, assign, field): #any-eq? / #any-match?
//   - For single-name nodes (func, struct, call): #eq? / #match?
//   - For wildcards: #match? / #any-match? with appropriate regex patterns
//   - For negation: #not-eq? / #not-match? / #any-not-eq? / #any-not-match?
//
// Parameters:
//   - capture: The Tree-sitter capture name (e.g., "@name", "@type", "@path")
//   - value: The identifier or pattern to match against
//   - not: Whether to negate the predicate
//   - nodeType: The DSL node type (for determining multi-name support)
//
// Returns:
//   - A formatted Tree-sitter predicate string
//
// Examples:
//   - BuildPredicate("@name", "TestFunc", false, "func") → `(#eq? @name "TestFunc")`
//   - BuildPredicate("@name", "Test*", false, "var") → `(#any-match? @name "^Test.*")`
//   - BuildPredicate("@name", "TestFunc", true, "const") → `(#any-not-eq? @name "TestFunc")`
func BuildPredicate(capture, value string, not bool, nodeType string) string {
	// Determine if this node type supports multiple names from AST identifier_list
	isMultiNameNode := nodeType == "var" || nodeType == "const" || nodeType == "assign" || nodeType == "field"

	// Select base operator based on pattern type and multi-name support
	var operator string
	if strings.Contains(value, "*") {
		// Wildcard pattern - use match predicates
		if isMultiNameNode && capture == "@name" {
			operator = "#any-match?"
		} else {
			operator = "#match?"
		}
		value = buildRegexFromWildcard(value)
	} else {
		// Exact match - use equality predicates
		if isMultiNameNode && capture == "@name" {
			operator = "#any-eq?"
		} else {
			operator = "#eq?"
		}
	}

	// Apply negation transformation
	if not {
		switch operator {
		case "#eq?":
			operator = "#not-eq?"
		case "#match?":
			operator = "#not-match?"
		case "#any-eq?":
			operator = "#any-not-eq?"
		case "#any-match?":
			operator = "#any-not-match?"
		}
	}

	// Format the predicate based on operator type
	if operator == "#eq?" || operator == "#not-eq?" || operator == "#any-eq?" || operator == "#any-not-eq?" {
		return fmt.Sprintf(`(%s %s "%s")`, operator, capture, value)
	}

	// For match predicates, the regex pattern is already formatted with quotes
	return fmt.Sprintf(`(%s %s %s)`, operator, capture, value)
}

// buildRegexFromWildcard converts a wildcard string to a regex pattern for Tree-sitter predicates.
// This internal function translates user-friendly wildcard syntax into proper regex patterns
// that can be used in Tree-sitter #match? predicates.
//
// Wildcard conversion patterns:
//   - "*text*" → ".*text.*" (contains)
//   - "*text" → ".*text$" (ends with)
//   - "text*" → "^text.*" (starts with)
//   - "prefix*suffix" → "^prefix.*suffix$" (starts and ends with)
//   - "literal" → "literal" (exact match, escaped)
//
// The function uses reEscape to safely escape literal parts of the pattern,
// ensuring that special regex characters in user input don't break the pattern.
//
// Parameters:
//   - wildcard: The wildcard pattern string
//
// Returns:
//   - A quoted regex pattern string suitable for Tree-sitter predicates
//
// Examples:
//   - buildRegexFromWildcard("Test*") → `"^Test.*"`
//   - buildRegexFromWildcard("*Handler") → `".*Handler$"`
//   - buildRegexFromWildcard("*User*") → `".*User.*"`
//   - buildRegexFromWildcard("Get*Data") → `"^Get.*Data$"`
func buildRegexFromWildcard(wildcard string) string {
	if wildcard == "*" {
		return `".*"`
	}
	if !strings.Contains(wildcard, "*") {
		return fmt.Sprintf(`"%s"`, reEscape(wildcard))
	}

	if strings.HasPrefix(wildcard, "*") && strings.HasSuffix(wildcard, "*") {
		// *contains*
		return fmt.Sprintf(`".*%s.*"`, reEscape(wildcard[1:len(wildcard)-1]))
	} else if strings.HasPrefix(wildcard, "*") {
		// *suffix
		return fmt.Sprintf(`".*%s$"`, reEscape(wildcard[1:]))
	} else if strings.HasSuffix(wildcard, "*") {
		// prefix*
		return fmt.Sprintf(`"^%s.*"`, reEscape(wildcard[:len(wildcard)-1]))
	} else {
		// prefix*suffix
		parts := strings.SplitN(wildcard, "*", 2)
		return fmt.Sprintf(`"^%s.*%s$"`, reEscape(parts[0]), reEscape(parts[1]))
	}
}
