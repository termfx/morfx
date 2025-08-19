package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/termfx/morfx/internal/lang/golang"
)

func TestScannerBasic(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test files
	testFiles := []string{"main.go", "utils.go", "README.md"}
	for _, file := range testFiles {
		err := os.WriteFile(file, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	cfg := Config{
		Provider: golang.NewProvider(),
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should find .go files but not .md files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}
}

func TestScannerWithGitignore(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create .gitignore
	gitignoreContent := "*.tmp\nignored.go\n"
	err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create test files
	testFiles := []string{"main.go", "ignored.go", "temp.tmp"}
	for _, file := range testFiles {
		if writeErr := os.WriteFile(file, []byte("package main"), 0o644); writeErr != nil {
			t.Fatalf("Failed to create test file %s: %v", file, writeErr)
		}
	}

	cfg := Config{
		Provider:    golang.NewProvider(),
		NoGitignore: false,
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should only find main.go (ignored.go and temp.tmp should be filtered)
	expectedCount := 1
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	if len(files) > 0 && filepath.Base(files[0]) != "main.go" {
		t.Errorf("Expected main.go, got %s", filepath.Base(files[0]))
	}
}

func TestScannerNoGitignore(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create .gitignore
	gitignoreContent := "*.tmp\nignored.go\n"
	err := os.WriteFile(".gitignore", []byte(gitignoreContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create test files
	testFiles := []string{"main.go", "ignored.go"}
	for _, file := range testFiles {
		writeErr := os.WriteFile(file, []byte("package main"), 0o644)
		if writeErr != nil {
			t.Fatalf("Failed to create test file %s: %v", file, writeErr)
		}
	}

	cfg := Config{
		Provider:    golang.NewProvider(),
		NoGitignore: true, // Disable gitignore filtering
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should find both .go files when gitignore is disabled
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}
}

func TestScannerIncludeExclude(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test files
	testFiles := []string{"main.go", "test_main.go", "utils.go"}
	for _, file := range testFiles {
		err := os.WriteFile(file, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	cfg := Config{
		Provider:     golang.NewProvider(),
		IncludeGlobs: []string{"test_*.go"},
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should only find test_main.go
	expectedCount := 1
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	if len(files) > 0 && filepath.Base(files[0]) != "test_main.go" {
		t.Errorf("Expected test_main.go, got %s", filepath.Base(files[0]))
	}
}

func TestScannerMaxBytes(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create small and large files
	smallContent := "package main"
	largeContent := make([]byte, 1000)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	err := os.WriteFile("small.go", []byte(smallContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	err = os.WriteFile("large.go", largeContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	cfg := Config{
		Provider: golang.NewProvider(),
		MaxBytes: 100, // Only allow files smaller than 100 bytes
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should only find small.go
	expectedCount := 1
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	if len(files) > 0 && filepath.Base(files[0]) != "small.go" {
		t.Errorf("Expected small.go, got %s", filepath.Base(files[0]))
	}
}

func TestScannerDirectorySkipping(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create directories that should be skipped
	skipDirs := []string{".git", "vendor", "node_modules"}
	for _, dir := range skipDirs {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create a .go file in the skipped directory
		filePath := filepath.Join(dir, "test.go")
		err = os.WriteFile(filePath, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create file in %s: %v", dir, err)
		}
	}

	// Create a normal .go file
	err := os.WriteFile("main.go", []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	cfg := Config{
		Provider: golang.NewProvider(),
	}
	s := New(cfg)

	files, err := s.ScanTargets(context.Background(), []string{"."})
	if err != nil {
		t.Errorf("ScanTargets() error = %v", err)
	}

	// Should only find main.go (files in skipped directories should be ignored)
	expectedCount := 1
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	if len(files) > 0 && filepath.Base(files[0]) != "main.go" {
		t.Errorf("Expected main.go, got %s", filepath.Base(files[0]))
	}
}
