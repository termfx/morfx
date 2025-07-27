package provider

import sitter "github.com/smacker/go-tree-sitter"

// LanguageProvider defines the contract for language-specific operations.
// Each supported language must implement this interface.
type LanguageProvider interface {
	// GetQuery translates a DSL node type and name into a Tree-sitter query.
	GetQuery(nodeType, nodeName string) (string, bool)

	// Aliases returns all names this provider responds to (e.g., "go", "golang").
	Aliases() []string

	// IsBlockLevelNode checks if a DSL node type is considered a block-level
	// element, which might affect formatting (e.g., adding newlines).
	IsBlockLevelNode(nodeType string) bool

	// GetDefaultIgnorePatterns provides idiomatic patterns for ignoring files
	// and symbols (e.g., test files).
	GetDefaultIgnorePatterns() (files []string, symbols []string)

	// Lang returns the canonical name of the language (e.g., "go").
	Lang() string

	// GetSitterLanguage returns the Tree-sitter language object for this provider.
	GetSitterLanguage() *sitter.Language

	// TranslateDSL translates a DSL query into a Tree-sitter query.
	TranslateDSL(query string) (string, error)
}
