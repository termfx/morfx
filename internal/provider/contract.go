package provider

import (
	"fmt"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/core"
)

// NodeMapping defines how universal node kinds map to language-specific AST nodes.
// This structure bridges the gap between universal concepts and language implementations.
type NodeMapping = core.NodeMapping

// LanguageProvider defines the minimal but complete interface that all language providers
// must implement. This interface enables true language abstraction by providing methods
// to translate between universal concepts and language-specific AST representations.
//
// Each method serves a specific purpose in the language abstraction layer:
// - Metadata methods provide basic language information
// - Translation methods convert between universal and language-specific concepts
// - AST helper methods extract information from Tree-sitter nodes
// - Scope detection methods handle hierarchical code organization
type LanguageProvider interface {
	// Lang returns the canonical language identifier (e.g., "go", "python", "javascript").
	// This is used for provider registration and lookup by the registry.
	Lang() string

	// Aliases returns alternative names for this language that users might use
	// (e.g., ["go", "golang"] or ["js", "javascript", "node", "nodejs"]).
	// This enables flexible language detection and user-friendly CLI experience.
	Aliases() []string

	// Extensions returns the file extensions associated with this language
	// (e.g., [".go"] or [".js", ".mjs", ".cjs"]).
	// Used for automatic language detection based on file extension.
	Extensions() []string

	// GetSitterLanguage returns the Tree-sitter language parser for this language.
	// This provides the low-level parsing capability for AST generation.
	GetSitterLanguage() *sitter.Language

	// TranslateKind converts a universal NodeKind to language-specific node mappings.
	// Returns multiple mappings because a single universal concept might correspond
	// to different AST node types in the target language.
	// Example: core.KindFunction might map to both "function_declaration" and "method_declaration".
	TranslateKind(kind core.NodeKind) []NodeMapping

	// TranslateQuery converts a universal Query to a Tree-sitter query string.
	// This is the core translation method that enables language-agnostic querying.
	// The provider translates universal concepts to language-specific Tree-sitter syntax.
	TranslateQuery(q *core.Query) (string, error)

	// GetNodeKind determines the universal NodeKind for a given Tree-sitter node.
	// This is the reverse of TranslateKind - it maps language-specific AST nodes
	// back to universal concepts for result generation.
	GetNodeKind(node *sitter.Node) core.NodeKind

	// GetNodeName extracts the identifier or name from a Tree-sitter node.
	// Different languages store names in different places within their AST,
	// so each provider implements language-specific extraction logic.
	GetNodeName(node *sitter.Node, source []byte) string

	// ParseAttributes extracts additional metadata from a Tree-sitter node as key-value pairs.
	// This can include type information, visibility modifiers, or other language-specific
	// attributes that might be useful for query matching or result enrichment.
	ParseAttributes(node *sitter.Node, source []byte) map[string]string

	// GetNodeScope determines the scope type (file, class, function, block) for a given node.
	// This enables scope-aware operations and hierarchical query capabilities.
	GetNodeScope(node *sitter.Node) core.ScopeType

	// FindEnclosingScope traverses up the AST to find the nearest enclosing node
	// of the specified scope type. This is essential for context-aware operations
	// and understanding code structure relationships.
	FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node

	// Additional methods needed for backward compatibility with existing codebase

	// NormalizeDSLKind translates language-specific DSL terms to universal kinds.
	// This allows providers to support language-specific vocabulary (e.g., "def" -> KindFunction).
	NormalizeDSLKind(dslKind string) core.NodeKind

	// IsBlockLevelNode determines if a node type should be treated as block-level for formatting.
	// This is used by the manipulator for proper code structure handling.
	IsBlockLevelNode(nodeType string) bool

	// GetDefaultIgnorePatterns returns default file and symbol patterns to ignore.
	// Used for filtering out test files, generated code, and other non-relevant matches.
	GetDefaultIgnorePatterns() (files []string, symbols []string)

	// OrganizeImports organizes and formats import statements in the source code.
	// Returns the modified source with properly organized imports.
	OrganizeImports(source []byte) ([]byte, error)

	// Format formats the source code according to language conventions.
	// Returns the formatted source code.
	Format(source []byte) ([]byte, error)

	// QuickCheck performs basic syntax and semantic validation on source code.
	// Returns a list of diagnostics (errors, warnings, info) found in the code.
	QuickCheck(source []byte) []core.QuickCheckDiagnostic

	// Additional methods for query optimization and cost estimation

	// OptimizeQuery optimizes a query for better performance.
	// This allows providers to rewrite queries for efficiency.
	OptimizeQuery(q *core.Query) *core.Query

	// EstimateQueryCost estimates the computational cost of executing a query.
	// This can be used for query planning and optimization decisions.
	EstimateQueryCost(q *core.Query) int
}

// BaseProvider provides common functionality that all language providers can embed.
// This struct contains shared implementation for common operations, reducing
// code duplication across providers while maintaining clean separation of concerns.
//
// Providers should embed BaseProvider and override methods as needed for
// language-specific behavior. The base implementation provides sensible defaults
// that work for many common cases.
type BaseProvider struct {
	// mappings stores the NodeKind to NodeMapping relationships built during initialization
	mappings map[core.NodeKind][]NodeMapping

	// cache stores translated queries to avoid repeated translation work
	cache map[string]string

	// cacheMu protects concurrent access to the cache
	cacheMu sync.RWMutex
}

// BuildMappings initializes the mappings from a slice of NodeMapping structs.
// This should be called during provider initialization to set up the translation
// tables that convert universal NodeKind values to language-specific mappings.
func (b *BaseProvider) BuildMappings(mappings []NodeMapping) {
	b.mappings = make(map[core.NodeKind][]NodeMapping)
	for _, mapping := range mappings {
		b.mappings[mapping.Kind] = append(b.mappings[mapping.Kind], mapping)
	}
}

// TranslateKind returns the node mappings for a given universal NodeKind.
// This is the core method that enables translation from universal concepts
// to language-specific AST node patterns.
func (b *BaseProvider) TranslateKind(kind core.NodeKind) []NodeMapping {
	if b.mappings == nil {
		return nil
	}
	return b.mappings[kind]
}

// CacheQuery stores a translated query in the cache for future reuse.
// This optimization avoids repeated translation of the same query patterns,
// which can be expensive for complex queries.
func (b *BaseProvider) CacheQuery(key, query string) {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	if b.cache == nil {
		b.cache = make(map[string]string)
	}
	b.cache[key] = query
}

// GetCachedQuery retrieves a previously cached query translation.
// Returns the cached query and a boolean indicating whether the key was found.
func (b *BaseProvider) GetCachedQuery(key string) (string, bool) {
	b.cacheMu.RLock()
	defer b.cacheMu.RUnlock()

	if b.cache == nil {
		return "", false
	}
	query, exists := b.cache[key]
	return query, exists
}

// ParseAttributes provides a default implementation that extracts basic node information.
// Most providers will want to override this to extract language-specific attributes,
// but this provides a reasonable baseline that includes node type and position data.
func (b *BaseProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	attrs := make(map[string]string)
	attrs["type"] = node.Type()
	attrs["start_line"] = fmt.Sprintf("%d", node.StartPoint().Row+1)
	attrs["end_line"] = fmt.Sprintf("%d", node.EndPoint().Row+1)
	attrs["start_col"] = fmt.Sprintf("%d", node.StartPoint().Column+1)
	attrs["end_col"] = fmt.Sprintf("%d", node.EndPoint().Column+1)
	return attrs
}

// GetNodeScope provides a default scope detection implementation based on common patterns.
// Most languages follow similar scoping rules, so this default implementation handles
// the most common cases. Providers can override this for language-specific behavior.
func (b *BaseProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
	switch {
	case b.isFileScope(node):
		return core.ScopeFile
	case b.isClassScope(node):
		return core.ScopeClass
	case b.isFunctionScope(node):
		return core.ScopeFunction
	case b.isBlockScope(node):
		return core.ScopeBlock
	default:
		// If we can't determine the scope, inherit from parent
		if parent := node.Parent(); parent != nil {
			return b.GetNodeScope(parent)
		}
		return core.ScopeFile
	}
}

// FindEnclosingScope provides a default implementation that traverses up the AST
// to find the nearest node of the specified scope type. This works for most languages
// since AST structure typically reflects scoping relationships.
func (b *BaseProvider) FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node {
	current := node.Parent()
	for current != nil {
		if b.GetNodeScope(current) == scope {
			return current
		}
		current = current.Parent()
	}
	return nil
}

// Helper methods for scope detection - these can be overridden by providers

// isFileScope checks if a node represents file-level scope
func (b *BaseProvider) isFileScope(node *sitter.Node) bool {
	nodeType := node.Type()
	return nodeType == "program" || nodeType == "source_file" || nodeType == "module"
}

// isClassScope checks if a node represents class-level scope
func (b *BaseProvider) isClassScope(node *sitter.Node) bool {
	nodeType := node.Type()
	return strings.Contains(nodeType, "class") ||
		strings.Contains(nodeType, "struct") ||
		strings.Contains(nodeType, "interface") ||
		nodeType == "type_declaration"
}

// isFunctionScope checks if a node represents function-level scope
func (b *BaseProvider) isFunctionScope(node *sitter.Node) bool {
	nodeType := node.Type()
	return strings.Contains(nodeType, "function") ||
		strings.Contains(nodeType, "method") ||
		strings.Contains(nodeType, "procedure") ||
		strings.Contains(nodeType, "def")
}

// isBlockScope checks if a node represents block-level scope
func (b *BaseProvider) isBlockScope(node *sitter.Node) bool {
	nodeType := node.Type()
	return nodeType == "block" ||
		nodeType == "statement_block" ||
		nodeType == "compound_statement" ||
		strings.Contains(nodeType, "_statement") && strings.Contains(nodeType, "if") ||
		strings.Contains(nodeType, "_statement") && strings.Contains(nodeType, "for") ||
		strings.Contains(nodeType, "_statement") && strings.Contains(nodeType, "while")
}

// ConvertWildcardToRegex converts wildcard patterns (* and ?) to regex patterns.
// This is a common operation needed by most providers for pattern matching in queries.
func (b *BaseProvider) ConvertWildcardToRegex(pattern string) string {
	// Escape regex special characters except * and ?
	escaped := strings.ReplaceAll(pattern, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "+", "\\+")
	escaped = strings.ReplaceAll(escaped, "^", "\\^")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")

	// Convert wildcards to regex
	escaped = strings.ReplaceAll(escaped, "*", ".*")
	escaped = strings.ReplaceAll(escaped, "?", ".")

	// Anchor the pattern to match entire strings
	return "^" + escaped + "$"
}

// BuildQueryFromMapping constructs a Tree-sitter query string from a NodeMapping and Query.
// This is a helper method that most providers can use to build their Tree-sitter queries
// from the mapping templates and query constraints.
func (b *BaseProvider) BuildQueryFromMapping(mapping NodeMapping, q *core.Query) string {
	// Handle pattern matching constraint
	patternConstraint := ""
	if q.Pattern != "" && q.Pattern != "*" {
		regexPattern := b.ConvertWildcardToRegex(q.Pattern)
		patternConstraint = fmt.Sprintf(`(#match? %s "%s")`, mapping.NameCapture, regexPattern)
	}

	// Handle type constraints from attributes
	typeConstraint := ""
	if typeAttr, hasType := q.Attributes["type"]; hasType && mapping.TypeCapture != "" {
		typeRegex := b.ConvertWildcardToRegex(typeAttr)
		typeConstraint = fmt.Sprintf(`(#match? %s "%s")`, mapping.TypeCapture, typeRegex)
	}

	// Combine all constraints
	constraints := []string{}
	if patternConstraint != "" {
		constraints = append(constraints, patternConstraint)
	}
	if typeConstraint != "" {
		constraints = append(constraints, typeConstraint)
	}

	constraintStr := strings.Join(constraints, " ")

	// Handle templates with multiple placeholders
	placeholderCount := strings.Count(mapping.Template, "%s")
	if placeholderCount > 1 {
		// For templates with multiple placeholders, use the same constraint for all
		placeholders := make([]any, placeholderCount)
		for i := range placeholders {
			placeholders[i] = constraintStr
		}
		return fmt.Sprintf(mapping.Template, placeholders...)
	}

	// Single placeholder or no placeholders
	if placeholderCount == 0 {
		return mapping.Template
	}
	return fmt.Sprintf(mapping.Template, constraintStr)
}

// NormalizeDSLKind provides a default implementation that returns the input as-is.
// Language-specific providers should override this to support their DSL vocabulary.
func (b *BaseProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
	return core.NodeKind(dslKind)
}

// IsBlockLevelNode provides a default implementation for determining block-level nodes.
// This can be overridden by providers for language-specific behavior.
func (b *BaseProvider) IsBlockLevelNode(nodeType string) bool {
	// Default implementation based on common patterns
	switch nodeType {
	case "block", "statement_block", "compound_statement":
		return true
	default:
		return strings.Contains(nodeType, "function") ||
			strings.Contains(nodeType, "class") ||
			strings.Contains(nodeType, "method") ||
			strings.Contains(nodeType, "if_statement") ||
			strings.Contains(nodeType, "for_statement") ||
			strings.Contains(nodeType, "while_statement")
	}
}

// GetDefaultIgnorePatterns provides default ignore patterns that work for most languages.
// Providers can override this to add language-specific ignore patterns.
func (b *BaseProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	files = []string{
		"*_test.*",
		"test_*.*",
		"*.test.*",
		"vendor/*",
		"node_modules/*",
		".git/*",
		"build/*",
		"dist/*",
	}
	symbols = []string{
		"test*",
		"Test*",
		"*_test",
		"*Test",
		"mock*",
		"Mock*",
	}
	return files, symbols
}

// OrganizeImports provides a default implementation (no-op).
// Language-specific providers should override this for proper import organization.
func (b *BaseProvider) OrganizeImports(source []byte) ([]byte, error) {
	// Default implementation: return source unchanged
	// Language-specific providers should override this
	return source, nil
}

// Format provides a default implementation (no-op).
// Language-specific providers should override this for proper code formatting.
func (b *BaseProvider) Format(source []byte) ([]byte, error) {
	// Default implementation: return source unchanged
	// Language-specific providers should override this
	return source, nil
}

// QuickCheck provides a default implementation (no diagnostics).
// Language-specific providers should override this for syntax/semantic checking.
func (b *BaseProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
	// Default implementation: return empty diagnostics
	// Language-specific providers should override this
	return []core.QuickCheckDiagnostic{}
}

// OptimizeQuery provides a default implementation (no optimization).
// Language-specific providers can override this for query optimization.
func (b *BaseProvider) OptimizeQuery(q *core.Query) *core.Query {
	// Default implementation: return query unchanged
	// Language-specific providers can override this for optimization
	return q
}

// EstimateQueryCost provides a default cost estimation implementation.
// Language-specific providers can override this for more accurate cost modeling.
func (b *BaseProvider) EstimateQueryCost(q *core.Query) int {
	// Simple heuristic: base cost + children cost
	cost := 1
	for _, child := range q.Children {
		cost += b.EstimateQueryCost(&child)
	}
	return cost
}
