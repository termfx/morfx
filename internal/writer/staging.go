package writer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/garaekz/fileman/internal/util"
)

// StagedChange represents a change that has been staged for commit.
type StagedChange struct {
	Path            string    `json:"path"`
	OriginalContent string    `json:"original_content"`
	ModifiedContent string    `json:"modified_content"`
	OriginalSHA256  string    `json:"original_sha256"`
	ModifiedSHA256  string    `json:"modified_sha256"`
	Timestamp       time.Time `json:"timestamp"`
	Operation       string    `json:"operation"`
	Query           string    `json:"query"`
}

// StagingWriter saves changes to .morfx/ directory for later commit.
type StagingWriter struct {
	stagingDir string
	changes    []StagedChange
	lockFile   *os.File
}

// NewStagingWriter creates a new staging writer.
func NewStagingWriter() *StagingWriter {
	w := &StagingWriter{
		stagingDir: ".morfx",
		changes:    make([]StagedChange, 0),
	}
	// Acquire lock on .morfx/.lock
	lockPath := filepath.Join(w.stagingDir, ".lock")
	os.MkdirAll(w.stagingDir, 0o755)
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err == nil {
		util.Flock(lockFile)
		w.lockFile = lockFile
	}
	return w
}

// WriteFile stages a file change instead of writing it immediately.
func (w *StagingWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	// Read original content
	var originalContent []byte
	if stat, err := os.Stat(path); err == nil && stat.Mode().IsRegular() {
		originalContent, _ = os.ReadFile(path)
	}

	// Create staged change
	change := StagedChange{
		Path:            path,
		OriginalContent: string(originalContent),
		ModifiedContent: string(content),
		OriginalSHA256:  w.sha256Hash(originalContent),
		ModifiedSHA256:  w.sha256Hash(content),
		Timestamp:       time.Now(),
		Operation:       "modify", // Could be extended for different operations
		Query:           "",       // Could be set from context
	}

	w.changes = append(w.changes, change)

	// Ensure staging directory exists
	if err := os.MkdirAll(w.stagingDir, 0o755); err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}
	// Lock is already held by NewStagingWriter

	// Save individual change file
	changeFile := filepath.Join(w.stagingDir, w.changeFileName(path))
	changeData, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling change data: %w", err)
	}

	// Retry mechanism for transient errors
	for i := 0; i < 3; i++ {
		err = os.WriteFile(changeFile, changeData, 0o644)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("writing change file: %w", err)
	}

	return nil
}

// Summary returns a summary of staged changes.
func (w *StagingWriter) Summary() string {
	if len(w.changes) == 0 {
		return "No changes staged."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Staged %d change(s) in %s/:\n", len(w.changes), w.stagingDir))

	for _, change := range w.changes {
		// Show diff preview
		diff := util.UnifiedDiff(change.OriginalContent, change.ModifiedContent, change.Path, 3)
		if diff != "" {
			sb.WriteString(fmt.Sprintf("\n%s", diff))
		}
	}

	sb.WriteString("\nRun 'morfx --commit' to apply these changes.\n")
	return sb.String()
}

// CommitWriter applies staged changes from .morfx/ directory.
type CommitWriter struct {
	stagingDir   string
	appliedFiles []string
	skippedFiles []string
	lockFile     *os.File
}

// NewCommitWriter creates a new commit writer.
func NewCommitWriter() *CommitWriter {
	w := &CommitWriter{
		stagingDir:   ".morfx",
		appliedFiles: make([]string, 0),
		skippedFiles: make([]string, 0),
	}
	// Acquire lock on .morfx/.lock
	lockPath := filepath.Join(w.stagingDir, ".lock")
	os.MkdirAll(w.stagingDir, 0o755)
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err == nil {
		util.Flock(lockFile)
		w.lockFile = lockFile
	}
	return w
}

// WriteFile is not used by CommitWriter - it applies staged changes instead.
func (w *CommitWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	// This method is not used by CommitWriter
	return fmt.Errorf("CommitWriter does not support WriteFile - use ApplyStagedChanges instead")
}

// ApplyStagedChanges reads and applies all staged changes.
func (w *CommitWriter) ApplyStagedChanges() error {
	if _, err := os.Stat(w.stagingDir); os.IsNotExist(err) {
		return fmt.Errorf("no staged changes found (no %s directory)", w.stagingDir)
	}

	// Read all change files
	entries, err := os.ReadDir(w.stagingDir)
	if err != nil {
		return fmt.Errorf("reading staging directory: %w", err)
	}

	jsonCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			jsonCount++
		}
	}
	if jsonCount == 0 {
		return fmt.Errorf("no staged changes found")
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		changeFile := filepath.Join(w.stagingDir, entry.Name())
		var lastErr error
		for i := 0; i < 3; i++ {
			lastErr = w.applyChangeFile(changeFile)
			if lastErr == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if lastErr != nil {
			return fmt.Errorf("applying change file %s: %w", changeFile, lastErr)
		}
	}

	// Clean up staging directory after successful commit
	for i := 0; i < 3; i++ {
		err = os.RemoveAll(w.stagingDir)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("cleaning up staging directory: %w", err)
	}

	return nil
}

// applyChangeFile applies a single staged change file.
func (w *CommitWriter) applyChangeFile(changeFile string) error {
	data, err := os.ReadFile(changeFile)
	if err != nil {
		return fmt.Errorf("reading change file: %w", err)
	}

	var change StagedChange
	if err := json.Unmarshal(data, &change); err != nil {
		return fmt.Errorf("unmarshaling change data: %w", err)
	}

	// Verify original file hasn't changed since staging
	if _, err := os.Stat(change.Path); err == nil {
		currentContent, err := os.ReadFile(change.Path)
		if err != nil {
			return fmt.Errorf("reading current file content: %w", err)
		}

		currentHash := w.sha256Hash(currentContent)
		if currentHash != change.OriginalSHA256 {
			w.skippedFiles = append(w.skippedFiles, change.Path)
			return fmt.Errorf("file %s has been modified since staging (hash mismatch)", change.Path)
		}
	}

	// Apply the change
	if err := util.WriteFileAtomic(change.Path, []byte(change.ModifiedContent), 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", change.Path, err)
	}

	w.appliedFiles = append(w.appliedFiles, change.Path)
	return nil
}

// Summary returns a summary of applied changes.
func (w *CommitWriter) Summary() string {
	var sb strings.Builder

	if len(w.appliedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Applied changes to %d file(s):\n", len(w.appliedFiles)))
		for _, path := range w.appliedFiles {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", path))
		}
	}

	if len(w.skippedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Skipped %d file(s) due to conflicts:\n", len(w.skippedFiles)))
		for _, path := range w.skippedFiles {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", path))
		}
	}

	if len(w.appliedFiles) == 0 && len(w.skippedFiles) == 0 {
		return "No changes were applied."
	}

	return sb.String()
}

// Helper methods

func (w *StagingWriter) sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (w *CommitWriter) sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (w *StagingWriter) changeFileName(path string) string {
	// Create a safe filename from the file path
	safe := strings.ReplaceAll(path, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	return fmt.Sprintf("change_%s.json", safe)
}
