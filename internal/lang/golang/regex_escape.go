package golang

import (
	"regexp"
)

// reEscape escapes special characters in a string for use in a regular expression.
// It uses regexp.QuoteMeta to handle all special regex metacharacters correctly,
// ensuring that literal strings are properly escaped when used in regex patterns.
//
// This function is essential for the DSL's wildcard pattern processing, where
// user-provided strings need to be safely embedded in Tree-sitter regex predicates.
//
// Special characters that are escaped include: \.+*?()|[]{}^$
//
// Usage examples:
//   - reEscape("hello.world") returns "hello\\.world"
//   - reEscape("func(*)") returns "func\\(\\*\\)"
//   - reEscape("$variable") returns "\\$variable"
//   - reEscape("test[0]") returns "test\\[0\\]"
//
// Common use cases in the DSL:
//   - Building regex patterns from wildcard expressions
//   - Escaping literal parts of function/variable names
//   - Ensuring safe regex construction in Tree-sitter predicates
//
// Example in wildcard processing:
//
//	pattern := "prefix*suffix"
//	parts := strings.Split(pattern, "*")
//	regex := fmt.Sprintf("^%s.*%s$", reEscape(parts[0]), reEscape(parts[1]))
//	// Results in: "^prefix.*suffix$" (safe for regex use)
func reEscape(s string) string {
	return regexp.QuoteMeta(s)
}
