// Package evaluator provides language-agnostic query evaluation capabilities.
//
// The UniversalEvaluator implements the core design principle: ONE evaluator
// implementation that works for ALL languages via dependency injection of
// language providers. This evaluator contains NO language-specific code
// whatsoever and delegates all translation and language-specific operations
// to the injected provider.
//
// Architecture Pattern: Dependency Injection
//
// The evaluator follows a clean dependency injection pattern where:
//  1. A language provider is injected at construction time
//  2. The provider handles ALL language-specific operations
//  3. The evaluator provides a universal interface that works identically
//     for all languages
//  4. Query translation, AST traversal, and result extraction are all
//     delegated to the provider
//
// This design ensures zero coupling between the core evaluation logic
// and any specific programming language, enabling true language agnosticism.
package evaluator

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// UniversalEvaluator provides language-agnostic query evaluation capabilities.
// It uses dependency injection to work with any language provider, ensuring
// that the same evaluator implementation works identically across all supported
// programming languages.
//
// The evaluator follows the core architectural principle: it contains NO
// language-specific code and delegates all language-specific operations
// to the injected provider. This enables true language agnosticism where
// adding support for a new language requires ONLY implementing a provider
// interface.
//
// Design Principles:
// - Single Responsibility: Only handles the evaluation workflow
// - Dependency Injection: Takes provider as injected dependency
// - Language Agnostic: Works identically for ALL languages
// - Provider Delegation: All language-specific logic handled by provider
type UniversalEvaluator struct {
	// provider is the injected language-specific provider that handles
	// all translation between universal concepts and language-specific
	// AST representations
	provider provider.LanguageProvider
}

// NewUniversalEvaluator creates a new evaluator instance with the given language provider.
// The provider is injected as a dependency and will handle all language-specific
// operations for this evaluator instance.
//
// This constructor follows the dependency injection pattern where the evaluator
// is completely agnostic to the specific language being processed - the provider
// encapsulates all language-specific knowledge.
//
// Parameters:
//   - p: The language provider that will handle language-specific operations
//
// Returns:
//   - *UniversalEvaluator: A new evaluator instance configured with the provider
//   - error: If the provider is invalid or cannot be initialized
//
// Example:
//
//	provider := &golang.GoProvider{}  // or any provider implementation
//	evaluator, err := NewUniversalEvaluator(provider)
//	if err != nil {
//	    return fmt.Errorf("failed to create evaluator: %w", err)
//	}
func NewUniversalEvaluator(p provider.LanguageProvider) (*UniversalEvaluator, error) {
	if p == nil {
		return nil, fmt.Errorf("language provider cannot be nil")
	}

	// Validate that the provider can provide the required Tree-sitter language
	sitterLang := p.GetSitterLanguage()
	if sitterLang == nil {
		return nil, fmt.Errorf("provider %s does not provide a valid Tree-sitter language", p.Lang())
	}

	return &UniversalEvaluator{
		provider: p,
	}, nil
}

// Evaluate executes a universal query against source code and returns language-agnostic results.
//
// This method implements the core architectural principle: it is IDENTICAL for ALL languages.
// The provider handles all language-specific translation and the evaluator provides a
// universal interface that works consistently regardless of the programming language.
//
// Evaluation Workflow:
// 1. Validate inputs (universal validation, no language-specific logic)
// 2. Parse source code using Tree-sitter via provider's language
// 3. Translate universal query to Tree-sitter query via provider
// 4. Execute Tree-sitter query against the AST
// 5. Process matches and create universal results via provider
// 6. Return language-agnostic result set
//
// Parameters:
//   - query: Universal query structure (language-agnostic)
//   - source: Source code to evaluate (as byte slice)
//
// Returns:
//   - *core.ResultSet: Universal result set containing matches
//   - error: If evaluation fails at any step
//
// The returned ResultSet contains only universal concepts (core.Result) with
// no language-specific dependencies, enabling consistent processing across
// all supported languages.
func (e *UniversalEvaluator) Evaluate(query *core.Query, source []byte) (*core.ResultSet, error) {
	// Input validation - universal checks only
	if query == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}
	if len(source) == 0 {
		return nil, fmt.Errorf("source code cannot be empty")
	}

	// Get Tree-sitter language from provider
	sitterLang := e.provider.GetSitterLanguage()
	if sitterLang == nil {
		return nil, fmt.Errorf("provider failed to provide Tree-sitter language")
	}

	// Create and configure Tree-sitter parser
	parser := sitter.NewParser()
	parser.SetLanguage(sitterLang)

	// Parse source code into AST
	tree := parser.Parse(nil, source)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source code")
	}
	defer tree.Close()

	// Translate universal query to Tree-sitter query via provider
	// This is where all language-specific translation happens
	tsQuery, err := e.provider.TranslateQuery(query)
	if err != nil {
		return nil, fmt.Errorf("provider failed to translate query: %w", err)
	}

	// Create Tree-sitter query from translated string
	q, err := sitter.NewQuery([]byte(tsQuery), sitterLang)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tree-sitter query: %w", err)
	}
	defer q.Close()

	// Execute Tree-sitter query
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	// Collect all matches into universal results
	results := make([]*core.Result, 0)
	qc.Exec(q, tree.RootNode())

	// Process each match using provider for language-specific extraction
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		// Process each capture in the match
		for _, capture := range m.Captures {
			node := capture.Node
			if node == nil {
				continue
			}

			// Create universal result using provider for language-specific extraction
			result, err := e.createUniversalResult(node, source, query)
			if err != nil {
				// Log error but continue processing other matches
				continue
			}

			if result != nil {
				results = append(results, result)
			}
		}
	}

	// Create and return universal result set
	resultSet := &core.ResultSet{
		Results:      results,
		TotalMatches: len(results),
	}

	return resultSet, nil
}

// createUniversalResult creates a universal Result from a Tree-sitter node using
// the provider for all language-specific extraction operations.
//
// This method demonstrates the delegation pattern where the evaluator handles
// the universal workflow while the provider handles all language-specific
// operations like name extraction, kind determination, and attribute parsing.
//
// Parameters:
//   - node: Tree-sitter node from the match
//   - source: Original source code (for text extraction)
//   - query: Original universal query (for context)
//
// Returns:
//   - *core.Result: Universal result with language-agnostic data
//   - error: If result creation fails
func (e *UniversalEvaluator) createUniversalResult(node *sitter.Node, source []byte, query *core.Query) (*core.Result, error) {
	if node == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	// Use provider to determine universal node kind
	// Provider maps language-specific AST node types to universal concepts
	nodeKind := e.provider.GetNodeKind(node)

	// Use provider to extract node name/identifier
	// Provider knows how to find names in language-specific AST structures
	nodeName := e.provider.GetNodeName(node, source)

	// Create universal location information
	location := core.Location{
		StartLine: int(node.StartPoint().Row) + 1, // Convert to 1-based
		EndLine:   int(node.EndPoint().Row) + 1,
		StartCol:  int(node.StartPoint().Column) + 1,
		EndCol:    int(node.EndPoint().Column) + 1,
		StartByte: int(node.StartByte()),
		EndByte:   int(node.EndByte()),
	}

	// Extract content from source
	content := string(source[node.StartByte():node.EndByte()])

	// Use provider to extract language-specific attributes
	// Provider can extract type information, visibility, etc.
	attributes := e.provider.ParseAttributes(node, source)

	// Convert provider attributes to universal metadata
	metadata := make(map[string]any)
	for key, value := range attributes {
		metadata[key] = value
	}

	// Add universal metadata
	metadata["node_type"] = node.Type()
	metadata["query_kind"] = string(query.Kind)
	metadata["query_pattern"] = query.Pattern

	// Use provider to determine scope information
	scope := e.provider.GetNodeScope(node)

	// Find parent context using provider
	var parentKind core.NodeKind
	var parentName string
	if parent := node.Parent(); parent != nil {
		parentKind = e.provider.GetNodeKind(parent)
		parentName = e.provider.GetNodeName(parent, source)
	}

	// Create universal result with no language-specific dependencies
	result := &core.Result{
		Kind:       nodeKind,
		Name:       nodeName,
		Location:   location,
		Content:    content,
		Metadata:   metadata,
		ParentKind: parentKind,
		ParentName: parentName,
		Scope:      scope,
	}

	return result, nil
}

// GetProvider returns the language provider used by this evaluator.
// This enables inspection of the provider for debugging or capability queries.
func (e *UniversalEvaluator) GetProvider() provider.LanguageProvider {
	return e.provider
}

// GetLanguage returns the canonical language identifier from the provider.
// This is a convenience method for getting the language name without
// directly accessing the provider.
func (e *UniversalEvaluator) GetLanguage() string {
	if e.provider == nil {
		return ""
	}
	return e.provider.Lang()
}
