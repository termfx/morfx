package writer

import (
	"fmt"
	"os"
	"strings"

	"github.com/termfx/morfx/internal/util"
)

// Writer provides an abstraction for file writing operations.
// It supports both dry-run mode (no actual writes) and commit mode (actual writes).
type Writer interface {
	WriteFile(path string, content []byte, perm os.FileMode) error
	Summary() string
}

// DryRunWriter tracks file changes without writing to disk.
type DryRunWriter struct {
	changes []FileChange
}

// FileChange represents a file that would be modified.
type FileChange struct {
	Path         string
	OriginalSize int
	NewSize      int
	BytesDiff    int
}

// NewDryRunWriter creates a new dry-run writer.
func NewDryRunWriter() *DryRunWriter {
	return &DryRunWriter{
		changes: make([]FileChange, 0),
	}
}

// WriteFile simulates writing a file and tracks the change.
func (w *DryRunWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	var originalSize int
	if stat, err := os.Stat(path); err == nil {
		originalSize = int(stat.Size())
	}

	newSize := len(content)
	bytesDiff := newSize - originalSize

	w.changes = append(w.changes, FileChange{
		Path:         path,
		OriginalSize: originalSize,
		NewSize:      newSize,
		BytesDiff:    bytesDiff,
	})

	return nil
}

// Summary returns a summary of changes that would be made.
func (w *DryRunWriter) Summary() string {
	if len(w.changes) == 0 {
		return "No changes would be made."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Would modify %d file(s):\n", len(w.changes)))

	totalBytesDiff := 0
	for _, change := range w.changes {
		totalBytesDiff += change.BytesDiff
		sign := "+"
		if change.BytesDiff < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("  %s (%s%d bytes)\n", change.Path, sign, change.BytesDiff))
	}

	sign := "+"
	if totalBytesDiff < 0 {
		sign = ""
	}
	sb.WriteString(fmt.Sprintf("Total: %s%d bytes\n", sign, totalBytesDiff))

	return sb.String()
}

// DiskWriter performs actual file writes to disk.
type DiskWriter struct {
	writtenFiles []string
}

// NewDiskWriter creates a new disk writer.
func NewDiskWriter() *DiskWriter {
	return &DiskWriter{
		writtenFiles: make([]string, 0),
	}
}

// WriteFile writes content to the specified file path.
func (w *DiskWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := util.WriteFileAtomic(path, content, perm); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}

	w.writtenFiles = append(w.writtenFiles, path)
	return nil
}

// Summary returns a summary of files that were written.
func (w *DiskWriter) Summary() string {
	if len(w.writtenFiles) == 0 {
		return "No files were written."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Successfully wrote %d file(s):\n", len(w.writtenFiles)))

	for _, path := range w.writtenFiles {
		sb.WriteString(fmt.Sprintf("  %s\n", path))
	}

	return sb.String()
}

// ReadOnlyWriter is used for get operations that don't modify files.
// It implements the Writer interface but doesn't track any changes.
type ReadOnlyWriter struct{}

// NewReadOnlyWriter creates a new read-only writer.
func NewReadOnlyWriter() *ReadOnlyWriter {
	return &ReadOnlyWriter{}
}

// WriteFile does nothing for read-only operations.
func (w *ReadOnlyWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	// Read-only operations don't write files
	return nil
}

// Summary returns an empty string for read-only operations.
func (w *ReadOnlyWriter) Summary() string {
	return ""
}
