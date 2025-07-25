package golang

import (
	"fmt"
	"strconv"

	"github.com/garaekz/fileman/internal/lang"
)

// goProvider implements the LanguageProvider interface for Go.
type goProvider struct{}

// templates holds the query templates for different Go node types.
// Each query uses an inner capture for the predicate and an outer '@target'
// capture for the node we want to operate on.
var templates = map[string]string{
	"function":  `((function_declaration name: (identifier) @name (#eq? @name "%s"))) @target`,
	"method":    `((method_declaration name: (identifier) @name (#eq? @name "%s"))) @target`,
	"import":    `((import_spec (interpreted_string_literal) @path (#eq? @path %s))) @target`,
	"type":      `((type_declaration name: (type_identifier) @name (#eq? @name "%s"))) @target`,
	"interface": `((type_declaration name: (type_identifier) @name (#eq? @name "%s") type: (interface_type))) @target`,
	"const":     `((const_declaration (const_spec name: (identifier) @name (#eq? @name "%s")))) @target`,
	"var":       `((var_declaration (var_spec name: (identifier) @name (#eq? @name "%s")))) @target`,
	"package":   `((package_clause name: (package_identifier) @name (#eq? @name "%s"))) @target`,
	"struct":    `((type_declaration (type_spec name: (type_identifier) @name (#eq? @name "%s") type: (struct_type)))) @target`,
}

// init registers the Go language provider upon package initialization.
func init() {
	lang.Register("go", &goProvider{})
	lang.Register("golang", &goProvider{})
}

// GetQuery returns the formatted query string for a given Go node type and name.
func (p *goProvider) GetQuery(nodeType, nodeName string) (string, bool) {
	template, ok := templates[nodeType]
	if !ok {
		return "", false
	}

	// For node types that represent string literals in the source code (like import paths),
	// the nodeName must be quoted before being injected into the query template.
	if nodeType == "import" {
		// For imports, we need to match the actual string literal which includes quotes
		quotedName := strconv.Quote(nodeName)
		return fmt.Sprintf(template, strconv.Quote(quotedName)), true
	}

	// For other nodes like identifiers, the name is used directly.
	return fmt.Sprintf(template, nodeName), true
}
