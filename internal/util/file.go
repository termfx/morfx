package util

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
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

// UnifiedDiff returns a unified diff of two strings.
func UnifiedDiff(from, to string, path string, context int, color bool) string {
	// This is a placeholder implementation. A real implementation would use a diff library.
	return `diff --git a/` + path + ` b/` + path + `
--- a/` + path + `
+++ b/` + path + `
@@ -1 +1 @@
- ` + from + `
+ ` + to + `
`
}
