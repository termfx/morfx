package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAtomicWriter_FreshIncompleteLockTimesOut(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "fresh-incomplete.txt")
	lockFile := testFile + ".lock"

	if err := os.WriteFile(lockFile, nil, 0o600); err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}

	config := DefaultAtomicConfig()
	config.BackupOriginal = false
	config.LockTimeout = 30 * time.Millisecond

	writer := NewAtomicWriter(config)

	err := writer.WriteFile(testFile, "content")
	if err == nil {
		t.Fatal("Expected timeout when lock file is fresh but incomplete")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}
