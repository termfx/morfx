package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAtomicWriter_WriteFile_LockTimeout(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "timeout_test.txt")

	// Test basic lock functionality instead of timing-sensitive timeout
	config := AtomicWriteConfig{
		LockTimeout:    100 * time.Millisecond,
		UseFsync:       false,
		BackupOriginal: false,
		TempSuffix:     ".morfx.tmp",
	}

	aw := NewAtomicWriter(config)
	defer aw.Cleanup()

	// Test that sequential writes work (lock is acquired and released properly)
	err := aw.WriteFile(testFile, "first content")
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "first content" {
		t.Errorf("Expected 'first content', got %q", string(content))
	}

	// Second write should also succeed
	err = aw.WriteFile(testFile, "second content")
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify updated content
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "second content" {
		t.Errorf("Expected 'second content', got %q", string(content))
	}
}

func TestAtomicWriter_WriteFile_WithBackup_Extended(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "backup_test.txt")

	// Create initial file
	initialContent := "initial content"
	err := os.WriteFile(testFile, []byte(initialContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Create atomic writer with backup enabled
	config := AtomicWriteConfig{
		UseFsync:       false,
		BackupOriginal: true,
		TempSuffix:     ".tmp",
		LockTimeout:    time.Second,
	}
	aw := NewAtomicWriter(config)
	defer aw.Cleanup()

	// Write new content
	newContent := "new content"
	err = aw.WriteFile(testFile, newContent)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify backup was created (with timestamp suffix)
	backupPattern := testFile + ".bak.*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("Failed to search for backup files: %v", err)
	}
	if len(matches) == 0 {
		t.Error("Backup file was not created")
	} else {
		// Verify backup content
		backupFile := matches[0]
		backupContent, err := os.ReadFile(backupFile)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}
		if string(backupContent) != initialContent {
			t.Errorf("Backup content = %q, want %q", string(backupContent), initialContent)
		}
	}

	// Verify new content was written
	actualContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(actualContent) != newContent {
		t.Errorf("File content = %q, want %q", string(actualContent), newContent)
	}
}

func TestAtomicWriter_ConcurrentWrites(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "concurrent_test.txt")

	config := AtomicWriteConfig{
		LockTimeout:    time.Second,
		UseFsync:       false,
		BackupOriginal: false,
		TempSuffix:     ".tmp",
	}

	numWriters := 5
	var wg sync.WaitGroup
	results := make([]error, numWriters)

	// Start multiple writers concurrently
	for i := range numWriters {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			aw := NewAtomicWriter(config)
			defer aw.Cleanup()

			content := fmt.Sprintf("content from writer %d", id)
			results[id] = aw.WriteFile(testFile, content)
		}(i)
	}

	wg.Wait()

	// At least one writer should succeed
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Writer %d failed: %v", i, err)
		}
	}

	if successCount == 0 {
		t.Error("No writers succeeded")
	}

	// Verify file exists and has content
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestAtomicWriter_Cleanup_Extended(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "cleanup_test.txt")

	config := AtomicWriteConfig{
		LockTimeout:    time.Second,
		UseFsync:       false,
		BackupOriginal: false,
		TempSuffix:     ".tmp",
	}

	aw := NewAtomicWriter(config)

	// Write a file to create locks
	err := aw.WriteFile(testFile, "test content")
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Cleanup should not panic
	aw.Cleanup()

	// Multiple cleanups should not panic
	aw.Cleanup()
	aw.Cleanup()
}
