package manipulator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
)

// Manipulator applies a single modification rule to content.
type Manipulator struct {
	Config   *model.Config
	Path     string // Path to the file being manipulated
	Original string // Original content before manipulation
	Data     []byte // Data to manipulate, e.g. file content
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

func Manipulate(cfg *model.Config, path, original string, data []byte) (*model.Result, error) {
	manip := &Manipulator{Config: cfg, Path: path, Original: original, Data: data}
	return manip.start()
}

func (m *Manipulator) fake(node *sitter.Node, start, end int) model.Change {
	ls, le := int(node.StartPoint().Row)+1, int(node.EndPoint().Row)+1
	return model.Change{
		RuleID:    m.Config.RuleID,
		LineStart: ls,
		LineEnd:   le,
		Start:     start,
		End:       end,
		Original:  "",                    // Empty for get operations
		New:       m.Original[start:end], // Put the found content in New
	}
}

func (m *Manipulator) apply(node *sitter.Node, content string, start, end, rewriteStart, rewriteEnd int) (Rewrite, error) {
	var bytes []byte
	switch m.Config.Operation {
	case model.OpReplace:
		bytes = []byte(m.Config.Replacement)
	case model.OpInsertBefore:
		replacement := m.Config.Replacement
		if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(
			nodeType,
		) {
			replacement = strings.TrimSpace(replacement)
			replacement = replacement + "\n\n"
		} else if !strings.HasSuffix(replacement, "\n") {
			replacement = replacement + "\n"
		}
		bytes = []byte(preserveIndentation(content, start, replacement))
		rewriteEnd = start // Insert at 'start', so the span to replace is empty
	case model.OpInsertAfter:
		replacement := m.Config.Replacement
		if nodeType := extractNodeType(m.Config.Pattern); m.Config.Provider.IsBlockLevelNode(
			nodeType,
		) {
			replacement = strings.TrimSpace(replacement)
			replacement = "\n\n" + replacement
		} else if !strings.HasPrefix(replacement, "\n") {
			replacement = "\n" + replacement
		}
		bytes = []byte(preserveIndentation(content, end, replacement))
		rewriteStart = end // Insert at 'end', so the span to replace is empty
	case model.OpDelete:
		bytes = []byte("")
	default:
		return Rewrite{}, model.Wrap(
			model.ErrInvalidOperation,
			fmt.Sprintf("unknown operation: %s", m.Config.Operation),
			nil,
		)
	}

	return Rewrite{
		RuleID:    m.Config.RuleID,
		Start:     rewriteStart,
		End:       rewriteEnd,
		NewText:   bytes,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
	}, nil
}

// apply executes the rule on the content and generates a list of rewrites.
func (m *Manipulator) start() (*model.Result, error) {
	matches, err := m.findMatches([]byte(m.Original))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	var rewrites []Rewrite
	var changes []model.Change
	for _, node := range matches {
		start, end := int(node.StartByte()), int(node.EndByte())
		rewriteStart := start // Default for replace/delete
		rewriteEnd := end     // Default for replace/delete

		if m.Config.Operation == model.OpGet {
			// For get operations, we just return the found content
			changes = append(changes, m.fake(node, start, end))
		} else {
			rewrite, err := m.apply(node, m.Original, start, end, rewriteStart, rewriteEnd)
			if err != nil {
				return nil, err
			}
			rewrites = append(rewrites, rewrite)
		}

	}

	sha := util.SHA1Hex(m.Data)
	res := &model.Result{
		File:            m.Path,
		Time:            time.Now().Format(time.RFC3339),
		SchemaVersion:   model.CurrentSchemaVersion,
		ToolVersion:     model.CurrentToolVersion,
		Success:         true,
		ModifiedCount:   len(changes),
		OriginalSHA1:    sha,
		OriginalContent: m.Original,
		Changes:         changes,
	}
	if m.Config.Operation == model.OpGet {
		res.ChangedBytes = 1 // No actual changes for get operations
		res.ModifiedContent = m.Original
		res.ModifiedSHA1 = sha + "-get" // Indicate this is a get operation
		return res, nil
	}

	// Apply all rewrites to the original content
	modified, changes := applyRewrites(m.Original, rewrites)
	res.ModifiedContent = modified
	res.ModifiedSHA1 = util.SHA1Hex([]byte(modified))
	res.ChangedBytes = util.SumChangedBytes(changes)
	return res, nil
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

// applyRewrites applies a slice of Rewrite operations to the original content.
// It returns the modified content and a slice of model.Change representing the transformations.
func applyRewrites(originalContent string, rewrites []Rewrite) (string, []model.Change) {
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

// getCached returns a (possibly cached) matcher for the rule.
// If not present it compiles/creates and stores atomically.
func getCached(cfg *model.Config) (*matcher.Matcher, error) {
	key := cfg.CacheKey()
	mu.RLock()
	if e, ok := data[key]; ok {
		mu.RUnlock()
		return e.mt, e.err
	}
	mu.RUnlock()

	// slow path – build matcher
	mt, err := matcher.New(cfg)
	mu.Lock()
	if err != nil {
		data[key] = &entry{mt: nil, err: err}
	} else {
		data[key] = &entry{mt: mt, err: err}
	}
	mu.Unlock()
	return data[key].mt, err
}
