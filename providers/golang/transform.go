package golang

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/termfx/morfx/core"
)

// Transform executes a transformation operation on Go source code
func (p *Provider) Transform(source string, op core.TransformOp) core.TransformResult {
	// Parse source using ParseCtx
	tree, err := p.parser.ParseCtx(context.TODO(), nil, []byte(source))
	if err != nil || tree == nil {
		return core.TransformResult{
			Error: fmt.Errorf("failed to parse source: %v", err),
		}
	}
	defer tree.Close()

	// Find targets
	targets := p.findTargets(tree.RootNode(), source, op.Target)
	if len(targets) == 0 {
		return core.TransformResult{
			Error: fmt.Errorf("no matches found for target"),
		}
	}

	// Use Pipeline for multiple targets (automatic optimization)
	if len(targets) > 1 {
		pipeline := p.NewTransformPipeline()
		return pipeline.BatchTransform(source, op, targets)
	}

	// Single target: use simple sequential transform
	confidence := p.calculateConfidence(op, targets, source)
	var modified string

	switch op.Method {
	case "replace":
		modified, err = p.doReplace(source, targets, op.Replacement)
	case "delete":
		modified, err = p.doDelete(source, targets)
	case "insert_before":
		modified, err = p.doInsertBefore(source, targets, op.Content)
	case "insert_after":
		modified, err = p.doInsertAfter(source, targets, op.Content)
	case "append":
		modified, err = p.doAppendToTarget(source, targets, op.Content)
	default:
		return core.TransformResult{
			Error: fmt.Errorf("unknown transform method: %s", op.Method),
		}
	}

	if err != nil {
		return core.TransformResult{
			Error: err,
		}
	}

	// Generate diff
	diff := p.generateDiff(source, modified)

	return core.TransformResult{
		Modified:   modified,
		Diff:       diff,
		Confidence: confidence,
		MatchCount: len(targets), // Track how many elements were actually transformed
	}
}

// findTargets finds all nodes matching the query
func (p *Provider) findTargets(root *sitter.Node, source string, query core.AgentQuery) []*sitter.Node {
	var targets []*sitter.Node

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		// Check if this node matches
		if p.nodeMatches(node, source, query) {
			targets = append(targets, node)
		}

		// Recurse children
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return targets
}

// nodeMatches checks if a node matches the query
func (p *Provider) nodeMatches(node *sitter.Node, source string, query core.AgentQuery) bool {
	// Map query type to AST node types
	nodeTypes := p.getNodeTypesForQuery(query.Type)

	// Check if node type matches
	typeMatches := false
	for _, nt := range nodeTypes {
		if node.Type() == nt {
			typeMatches = true
			break
		}
	}

	if !typeMatches {
		return false
	}

	// Check name if specified
	if query.Name != "" {
		name := p.extractNodeName(node, source)
		if !p.matchesPattern(name, query.Name) {
			return false
		}
	}

	return true
}

// extractNodeName gets the name from various node types
func (p *Provider) extractNodeName(node *sitter.Node, source string) string {
	// Try standard name field
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}

	// For other nodes, try first identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return source[child.StartByte():child.EndByte()]
		}
	}

	return ""
}

// doReplace performs replacement transformation
func (p *Provider) doReplace(source string, targets []*sitter.Node, replacement string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets to replace")
	}

	// Sort targets by position (reverse order to maintain positions)
	sortedTargets := make([]*sitter.Node, len(targets))
	copy(sortedTargets, targets)
	// Sort reverse order to replace from end to beginning
	for i := 0; i < len(sortedTargets)-1; i++ {
		for j := i + 1; j < len(sortedTargets); j++ {
			if sortedTargets[i].StartByte() < sortedTargets[j].StartByte() {
				sortedTargets[i], sortedTargets[j] = sortedTargets[j], sortedTargets[i]
			}
		}
	}

	// Replace each target
	result := source
	for _, target := range sortedTargets {
		before := result[:target.StartByte()]
		after := result[target.EndByte():]
		result = before + replacement + after
	}

	return result, nil
}

// doDelete performs deletion transformation
func (p *Provider) doDelete(source string, targets []*sitter.Node) (string, error) {
	// Delete is replace with empty string
	return p.doReplace(source, targets, "")
}

// doInsertBefore performs insertion before target
func (p *Provider) doInsertBefore(source string, targets []*sitter.Node, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for insertion")
	}

	// Sort targets reverse order
	sortedTargets := make([]*sitter.Node, len(targets))
	copy(sortedTargets, targets)
	for i := 0; i < len(sortedTargets)-1; i++ {
		for j := i + 1; j < len(sortedTargets); j++ {
			if sortedTargets[i].StartByte() < sortedTargets[j].StartByte() {
				sortedTargets[i], sortedTargets[j] = sortedTargets[j], sortedTargets[i]
			}
		}
	}

	// Insert before each target
	result := source
	for _, target := range sortedTargets {
		before := result[:target.StartByte()]
		after := result[target.StartByte():]

		// Preserve indentation
		indent := p.getIndentation(source, target)
		contentWithIndent := indent + content + "\n"

		result = before + contentWithIndent + after
	}

	return result, nil
}

// doInsertAfter performs insertion after target
func (p *Provider) doInsertAfter(source string, targets []*sitter.Node, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for insertion")
	}

	// Sort targets reverse order
	sortedTargets := make([]*sitter.Node, len(targets))
	copy(sortedTargets, targets)
	for i := 0; i < len(sortedTargets)-1; i++ {
		for j := i + 1; j < len(sortedTargets); j++ {
			if sortedTargets[i].StartByte() < sortedTargets[j].StartByte() {
				sortedTargets[i], sortedTargets[j] = sortedTargets[j], sortedTargets[i]
			}
		}
	}

	// Insert after each target
	result := source
	for _, target := range sortedTargets {
		before := result[:target.EndByte()]
		after := result[target.EndByte():]

		// Preserve indentation
		indent := p.getIndentation(source, target)
		contentWithIndent := "\n" + indent + content

		result = before + contentWithIndent + after
	}

	return result, nil
}

// getIndentation extracts indentation for a node
func (p *Provider) getIndentation(source string, node *sitter.Node) string {
	line := node.StartPoint().Row
	lineStart := 0
	currentLine := uint32(0)

	// Find start of the line
	for i, ch := range source {
		if currentLine == line {
			lineStart = i
			break
		}
		if ch == '\n' {
			currentLine++
		}
	}

	// Extract indentation
	indent := ""
	for i := lineStart; i < len(source); i++ {
		if source[i] == ' ' || source[i] == '\t' {
			indent += string(source[i])
		} else {
			break
		}
	}

	return indent
}

// calculateConfidence calculates transformation confidence
func (p *Provider) calculateConfidence(op core.TransformOp, targets []*sitter.Node, source string) core.ConfidenceScore {
	score := 1.0
	factors := []core.ConfidenceFactor{}

	// Factor 1: Number of targets
	if len(targets) == 1 {
		score += 0.1
		factors = append(factors, core.ConfidenceFactor{
			Name:   "single_target",
			Impact: 0.1,
			Reason: "Only one target found, unambiguous",
		})
	} else if len(targets) > 5 {
		score -= 0.3
		factors = append(factors, core.ConfidenceFactor{
			Name:   "multiple_targets",
			Impact: -0.3,
			Reason: fmt.Sprintf("Operation affects %d locations", len(targets)),
		})
	}
	// Factor 2: Operation type
	switch op.Method {
	case "delete":
		score -= 0.2
		factors = append(factors, core.ConfidenceFactor{
			Name:   "delete_operation",
			Impact: -0.2,
			Reason: "Delete operations are destructive",
		})
	case "replace":
		// Check if replacing exported function
		if len(targets) > 0 && p.isExported(p.extractNodeName(targets[0], source)) {
			score -= 0.2
			factors = append(factors, core.ConfidenceFactor{
				Name:   "exported_api",
				Impact: -0.2,
				Reason: "Modifying exported API",
			})
		}
	}

	// Factor 3: Pattern specificity
	if strings.Contains(op.Target.Name, "*") {
		score -= 0.15
		factors = append(factors, core.ConfidenceFactor{
			Name:   "wildcard_pattern",
			Impact: -0.15,
			Reason: "Wildcard patterns may match unintended targets",
		})
	}

	// Normalize score
	if score < 0 {
		score = 0
	} else if score > 1 {
		score = 1
	}

	// Determine level
	level := "high"
	if score < 0.8 {
		level = "medium"
	}
	if score < 0.5 {
		level = "low"
	}

	return core.ConfidenceScore{
		Score:   score,
		Level:   level,
		Factors: factors,
	}
}

// isExported checks if identifier is exported (starts with capital)
func (p *Provider) isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// generateDiff creates a unified diff
func (p *Provider) generateDiff(original, modified string) string {
	if original == modified {
		return ""
	}

	// Simple line-based diff for MVP
	originalLines := strings.Split(original, "\n")
	modifiedLines := strings.Split(modified, "\n")

	diff := "--- original\n+++ modified\n"

	// Find first difference
	firstDiff := -1
	for i := 0; i < len(originalLines) && i < len(modifiedLines); i++ {
		if originalLines[i] != modifiedLines[i] {
			firstDiff = i
			break
		}
	}

	if firstDiff == -1 {
		// Length difference
		if len(originalLines) > len(modifiedLines) {
			firstDiff = len(modifiedLines)
		} else {
			firstDiff = len(originalLines)
		}
	}

	// Show context around changes (simplified)
	start := firstDiff - 2
	if start < 0 {
		start = 0
	}

	diff += fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		start+1, len(originalLines)-start,
		start+1, len(modifiedLines)-start)

	// Add some context lines
	for i := start; i < firstDiff && i < len(originalLines); i++ {
		diff += " " + originalLines[i] + "\n"
	}

	// Show changes
	if firstDiff < len(originalLines) {
		for i := firstDiff; i < len(originalLines) && i < firstDiff+5; i++ {
			diff += "-" + originalLines[i] + "\n"
		}
	}
	if firstDiff < len(modifiedLines) {
		for i := firstDiff; i < len(modifiedLines) && i < firstDiff+5; i++ {
			diff += "+" + modifiedLines[i] + "\n"
		}
	}

	return diff
}

// getNodeTypesForQuery maps query types to AST node types
func (p *Provider) getNodeTypesForQuery(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_declaration", "method_declaration"}
	case "struct":
		return []string{"type_spec"} // Need additional check for struct type
	case "interface":
		return []string{"type_spec"} // Need additional check for interface type
	case "variable", "var":
		return []string{"var_declaration", "short_var_declaration"}
	case "constant", "const":
		return []string{"const_declaration"}
	case "import":
		return []string{"import_declaration"}
	case "type":
		return []string{"type_declaration", "type_spec"}
	case "method":
		return []string{"method_declaration"}
	case "field":
		return []string{"field_declaration"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// doAppendToTarget appends content to the end of target scope
func (p *Provider) doAppendToTarget(source string, targets []*sitter.Node, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for append")
	}

	// For append, we only use first target
	target := targets[0]

	// Find the appropriate insertion point based on target type
	var insertPos uint32
	var needsNewline bool

	switch target.Type() {
	case "type_spec":
		// For struct/interface, append inside the body
		if typeNode := target.ChildByFieldName("type"); typeNode != nil {
			if typeNode.Type() == "struct_type" || typeNode.Type() == "interface_type" {
				// Find closing brace
				insertPos = typeNode.EndByte() - 1 // Before }
				needsNewline = true
			} else {
				// Not a struct/interface, append after
				insertPos = target.EndByte()
			}
		} else {
			insertPos = target.EndByte()
		}

	case "function_declaration", "method_declaration":
		// For functions, append inside body
		if body := target.ChildByFieldName("body"); body != nil {
			insertPos = body.EndByte() - 1 // Before closing }
			needsNewline = true
		} else {
			// No body, append after
			insertPos = target.EndByte()
		}

	default:
		// Default: append after target
		insertPos = target.EndByte()
	}

	// Build insertion
	before := source[:insertPos]
	after := source[insertPos:]

	// Add proper formatting
	indent := p.getIndentation(source, target)
	var insertion string

	if needsNewline {
		// Inside a scope - detect if tabs or spaces from existing content
		innerIndent := p.detectInnerIndentation(source, target)
		insertion = "\n" + innerIndent + content + "\n" + indent
	} else {
		// After target
		insertion = "\n\n" + content
	}

	return before + insertion + after, nil
}

// detectInnerIndentation finds the indentation used inside a scope
func (p *Provider) detectInnerIndentation(source string, node *sitter.Node) string {
	// Look at the first child inside the scope to detect indentation
	startByte := node.StartByte()
	endByte := node.EndByte()

	if startByte >= endByte {
		return "\t" // Default to tab
	}

	// Find first line after opening brace
	content := source[startByte:endByte]
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			// Count leading whitespace
			leadingSpace := len(line) - len(strings.TrimLeft(line, " \t"))
			if leadingSpace > 0 {
				return line[:leadingSpace]
			}
		}
	}

	// Default: parent indent + tab
	parentIndent := p.getIndentation(source, node)
	return parentIndent + "\t"
}
