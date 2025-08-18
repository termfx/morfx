package types

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// Universal DSL concepts that ALL languages must map
type NodeKind string

const (
	KindFunction   NodeKind = "function"
	KindVariable   NodeKind = "variable"
	KindClass      NodeKind = "class"
	KindMethod     NodeKind = "method"
	KindImport     NodeKind = "import"
	KindConstant   NodeKind = "constant"
	KindField      NodeKind = "field"
	KindCall       NodeKind = "call"
	KindAssignment NodeKind = "assignment"
	KindCondition  NodeKind = "condition"
	KindLoop       NodeKind = "loop"
	KindBlock      NodeKind = "block"
	KindComment    NodeKind = "comment"
	KindDecorator  NodeKind = "decorator"
	KindType       NodeKind = "type"
)

// Universal query structure - no language specifics
type Query struct {
	Kind       NodeKind          // Universal node kind
	Pattern    string            // Name/identifier pattern
	Attributes map[string]string // type, visibility, etc.
	Operator   string            // &&, ||, >, !
	Children   []Query           // Nested queries
	Scope      ScopeType         // Where to apply operations
	Raw        string            // Original DSL string
}

// Scope types universal across languages
type ScopeType string

const (
	ScopeFile      ScopeType = "file"
	ScopeClass     ScopeType = "class"
	ScopeFunction  ScopeType = "function"
	ScopeBlock     ScopeType = "block"
	ScopeNamespace ScopeType = "namespace"
)

// Result is language-agnostic
type Result struct {
	Node     *sitter.Node
	Kind     NodeKind
	Name     string
	Location Location
	Metadata map[string]any
}

type Location struct {
	File      string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
}

// ResultSet manages a collection of results
type ResultSet struct {
	results []*Result
	index   map[string]*Result // For fast lookups
}

// NewResultSet creates a new result set
func NewResultSet() *ResultSet {
	return &ResultSet{
		results: make([]*Result, 0),
		index:   make(map[string]*Result),
	}
}

// Add adds a result to the set
func (rs *ResultSet) Add(result *Result) {
	key := fmt.Sprintf("%s:%s:%s:%d:%d", result.Location.File, result.Kind, result.Name, result.Location.StartLine, result.Location.StartCol)
	if _, exists := rs.index[key]; !exists {
		rs.results = append(rs.results, result)
		rs.index[key] = result
	}
}

// All returns all results
func (rs *ResultSet) All() []*Result {
	return rs.results
}

// Count returns the number of results
func (rs *ResultSet) Count() int {
	return len(rs.results)
}

// Filter filters results by a predicate function
func (rs *ResultSet) Filter(predicate func(*Result) bool) *ResultSet {
	filtered := NewResultSet()
	for _, result := range rs.results {
		if predicate(result) {
			filtered.Add(result)
		}
	}
	return filtered
}

// Merge combines two result sets
func (rs *ResultSet) Merge(other *ResultSet) *ResultSet {
	merged := NewResultSet()
	for _, result := range rs.results {
		merged.Add(result)
	}
	for _, result := range other.results {
		merged.Add(result)
	}
	return merged
}
