package writer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/termfx/morfx/internal/util"
)

// -----------------------------------------------------------------------------
// Types & helpers
// -----------------------------------------------------------------------------

// StagedChange represents a single file change stored in the staging area.
//
// NOTE: The diff preview is not stored to keep the JSON small; we can always
// generate it on‑demand from OriginalContent ↔ ModifiedContent.
// -----------------------------------------------------------------------------
type StagedChange struct {
	Path            string    `json:"path"`
	OriginalContent string    `json:"original_content"`
	ModifiedContent string    `json:"modified_content"`
	OriginalSHA256  string    `json:"original_sha256"`
	ModifiedSHA256  string    `json:"modified_sha256"`
	SizeDelta       int64     `json:"size_delta"`
	Timestamp       time.Time `json:"timestamp"`
	Operation       string    `json:"operation"` // "modify" | "create" | "delete"
	Query           string    `json:"query"`     // optional DSL that triggered the change
}

// sha256Hex returns the SHA‑256 of data as hex string; empty string for nil slice.
func sha256Hex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// -----------------------------------------------------------------------------
// StagingWriter — stores changes under the .morfx/ directory (no fs mutation)
// -----------------------------------------------------------------------------

type StagingWriter struct {
	stagingDir string
	mu         sync.Mutex
	changes    []StagedChange
}

func NewStagingWriter() *StagingWriter {
	return &StagingWriter{
		stagingDir: ".morfx",
		changes:    make([]StagedChange, 0, 8),
	}
}

// WriteFile records the desired content under the staging dir; it never modifies
// the target path on disk.
func (w *StagingWriter) WriteFile(path string, content []byte, _ os.FileMode) error {
	// read current file (best‑effort)
	originalContent, _ := os.ReadFile(path) // ignore err: if not exist, treat as create

	change := StagedChange{
		Path:            path,
		OriginalContent: string(originalContent),
		ModifiedContent: string(content),
		OriginalSHA256:  sha256Hex(originalContent),
		ModifiedSHA256:  sha256Hex(content),
		SizeDelta:       int64(len(content)) - int64(len(originalContent)),
		Timestamp:       time.Now(),
		Operation:       "modify", // we currently only support modify/create
	}

	w.mu.Lock()
	w.changes = append(w.changes, change)
	w.mu.Unlock()

	if err := os.MkdirAll(w.stagingDir, 0o755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	changeFile := filepath.Join(w.stagingDir, safeFileName(path))
	data, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal change: %w", err)
	}
	if err := os.WriteFile(changeFile, data, 0o644); err != nil {
		return fmt.Errorf("write change file: %w", err)
	}
	return nil
}

// Summary returns a unified diff preview for all staged changes.
func (w *StagingWriter) Summary() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.changes) == 0 {
		return "No changes staged."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Staged %d change(s) in %s/:\n", len(w.changes), w.stagingDir))
	for _, c := range w.changes {
		diff := util.UnifiedDiff(c.OriginalContent, c.ModifiedContent, c.Path, 3)
		if diff != "" {
			sb.WriteString("\n" + diff)
		}
	}
	sb.WriteString("\nRun 'morfx --commit' to apply these changes.\n")
	return sb.String()
}

func safeFileName(path string) string {
	// Produce a filename safe within staging dir
	rep := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return fmt.Sprintf("change_%s.json", rep.Replace(path))
}

// -----------------------------------------------------------------------------
// CommitWriter — applies staged JSON files atomically & safely
// -----------------------------------------------------------------------------

type CommitWriter struct {
	stagingDir   string
	appliedFiles []string
	skippedFiles []string
}

func NewCommitWriter() *CommitWriter {
	return &CommitWriter{
		stagingDir:   ".morfx",
		appliedFiles: make([]string, 0, 8),
		skippedFiles: make([]string, 0, 8),
	}
}

// WriteFile is not supported; use ApplyStagedChanges
func (*CommitWriter) WriteFile(string, []byte, os.FileMode) error {
	return errors.New("CommitWriter does not support WriteFile; call ApplyStagedChanges")
}

func (w *CommitWriter) ApplyStagedChanges() error {
	entries, err := os.ReadDir(w.stagingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no staged changes (no %s dir)", w.stagingDir)
		}
		return err
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if err := w.applyChangeFile(filepath.Join(w.stagingDir, e.Name())); err != nil {
			return err // abort on first error; user can rerun after fixing
		}
	}
	// Remove staging dir only if everything applied OK
	return os.RemoveAll(w.stagingDir)
}

func (w *CommitWriter) applyChangeFile(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var ch StagedChange
	if err := json.Unmarshal(data, &ch); err != nil {
		return err
	}

	// Verify file hasn’t changed since staging
	currentContent, _ := os.ReadFile(ch.Path) // ignore err if not exist
	if sha256Hex(currentContent) != ch.OriginalSHA256 {
		w.skippedFiles = append(w.skippedFiles, ch.Path)
		return fmt.Errorf("file %s modified since staging; aborting", ch.Path)
	}

	// Write atomically
	if err := util.WriteFileAtomic(ch.Path, []byte(ch.ModifiedContent), 0o644); err != nil {
		return err
	}
	w.appliedFiles = append(w.appliedFiles, ch.Path)
	return nil
}

func (w *CommitWriter) Summary() string {
	var sb strings.Builder
	if len(w.appliedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Applied %d file(s):\n", len(w.appliedFiles)))
		for _, p := range w.appliedFiles {
			sb.WriteString("  ✓ " + p + "\n")
		}
	}
	if len(w.skippedFiles) > 0 {
		sb.WriteString(fmt.Sprintf("Skipped %d file(s) due to conflicts:\n", len(w.skippedFiles)))
		for _, p := range w.skippedFiles {
			sb.WriteString("  ✗ " + p + "\n")
		}
	}
	if sb.Len() == 0 {
		return "No changes were applied."
	}
	return sb.String()
}
