package core

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
)

// Manipulator applies a single modification rule to content.
type Manipulator struct {
	Config model.ModificationConfig
}

type entry struct {
	mt  matcher.Matcher
	err error
}

var (
	mu   sync.RWMutex
	data = make(map[string]*entry) // key = ruleID + fingerprint(pattern/flags)
)

// NewManipulator creates a new manipulator for a given rule.
func NewManipulator(cfg model.ModificationConfig) *Manipulator {
	return &Manipulator{Config: cfg}
}

// Apply executes the rule on the content and delegates to the appropriate helpers.
// – If NormalizeWhitespace is false, works on original text (traditional regex).
// – If NormalizeWhitespace is true, collapses whitespace in content and pattern,
//
//	matches on the normalized version, then remaps to original.
//
// – If LiteralPattern is true, the pattern (already normalized or not) is escaped
//
//	with regexp.QuoteMeta to treat it as literal text.
func (m *Manipulator) Apply(content string) (string, []model.Change, error) {
	cfg := m.Config
	pattern := cfg.Pattern

	// ---------------------------------------------------------
	// 1. Path WITHOUT whitespace normalization
	// ---------------------------------------------------------
	if !cfg.NormalizeWhitespace {
		return m.applyNoNormalize(content, pattern)
	}

	// ---------------------------------------------------------
	// 2. Path WITH whitespace normalization
	// ---------------------------------------------------------
	//   a) Normalize original content
	normContent, n2o, o2n := util.NormalizeWhitespace(content)
	//   b) Normalize the pattern to collapse its whitespace
	normPattern, _, _ := util.NormalizeWhitespace(pattern)
	if cfg.LiteralPattern {
		normPattern = regexp.QuoteMeta(normPattern)
	}

	//   c) Flags (?m)(?s)
	if cfg.Multiline {
		normPattern = "(?m)" + normPattern
	}
	if cfg.DotAll {
		normPattern = "(?s)" + normPattern
	}

	//   d) Compile
	re, err := regexp.Compile(normPattern)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", model.ErrInvalidRegex, err)
	}

	//   e) Search in normalized content
	matchesNorm := re.FindAllStringSubmatchIndex(normContent, -1)

	//   f) Remap spans to original
	matchesOrig := util.RemapAllMatches(matchesNorm, n2o, o2n)

	//   g) Parse occurrences
	occ, err := parseOccurrences(cfg.Occurrences)
	if err != nil {
		return "", nil, err
	}

	//   h) Context filtering and final application on ORIGINAL
	if cfg.Context != nil {
		return m.applyWithContextOnOriginal(content, re, matchesOrig, occ)
	}
	return m.applyMatchesOnOriginal(content, re, matchesOrig, occ)
}

func (m *Manipulator) applyNoNormalize(content, pattern string) (string, []model.Change, error) {
	if m.Config.LiteralPattern {
		pattern = regexp.QuoteMeta(pattern)
	}
	if m.Config.Multiline {
		pattern = "(?m)" + pattern
	}
	if m.Config.DotAll {
		pattern = "(?s)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", nil, Wrap(ErrInvalidRegex,
			"failed to compile regex pattern", err)
	}

	occ, err := parseOccurrences(m.Config.Occurrences)
	if err != nil {
		return "", nil, err
	}

	if m.Config.Context != nil {
		return m.applyWithContext(content, re, occ)
	}
	return m.applySimple(content, re, occ)
}

// Igual que applySimple pero usando los índices ya remapeados.
func (m *Manipulator) applyMatchesOnOriginal(
	content string,
	re *regexp.Regexp,
	matches [][]int,
	occ model.OccurrenceSpec,
) (string, []model.Change, error) {
	// Limpia los -1 y aplica occ
	filtered := make([][]int, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 || m[0] < 0 || m[1] < 0 {
			continue
		}
		filtered = append(filtered, m)
	}

	if len(filtered) == 0 {
		return content, nil, nil
	}

	if occ.Max != -1 && len(filtered) > occ.Max {
		filtered = filtered[:occ.Max]
	}

	return m.applyMatches(content, re, filtered)
}

// Igual que filterMatchesByContext, pero recibe matches ya remapeados a original.
func (m *Manipulator) applyWithContextOnOriginal(
	content string,
	re *regexp.Regexp,
	matches [][]int,
	occ model.OccurrenceSpec,
) (string, []model.Change, error) {
	allMatches := matches
	if len(allMatches) == 0 {
		return content, nil, nil
	}

	validMatches, err := m.filterMatchesByContext(content, allMatches)
	if err != nil {
		return "", nil, err
	}

	if len(validMatches) == 0 {
		return content, nil, nil
	}

	if occ.Max != -1 && len(validMatches) > occ.Max {
		validMatches = validMatches[:occ.Max]
	}

	return m.applyMatches(content, re, validMatches)
}

// applySimple applies modifications by iterating matches from right to left.
func (m *Manipulator) applySimple(
	content string,
	re *regexp.Regexp,
	occ model.OccurrenceSpec,
) (string, []model.Change, error) {
	matches := re.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil, nil
	}

	if occ.Max != -1 && len(matches) > occ.Max {
		matches = matches[:occ.Max]
	}

	return m.applyMatches(content, re, matches)
}

// applyWithContext filters matches by context and then applies modifications.
func (m *Manipulator) applyWithContext(
	content string,
	re *regexp.Regexp,
	occ model.OccurrenceSpec,
) (string, []model.Change, error) {
	allMatches := re.FindAllStringSubmatchIndex(content, -1)
	if len(allMatches) == 0 {
		return content, nil, nil
	}

	validMatches, err := m.filterMatchesByContext(content, allMatches)
	if err != nil {
		return "", nil, err
	}

	if len(validMatches) == 0 {
		return content, nil, nil
	}

	if occ.Max != -1 && len(validMatches) > occ.Max {
		validMatches = validMatches[:occ.Max]
	}

	return m.applyMatches(content, re, validMatches)
}

// applyMatches performs the actual byte-level modifications for a given set of matches.
func (m *Manipulator) applyMatches(
	content string,
	re *regexp.Regexp,
	matches [][]int,
) (string, []model.Change, error) {
	lineIdx := computeLineIndex(content)
	b := []byte(content)
	var changes []model.Change

	// Process in reverse to avoid recalculating offsets
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match[0], match[1]

		origBytes := b[start:end]
		var newBytes []byte

		switch m.Config.Operation {
		case model.OpReplace:
			newBytes = re.ExpandString(nil, m.Config.Replacement, content, match)
		case model.OpInsertBefore:
			ins := preserveIndentation(content, start, m.Config.Replacement)
			if !dedupeInsert(b, start, []byte(ins), true) {
				continue
			}
			newBytes = append([]byte(ins), origBytes...)

		case model.OpInsertAfter:
			ins := preserveIndentation(content, end, m.Config.Replacement)
			if !dedupeInsert(b, end, []byte(ins), false) {
				continue
			}
			newBytes = append(origBytes, []byte(ins)...)
		case model.OpDelete:
			newBytes = []byte("")
		default:
			return "", nil, CLIError{
				Code:    ErrInvalidOperation,
				Message: fmt.Sprintf("unknown operation: %s", m.Config.Operation),
			}
		}

		b = util.Splice(b, start, end, newBytes)

		ls, le := byteToLineRange(lineIdx, start, end)
		changes = append(changes, model.Change{
			RuleID:    m.Config.RuleID,
			LineStart: ls,
			LineEnd:   le,
			Start:     start,
			End:       end,
			Original:  string(origBytes),
			New:       string(newBytes),
		})
	}

	util.ReverseChanges(changes)
	return string(b), changes, nil
}

func (m *Manipulator) filterMatchesByContext(content string, allMatches [][]int) ([][]int, error) {
	ctx := m.Config.Context
	var beforeRe, afterRe *regexp.Regexp
	var err error

	if ctx.Before != "" {
		beforeRe, err = regexp.Compile(ctx.Before)
		if err != nil {
			return nil, CLIError{
				Code:    ErrInvalidRegex,
				Message: fmt.Sprintf("invalid before context regex: %s", ctx.Before),
			}
		}
	}
	if ctx.After != "" {
		afterRe, err = regexp.Compile(ctx.After)
		if err != nil {
			return nil, CLIError{
				Code:    ErrInvalidRegex,
				Message: fmt.Sprintf("invalid after context regex: %s", ctx.After),
			}
		}
	}

	lines := strings.Split(content, "\n")
	lineIdx := computeLineIndex(content)
	var valid [][]int

	for _, match := range allMatches {
		start, end := match[0], match[1]
		ls, _ := byteToLineRange(lineIdx, start, end)

		// Check "before" context
		if beforeRe != nil {
			startLine := ls - 1 // 0-indexed
			windowStart := 0
			if ctx.WindowBefore > 0 && startLine-ctx.WindowBefore > 0 {
				windowStart = startLine - ctx.WindowBefore
			}
			window := lines[windowStart:startLine]
			if !beforeRe.MatchString(strings.Join(window, "\n")) {
				continue
			}
		}

		// Check "after" context
		if afterRe != nil {
			// Find the line where the match ends
			_, le := byteToLineRange(lineIdx, start, end)
			endLine := le - 1 // 0-indexed

			windowEnd := len(lines)
			if ctx.WindowAfter > 0 && endLine+1+ctx.WindowAfter < len(lines) {
				windowEnd = endLine + 1 + ctx.WindowAfter
			}
			window := lines[endLine+1 : windowEnd]
			if !afterRe.MatchString(strings.Join(window, "\n")) {
				continue
			}
		}
		valid = append(valid, match)
	}
	return valid, nil
}

// GetCached returns a (possibly cached) matcher for the rule.
// If not present it compiles/creates and stores atomically.
func GetCached(cfg model.ModificationConfig) (matcher.Matcher, error) {
	key := cacheKey(cfg)
	mu.RLock()
	if e, ok := data[key]; ok {
		mu.RUnlock()
		return e.mt, e.err
	}
	mu.RUnlock()

	// slow path – build matcher
	mt, err := buildMatcher(cfg) // existing helper in core
	mu.Lock()
	data[key] = &entry{mt: mt, err: err}
	mu.Unlock()
	return mt, err
}

// cacheKey creates a unique fingerprint for a rule's relevant fields.
func cacheKey(cfg model.ModificationConfig) string {
	return cfg.RuleID + "|" + cfg.Pattern + "|" +
		boolToStr(cfg.UseAST) + "|" + cfg.Lang + "|" +
		boolToStr(cfg.LiteralPattern) + "|" +
		boolToStr(cfg.NormalizeWhitespace) + "|" +
		boolToStr(cfg.Multiline) + boolToStr(cfg.DotAll)
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// findMatches abstracts calling the selected matcher engine and returning
// [][]int compatible with the existing applyMatches flow (start, end only).
func (m *Manipulator) findMatchesBytes(b []byte) ([][]int, error) {
	mat, err := GetCached(m.Config)
	if err != nil {
		return nil, err
	}

	spans, err := mat.Find(b)
	if err != nil {
		return nil, err
	}
	out := make([][]int, len(spans))
	for i, s := range spans {
		out[i] = []int{s.Start, s.End}
	}
	return out, nil
}

// ApplyAST is a thin wrapper that swaps out the regex path with the new matcher
// while re‑using existing modification logic.
func (m *Manipulator) ApplyAST(content string) (string, []model.Change, error) {
	matches, err := m.findMatchesBytes([]byte(content))
	if err != nil {
		return "", nil, err
	}
	if len(matches) == 0 {
		return content, nil, nil // nothing to do
	}

	// Re‑use existing applyMatches with dummy regexp since replacement for AST
	// is currently only delete / insert raw text.
	// For replace operations we simply ignore capture groups (not yet supported).
	// For now, we don't leverage capture groups in AST mode. We just need a
	// placeholder *regexp.Regexp to satisfy applyMatches.
	dummy := regexp.MustCompile("")
	return m.applyMatches(content, dummy, matches)
}

// --- Helpers ---

func parseOccurrences(s string) (model.OccurrenceSpec, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "all":
		return model.OccurrenceSpec{Max: -1}, nil
	case "first":
		return model.OccurrenceSpec{Max: 1}, nil
	default:
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return model.OccurrenceSpec{}, CLIError{
				Code:    ErrInvalidOccurrences,
				Message: fmt.Sprintf("invalid occurrences value: %q", s),
			}
		}
		return model.OccurrenceSpec{Max: n}, nil
	}
}

func preserveIndentation(content string, position int, text string) string {
	lineStart := strings.LastIndex(content[:position], "\n") + 1
	indent := util.TakeIndent(content[lineStart:position])

	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}

	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" {
			lines[i] = indent + strings.TrimPrefix(lines[i], "\r")
		}
	}
	return strings.Join(lines, lineEnding)
}

func computeLineIndex(content string) []int {
	idx := []int{0}
	pos := 0
	for {
		i := strings.IndexByte(content[pos:], '\n')
		if i == -1 {
			break
		}
		pos += i + 1
		idx = append(idx, pos)
	}
	return idx
}

func byteToLine(lineIdx []int, pos int) int {
	lo, hi := 0, len(lineIdx)-1
	line := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if lineIdx[mid] > pos {
			hi = mid - 1
		} else {
			line = mid
			lo = mid + 1
		}
	}
	return line + 1 // 1-based
}

func byteToLineRange(lineIdx []int, start, end int) (int, int) {
	return byteToLine(lineIdx, start), byteToLine(lineIdx, end-1) // end is exclusive
}

// dedupeInsert ensures we don't insert duplicate text if it's already present
// at the given position (prefix for insert-before, suffix for insert-after).
func dedupeInsert(buf []byte, pos int, insert []byte, before bool) bool {
	if before {
		// Check prefix ending at pos
		if pos >= len(insert) && bytes.Equal(buf[pos-len(insert):pos], insert) {
			return false // duplicate, skip
		}
	} else {
		// Check suffix starting at pos
		if pos+len(insert) <= len(buf) && bytes.Equal(buf[pos:pos+len(insert)], insert) {
			return false
		}
	}
	return true // safe to insert
}
