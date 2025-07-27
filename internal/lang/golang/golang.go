package golang

import (
	"fmt"
	"strconv"

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

// templates contains the mappings from the DSL to the Tree-sitter queries.
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

// goBlockLevelNodes defines the block-level nodes for Go language.
var goBlockLevelNodes = map[string]struct{}{
	"func": {}, "method": {}, "type": {}, "struct": {}, "interface": {}, "const": {}, "var": {}, "package": {},
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

	// Imports need special handling for quotes.
	if nodeType == "import" {
		quotedName := strconv.Quote(nodeName)
		return fmt.Sprintf(template, strconv.Quote(quotedName)), true
	}

	return fmt.Sprintf(template, nodeName), true
}
