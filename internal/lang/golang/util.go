package golang

import "strings"

// CleanQueryString removes all newline, carriage return, and tab characters from a string.
func CleanQueryString(query string) string {
	query = strings.ReplaceAll(query, "\n", "")
	query = strings.ReplaceAll(query, "\r", "")
	query = strings.ReplaceAll(query, "\t", "")
	return query
}

// HasNegationPredicates checks if a Tree-sitter query contains negation predicates.
// Phase 2 uses only portable negation predicates: #not-eq?, #not-match?, #any-not-eq?, #any-not-match?
func HasNegationPredicates(query string) bool {
	return strings.Contains(query, "#not-eq?") ||
		strings.Contains(query, "#not-match?") ||
		strings.Contains(query, "#any-not-eq?") ||
		strings.Contains(query, "#any-not-match?")
}
