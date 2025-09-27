package base

import (
	"fmt"
	"math"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pmezard/go-difflib/difflib"
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/providers"
)

// LanguageConfig defines language-specific behavior that must be implemented
type LanguageConfig interface {
	// Metadata
	Language() string
	Extensions() []string
	GetLanguage() *sitter.Language

	// Language-specific AST mapping
	MapQueryTypeToNodeTypes(queryType string) []string
	ExtractNodeName(node *sitter.Node, source string) string
	IsExported(name string) bool // For confidence calculation

	// For discovery/specification
	SupportedQueryTypes() []string
}

// SmartAppendConfig allows language configs to provide smarter append behaviour.
type SmartAppendConfig interface {
	SmartAppend(source string, target *sitter.Node, content string) (string, bool)
}

// Provider provides common functionality for all language providers
type Provider struct {
	config LanguageConfig
	cache  *ASTCache
	pool   *sync.Pool
	stats  providerStats
}

type providerStats struct {
	borrowCount atomic.Int64
	returnCount atomic.Int64
	active      atomic.Int64
}

// New creates a base provider with language-specific config
func New(config LanguageConfig) *Provider {
	lang := config.GetLanguage()
	if lang == nil {
		panic(fmt.Sprintf("Failed to load %s language for tree-sitter", config.Language()))
	}

	pool := &sync.Pool{
		New: func() any {
			parser := sitter.NewParser()
			parser.SetLanguage(lang)
			return parser
		},
	}

	return &Provider{
		config: config,
		cache:  GlobalCache,
		pool:   pool,
	}
}

// Language returns language identifier
func (p *Provider) Language() string {
	return p.config.Language()
}

// Extensions returns supported file extensions
func (p *Provider) Extensions() []string {
	return p.config.Extensions()
}

// SupportedQueryTypes lists human-friendly query types/aliases
func (p *Provider) SupportedQueryTypes() []string {
	return p.config.SupportedQueryTypes()
}

// borrowParser retrieves a parser instance from the pool.
func (p *Provider) borrowParser() *sitter.Parser {
	parser := p.pool.Get().(*sitter.Parser)
	p.stats.borrowCount.Add(1)
	p.stats.active.Add(1)
	return parser
}

// releaseParser returns a parser instance to the pool.
func (p *Provider) releaseParser(parser *sitter.Parser) {
	if parser != nil {
		p.stats.returnCount.Add(1)
		p.stats.active.Add(-1)
		p.pool.Put(parser)
	}
}

// Stats returns the current parser pool metrics for this provider.
func (p *Provider) Stats() providers.Stats {
	return providers.Stats{
		BorrowCount: p.stats.borrowCount.Load(),
		ReturnCount: p.stats.returnCount.Load(),
		Active:      p.stats.active.Load(),
	}
}

// Query finds code elements matching the query
func (p *Provider) Query(source string, query core.AgentQuery) core.QueryResult {
	parser := p.borrowParser()
	defer p.releaseParser(parser)

	tree, hit := p.cache.GetOrParse(parser, []byte(source))
	if tree == nil {
		if hit {
			return core.QueryResult{Error: fmt.Errorf("failed to copy cached tree")}
		}
		return core.QueryResult{Error: fmt.Errorf("failed to parse source")}
	}
	defer tree.Close()

	// Check for syntax errors first
	var errors []string
	p.findErrors(tree.RootNode(), source, &errors)
	if len(errors) > 0 {
		return core.QueryResult{Error: fmt.Errorf("syntax errors in source: %v", errors)}
	}

	var matches []core.Match
	p.walkTree(tree.RootNode(), source, query, &matches)

	return core.QueryResult{
		Matches: matches,
		Total:   len(matches),
	}
}

// walkTree recursively walks AST looking for matches
func (p *Provider) walkTree(node *sitter.Node, source string, query core.AgentQuery, matches *[]core.Match) {
	if match := p.checkNode(node, source, query); match != nil {
		*matches = append(*matches, *match)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		p.walkTree(child, source, query, matches)
	}
}

// checkNode checks if a node matches the query using language-specific mapping
func (p *Provider) checkNode(node *sitter.Node, source string, query core.AgentQuery) *core.Match {
	nodeType := node.Type()

	// Get valid node types for this query from language config
	validTypes := p.config.MapQueryTypeToNodeTypes(query.Type)
	if !slices.Contains(validTypes, nodeType) {
		return nil
	}

	// Extract name using language-specific logic
	name := p.config.ExtractNodeName(node, source)
	if name == "" {
		name = "anonymous"
	}

	// Check name pattern
	if !p.matchesPattern(name, query.Name) {
		return nil
	}

	return &core.Match{
		Type: query.Type,
		Name: name,
		Location: core.Location{
			Line:      int(node.StartPoint().Row) + 1,
			Column:    int(node.StartPoint().Column) + 1,
			EndLine:   int(node.EndPoint().Row) + 1,
			EndColumn: int(node.EndPoint().Column) + 1,
		},
		Content: source[node.StartByte():node.EndByte()],
	}
}

// Transform applies a transformation operation
func (p *Provider) Transform(source string, op core.TransformOp) core.TransformResult {
	parser := p.borrowParser()
	defer p.releaseParser(parser)

	tree, hit := p.cache.GetOrParse(parser, []byte(source))
	if tree == nil {
		err := fmt.Errorf("failed to parse source")
		if hit {
			err = fmt.Errorf("failed to copy cached tree")
		}
		return core.TransformResult{Error: err}
	}
	defer tree.Close()

	// Find targets
	matches := p.findTargets(tree.RootNode(), source, op.Target)
	if len(matches) == 0 {
		return core.TransformResult{
			Error: fmt.Errorf("no matches found for target"),
		}
	}

	// Extract nodes for transformation operations
	nodes := make([]*sitter.Node, len(matches))
	for i, match := range matches {
		nodes[i] = match.Node
	}

	// Calculate confidence
	confidence := p.calculateConfidence(op, nodes, source)
	var (
		modified string
		err      error
	)

	switch op.Method {
	case "replace":
		modified, err = p.doReplace(source, nodes, op.Replacement)
	case "delete":
		modified, err = p.doDelete(source, nodes)
	case "insert_before":
		modified, err = p.doInsertBefore(source, nodes, op.Content)
	case "insert_after":
		modified, err = p.doInsertAfter(source, nodes, op.Content)
	case "append":
		modified, err = p.doAppendToTarget(source, nodes, op.Content)
	default:
		return core.TransformResult{
			Error: fmt.Errorf("unknown transform method: %s", op.Method),
		}
	}

	if err != nil {
		return core.TransformResult{Error: err}
	}

	// Generate diff
	diff := p.generateDiff(source, modified)

	p.adjustConfidence(&confidence, op, source, modified, nodes)

	return core.TransformResult{
		Modified:   modified,
		Diff:       diff,
		Confidence: confidence,
		MatchCount: len(matches), // Now shows actual match count including expansions
	}
}

// Validate checks syntax
func (p *Provider) Validate(source string) providers.ValidationResult {
	parser := p.borrowParser()
	defer p.releaseParser(parser)

	tree := parser.Parse(nil, []byte(source))
	if tree == nil {
		return providers.ValidationResult{
			Valid:  false,
			Errors: []string{"Failed to parse source"},
		}
	}
	defer tree.Close()

	var errors []string
	p.findErrors(tree.RootNode(), source, &errors)

	return providers.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// matchesPattern checks if name matches pattern (with wildcards)
func (p *Provider) matchesPattern(name, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}

	return matched
}

// findTargets finds all matches for the query with proper expansion
func (p *Provider) findTargets(root *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if p.nodeMatches(node, source, query) {
			// Expand node into multiple matches if needed
			expanded := p.expandMatches(node, source, query)
			matches = append(matches, expanded...)
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return matches
}

// expandMatches converts a node into one or more matches
func (p *Provider) expandMatches(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	// Check if provider has custom expansion logic
	if expander, ok := p.config.(interface {
		ExpandMatches(*sitter.Node, string, core.AgentQuery) []core.CodeMatch
	}); ok {
		return expander.ExpandMatches(node, source, query)
	}

	// Default: single match
	name := p.config.ExtractNodeName(node, source)
	return []core.CodeMatch{{
		Node:      node,
		Name:      name,
		Type:      query.Type,
		NodeType:  node.Type(),
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
		Line:      node.StartPoint().Row,
		Column:    node.StartPoint().Column,
	}}
}

// nodeMatches checks if a node matches the query with provider-specific validation
func (p *Provider) nodeMatches(node *sitter.Node, source string, query core.AgentQuery) bool {
	nodeTypes := p.config.MapQueryTypeToNodeTypes(query.Type)
	typeMatches := slices.Contains(nodeTypes, node.Type())

	if !typeMatches {
		return false
	}

	// Provider-specific validation (e.g., Go struct vs interface in type_spec)
	if validator, ok := p.config.(interface {
		ValidateTypeSpec(*sitter.Node, string, string) bool
	}); ok {
		if !validator.ValidateTypeSpec(node, source, query.Type) {
			return false
		}
	}

	if query.Name != "" {
		name := p.config.ExtractNodeName(node, source)
		if !p.matchesPattern(name, query.Name) {
			return false
		}
	}

	return true
}

// sortTargetsDescending sorts nodes by start byte in descending order (reverse)
func sortTargetsDescending(targets []*sitter.Node) []*sitter.Node {
	sorted := make([]*sitter.Node, len(targets))
	copy(sorted, targets)
	slices.SortFunc(sorted, func(a, b *sitter.Node) int {
		return int(b.StartByte() - a.StartByte()) // Reverse order
	})
	return sorted
}

// doReplace performs replacement transformation
func (p *Provider) doReplace(source string, targets []*sitter.Node, replacement string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets to replace")
	}

	// Sort targets by position (reverse order to maintain positions)
	sortedTargets := sortTargetsDescending(targets)

	// Replace each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		startPos := int(target.StartByte())
		endPos := int(target.EndByte())

		// Safety bounds check
		if startPos > len(result) || endPos > len(result) || startPos < 0 || endPos < 0 {
			continue
		}

		before := result[:startPos]
		after := result[endPos:]
		result = before + replacement + after
	}

	return result, nil
}

// doDelete performs deletion transformation
func (p *Provider) doDelete(source string, targets []*sitter.Node) (string, error) {
	return p.doReplace(source, targets, "")
}

// doInsertBefore performs insertion before target
func (p *Provider) doInsertBefore(source string, targets []*sitter.Node, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for insertion")
	}

	// Sort targets reverse order
	sortedTargets := sortTargetsDescending(targets)

	// Insert before each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		startPos := int(target.StartByte())

		// Safety bounds check
		if startPos > len(result) || startPos < 0 {
			continue
		}

		before := result[:startPos]
		after := result[startPos:]

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
	sortedTargets := sortTargetsDescending(targets)

	// Insert after each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		endPos := int(target.EndByte())

		// Safety bounds check
		if endPos > len(result) || endPos < 0 {
			continue
		}

		before := result[:endPos]
		after := result[endPos:]

		// Preserve indentation
		indent := p.getIndentation(source, target)
		contentWithIndent := "\n" + indent + content

		result = before + contentWithIndent + after
	}

	return result, nil
}

// doAppendToTarget appends content to the end of target scope
func (p *Provider) doAppendToTarget(source string, targets []*sitter.Node, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for append")
	}

	// For append, we only use first target
	target := targets[0]

	if smart, ok := p.config.(SmartAppendConfig); ok {
		if modified, handled := smart.SmartAppend(source, target, content); handled {
			return modified, nil
		}
	}

	// This is language-agnostic - append after target
	insertPos := target.EndByte()

	before := source[:insertPos]
	after := source[insertPos:]

	// Add proper formatting
	insertion := "\n\n" + content

	return before + insertion + after, nil
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
func (p *Provider) calculateConfidence(
	op core.TransformOp,
	targets []*sitter.Node,
	source string,
) core.ConfidenceScore {
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
		// Check if deleting exported function
		if len(targets) > 0 {
			name := p.config.ExtractNodeName(targets[0], source)
			if p.config.IsExported(name) {
				score -= 0.3
				factors = append(factors, core.ConfidenceFactor{
					Name:   "delete_exported_api",
					Impact: -0.3,
					Reason: "Deleting exported API is dangerous",
				})
			}
		}
	case "replace":
		// Check if replacing exported function using language-specific logic
		if len(targets) > 0 {
			name := p.config.ExtractNodeName(targets[0], source)
			if p.config.IsExported(name) {
				score -= 0.2
				factors = append(factors, core.ConfidenceFactor{
					Name:   "exported_api",
					Impact: -0.2,
					Reason: "Modifying exported API",
				})
			}
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

	return core.ConfidenceScore{
		Score:   score,
		Level:   confidenceLevel(score),
		Factors: factors,
	}
}

func (p *Provider) adjustConfidence(conf *core.ConfidenceScore, op core.TransformOp, original, modified string, targets []*sitter.Node) {
	if conf == nil {
		return
	}

	if validation := p.Validate(modified); !validation.Valid {
		conf.Score -= 0.4
		conf.Factors = append(conf.Factors, core.ConfidenceFactor{
			Name:   "post_validation_failed",
			Impact: -0.4,
			Reason: "Transformed code failed syntax validation",
		})
	}

	if len(targets) > 3 {
		conf.Score -= 0.1
		conf.Factors = append(conf.Factors, core.ConfidenceFactor{
			Name:   "large_target_set",
			Impact: -0.1,
			Reason: fmt.Sprintf("Operation affected %d nodes", len(targets)),
		})
	}

	if strings.Count(op.Target.Name, "*") > 1 {
		conf.Score -= 0.1
		conf.Factors = append(conf.Factors, core.ConfidenceFactor{
			Name:   "broad_wildcard",
			Impact: -0.1,
			Reason: "Wildcard pattern is very broad",
		})
	}

	if len(modified) > 0 && len(original) > 0 {
		delta := math.Abs(float64(len(modified)-len(original))) / float64(len(original))
		if delta > 0.3 {
			conf.Score -= 0.1
			conf.Factors = append(conf.Factors, core.ConfidenceFactor{
				Name:   "large_size_delta",
				Impact: -0.1,
				Reason: "Transformation changed file size significantly",
			})
		}
	}

	clampConfidence(conf)
}

func clampConfidence(conf *core.ConfidenceScore) {
	if conf.Score < 0 {
		conf.Score = 0
	} else if conf.Score > 1 {
		conf.Score = 1
	}
	conf.Level = confidenceLevel(conf.Score)
}

func confidenceLevel(score float64) string {
	switch {
	case score < 0.5:
		return "low"
	case score < 0.8:
		return "medium"
	default:
		return "high"
	}
}

// generateDiff creates a unified diff
func (p *Provider) generateDiff(original, modified string) string {
	if original == modified {
		return ""
	}

	diff := difflib.UnifiedDiff{
		A:        strings.Split(original, "\n"),
		B:        strings.Split(modified, "\n"),
		FromFile: "original",
		ToFile:   "modified",
		Context:  3,
	}

	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Sprintf("--- original\n+++ modified\n@@ changes @@\n%d bytes -> %d bytes",
			len(original), len(modified))
	}

	return text
}

// findErrors looks for syntax errors in AST
func (p *Provider) findErrors(node *sitter.Node, source string, errors *[]string) {
	if node.Type() == "ERROR" {
		*errors = append(*errors, fmt.Sprintf(
			"Syntax error at line %d, column %d",
			node.StartPoint().Row+1,
			node.StartPoint().Column+1,
		))
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.findErrors(node.Child(i), source, errors)
	}
}
