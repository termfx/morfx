package util

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandGlobs expands a list of file paths, including those with glob patterns.
func ExpandGlobs(files []string) []string {
	var expandedFiles []string
	for _, file := range files {
		if matches, err := filepath.Glob(file); err == nil {
			expandedFiles = append(expandedFiles, matches...)
		} else {
			expandedFiles = append(expandedFiles, file)
		}
	}
	return expandedFiles
}

// SHA1Hex calculates the SHA1 hash of a byte slice and returns it as a hex string.
func SHA1Hex(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// SHA1FileHex calculates the SHA1 hash of a file's content and returns it as a hex string.
func SHA1FileHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RaceDetected checks if a file has been modified since it was last read.
func RaceDetected(before, after os.FileInfo) bool {
	return before.ModTime() != after.ModTime() || before.Size() != after.Size()
}

// WriteFileAtomic writes data to a file atomically.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpFile.Name(), path)
}

// UnifiedDiff returns a unified diff of two strings in unified diff format.
// Generates stable headers and minimal hunks according to Morfx Core Spec.
func UnifiedDiff(from, to, path string, context int) string {
	if from == to {
		return ""
	}

	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")

	// Remove trailing empty line if present (common in text files)
	if len(fromLines) > 0 && fromLines[len(fromLines)-1] == "" {
		fromLines = fromLines[:len(fromLines)-1]
	}
	if len(toLines) > 0 && toLines[len(toLines)-1] == "" {
		toLines = toLines[:len(toLines)-1]
	}

	hunks := generateDiffHunks(fromLines, toLines, context)
	if len(hunks) == 0 {
		return ""
	}

	var buf bytes.Buffer
	// Stable headers without ANSI colors
	buf.WriteString("--- a/" + path + "\n")
	buf.WriteString("+++ b/" + path + "\n")

	for _, hunk := range hunks {
		buf.WriteString(hunk)
	}

	return buf.String()
}

// generateDiffHunks creates minimal unified diff hunks with proper line numbering.
func generateDiffHunks(fromLines, toLines []string, context int) []string {
	// Simple line-based diff using LCS (Longest Common Subsequence)
	operations := computeLineDiff(fromLines, toLines)

	// If no changes, return empty
	hasChanges := false
	for _, op := range operations {
		if op.Type != "equal" {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return nil
	}

	// Create a single hunk with all operations
	var hunk strings.Builder

	// Calculate hunk boundaries
	firstChange := -1
	lastChange := -1
	for i, op := range operations {
		if op.Type != "equal" {
			if firstChange == -1 {
				firstChange = i
			}
			lastChange = i
		}
	}

	if firstChange == -1 {
		return nil
	}

	// Determine hunk start and end with context
	hunkStart := max(0, firstChange-context)
	hunkEnd := min(len(operations)-1, lastChange+context)

	// Calculate line numbers for hunk header
	hunkStartFrom := 1
	hunkStartTo := 1
	hunkFromCount := 0
	hunkToCount := 0

	if hunkStart < len(operations) {
		op := operations[hunkStart]
		if op.FromLine > 0 {
			hunkStartFrom = op.FromLine
		}
		if op.ToLine > 0 {
			hunkStartTo = op.ToLine
		}
	}

	// Count lines in hunk
	for i := hunkStart; i <= hunkEnd; i++ {
		op := operations[i]
		if op.Type == "equal" || op.Type == "delete" {
			hunkFromCount++
		}
		if op.Type == "equal" || op.Type == "insert" {
			hunkToCount++
		}
	}

	// Generate hunk content
	for i := hunkStart; i <= hunkEnd; i++ {
		op := operations[i]
		switch op.Type {
		case "equal":
			hunk.WriteString(" " + op.Text + "\n")
		case "delete":
			hunk.WriteString("-" + op.Text + "\n")
		case "insert":
			hunk.WriteString("+" + op.Text + "\n")
		}
	}

	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunkStartFrom, hunkFromCount, hunkStartTo, hunkToCount)
	return []string{header + hunk.String()}
}

// computeLineDiff computes line-based differences using a simple algorithm
func computeLineDiff(fromLines, toLines []string) []DiffOp {
	var operations []DiffOp

	// Simple Myers-like algorithm for line comparison
	fromIdx := 0
	toIdx := 0
	fromLineNum := 1
	toLineNum := 1

	for fromIdx < len(fromLines) || toIdx < len(toLines) {
		if fromIdx >= len(fromLines) {
			// Only insertions left
			operations = append(operations, DiffOp{
				Type:     "insert",
				FromLine: 0,
				ToLine:   toLineNum,
				Text:     toLines[toIdx],
			})
			toIdx++
			toLineNum++
		} else if toIdx >= len(toLines) {
			// Only deletions left
			operations = append(operations, DiffOp{
				Type:     "delete",
				FromLine: fromLineNum,
				ToLine:   0,
				Text:     fromLines[fromIdx],
			})
			fromIdx++
			fromLineNum++
		} else if fromLines[fromIdx] == toLines[toIdx] {
			// Lines are equal
			operations = append(operations, DiffOp{
				Type:     "equal",
				FromLine: fromLineNum,
				ToLine:   toLineNum,
				Text:     fromLines[fromIdx],
			})
			fromIdx++
			toIdx++
			fromLineNum++
			toLineNum++
		} else {
			// Lines differ - look ahead to see if we can find a match
			foundMatch := false

			// Look for the current fromLine in the remaining toLines
			for lookAhead := toIdx + 1; lookAhead < len(toLines) && lookAhead < toIdx+5; lookAhead++ {
				if fromLines[fromIdx] == toLines[lookAhead] {
					// Found a match - insert the lines before it
					for insertIdx := toIdx; insertIdx < lookAhead; insertIdx++ {
						operations = append(operations, DiffOp{
							Type:     "insert",
							FromLine: 0,
							ToLine:   toLineNum,
							Text:     toLines[insertIdx],
						})
						toLineNum++
					}
					toIdx = lookAhead
					foundMatch = true
					break
				}
			}

			if !foundMatch {
				// Look for the current toLine in the remaining fromLines
				for lookAhead := fromIdx + 1; lookAhead < len(fromLines) && lookAhead < fromIdx+5; lookAhead++ {
					if toLines[toIdx] == fromLines[lookAhead] {
						// Found a match - delete the lines before it
						for deleteIdx := fromIdx; deleteIdx < lookAhead; deleteIdx++ {
							operations = append(operations, DiffOp{
								Type:     "delete",
								FromLine: fromLineNum,
								ToLine:   0,
								Text:     fromLines[deleteIdx],
							})
							fromLineNum++
						}
						fromIdx = lookAhead
						foundMatch = true
						break
					}
				}
			}

			if !foundMatch {
				// No match found - treat as replacement (delete + insert)
				operations = append(operations, DiffOp{
					Type:     "delete",
					FromLine: fromLineNum,
					ToLine:   0,
					Text:     fromLines[fromIdx],
				})
				operations = append(operations, DiffOp{
					Type:     "insert",
					FromLine: 0,
					ToLine:   toLineNum,
					Text:     toLines[toIdx],
				})
				fromIdx++
				toIdx++
				fromLineNum++
				toLineNum++
			}
		}
	}

	return operations
}

// DiffOp represents a single diff operation
type DiffOp struct {
	Type     string // "equal", "delete", "insert"
	FromLine int    // 1-based line number in original (0 for inserts)
	ToLine   int    // 1-based line number in new (0 for deletes)
	Text     string // line content
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
