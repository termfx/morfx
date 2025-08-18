package evaluator

import (
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/types"
)

// UniversalEvaluator provides language-agnostic query evaluation capabilities.
// It works with any language provider to execute DSL queries against source code
// using Tree-sitter for parsing and traversal.
type UniversalEvaluator struct {
	// provider is the language-specific provider for DSL translation
	provider types.LanguageProvider
	// parser is the Tree-sitter parser for the target language
	parser *sitter.Parser
	// language is the Tree-sitter language definition
	language *sitter.Language
}

// NewUniversalEvaluator creates a new evaluator instance for the given language provider
func NewUniversalEvaluator(p types.LanguageProvider) (*UniversalEvaluator, error) {
	if p == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}

	lang := p.GetSitterLanguage()
	if lang == nil {
		return nil, fmt.Errorf("provider must return a valid Tree-sitter language")
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	return &UniversalEvaluator{
		provider: p,
		parser:   parser,
		language: lang,
	}, nil
}

// EvaluateQuery executes a universal query against source code and returns results
func (e *UniversalEvaluator) EvaluateQuery(query *types.Query, source []byte) (*types.ResultSet, error) {
	if query == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}
	if len(source) == 0 {
		return nil, fmt.Errorf("source code cannot be empty")
	}

	// Parse source code into AST
	tree := e.parser.Parse(nil, source)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source code")
	}
	defer tree.Close()

	// Translate universal query to Tree-sitter query
	tsQuery, err := e.provider.TranslateQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	// Execute Tree-sitter query
	q, err := sitter.NewQuery([]byte(tsQuery), e.language)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tree-sitter query: %w", err)
	}
	defer q.Close()

	// Execute query and collect results
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	resultSet := types.NewResultSet()
	qc.Exec(q, tree.RootNode())

	// Process matches
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

			// Create result from node
			result := e.createResultFromNode(node, source, query)
			if result != nil {
				resultSet.Add(result)
			}
		}
	}

	// Apply post-processing filters
	return e.applyFilters(resultSet, query), nil
}

// EvaluateMultipleQueries executes multiple queries and combines results
func (e *UniversalEvaluator) EvaluateMultipleQueries(queries []*types.Query, source []byte) (*types.ResultSet, error) {
	if len(queries) == 0 {
		return types.NewResultSet(), nil
	}

	combinedResults := types.NewResultSet()

	for _, query := range queries {
		results, err := e.EvaluateQuery(query, source)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate query %s: %w", query.Raw, err)
		}
		combinedResults = combinedResults.Merge(results)
	}

	return combinedResults, nil
}

// EvaluateLogicalQuery handles logical operations (AND, OR, NOT)
func (e *UniversalEvaluator) EvaluateLogicalQuery(query *types.Query, source []byte) (*types.ResultSet, error) {
	if query.Operator == "" {
		return e.EvaluateQuery(query, source)
	}

	switch {
	case strings.HasPrefix(query.Operator, "!"):
		return e.evaluateNegation(query, source)
	case query.Operator == "&&":
		return e.evaluateAnd(query, source)
	case query.Operator == "||":
		return e.evaluateOr(query, source)
	case query.Operator == ">":
		return e.evaluateHierarchical(query, source)
	default:
		return e.EvaluateQuery(query, source)
	}
}

// evaluateNegation handles negated queries
func (e *UniversalEvaluator) evaluateNegation(query *types.Query, source []byte) (*types.ResultSet, error) {
	// Create a copy of the query without negation
	positiveQuery := *query
	positiveQuery.Operator = strings.TrimPrefix(query.Operator, "!")

	// Get all possible nodes of the same kind
	allNodes, err := e.getAllNodesOfKind(query.Kind, source)
	if err != nil {
		return nil, err
	}

	// Get nodes that match the positive query
	matchingNodes, err := e.EvaluateQuery(&positiveQuery, source)
	if err != nil {
		return nil, err
	}

	// Return nodes that don't match
	return e.subtractResults(allNodes, matchingNodes), nil
}

// evaluateAnd handles AND operations
func (e *UniversalEvaluator) evaluateAnd(query *types.Query, source []byte) (*types.ResultSet, error) {
	if len(query.Children) < 2 {
		return nil, fmt.Errorf("AND operation requires at least 2 operands")
	}

	// Start with results from first query
	results, err := e.EvaluateLogicalQuery(&query.Children[0], source)
	if err != nil {
		return nil, err
	}

	// Intersect with results from remaining queries
	for i := 1; i < len(query.Children); i++ {
		nextResults, err := e.EvaluateLogicalQuery(&query.Children[i], source)
		if err != nil {
			return nil, err
		}
		results = e.intersectResults(results, nextResults)
	}

	return results, nil
}

// evaluateOr handles OR operations
func (e *UniversalEvaluator) evaluateOr(query *types.Query, source []byte) (*types.ResultSet, error) {
	if len(query.Children) < 2 {
		return nil, fmt.Errorf("OR operation requires at least 2 operands")
	}

	combinedResults := types.NewResultSet()

	// Union results from all queries
	for _, childQuery := range query.Children {
		results, err := e.EvaluateLogicalQuery(&childQuery, source)
		if err != nil {
			return nil, err
		}
		combinedResults = combinedResults.Merge(results)
	}

	return combinedResults, nil
}

// evaluateHierarchical handles parent > child relationships
func (e *UniversalEvaluator) evaluateHierarchical(query *types.Query, source []byte) (*types.ResultSet, error) {
	if len(query.Children) == 0 {
		return nil, fmt.Errorf("hierarchical query requires parent specification")
	}

	parentQuery := &query.Children[0]

	// Find parent nodes
	parentResults, err := e.EvaluateLogicalQuery(parentQuery, source)
	if err != nil {
		return nil, err
	}

	// Find child nodes within parent contexts
	childResults := types.NewResultSet()
	for _, parentResult := range parentResults.All() {
		// Create child query within parent scope
		childQuery := *query
		childQuery.Operator = "" // Remove hierarchy operator
		childQuery.Children = nil
		childQuery.Scope = types.ScopeBlock // Limit to parent scope

		// This would need scope-limited evaluation
		// For now, we'll do a simple implementation
		results, err := e.EvaluateQuery(&childQuery, source)
		if err != nil {
			continue
		}

		// Filter results to only those within parent node
		for _, result := range results.All() {
			if e.isNodeWithinParent(result.Node, parentResult.Node) {
				childResults.Add(result)
			}
		}
	}

	return childResults, nil
}

// createResultFromNode creates a Result object from a Tree-sitter node
func (e *UniversalEvaluator) createResultFromNode(node *sitter.Node, source []byte, query *types.Query) *types.Result {
	if node == nil {
		return nil
	}

	// Get node kind using provider
	nodeKind := e.provider.GetNodeKind(node)

	// Get node name using provider
	nodeName := e.provider.GetNodeName(node, source)

	// Create location info
	location := types.Location{
		StartLine: int(node.StartPoint().Row) + 1, // Convert to 1-based
		EndLine:   int(node.EndPoint().Row) + 1,
		StartCol:  int(node.StartPoint().Column),
		EndCol:    int(node.EndPoint().Column),
	}

	// Create metadata
	metadata := map[string]any{
		"node_type":     node.Type(),
		"byte_range":    fmt.Sprintf("%d-%d", node.StartByte(), node.EndByte()),
		"query_kind":    string(query.Kind),
		"query_pattern": query.Pattern,
	}

	return &types.Result{
		Node:     node,
		Kind:     types.NodeKind(string(nodeKind)),
		Name:     nodeName,
		Location: location,
		Metadata: metadata,
	}
}

// applyFilters applies post-processing filters to results
func (e *UniversalEvaluator) applyFilters(results *types.ResultSet, query *types.Query) *types.ResultSet {
	if query.Pattern == "" && len(query.Attributes) == 0 {
		return results
	}

	filteredResults := types.NewResultSet()

	for _, result := range results.All() {
		if e.matchesFilters(result, query) {
			filteredResults.Add(result)
		}
	}

	return filteredResults
}

// matchesFilters checks if a result matches the query filters
func (e *UniversalEvaluator) matchesFilters(result *types.Result, query *types.Query) bool {
	// Check pattern matching
	if query.Pattern != "" && query.Pattern != "*" {
		if !e.matchesPattern(result.Name, query.Pattern) {
			return false
		}
	}

	// Check attribute filters
	for key, value := range query.Attributes {
		if !e.matchesAttribute(result, key, value) {
			return false
		}
	}

	return true
}

// matchesPattern checks if a name matches a pattern (with wildcard support)
func (e *UniversalEvaluator) matchesPattern(name, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple wildcard matching - can be enhanced
	if strings.Contains(pattern, "*") {
		// Convert to regex-like matching
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		matched, _ := regexp.MatchString("^"+pattern+"$", name)
		return matched
	}

	return name == pattern
}

// matchesAttribute checks if a result matches an attribute filter
func (e *UniversalEvaluator) matchesAttribute(result *types.Result, key, value string) bool {
	// This would need to be implemented based on specific attribute types
	// For now, return true as a placeholder
	return true
}

// Helper methods for set operations

// getAllNodesOfKind gets all nodes of a specific kind
func (e *UniversalEvaluator) getAllNodesOfKind(kind types.NodeKind, source []byte) (*types.ResultSet, error) {
	// Create a wildcard query for the kind
	wildcardQuery := &types.Query{
		Kind:    kind,
		Pattern: "*",
	}

	return e.EvaluateQuery(wildcardQuery, source)
}

// subtractResults returns results in first set but not in second
func (e *UniversalEvaluator) subtractResults(first, second *types.ResultSet) *types.ResultSet {
	result := types.NewResultSet()
	secondNodes := make(map[*sitter.Node]bool)

	// Build lookup map for second set
	for _, r := range second.All() {
		secondNodes[r.Node] = true
	}

	// Add nodes from first set that aren't in second
	for _, r := range first.All() {
		if !secondNodes[r.Node] {
			result.Add(r)
		}
	}

	return result
}

// intersectResults returns results present in both sets
func (e *UniversalEvaluator) intersectResults(first, second *types.ResultSet) *types.ResultSet {
	result := types.NewResultSet()
	secondNodes := make(map[*sitter.Node]bool)

	// Build lookup map for second set
	for _, r := range second.All() {
		secondNodes[r.Node] = true
	}

	// Add nodes from first set that are also in second
	for _, r := range first.All() {
		if secondNodes[r.Node] {
			result.Add(r)
		}
	}

	return result
}

// isNodeWithinParent checks if a node is within the scope of a parent node
func (e *UniversalEvaluator) isNodeWithinParent(child, parent *sitter.Node) bool {
	if child == nil || parent == nil {
		return false
	}

	// Check if child's byte range is within parent's byte range
	return child.StartByte() >= parent.StartByte() && child.EndByte() <= parent.EndByte()
}

// GetProvider returns the language provider used by this evaluator
func (e *UniversalEvaluator) GetProvider() types.LanguageProvider {
	return e.provider
}

// GetLanguage returns the Tree-sitter language used by this evaluator
func (e *UniversalEvaluator) GetLanguage() *sitter.Language {
	return e.language
}
