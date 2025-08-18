package types

import sitter "github.com/smacker/go-tree-sitter"

// LanguageProvider defines the contract for language-specific operations.
// This interface is in the types package to avoid circular dependencies.
type LanguageProvider interface {
	// Basic metadata
	Lang() string         // "go", "python", "php"
	Aliases() []string    // Alternative names
	Extensions() []string // File extensions
	GetSitterLanguage() *sitter.Language

	// DSL Translation - the heart of the provider
	TranslateKind(kind NodeKind) []NodeMapping
	TranslateQuery(q *Query) (string, error)

	// Language-specific DSL support
	NormalizeDSLKind(dslKind string) NodeKind // Translate language DSL to universal
	GetSupportedDSLKinds() []string           // Return language-specific DSL vocabulary

	// Parsing helpers
	ParseAttributes(node *sitter.Node, source []byte) map[string]string
	GetNodeKind(node *sitter.Node) NodeKind
	GetNodeName(node *sitter.Node, source []byte) string

	// Language-specific optimizations (optional)
	OptimizeQuery(q *Query) *Query
	EstimateQueryCost(q *Query) int

	// Scope detection
	GetNodeScope(node *sitter.Node) ScopeType
	FindEnclosingScope(node *sitter.Node, scope ScopeType) *sitter.Node

	// Code structure helpers
	IsBlockLevelNode(nodeType string) bool
	GetDefaultIgnorePatterns() (files []string, symbols []string)
}

// NodeMapping maps universal kinds to language-specific AST nodes
type NodeMapping struct {
	Kind        NodeKind
	NodeTypes   []string          // AST node types in this language
	NameCapture string            // How to capture the name
	TypeCapture string            // How to capture the type
	Template    string            // Tree-sitter query template
	Attributes  map[string]string // Additional captures
}
