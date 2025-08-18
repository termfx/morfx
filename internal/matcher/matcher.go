package matcher

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/types"
)

// Matcher uses Tree-sitter queries to match AST nodes.
type Matcher struct {
	lang  *sitter.Language
	query *sitter.Query
}

// New builds an Matcher for the given query and language identifier.
func New(cfg *model.Config) (*Matcher, error) {
	language := cfg.Provider.GetSitterLanguage()

	q, err := sitter.NewQuery([]byte(cfg.Pattern), language)
	if err != nil {
		return nil, err
	}
	return &Matcher{lang: language, query: q}, nil
}

// Find parses the source and returns the Tree-sitter nodes of the '@target' captures.
func (a *Matcher) Find(src []byte) ([]*sitter.Node, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(a.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(a.query, tree.RootNode())

	var nodes []*sitter.Node
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		// Filter out predicates before iterating captures
		match = cursor.FilterPredicates(match, src)
		for _, cap := range match.Captures {
			captureName := a.query.CaptureNameForId(cap.Index)
			// We only care about the final node designated as '@target'.
			if captureName == "target" {
				nodes = append(nodes, cap.Node)
				// Found the target for this match, break inner loop to avoid duplicates.
				break
			}
		}
	}
	return nodes, nil
}

// CompoundMatcher extends the basic Matcher to support compound boolean queries
// using the new MORFX Boolean Operators implementation.
type CompoundMatcher struct {
	*Matcher
	provider types.LanguageProvider
}

// NewCompound creates a new CompoundMatcher that supports boolean operations.
// This matcher can handle AND, OR, NOT, and hierarchical queries.
func NewCompound(cfg *model.Config) (*CompoundMatcher, error) {
	baseMatcher, err := New(cfg)
	if err != nil {
		return nil, err
	}

	return &CompoundMatcher{
		Matcher:  baseMatcher,
		provider: cfg.Provider,
	}, nil
}

// FindCompound evaluates compound DSL queries and returns structured results.
// This method supports the full range of boolean operations and provides
// enhanced result metadata.
func (cm *CompoundMatcher) FindCompound(query string, src []byte) (*CompoundResult, error) {
	// Check if provider supports compound queries
	if compoundProvider, ok := cm.provider.(CompoundQueryProvider); ok {
		resultSet, err := compoundProvider.EvaluateQuery(query, src)
		if err != nil {
			return nil, fmt.Errorf("compound query evaluation failed: %w", err)
		}

		return &CompoundResult{
			ResultSet: resultSet,
			Query:     query,
			Source:    src,
		}, nil
	}

	// Fallback to simple query for providers that don't support compound queries
	nodes, err := cm.Find(src)
	if err != nil {
		return nil, err
	}

	// Convert simple results to compound format
	return &CompoundResult{
		Nodes:  nodes,
		Query:  query,
		Source: src,
	}, nil
}

// CompoundQueryProvider interface for providers that support compound queries
type CompoundQueryProvider interface {
	types.LanguageProvider
	EvaluateQuery(query string, source []byte) (any, error)
	SupportsCompoundQueries() bool
	ValidateQuery(query string) error
}

// CompoundResult represents the result of a compound query evaluation
type CompoundResult struct {
	ResultSet any            // ResultSet from evaluator (if compound)
	Nodes     []*sitter.Node // Simple nodes (fallback)
	Query     string         // Original query
	Source    []byte         // Source code
	Metadata  map[string]any // Additional metadata
}

// GetNodes extracts nodes from either compound or simple results
func (cr *CompoundResult) GetNodes() []*sitter.Node {
	if cr.Nodes != nil {
		return cr.Nodes
	}

	// Extract nodes from ResultSet if available
	// This would need to be implemented based on the actual ResultSet interface
	return nil
}

// IsCompound returns true if this result came from a compound query
func (cr *CompoundResult) IsCompound() bool {
	return cr.ResultSet != nil
}
