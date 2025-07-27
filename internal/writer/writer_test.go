package writer

import (
	"os"
	"testing"
	"time"
)

func TestDryRunWriter(t *testing.T) {
	w := NewDryRunWriter()

	// Test WriteFile doesn't actually write
	err := w.WriteFile("nonexistent.txt", []byte("test"), 0o644)
	if err != nil {
		t.Errorf("DryRunWriter.WriteFile() error = %v", err)
	}

	// Verify file wasn't created
	if _, err := os.Stat("nonexistent.txt"); !os.IsNotExist(err) {
		t.Error("DryRunWriter should not create files")
	}

	// Test Summary
	summary := w.Summary()
	if summary == "" {
		t.Error("DryRunWriter.Summary() should return non-empty string")
	}
}

func TestStagingWriter(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test file
	testFile := "test.txt"
	originalContent := "original content"
	err := os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w := NewStagingWriter()
	modifiedContent := "modified content"

	// Test WriteFile stages changes
	err = w.WriteFile(testFile, []byte(modifiedContent), 0o644)
	if err != nil {
		t.Errorf("StagingWriter.WriteFile() error = %v", err)
	}

	// Verify staging directory exists
	if _, err := os.Stat(".morfx"); os.IsNotExist(err) {
		t.Error("Staging directory should be created")
	}

	// Verify original file unchanged
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != originalContent {
		t.Error("Original file should not be modified by staging")
	}

	// Test Summary
	summary := w.Summary()
	if summary == "" {
		t.Error("StagingWriter.Summary() should return non-empty string")
	}
}

func TestCommitWriter(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test file and stage a change
	testFile := "test.txt"
	originalContent := "original content"
	modifiedContent := "modified content"

	err := os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage a change
	stagingWriter := NewStagingWriter()
	err = stagingWriter.WriteFile(testFile, []byte(modifiedContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to stage change: %v", err)
	}

	// Apply staged changes
	commitWriter := NewCommitWriter()
	err = commitWriter.ApplyStagedChanges()
	if err != nil {
		t.Errorf("CommitWriter.ApplyStagedChanges() error = %v", err)
	}

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != modifiedContent {
		t.Errorf("File content = %q, want %q", string(content), modifiedContent)
	}

	// Verify staging directory was cleaned up
	if _, err := os.Stat(".morfx"); !os.IsNotExist(err) {
		t.Error("Staging directory should be cleaned up after commit")
	}

	// Test Summary
	summary := commitWriter.Summary()
	if summary == "" {
		t.Error("CommitWriter.Summary() should return non-empty string")
	}
}

func TestCommitWriterNoStagedChanges(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	commitWriter := NewCommitWriter()
	err := commitWriter.ApplyStagedChanges()
	if err == nil {
		t.Error("CommitWriter.ApplyStagedChanges() should error when no staged changes")
	}
}

func TestStagedChangeJSON(t *testing.T) {
	change := StagedChange{
		Path:            "test.go",
		OriginalContent: "original",
		ModifiedContent: "modified",
		OriginalSHA256:  "abc123",
		ModifiedSHA256:  "def456",
		Timestamp:       time.Now(),
		Operation:       "modify",
		Query:           "func:main",
	}

	// Test that all fields are populated
	if change.Path == "" {
		t.Error("Path should not be empty")
	}
	if change.OriginalContent == "" {
		t.Error("OriginalContent should not be empty")
	}
	if change.ModifiedContent == "" {
		t.Error("ModifiedContent should not be empty")
	}
}
