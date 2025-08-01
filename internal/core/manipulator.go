package core

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
)

// Manipulator applies a single modification rule to content.
type Manipulator struct {
	Config *model.Config
}

type entry struct {
	mt  *matcher.Matcher
	err error
}

// Rewrite represents a single modification operation on the source code.
type Rewrite struct {
	RuleID    string
	Start     int
	End       int
	NewText   []byte
	LineStart int
	LineEnd   int
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
	matches, err := m.findMatches([]byte(content))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	var changes []model.Change
	for _, node := range matches {
		start, end := int(node.StartByte()), int(node.EndByte())
		matchedContent := content[start:end]

		ls, le := int(node.StartPoint().Row)+1, int(node.EndPoint().Row)+1
		changes = append(changes, model.Change{
			RuleID:    m.Config.RuleID,
			LineStart: ls,
			LineEnd:   le,
			Start:     start,
			End:       end,
			Original:  "",             // Empty for get operations
			New:       matchedContent, // Put the found content in New
		})
	}

	return changes, nil
}

// Apply executes the rule on the content and generates a list of rewrites.
func (m *Manipulator) Apply(content string) ([]Rewrite, error) {
	matches, err := m.findMatches([]byte(content))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	var rewrites []Rewrite

	for _, node := range matches {
		start := int(node.StartByte())
		end := int(node.EndByte())
		var newBytes []byte
		rewriteStart := start // Default for replace/delete
		rewriteEnd := end     // Default for replace/delete

		switch m.Config.Operation {
		case model.OpGet:
			// For OpGet, we don't generate a rewrite, as it's a read-only operation.
			continue
		case model.OpReplace:
			newBytes = []byte(m.Config.Replacement)
		case model.OpInsertBefore:
			replacement := m.Config.Replacement
			if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(nodeType) {
				replacement = strings.TrimSpace(replacement)
				replacement = replacement + "\n\n"
			} else if !strings.HasSuffix(replacement, "\n") {
				replacement = replacement + "\n"
			}
			newBytes = []byte(preserveIndentation(content, start, replacement))
			rewriteEnd = start // Insert at 'start', so the span to replace is empty

		case model.OpInsertAfter:
			replacement := m.Config.Replacement
			if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(nodeType) {
				replacement = strings.TrimSpace(replacement)
				replacement = "\n\n" + replacement
			} else if !strings.HasPrefix(replacement, "\n") {
				replacement = "\n" + replacement
			}
			newBytes = []byte(preserveIndentation(content, end, replacement))
			rewriteStart = end // Insert at 'end', so the span to replace is empty

		case model.OpDelete:
			newBytes = []byte("")
		default:
			return nil, model.Wrap(model.ErrInvalidOperation, fmt.Sprintf("unknown operation: %s", m.Config.Operation), nil)
		}

		rewrites = append(rewrites, Rewrite{
			RuleID:    m.Config.RuleID,
			Start:     rewriteStart,
			End:       rewriteEnd,
			NewText:   newBytes,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
		})
	}

	return rewrites, nil
}

// findMatches abstracts calling the selected matcher engine and returning
// []*sitter.Node compatible with the existing applyMatches flow.
func (m *Manipulator) findMatches(b []byte) ([]*sitter.Node, error) {
	mat, err := getCached(m.Config)
	if err != nil {
		return nil, err
	}
	if mat == nil {
		return nil, nil
	}

	nodes, err := mat.Find(b)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// ApplyRewrites applies a slice of Rewrite operations to the original content.
// It returns the modified content and a slice of model.Change representing the transformations.
func ApplyRewrites(originalContent string, rewrites []Rewrite) (string, []model.Change) {
	b := []byte(originalContent)
	var changes []model.Change

	// Sort rewrites in reverse order to apply them from end to start
	// This prevents issues with byte offsets changing for subsequent rewrites.
	for i, j := 0, len(rewrites)-1; i < j; i, j = i+1, j-1 {
		rewrites[i], rewrites[j] = rewrites[j], rewrites[i]
	}

	for _, r := range rewrites {
		origBytes := b[r.Start:r.End]
		b = util.Splice(b, r.Start, r.End, r.NewText)

		changes = append(changes, model.Change{
			RuleID:    r.RuleID,
			LineStart: r.LineStart,
			LineEnd:   r.LineEnd,
			Start:     r.Start,
			End:       r.End,
			Original:  string(origBytes),
			New:       string(r.NewText),
		})
	}

	util.ReverseChanges(changes) // Reverse again to get chronological order
	return string(b), changes
}

// --- Helpers ---

// extractNodeType extracts the node type from a DSL pattern.
// For example, "func:Init" returns "func", "!struct:User" returns "struct".
func extractNodeType(pattern string) string {
	// Remove negation prefix if present
	if after, ok := strings.CutPrefix(pattern, "!"); ok {
		pattern = after
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
	for i, line := range lines {
		if i == 0 {
			// First line gets the existing indentation
			lines[i] = indent + strings.TrimPrefix(line, "\r")
		} else if line != "" {
			// Subsequent non-empty lines get the same indentation
			lines[i] = indent + strings.TrimPrefix(line, "\r")
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
