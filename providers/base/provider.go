package base

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pmezard/go-difflib/difflib"
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/providers"
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

// QueryTypeNormalizer lets providers own DSL/query aliases for their language.
type QueryTypeNormalizer interface {
	NormalizeQueryType(queryType string) string
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
			return newParserAdapter(lang)
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
func (p *Provider) borrowParser() *parserAdapter {
	parser, ok := p.pool.Get().(*parserAdapter)
	if !ok || parser == nil {
		parser = newParserAdapter(p.config.GetLanguage())
	}

	p.stats.borrowCount.Add(1)
	p.stats.active.Add(1)

	return parser
}

// releaseParser returns a parser instance to the pool.
func (p *Provider) releaseParser(parser *parserAdapter) {
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
	query = p.normalizeQuery(query)

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

	targets := p.findTargets(tree.RootNode(), source, query)
	matches := make([]core.Match, 0, len(targets))
	for _, target := range targets {
		matches = append(matches, p.targetToMatch(source, target))
	}

	return core.QueryResult{
		Matches: matches,
		Total:   len(matches),
	}
}

// Transform applies a transformation operation
func (p *Provider) Transform(source string, op core.TransformOp) core.TransformResult {
	op.Target = p.normalizeQuery(op.Target)

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

	// For append without a target, use root node directly
	if op.Method == "append" && op.Target.Type == "" && op.Target.Name == "" {
		root := tree.RootNode()
		confidence := core.ConfidenceScore{
			Score: 1.0,
			Level: "high",
			Factors: []core.ConfidenceFactor{{
				Name:   "append_to_root",
				Impact: 0.0,
				Reason: "Appending to end of file (no target specified)",
			}},
		}

		modified, err := p.doAppendToTarget(source, []Target{NewTarget(root, "", "")}, op.Content)
		if err != nil {
			return core.TransformResult{Error: err}
		}

		diff := p.generateDiff(source, modified)
		return core.TransformResult{
			Modified:   modified,
			Diff:       diff,
			Confidence: confidence,
			MatchCount: 1,
		}
	}

	// Find targets
	matches := p.findTargets(tree.RootNode(), source, op.Target)
	if len(matches) == 0 {
		return core.TransformResult{
			Error: core.ErrNoMatchesFound,
		}
	}

	// Calculate confidence
	confidence := p.calculateConfidence(op, matches, source)
	var (
		modified string
		err      error
	)

	switch op.Method {
	case "replace":
		modified, err = p.doReplace(source, matches, op.Replacement)
	case "delete":
		modified, err = p.doDelete(source, matches)
	case "insert_before":
		modified, err = p.doInsertBefore(source, matches, op.Content)
	case "insert_after":
		modified, err = p.doInsertAfter(source, matches, op.Content)
	case "append":
		modified, err = p.doAppendToTarget(source, matches, op.Content)
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

	p.adjustConfidence(&confidence, op, source, modified, matches)

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

	tree := parser.Parse([]byte(source))
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
	matched, _ := p.matchPatternCaptures(name, pattern)
	return matched
}

func (p *Provider) matchPatternCaptures(name, pattern string) (bool, map[string]string) {
	if pattern == "" || pattern == "*" {
		return true, nil
	}

	if strings.Contains(pattern, "$") {
		return matchCapturePattern(name, pattern)
	}

	matched, err := path.Match(pattern, name)
	if err != nil {
		return false, nil
	}

	return matched, nil
}

func matchCapturePattern(name, pattern string) (bool, map[string]string) {
	var (
		builder strings.Builder
		names   []string
	)
	builder.WriteString("^")
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			builder.WriteString(".*")
			i++
		case '$':
			start := i + 1
			end := start
			for end < len(pattern) {
				ch := pattern[end]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
					end++
					continue
				}
				break
			}
			if end == start {
				builder.WriteString(regexp.QuoteMeta("$"))
				i++
				continue
			}
			names = append(names, pattern[start:end])
			if end < len(pattern) {
				builder.WriteString("(.+?)")
			} else {
				builder.WriteString("(.+)")
			}
			i = end
		default:
			builder.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	builder.WriteString("$")

	re, err := regexp.Compile(builder.String())
	if err != nil {
		return false, nil
	}
	matches := re.FindStringSubmatch(name)
	if matches == nil {
		return false, nil
	}
	captures := make(map[string]string, len(names))
	for i, captureName := range names {
		captures[captureName] = matches[i+1]
	}
	return true, captures
}

// findTargets finds all matches for the query with proper expansion.
func (p *Provider) findTargets(root *sitter.Node, source string, query core.AgentQuery) []Target {
	switch strings.ToUpper(strings.TrimSpace(query.Operator)) {
	case "AND":
		return p.findIntersectionTargets(root, source, query.Operands)
	case "OR":
		return p.findUnionTargets(root, source, query.Operands)
	case "NOT":
		return p.findNegatedTargets(root, source, query.Operands)
	default:
		return p.findSimpleTargets(root, source, query)
	}
}

func (p *Provider) normalizeQuery(query core.AgentQuery) core.AgentQuery {
	if normalizer, ok := p.config.(QueryTypeNormalizer); ok {
		query.Type = normalizer.NormalizeQueryType(query.Type)
	}
	if query.Contains != nil {
		child := p.normalizeQuery(*query.Contains)
		query.Contains = &child
	}
	if len(query.Operands) > 0 {
		operands := make([]core.AgentQuery, len(query.Operands))
		for i, operand := range query.Operands {
			operands[i] = p.normalizeQuery(operand)
		}
		query.Operands = operands
	}
	return query
}

func (p *Provider) findSimpleTargets(root *sitter.Node, source string, query core.AgentQuery) []Target {
	var matches []Target
	seen := make(map[string]struct{})

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if targets := p.candidateTargets(node, source, query); len(targets) > 0 {
			for _, target := range targets {
				key := semanticTargetKey(query.Type, target)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				matches = append(matches, target)
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}

	walk(root)
	return matches
}

func (p *Provider) findTargetsBelow(root *sitter.Node, source string, query core.AgentQuery) []Target {
	var matches []Target
	for i := 0; i < int(root.ChildCount()); i++ {
		matches = append(matches, p.findTargets(root.Child(i), source, query)...)
	}
	return matches
}

func (p *Provider) findUnionTargets(root *sitter.Node, source string, operands []core.AgentQuery) []Target {
	var matches []Target
	seen := make(map[string]struct{})
	for _, operand := range operands {
		for _, target := range p.findTargets(root, source, operand) {
			key := semanticTargetKey(target.Type, target)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			matches = append(matches, target)
		}
	}
	return matches
}

func (p *Provider) findIntersectionTargets(root *sitter.Node, source string, operands []core.AgentQuery) []Target {
	if len(operands) == 0 {
		return nil
	}

	current := p.findTargets(root, source, operands[0])
	for _, operand := range operands[1:] {
		next := p.findTargets(root, source, operand)
		nextKeys := make(map[string]struct{}, len(next))
		for _, target := range next {
			nextKeys[semanticTargetKey(target.Type, target)] = struct{}{}
		}

		filtered := current[:0]
		for _, target := range current {
			if _, exists := nextKeys[semanticTargetKey(target.Type, target)]; exists {
				filtered = append(filtered, target)
			}
		}
		current = filtered
	}
	return current
}

func (p *Provider) findNegatedTargets(root *sitter.Node, source string, operands []core.AgentQuery) []Target {
	if len(operands) != 1 {
		return nil
	}

	operand := operands[0]
	if strings.TrimSpace(operand.Type) == "" {
		return nil
	}

	allQuery := operand
	allQuery.Name = "*"
	allQuery.Contains = nil
	allQuery.Operator = ""
	allQuery.Operands = nil
	allQuery.Attributes = nil

	excluded := p.findTargets(root, source, operand)
	excludedKeys := make(map[string]struct{}, len(excluded))
	for _, target := range excluded {
		excludedKeys[semanticTargetKey(target.Type, target)] = struct{}{}
	}

	var matches []Target
	for _, target := range p.findTargets(root, source, allQuery) {
		if _, exists := excludedKeys[semanticTargetKey(target.Type, target)]; !exists {
			matches = append(matches, target)
		}
	}
	return matches
}

func semanticTargetKey(queryType string, target Target) string {
	return fmt.Sprintf("%s:%d:%d:%s", queryType, target.StartByte, target.EndByte, target.Name)
}

// expandMatches converts a node into one or more matches
func (p *Provider) expandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	if expander, ok := p.config.(interface {
		ExpandMatches(*sitter.Node, string, core.AgentQuery) []Target
	}); ok {
		return expander.ExpandMatches(node, source, query)
	}

	name := p.config.ExtractNodeName(node, source)
	return []Target{NewTarget(node, query.Type, name)}
}

func (p *Provider) candidateTargets(node *sitter.Node, source string, query core.AgentQuery) []Target {
	if !p.nodeMatches(node, source, query.Type) {
		return nil
	}

	targets := p.expandMatches(node, source, query)
	if len(targets) == 0 {
		return nil
	}

	filtered := make([]Target, 0, len(targets))
	for _, target := range targets {
		name := target.Name
		if name == "" {
			name = "anonymous"
			target.Name = name
		}
		matched, captures := p.matchPatternCaptures(name, query.Name)
		if matched &&
			p.matchesAttributes(target, source, query.Attributes) &&
			p.matchesContains(target, source, query.Contains, query.ContainsDirect) {
			target.Captures = captures
			filtered = append(filtered, target)
		}
	}

	return filtered
}

func (p *Provider) matchesAttributes(target Target, source string, attributes map[string]string) bool {
	if len(attributes) == 0 {
		return true
	}

	providerAttributes := make(map[string]string, len(attributes))
	for key, value := range attributes {
		switch {
		case key == "text":
			if !matchTextAttribute(nodeContent(target.Node, source), value) {
				return false
			}
		case key == "source":
			if !matchTextAttribute(target.Name, value) && !matchTextAttribute(nodeContent(target.Node, source), value) {
				return false
			}
		case key == "arg" || strings.HasPrefix(key, "arg"):
			if !matchArgumentAttribute(target.Node, source, key, value) {
				return false
			}
		case key == "before":
			if !p.matchesSiblingPredicate(target.Node, source, value, true) {
				return false
			}
		case key == "after":
			if !p.matchesSiblingPredicate(target.Node, source, value, false) {
				return false
			}
		default:
			providerAttributes[key] = value
		}
	}
	if len(providerAttributes) == 0 {
		return true
	}

	if validator, ok := p.config.(interface {
		ValidateQueryAttributes(Target, string, map[string]string) bool
	}); ok {
		return validator.ValidateQueryAttributes(target, source, providerAttributes)
	}

	return true
}

func nodeContent(node *sitter.Node, source string) string {
	if node == nil {
		return ""
	}
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	return source[start:end]
}

func matchTextAttribute(actual, pattern string) bool {
	actual = stripAttributeQuotes(strings.TrimSpace(actual))
	pattern = stripAttributeQuotes(strings.TrimSpace(pattern))
	if pattern == "" {
		return actual == ""
	}
	if strings.ContainsAny(pattern, "*?[") {
		matched, err := path.Match(pattern, actual)
		return err == nil && matched
	}
	return actual == pattern || strings.Contains(actual, pattern)
}

func stripAttributeQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') || (first == '`' && last == '`') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func matchArgumentAttribute(node *sitter.Node, source, key, pattern string) bool {
	args := argumentTexts(node, source)
	if len(args) == 0 {
		return false
	}
	if key == "arg" {
		for _, arg := range args {
			if matchTextAttribute(arg, pattern) {
				return true
			}
		}
		return false
	}

	indexText := strings.TrimPrefix(key, "arg")
	var index int
	for _, ch := range indexText {
		if ch < '0' || ch > '9' {
			return false
		}
		index = index*10 + int(ch-'0')
	}
	if index < 0 || index >= len(args) {
		return false
	}
	return matchTextAttribute(args[index], pattern)
}

func argumentTexts(node *sitter.Node, source string) []string {
	if node == nil {
		return nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		childType := child.Type()
		if childType == "arguments" || childType == "argument_list" || strings.Contains(childType, "arguments") {
			args := make([]string, 0, child.NamedChildCount())
			for j := 0; j < int(child.NamedChildCount()); j++ {
				arg := child.NamedChild(j)
				text := stripAttributeQuotes(nodeContent(arg, source))
				if text != "" {
					args = append(args, text)
				}
			}
			return args
		}
	}
	return nil
}

func (p *Provider) matchesSiblingPredicate(node *sitter.Node, source, dsl string, before bool) bool {
	if node == nil || node.Parent() == nil {
		return false
	}
	query, err := core.ParseDSL(dsl)
	if err != nil {
		return false
	}

	parent := node.Parent()
	for i := 0; i < int(parent.ChildCount()); i++ {
		sibling := parent.Child(i)
		if sibling == nil || sibling == node {
			continue
		}
		targets := p.findTargets(sibling, source, query)
		for _, target := range targets {
			if before && target.StartByte > node.EndByte() {
				return true
			}
			if !before && target.EndByte < node.StartByte() {
				return true
			}
		}
	}
	return false
}

func (p *Provider) matchesContains(target Target, source string, child *core.AgentQuery, direct bool) bool {
	if child == nil {
		return true
	}
	if target.Node == nil {
		return false
	}
	if direct {
		return p.matchesDirectChild(target.Node, source, *child)
	}
	return len(p.findTargetsBelow(target.Node, source, *child)) > 0
}

func (p *Provider) matchesDirectChild(parent *sitter.Node, source string, query core.AgentQuery) bool {
	for _, child := range directSemanticChildren(parent) {
		if len(p.candidateTargets(child, source, query)) > 0 {
			return true
		}
	}
	return false
}

func directSemanticChildren(parent *sitter.Node) []*sitter.Node {
	if parent == nil {
		return nil
	}

	var children []*sitter.Node
	for i := 0; i < int(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child == nil {
			continue
		}
		if isTransparentContainer(child.Type()) {
			for j := 0; j < int(child.ChildCount()); j++ {
				if grandchild := child.Child(j); grandchild != nil {
					children = append(children, grandchild)
				}
			}
			continue
		}
		children = append(children, child)
	}
	return children
}

func isTransparentContainer(nodeType string) bool {
	switch nodeType {
	case "class_body", "declaration_list", "statement_block", "block", "compound_statement", "body":
		return true
	default:
		return false
	}
}

// nodeMatches checks if a node matches the query with provider-specific validation
func (p *Provider) nodeMatches(node *sitter.Node, source string, queryType string) bool {
	nodeTypes := p.config.MapQueryTypeToNodeTypes(queryType)
	typeMatches := slices.Contains(nodeTypes, node.Type())

	if !typeMatches {
		return false
	}

	return p.passesProviderValidation(node, source, queryType)
}

func (p *Provider) passesProviderValidation(node *sitter.Node, source string, queryType string) bool {
	if validator, ok := p.config.(interface {
		ValidateQueryNode(*sitter.Node, string, string) bool
	}); ok {
		if !validator.ValidateQueryNode(node, source, queryType) {
			return false
		}
	}

	if validator, ok := p.config.(interface {
		ValidateTypeSpec(*sitter.Node, string, string) bool
	}); ok {
		if !validator.ValidateTypeSpec(node, source, queryType) {
			return false
		}
	}

	if validator, ok := p.config.(interface {
		ValidateAssignment(*sitter.Node, string, string) bool
	}); ok {
		if !validator.ValidateAssignment(node, source, queryType) {
			return false
		}
	}

	return true
}

func (p *Provider) targetToMatch(source string, target Target) core.Match {
	location := core.Location{
		Line:   int(target.Line) + 1,
		Column: int(target.Column) + 1,
	}

	if target.Node != nil {
		location.EndLine = int(target.Node.EndPoint().Row) + 1
		location.EndColumn = int(target.Node.EndPoint().Column) + 1
	}

	content := ""
	start := int(target.StartByte)
	end := int(target.EndByte)
	if start >= 0 && end >= 0 && start <= end && end <= len(source) {
		content = source[start:end]
	}

	return core.Match{
		Type:     target.Type,
		Name:     target.Name,
		Location: location,
		Content:  content,
		Captures: target.Captures,
	}
}

// sortTargetsDescending sorts nodes by start byte in descending order (reverse)
func sortTargetsDescending(targets []Target) []Target {
	sorted := make([]Target, len(targets))
	copy(sorted, targets)
	sort.SliceStable(sorted, func(i, j int) bool {
		a := sorted[i]
		b := sorted[j]

		if a.StartByte != b.StartByte {
			return a.StartByte > b.StartByte
		}
		if a.EndByte != b.EndByte {
			return a.EndByte > b.EndByte
		}

		return false
	})
	return sorted
}

// doReplace performs replacement transformation
func (p *Provider) doReplace(source string, targets []Target, replacement string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets to replace")
	}

	// Sort targets by position (reverse order to maintain positions)
	sortedTargets := sortTargetsDescending(targets)

	// Replace each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		startPos := int(target.StartByte)
		endPos := int(target.EndByte)

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
func (p *Provider) doDelete(source string, targets []Target) (string, error) {
	return p.doReplace(source, targets, "")
}

// doInsertBefore performs insertion before target
func (p *Provider) doInsertBefore(source string, targets []Target, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for insertion")
	}

	// Sort targets reverse order
	sortedTargets := sortTargetsDescending(targets)

	// Insert before each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		startPos := int(target.StartByte)

		// Safety bounds check
		if startPos > len(result) || startPos < 0 {
			continue
		}

		before := result[:startPos]
		after := result[startPos:]

		// Preserve indentation
		indent := p.getIndentation(source, target.Node)
		contentWithIndent := indent + content + "\n"

		result = before + contentWithIndent + after
	}

	return result, nil
}

// doInsertAfter performs insertion after target
func (p *Provider) doInsertAfter(source string, targets []Target, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for insertion")
	}

	// Sort targets reverse order
	sortedTargets := sortTargetsDescending(targets)

	// Insert after each target (from end to start to preserve positions)
	result := source
	for _, target := range sortedTargets {
		endPos := int(target.EndByte)

		// Safety bounds check
		if endPos > len(result) || endPos < 0 {
			continue
		}

		before := result[:endPos]
		after := result[endPos:]

		// Preserve indentation
		indent := p.getIndentation(source, target.Node)
		contentWithIndent := "\n" + indent + content

		result = before + contentWithIndent + after
	}

	return result, nil
}

// doAppendToTarget appends content to the end of target scope
func (p *Provider) doAppendToTarget(source string, targets []Target, content string) (string, error) {
	if len(targets) == 0 {
		return source, fmt.Errorf("no targets for append")
	}

	// For append, we only use first target
	target := targets[0]

	if smart, ok := p.config.(SmartAppendConfig); ok {
		if modified, handled := smart.SmartAppend(source, target.Node, content); handled {
			return modified, nil
		}
	}

	// This is language-agnostic - append after target
	insertPos := int(target.EndByte)

	before := source[:insertPos]
	after := source[insertPos:]

	// Add proper formatting
	insertion := "\n\n" + content

	return before + insertion + after, nil
}

// getIndentation extracts indentation for a node
func (p *Provider) getIndentation(source string, node *sitter.Node) string {
	if node == nil {
		return ""
	}

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
	targets []Target,
	_ string,
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
		score -= 0.1
		factors = append(factors, core.ConfidenceFactor{
			Name:   "delete_operation",
			Impact: -0.1,
			Reason: "Delete operations are destructive",
		})
		// Check if deleting exported function
		if len(targets) > 0 {
			if p.config.IsExported(targets[0].Name) {
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
			if p.config.IsExported(targets[0].Name) {
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

func (p *Provider) adjustConfidence(conf *core.ConfidenceScore, op core.TransformOp, original, modified string, targets []Target) {
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
