package writer

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/garaekz/fileman/internal/util"
)

// InteractiveWriter shows diffs and asks for user confirmation before writing.
type InteractiveWriter struct {
	diskWriter   *DiskWriter
	dryRunWriter *DryRunWriter
	confirmed    []string
	rejected     []string
}

// NewInteractiveWriter creates a new interactive writer.
func NewInteractiveWriter() *InteractiveWriter {
	return &InteractiveWriter{
		diskWriter:   NewDiskWriter(),
		dryRunWriter: NewDryRunWriter(),
		confirmed:    make([]string, 0),
		rejected:     make([]string, 0),
	}
}

// WriteFile shows a diff and asks for user confirmation before writing.
func (w *InteractiveWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	// Read original content for diff
	var originalContent []byte
	if stat, err := os.Stat(path); err == nil && stat.Mode().IsRegular() {
		originalContent, _ = os.ReadFile(path)
	}

	// Show diff
	diff := util.UnifiedDiff(string(originalContent), string(content), path, 3)
	if diff != "" {
		fmt.Print(diff)
		fmt.Printf("\nApply changes to %s? [y/N/q]: ", path)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading user input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			w.confirmed = append(w.confirmed, path)
			return w.diskWriter.WriteFile(path, content, perm)
		case "q", "quit":
			return fmt.Errorf("user cancelled operation")
		default:
			w.rejected = append(w.rejected, path)
			return w.dryRunWriter.WriteFile(path, content, perm)
		}
	}

	// No changes, just track as dry-run
	return w.dryRunWriter.WriteFile(path, content, perm)
}

// Summary returns a summary of user decisions.
func (w *InteractiveWriter) Summary() string {
	var sb strings.Builder

	if len(w.confirmed) > 0 {
		sb.WriteString(fmt.Sprintf("Applied changes to %d file(s):\n", len(w.confirmed)))
		for _, path := range w.confirmed {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", path))
		}
	}

	if len(w.rejected) > 0 {
		sb.WriteString(fmt.Sprintf("Rejected changes to %d file(s):\n", len(w.rejected)))
		for _, path := range w.rejected {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", path))
		}
	}

	if len(w.confirmed) == 0 && len(w.rejected) == 0 {
		return "No changes were proposed."
	}

	return sb.String()
}
