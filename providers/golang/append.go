package golang

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/termfx/morfx/core"
)

// SmartAppend adds content at the optimal location based on content type
func (p *Provider) SmartAppend(source string, content string) core.TransformResult {
	// Parse source
	tree, err := p.parser.ParseCtx(context.TODO(), nil, []byte(source))
	if err != nil || tree == nil {
		return core.TransformResult{
			Error: fmt.Errorf("failed to parse source"),
		}
	}
	defer tree.Close()

	// Detect what type of content we're appending
	contentType := p.detectContentType(content)

	// Find the last occurrence of similar content
	var insertPoint *InsertionPoint

	switch contentType {
	case "method":
		// Find receiver and last method with same receiver
		receiver := p.extractReceiver(content)
		insertPoint = p.findLastMethod(tree, source, receiver)

	case "function":
		// Find last top-level function
		insertPoint = p.findLastFunction(tree, source)

	case "type":
		// Find last type declaration
		insertPoint = p.findLastType(tree, source)

	default:
		// Default to end of file
		insertPoint = &InsertionPoint{
			Position: uint32(len(source)),
			Strategy: "End of file",
			Prefix:   "\n\n",
			Suffix:   "",
		}
	}

	// Build modified source
	modified := p.insertAtPoint(source, content, insertPoint)

	// Calculate confidence
	confidence := p.calculateAppendConfidence(insertPoint, content)

	// Generate diff
	diff := p.generateDiff(source, modified)

	return core.TransformResult{
		Modified:   modified,
		Diff:       diff,
		Confidence: confidence,
		MatchCount: 1,
		Metadata: map[string]interface{}{
			"strategy": insertPoint.Strategy,
			"position": insertPoint.Position,
		},
	}
}

// detectContentType analyzes what kind of Go code this is
func (p *Provider) detectContentType(content string) string {
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "func (") {
		return "method"
	}
	if strings.HasPrefix(trimmed, "func ") {
		return "function"
	}
	if strings.HasPrefix(trimmed, "type ") {
		return "type"
	}
	if strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
		return "declaration"
	}

	return "unknown"
}

// findLastMethod finds the last method with matching receiver
func (p *Provider) findLastMethod(tree *sitter.Tree, source string, receiver string) *InsertionPoint {
	root := tree.RootNode()

	var lastMethod *sitter.Node
	var lastPos uint32

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_declaration" {
			// Check if receiver matches
			if receiverNode := node.ChildByFieldName("receiver"); receiverNode != nil {
				methodReceiver := p.extractReceiverFromNode(receiverNode, source)
				if methodReceiver == receiver || receiver == "" {
					if node.EndByte() > lastPos {
						lastMethod = node
						lastPos = node.EndByte()
					}
				}
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)

	if lastMethod != nil {
		return &InsertionPoint{
			Position: lastMethod.EndByte(),
			Strategy: fmt.Sprintf("After last method of %s", receiver),
			Prefix:   "\n\n",
			Suffix:   "",
		}
	}

	// No methods found, append to end
	return &InsertionPoint{
		Position: uint32(len(source)),
		Strategy: "End of file (no methods found)",
		Prefix:   "\n\n",
		Suffix:   "",
	}
}

// findLastFunction finds the last top-level function
func (p *Provider) findLastFunction(tree *sitter.Tree, source string) *InsertionPoint {
	root := tree.RootNode()

	var lastFunc *sitter.Node
	var lastPos uint32

	var walk func(*sitter.Node, int)
	walk = func(node *sitter.Node, depth int) {
		if node.Type() == "function_declaration" && depth == 1 {
			if node.EndByte() > lastPos {
				lastFunc = node
				lastPos = node.EndByte()
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i), depth+1)
		}
	}

	walk(root, 0)

	if lastFunc != nil {
		return &InsertionPoint{
			Position: lastFunc.EndByte(),
			Strategy: "After last function",
			Prefix:   "\n\n",
			Suffix:   "",
		}
	}

	return &InsertionPoint{
		Position: uint32(len(source)),
		Strategy: "End of file (no functions found)",
		Prefix:   "\n\n",
		Suffix:   "",
	}
}

// findLastType finds the last type declaration
func (p *Provider) findLastType(tree *sitter.Tree, source string) *InsertionPoint {
	root := tree.RootNode()

	var lastType *sitter.Node
	var lastPos uint32

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "type_declaration" || node.Type() == "type_spec" {
			if node.EndByte() > lastPos {
				lastType = node
				lastPos = node.EndByte()
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)

	if lastType != nil {
		return &InsertionPoint{
			Position: lastType.EndByte(),
			Strategy: "After last type",
			Prefix:   "\n\n",
			Suffix:   "",
		}
	}

	return &InsertionPoint{
		Position: uint32(len(source)),
		Strategy: "End of file (no types found)",
		Prefix:   "\n\n",
		Suffix:   "",
	}
}

// InsertionPoint describes where to insert
type InsertionPoint struct {
	Position uint32
	Strategy string
	Prefix   string
	Suffix   string
}

// insertAtPoint inserts content at the specified point
func (p *Provider) insertAtPoint(source, content string, point *InsertionPoint) string {
	before := source[:point.Position]
	after := source[point.Position:]

	insertion := point.Prefix + content + point.Suffix

	return before + insertion + after
}

// extractReceiver gets receiver type from method signature
func (p *Provider) extractReceiver(methodSig string) string {
	start := strings.Index(methodSig, "(")
	end := strings.Index(methodSig, ")")

	if start > 0 && end > start {
		receiver := methodSig[start+1 : end]
		parts := strings.Fields(receiver)
		if len(parts) >= 2 {
			typeName := strings.TrimPrefix(parts[1], "*")
			return typeName
		}
	}

	return ""
}

// extractReceiverFromNode gets receiver from AST node
func (p *Provider) extractReceiverFromNode(receiverNode *sitter.Node, source string) string {
	receiverText := source[receiverNode.StartByte():receiverNode.EndByte()]
	return p.extractReceiver("func " + receiverText + " dummy()")
}

// calculateAppendConfidence scores the append operation
func (p *Provider) calculateAppendConfidence(point *InsertionPoint, content string) core.ConfidenceScore {
	score := 1.0
	factors := []core.ConfidenceFactor{}

	// All appends at end of file or after similar elements are safe
	if strings.Contains(point.Strategy, "After last") {
		factors = append(factors, core.ConfidenceFactor{
			Name:   "semantic_position",
			Impact: 0.05,
			Reason: "Appending after similar elements",
		})
		score = 1.0
	}

	// Large insertions slightly lower confidence
	if lines := strings.Count(content, "\n"); lines > 20 {
		score -= 0.1
		factors = append(factors, core.ConfidenceFactor{
			Name:   "large_content",
			Impact: -0.1,
			Reason: fmt.Sprintf("Inserting %d lines", lines),
		})
	}

	level := "high"
	if score < 0.8 {
		level = "medium"
	}

	return core.ConfidenceScore{
		Score:   score,
		Level:   level,
		Factors: factors,
	}
}
