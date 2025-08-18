package provider

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/types"
)

// NodeMapping is now an alias to types.NodeMapping
type NodeMapping = types.NodeMapping

// LanguageProvider is now just an alias to types.LanguageProvider
type LanguageProvider = types.LanguageProvider

// Base provider with common functionality
type BaseProvider struct {
	mappings map[types.NodeKind][]NodeMapping
	cache    map[string]string
}

// Helper methods that all providers can use
func (b *BaseProvider) BuildMappings(mappings []NodeMapping) {
	b.mappings = make(map[types.NodeKind][]NodeMapping)
	for _, m := range mappings {
		b.mappings[m.Kind] = append(b.mappings[m.Kind], m)
	}
}

func (b *BaseProvider) CacheQuery(key, query string) {
	if b.cache == nil {
		b.cache = make(map[string]string)
	}
	b.cache[key] = query
}

// GetCachedQuery retrieves a cached query
func (b *BaseProvider) GetCachedQuery(key string) (string, bool) {
	if b.cache == nil {
		return "", false
	}
	query, exists := b.cache[key]
	return query, exists
}

// TranslateKind returns the node mappings for a given kind
func (b *BaseProvider) TranslateKind(kind types.NodeKind) []NodeMapping {
	if b.mappings == nil {
		return nil
	}
	return b.mappings[kind]
}

// ParseAttributes provides a default implementation
func (b *BaseProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	attrs := make(map[string]string)
	attrs["type"] = node.Type()
	attrs["start_line"] = string(rune(node.StartPoint().Row))
	attrs["end_line"] = string(rune(node.EndPoint().Row))
	return attrs
}

// OptimizeQuery provides a default implementation (no optimization)
func (b *BaseProvider) OptimizeQuery(q *types.Query) *types.Query {
	return q
}

// EstimateQueryCost provides a default cost estimation
func (b *BaseProvider) EstimateQueryCost(q *types.Query) int {
	// Simple heuristic: base cost + children cost
	cost := 1
	for _, child := range q.Children {
		cost += b.EstimateQueryCost(&child)
	}
	return cost
}

// GetNodeScope provides a default implementation
func (b *BaseProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	// Default implementation - providers can override
	switch node.Type() {
	case "function_declaration", "method_declaration":
		return "function"
	case "type_declaration", "struct_type", "class_definition":
		return "class"
	case "block":
		return "block"
	default:
		return "file"
	}
}

// FindEnclosingScope provides a default implementation
func (b *BaseProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	current := node.Parent()
	for current != nil {
		if b.GetNodeScope(current) == scope {
			return current
		}
		current = current.Parent()
	}
	return nil
}

// IsBlockLevelNode determines if a node type should be treated as block-level for formatting
func (b *BaseProvider) IsBlockLevelNode(nodeType string) bool {
	// Default implementation
	switch nodeType {
	case "block", "function_declaration", "class_definition", "method_declaration":
		return true
	default:
		return false
	}
}

// GetDefaultIgnorePatterns provides default ignore patterns
func (b *BaseProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	// Default patterns that most languages can use
	files = []string{
		"*_test.*",
		"test_*.*",
		"*.test.*",
		"vendor/*",
		"node_modules/*",
		".git/*",
	}
	symbols = []string{
		"test*",
		"Test*",
		"*_test",
		"*Test",
	}
	return files, symbols
}

// NormalizeDSLKind provides default implementation (no transformation)
func (b *BaseProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	return types.NodeKind(dslKind)
}

// GetSupportedDSLKinds provides default implementation (empty list)
func (b *BaseProvider) GetSupportedDSLKinds() []string {
	return []string{}
}
