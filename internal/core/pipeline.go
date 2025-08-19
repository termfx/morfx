package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
)

// Edit represents a single code modification
type Edit struct {
	// StartByte is the starting byte offset
	StartByte int
	// EndByte is the ending byte offset
	EndByte int
	// NewText is the replacement text
	NewText string
	// Operation is the type of edit
	Operation Operation
	// Priority for stable ordering (lower = higher priority)
	Priority int
}

// Pipeline implements the deterministic Apply Pipeline
type Pipeline struct {
	provider      PipelineProvider
	fuzzyResolver *FuzzyResolver
}

// NewPipeline creates a new pipeline instance
func NewPipeline(provider PipelineProvider) *Pipeline {
	return &Pipeline{
		provider:      provider,
		fuzzyResolver: NewFuzzyResolver(),
	}
}

// Apply executes the 8-step deterministic pipeline
func (p *Pipeline) Apply(input Input) (*PipelineResult, error) {
	start := time.Now()
	stats := Stats{
		BytesProcessed: int64(len(input.CodeIn)),
		LinesProcessed: strings.Count(input.CodeIn, "\n") + 1,
	}

	result := &PipelineResult{
		Status:      StatusSuccess,
		ResolvedOp:  input.Op,
		CodeOut:     input.CodeIn,
		Stats:       stats,
		Diagnostics: []Diagnostic{},
		Engine: Engine{
			Version:   "1.0.0", // TODO: Get from build info
			Provider:  p.provider.Lang(),
			Timestamp: time.Now(),
		},
		Fuzzy: FuzzyMatch{Used: false},
	}

	// Step 1: Parse
	var tree *sitter.Tree
	languageInterface := p.provider.GetSitterLanguage()
	if languageInterface != nil {
		// Cast the interface{} back to *sitter.Language
		language, ok := languageInterface.(*sitter.Language)
		if !ok {
			return p.errorResult(result, fmt.Errorf("invalid sitter language type"))
		}

		parser := sitter.NewParser()
		parser.SetLanguage(language)
		var err error
		tree, err = parser.ParseCtx(context.TODO(), nil, []byte(input.CodeIn))
		if err != nil {
			return p.errorResult(result, fmt.Errorf("parse error: %w", err))
		}
		defer tree.Close()
	} else {
		// For testing or languages without tree-sitter support
		// Skip parsing and continue with mock operations
	}

	// Step 2: Resolve Operation
	resolvedOp, err := p.resolveOperation(input.Op, input.Query)
	if err != nil {
		return p.errorResult(result, fmt.Errorf("operation resolution error: %w", err))
	}
	result.ResolvedOp = resolvedOp

	// Step 3: Select Anchors
	anchors, err := p.selectAnchors(tree, input.Query, []byte(input.CodeIn))
	if err != nil {
		// Try fuzzy resolution if enabled
		if input.Options != nil && input.Options.Fuzz {
			fuzzyAnchors, fuzzyMatch, fuzzyErr := p.fuzzyResolver.Resolve(
				tree, input.Query, []byte(input.CodeIn), p.provider, input.Options.MaxFuzzDistance)
			if fuzzyErr == nil {
				anchors = fuzzyAnchors
				result.Fuzzy = *fuzzyMatch
			} else {
				return p.errorResult(result, fmt.Errorf("anchor selection error: %w", err))
			}
		} else {
			return p.errorResult(result, fmt.Errorf("anchor selection error: %w", err))
		}
	}
	result.Stats.MatchesFound = len(anchors)

	// Step 4: Plan Edits
	edits, err := p.planEdits(anchors, resolvedOp, input.Repl, []byte(input.CodeIn))
	if err != nil {
		return p.errorResult(result, fmt.Errorf("edit planning error: %w", err))
	}

	// Check for overlaps
	overlaps := p.detectOverlaps(edits)
	result.Stats.OverlapsDetected = len(overlaps)
	if len(overlaps) > 0 {
		return p.errorResult(result, fmt.Errorf("edit conflicts detected: %d overlaps", len(overlaps)))
	}

	// Step 5: Apply Edits
	modifiedCode, err := p.applyEdits([]byte(input.CodeIn), edits)
	if err != nil {
		return p.errorResult(result, fmt.Errorf("edit application error: %w", err))
	}
	result.Stats.EditsApplied = len(edits)

	// Step 6: Post-process
	if input.Options == nil || !input.Options.SkipValidation {
		if validateErr := p.validateResult(modifiedCode, input.Repl); validateErr != nil {
			result.Status = StatusPartial
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("validation warning: %v", err),
			})
		}
	}

	processedCode, err := p.postProcess(modifiedCode, input.Options)
	if err != nil {
		result.Status = StatusPartial
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Message:  fmt.Sprintf("post-processing warning: %v", err),
		})
		processedCode = modifiedCode // Use unprocessed code
	}

	// Step 7: Generate Diff
	diff, err := p.generateDiff([]byte(input.CodeIn), processedCode)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Message:  fmt.Sprintf("diff generation warning: %v", err),
		})
	} else {
		result.Diff = diff
	}

	// Step 8: Finalize Result
	result.CodeOut = string(processedCode)
	result.Hash = fmt.Sprintf("%x", sha256.Sum256(processedCode))
	result.Stats.Duration = time.Since(start)

	return result, nil
}

// resolveOperation determines the actual operation to perform
func (p *Pipeline) resolveOperation(op Operation, query string) (Operation, error) {
	// For now, return the operation as-is
	// Future: Add logic for operation inference based on query
	_ = query // TODO: Use query for operation inference
	switch op {
	case InsertBefore, InsertAfter, Replace, Delete, AppendToBody:
		return op, nil
	default:
		return "", fmt.Errorf("unsupported operation: %s", op)
	}
}

// selectAnchors finds target locations using the query
func (p *Pipeline) selectAnchors(tree *sitter.Tree, query string, source []byte) ([]*sitter.Node, error) {
	// Convert DSL query to Tree-sitter query
	dslQuery := Query{} // TODO: Parse DSL query
	tsQuery, err := p.provider.TranslateQuery(&dslQuery)
	if err != nil {
		return nil, fmt.Errorf("query translation failed: %w", err)
	}

	// Execute Tree-sitter query
	languageInterface := p.provider.GetSitterLanguage()
	if languageInterface == nil {
		// For testing or languages without tree-sitter support
		// Return empty anchors
		return []*sitter.Node{}, nil
	}

	// Cast the interface{} back to *sitter.Language
	language, ok := languageInterface.(*sitter.Language)
	if !ok {
		return nil, fmt.Errorf("invalid sitter language type")
	}

	q, err := sitter.NewQuery([]byte(tsQuery), language)
	if err != nil {
		return nil, fmt.Errorf("invalid tree-sitter query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	// Execute query and collect matches
	var anchors []*sitter.Node
	qc.Exec(q, tree.RootNode())
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			anchors = append(anchors, capture.Node)
		}
	}

	if len(anchors) == 0 {
		return nil, fmt.Errorf("no matches found for query: %s", query)
	}

	return anchors, nil
}

// planEdits creates edit operations for the anchors
func (p *Pipeline) planEdits(anchors []*sitter.Node, op Operation, repl string, source []byte) ([]Edit, error) {
	var edits []Edit

	for i, anchor := range anchors {
		var edit Edit
		switch op {
		case InsertBefore:
			edit = Edit{
				StartByte: int(anchor.StartByte()),
				EndByte:   int(anchor.StartByte()),
				NewText:   repl,
				Operation: op,
				Priority:  i,
			}
		case InsertAfter:
			edit = Edit{
				StartByte: int(anchor.EndByte()),
				EndByte:   int(anchor.EndByte()),
				NewText:   repl,
				Operation: op,
				Priority:  i,
			}
		case Replace:
			edit = Edit{
				StartByte: int(anchor.StartByte()),
				EndByte:   int(anchor.EndByte()),
				NewText:   repl,
				Operation: op,
				Priority:  i,
			}
		case Delete:
			edit = Edit{
				StartByte: int(anchor.StartByte()),
				EndByte:   int(anchor.EndByte()),
				NewText:   "",
				Operation: op,
				Priority:  i,
			}
		case AppendToBody:
			appendPoint, err := p.provider.AppendPoint(anchor, source)
			if err != nil {
				return nil, fmt.Errorf("failed to find append point: %w", err)
			}
			edit = Edit{
				StartByte: appendPoint,
				EndByte:   appendPoint,
				NewText:   repl,
				Operation: op,
				Priority:  i,
			}
		default:
			return nil, fmt.Errorf("unsupported operation: %s", op)
		}

		edits = append(edits, edit)
	}

	return edits, nil
}

// detectOverlaps finds conflicting edits
func (p *Pipeline) detectOverlaps(edits []Edit) []string {
	// Sort edits by start byte for efficient overlap detection
	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		return sortedEdits[i].StartByte < sortedEdits[j].StartByte
	})

	var overlaps []string
	for i := 0; i < len(sortedEdits)-1; i++ {
		current := sortedEdits[i]
		next := sortedEdits[i+1]

		// Check if current edit overlaps with next
		if current.EndByte > next.StartByte {
			overlaps = append(overlaps, fmt.Sprintf(
				"overlap between edits at bytes %d-%d and %d-%d",
				current.StartByte, current.EndByte,
				next.StartByte, next.EndByte))
		}
	}

	return overlaps
}

// applyEdits applies all edits to the source code
func (p *Pipeline) applyEdits(source []byte, edits []Edit) ([]byte, error) {
	// Sort edits by start byte in reverse order for safe application
	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].StartByte == sortedEdits[j].StartByte {
			return sortedEdits[i].Priority < sortedEdits[j].Priority
		}
		return sortedEdits[i].StartByte > sortedEdits[j].StartByte
	})

	result := make([]byte, len(source))
	copy(result, source)

	for _, edit := range sortedEdits {
		if edit.StartByte < 0 || edit.EndByte > len(result) {
			return nil, fmt.Errorf("edit out of bounds: %d-%d", edit.StartByte, edit.EndByte)
		}

		// Apply the edit
		newResult := make([]byte, 0, len(result)+len(edit.NewText))
		newResult = append(newResult, result[:edit.StartByte]...)
		newResult = append(newResult, []byte(edit.NewText)...)
		newResult = append(newResult, result[edit.EndByte:]...)
		result = newResult
	}

	return result, nil
}

// validateResult performs basic validation on the result
func (p *Pipeline) validateResult(code []byte, snippet string) error {
	return p.provider.ValidateSnippet(snippet, nil, code)
}

// postProcess applies formatting and import organization
func (p *Pipeline) postProcess(code []byte, options *InputOptions) ([]byte, error) {
	result := code
	var err error

	// Step 1: Organize imports
	if options == nil || !options.SkipImports {
		result, err = p.provider.OrganizeImports(result)
		if err != nil {
			return code, fmt.Errorf("import organization failed: %w", err)
		}
	}

	// Step 2: Format code
	if options == nil || !options.SkipFormat {
		result, err = p.provider.Format(result)
		if err != nil {
			return code, fmt.Errorf("formatting failed: %w", err)
		}
	}

	// Step 3: Quick check
	diagnostics := p.provider.QuickCheck(result)
	if len(diagnostics) > 0 {
		// Log diagnostics but don't fail
		for _, diag := range diagnostics {
			if diag.Severity == "error" {
				return code, fmt.Errorf("quick check failed: %s", diag.Message)
			}
		}
	}

	return result, nil
}

// generateDiff creates a unified diff
func (p *Pipeline) generateDiff(original, modified []byte) (string, error) {
	// Simple diff implementation - in production, use a proper diff library
	if string(original) == string(modified) {
		return "", nil
	}

	// Basic unified diff format
	diff := fmt.Sprintf("--- original\n+++ modified\n@@ -1,%d +1,%d @@\n",
		strings.Count(string(original), "\n")+1,
		strings.Count(string(modified), "\n")+1)

	origLines := strings.Split(string(original), "\n")
	modLines := strings.Split(string(modified), "\n")

	// Simple line-by-line comparison
	maxLines := max(len(modLines), len(origLines))

	for i := range maxLines {
		origLine := ""
		modLine := ""

		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(modLines) {
			modLine = modLines[i]
		}

		if origLine != modLine {
			if origLine != "" {
				diff += "-" + origLine + "\n"
			}
			if modLine != "" {
				diff += "+" + modLine + "\n"
			}
		} else {
			diff += " " + origLine + "\n"
		}
	}

	return diff, nil
}

// errorResult creates an error result
func (p *Pipeline) errorResult(result *PipelineResult, err error) (*PipelineResult, error) {
	result.Status = StatusError
	result.Diagnostics = append(result.Diagnostics, Diagnostic{
		Severity: "error",
		Message:  err.Error(),
	})
	return result, err
}
