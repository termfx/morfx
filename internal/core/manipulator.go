package core

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
)

// Manipulator applies a single modification rule to content.
type Manipulator struct {
	Config *model.Config
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
func NewManipulator(cfg *model.Config) *Manipulator {
	return &Manipulator{Config: cfg}
}

// ApplyHarmless is a special case for CLI testing where we run the rule without
// actually modifying the content. It returns the changes that would be made.
func (m *Manipulator) ApplyHarmless(content string) ([]model.Change, error) {
	matches, err := m.findMatchesBytes([]byte(content))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	lineIdx := computeLineIndex(content)
	b := []byte(content)
	var changes []model.Change
	for _, match := range matches {
		start, end := match[0], match[1]
		origBytes := b[start:end]
		var newBytes []byte

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

	return changes, nil
}

// Apply executes the rule on the content and delegates to the appropriate helpers.
func (m *Manipulator) Apply(content string) (string, []model.Change, error) {
	matches, err := m.findMatchesBytes([]byte(content))
	if err != nil {
		return "", nil, err
	}
	if len(matches) == 0 {
		return content, nil, nil
	}
	lineIdx := computeLineIndex(content)
	b := []byte(content)
	var changes []model.Change

	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match[0], match[1]
		origBytes := b[start:end]
		var newBytes []byte

		switch m.Config.Operation {
		case model.OpGet:
			newBytes = origBytes
		case model.OpReplace:
			newBytes = []byte(m.Config.Replacement)
		case model.OpInsertBefore:
			replacement := m.Config.Replacement
			// Ensure we add a newline after the replacement if inserting before a block-level node
			if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(nodeType) {
				replacement = strings.TrimSpace(replacement)
				replacement = replacement + "\n\n"
			}
			ins := preserveIndentation(content, start, replacement)
			newBytes = append([]byte(ins), origBytes...)

		case model.OpInsertAfter:
			replacement := m.Config.Replacement
			// Ensure we add a newline before the replacement if inserting after a block-level node
			if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(nodeType) {
				replacement = strings.TrimSpace(replacement)
				replacement = "\n\n" + replacement
			}
			ins := preserveIndentation(content, end, replacement)
			newBytes = append(origBytes, []byte(ins)...)
		case model.OpDelete:
			newBytes = []byte("")
		default:
			return "", nil, model.Wrap(model.ErrInvalidOperation, fmt.Sprintf("unknown operation: %s", m.Config.Operation), nil)
		}

		// For 'get', we don't actually splice the bytes.
		if m.Config.Operation != model.OpGet {
			b = util.Splice(b, start, end, newBytes)
		}

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

// findMatchesBytes abstracts calling the selected matcher engine and returning
// [][]int compatible with the existing applyMatches flow (start, end only).
func (m *Manipulator) findMatchesBytes(b []byte) ([][]int, error) {
	mat, err := getCached(m.Config)
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

// --- Helpers ---

// extractNodeType extracts the node type from a DSL pattern.
// For example, "func:Init" returns "func", "!struct:User" returns "struct".
func extractNodeType(pattern string) string {
	// Remove negation prefix if present
	if strings.HasPrefix(pattern, "!") {
		pattern = strings.TrimPrefix(pattern, "!")
	}

	// Split by > for parent/child relationships and take the first part
	parts := strings.Split(pattern, ">")
	if len(parts) == 0 {
		return ""
	}

	// Split by : to get node type
	firstPart := strings.TrimSpace(parts[0])
	nodeParts := strings.SplitN(firstPart, ":", 2)
	if len(nodeParts) < 1 {
		return ""
	}

	return strings.TrimSpace(nodeParts[0])
}

func preserveIndentation(content string, position int, text string) string {
	lineStart := strings.LastIndex(content[:position], "\n") + 1
	indent := util.TakeIndent(content[lineStart:position])

	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[0] != "" {
		lines[0] = indent + strings.TrimPrefix(lines[0], "\r")
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" {
			lines[i] = indent + strings.TrimPrefix(lines[i], "\r")
		}
	}
	return strings.Join(lines, lineEnding)
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
