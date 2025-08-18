package util

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
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
func UnifiedDiff(from, to, path string, context int) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(from, to, false)

	// Generate diff hunks
	var buf bytes.Buffer
	buf.WriteString("\033[1;36m--- a/" + path + "\033[0m\n")
	buf.WriteString("\033[1;32m+++ b/" + path + "\033[0m\n")

	i, j := 0, 0
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			// Context lines
			lines := strings.Split(d.Text, "\n")
			for k := 0; k < len(lines)-1; k++ {
				if context > 0 {
					buf.WriteString(" " + lines[k] + "\n")
				}
				i++
				j++
			}
		case diffmatchpatch.DiffDelete:
			for line := range strings.SplitSeq(d.Text, "\n") {
				if line != "" {
					buf.WriteString("\033[1;31m-" + line + "\033[0m\n")
				}
				i++
			}
		case diffmatchpatch.DiffInsert:
			for line := range strings.SplitSeq(d.Text, "\n") {
				if line != "" {
					buf.WriteString("\033[1;32m+" + line + "\033[0m\n")
				}
				j++
			}
		}
	}

	return buf.String()
}
