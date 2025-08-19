package types

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/core"
)

// Type aliases for core contracts to maintain compatibility
// These allow existing code to continue working while using the new pure contracts
type (
	NodeKind  = core.NodeKind
	Query     = core.Query
	ScopeType = core.ScopeType
	Location  = core.Location
)

// Constants are re-exported for backward compatibility
const (
	KindFunction   = core.KindFunction
	KindVariable   = core.KindVariable
	KindClass      = core.KindClass
	KindMethod     = core.KindMethod
	KindImport     = core.KindImport
	KindConstant   = core.KindConstant
	KindField      = core.KindField
	KindCall       = core.KindCall
	KindAssignment = core.KindAssignment
	KindCondition  = core.KindCondition
	KindLoop       = core.KindLoop
	KindBlock      = core.KindBlock
	KindComment    = core.KindComment
	KindDecorator  = core.KindDecorator
	KindType       = core.KindType
	KindInterface  = core.KindInterface
	KindEnum       = core.KindEnum
	KindParameter  = core.KindParameter
	KindReturn     = core.KindReturn
	KindThrow      = core.KindThrow
	KindTryCatch   = core.KindTryCatch
)

const (
	ScopeFile      = core.ScopeFile
	ScopeClass     = core.ScopeClass
	ScopeFunction  = core.ScopeFunction
	ScopeBlock     = core.ScopeBlock
	ScopeNamespace = core.ScopeNamespace
	ScopePackage   = core.ScopePackage
)

// Result extends the core Result with tree-sitter specific functionality
// This bridges the gap between pure contracts and tree-sitter implementation
type Result struct {
	*core.Result
	Node *sitter.Node // Tree-sitter node for language-specific operations
}

// NewResult creates a new Result from core.Result and sitter.Node
func NewResult(coreResult *core.Result, node *sitter.Node) *Result {
	return &Result{
		Result: coreResult,
		Node:   node,
	}
}

// ToCoreResult converts this Result to a pure core.Result
func (r *Result) ToCoreResult() *core.Result {
	return r.Result
}

// ResultSet manages a collection of results with tree-sitter nodes
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

// ToCoreResultSet converts this ResultSet to a core.ResultSet
func (rs *ResultSet) ToCoreResultSet() *core.ResultSet {
	coreResults := make([]*core.Result, len(rs.results))
	for i, result := range rs.results {
		coreResults[i] = result.ToCoreResult()
	}
	return &core.ResultSet{
		Results:      coreResults,
		TotalMatches: len(coreResults),
	}
}
