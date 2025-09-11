package golang

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
)

// EXPERIMENTAL: ParallelQuery has significant overhead due to tree-sitter's
// thread-safety limitations. For most cases, use Query() which is more
// efficient (6.8ms vs 13.5ms in benchmarks).
//
// This method exists primarily for benchmarking and experimentation.
// Tree-sitter maintains a mutable internal cache that causes race conditions
// with concurrent access, forcing sequential data extraction.

// NodeData contains extracted node information for parallel processing
type NodeData struct {
	Type      string
	Name      string
	StartLine int
	StartCol  int
}

// ParallelQuery executes queries with safe concurrent processing (EXPERIMENTAL)
func (p *Provider) ParallelQuery(source string, query core.AgentQuery) core.QueryResult {
	tree, err := p.parser.ParseCtx(context.TODO(), nil, []byte(source))
	if err != nil || tree == nil {
		return core.QueryResult{Error: fmt.Errorf("parse failed: %v", err)}
	}
	defer tree.Close()

	// Phase 1: Extract node data sequentially (tree-sitter is not thread-safe)
	nodes := p.extractAllNodes(tree.RootNode(), source)

	// Phase 2: Process nodes in parallel (no tree access needed)
	numWorkers := runtime.NumCPU()
	workChan := make(chan NodeData, len(nodes))
	resultChan := make(chan core.Match, len(nodes))

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for nodeData := range workChan {
				if match := p.processNodeData(nodeData, query); match != nil {
					resultChan <- *match
				}
			}
		}()
	}

	// Queue all work
	for _, node := range nodes {
		workChan <- node
	}
	close(workChan)

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var matches []core.Match
	for match := range resultChan {
		matches = append(matches, match)
	}

	return core.QueryResult{Matches: matches}
}

// extractAllNodes walks the tree sequentially and extracts node data
func (p *Provider) extractAllNodes(root *sitter.Node, source string) []NodeData {
	var nodes []NodeData

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		// Extract data while we have safe access to the tree
		nodeData := NodeData{
			Type:      node.Type(),
			Name:      p.extractNodeName(node, source),
			StartLine: int(node.StartPoint().Row) + 1,
			StartCol:  int(node.StartPoint().Column) + 1,
		}
		nodes = append(nodes, nodeData)

		// Recurse through children
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return nodes
}

// processNodeData checks if extracted node data matches the query (no tree access)
func (p *Provider) processNodeData(nodeData NodeData, query core.AgentQuery) *core.Match {
	// Map natural language terms to AST node types
	typeMatches := false
	switch query.Type {
	case "function", "func":
		typeMatches = nodeData.Type == "function_declaration" || nodeData.Type == "method_declaration"
	case "struct":
		typeMatches = nodeData.Type == "type_spec" // More complex check needed but simplified for parallel safety
	case "interface":
		typeMatches = nodeData.Type == "type_spec" // Simplified
	case "variable", "var":
		typeMatches = nodeData.Type == "var_declaration" || nodeData.Type == "short_var_declaration"
	case "constant", "const":
		typeMatches = nodeData.Type == "const_declaration"
	case "import":
		typeMatches = nodeData.Type == "import_declaration"
	}

	if !typeMatches {
		return nil
	}

	// Check name pattern if specified
	if query.Name != "" && !p.matchesPattern(nodeData.Name, query.Name) {
		return nil
	}

	return &core.Match{
		Type: query.Type,
		Name: nodeData.Name,
		Location: core.Location{
			Line:   nodeData.StartLine,
			Column: nodeData.StartCol,
		},
	}
}
