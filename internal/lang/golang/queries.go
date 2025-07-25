package golang

import (
	"fmt"
	"strconv"

	"github.com/garaekz/fileman/internal/lang"
)

// goProvider implements the LanguageProvider interface for Go.
type goProvider struct{}

// templates holds the query templates for different Go node types.
var templates = map[string]string{
	"function":  `(function_declaration name: (identifier) @name (#eq? @name "%s"))`,
	"method":    `(method_declaration receiver: (* (identifier)) name: (identifier) @name (#eq? @name "%s"))`,
	"import":    `(import_spec path: (interpreted_string_literal) @path (#eq? @path %s))`,
	"type":      `(type_declaration name: (type_identifier) @name (#eq? @name "%s"))`,
	"struct":    `(type_declaration name: (type_identifier) @name (#eq? @name "%s") body: (struct_type))`,
	"interface": `(type_declaration name: (type_identifier) @name (#eq? @name "%s") body: (interface_type))`,
	"const":     `(const_declaration name: (identifier) @name (#eq? @name "%s"))`,
	"var":       `(var_declaration name: (identifier) @name (#eq? @name "%s"))`,
	"package":   `(package_clause name: (identifier) @name (#eq? @name "%s"))`,
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
		return fmt.Sprintf(template, strconv.Quote(nodeName)), true
	}

	// For other nodes like identifiers, the name is used directly.
	return fmt.Sprintf(template, nodeName), true
}
