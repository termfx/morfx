package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileWalker_ValidateScope_InvalidPath(t *testing.T) {
	walker := NewFileWalker()

	tests := []struct {
		name        string
		scope       FileScope
		expectError bool
	}{
		{
			name:        "empty path",
			scope:       FileScope{Path: ""},
			expectError: true,
		},
		{
			name:        "nonexistent path",
			scope:       FileScope{Path: "/nonexistent/directory"},
			expectError: true,
		},
		{
			name:        "file instead of directory",
			scope:       FileScope{Path: "/etc/passwd"}, // Assuming this is a file
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := walker.validateScope(tt.scope)
			if (err != nil) != tt.expectError {
				t.Errorf("validateScope() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestFileWalker_Walk_CancelledContext(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create some test files
	for i := range 5 {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(testFile, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go"},
		Language: "go",
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Consume results (may be none due to cancellation)
	count := 0
	for result := range results {
		count++
		if result.Error != nil {
			t.Logf("Result error: %v", result.Error)
		}
	}

	// Due to context cancellation, we may get 0 or partial results
	t.Logf("Got %d results with cancelled context", count)
}

func TestFileWalker_Walk_MaxDepthLimit(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create nested directory structure
	level1Dir := filepath.Join(tempDir, "level1")
	level2Dir := filepath.Join(level1Dir, "level2")
	level3Dir := filepath.Join(level2Dir, "level3")

	err := os.MkdirAll(level3Dir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}

	// Create files at each level
	files := []string{
		filepath.Join(tempDir, "root.go"),
		filepath.Join(level1Dir, "level1.go"),
		filepath.Join(level2Dir, "level2.go"),
		filepath.Join(level3Dir, "level3.go"),
	}

	for _, file := range files {
		err := os.WriteFile(file, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	ctx := context.Background()
	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go"},
		MaxDepth: 2, // Should stop at level2
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	foundFiles := make(map[string]bool)
	for result := range results {
		if result.Error == nil {
			foundFiles[result.Path] = true
		}
	}

	// Should find root.go, level1.go, level2.go but NOT level3.go
	expectedFiles := []string{
		filepath.Join(tempDir, "root.go"),
		filepath.Join(level1Dir, "level1.go"),
		filepath.Join(level2Dir, "level2.go"),
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected to find file: %s", expected)
		}
	}

	// Should NOT find level3.go
	level3File := filepath.Join(level3Dir, "level3.go")
	if foundFiles[level3File] {
		t.Errorf("Should not find file beyond max depth: %s", level3File)
	}
}

func TestFileWalker_Walk_MaxFilesLimit(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create more files than the limit
	numFiles := 10
	for i := range numFiles {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(testFile, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	ctx := context.Background()
	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go"},
		MaxFiles: 5, // Limit to 5 files
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	count := 0
	for result := range results {
		if result.Error == nil {
			count++
		}
	}

	// Should have at most 5 files
	if count > 5 {
		t.Errorf("Expected at most 5 files, got %d", count)
	}
}

func TestFileWalker_Walk_PermissionDenied(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create a directory we can't read
	restrictedDir := filepath.Join(tempDir, "restricted")
	err := os.Mkdir(restrictedDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create restricted directory: %v", err)
	}

	// Create a file in the restricted directory
	testFile := filepath.Join(restrictedDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make directory unreadable
	err = os.Chmod(restrictedDir, 0o000)
	if err != nil {
		t.Fatalf("Failed to make directory unreadable: %v", err)
	}
	defer os.Chmod(restrictedDir, 0o755) // Restore for cleanup

	ctx := context.Background()
	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go"},
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Should complete without crashing, but may skip inaccessible directories
	count := 0
	for result := range results {
		count++
		if result.Error != nil {
			t.Logf("Result error: %v", result.Error)
		}
	}

	// The walker should handle permission errors gracefully
	t.Logf("Processed %d results with permission restrictions", count)
}

func TestFileWalker_MatchPattern_DoubleAsterisk(t *testing.T) {
	walker := NewFileWalker()

	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"**/*.go", "src/main.go", true},     // Fixed: ** should match any directory depth
		{"**/*.go", "src/pkg/util.go", true}, // Fixed: ** should match any directory depth
		{"**/*.go", "main.go", true},         // Fixed: ** should match zero directories too
		{"**/*.go", "main.py", false},
		{"src/**/*.go", "src/main.go", true},     // Fixed: should match within src/
		{"src/**/*.go", "src/pkg/util.go", true}, // Fixed: should match within src/
		{"src/**/*.go", "other/main.go", false},
		{"**", "any/path/file.txt", true},
		{"**/test", "src/test", true},
		{"**/test", "src/test.go", false}, // Fixed: test != test.go
		{"prefix/**", "prefix/any/file", true},
		{"prefix/**", "other/prefix/file", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.pattern, tt.path), func(t *testing.T) {
			result := walker.matchPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, expected %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestFileWalker_MatchPattern_SimplePatterns(t *testing.T) {
	walker := NewFileWalker()

	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", true}, // Matches basename
		{"*.py", "main.go", false},
		{"test*", "test_file.go", true},
		{"test*", "src/test_file.go", true}, // Matches basename
		{"*/main.go", "src/main.go", true},
		{"*/main.go", "main.go", false},
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "other/main.go", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.pattern, tt.path), func(t *testing.T) {
			result := walker.matchPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, expected %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestFileWalker_IncludeExclude_EdgeCases(t *testing.T) {
	walker := NewFileWalker()

	tests := []struct {
		name     string
		path     string
		include  []string
		exclude  []string
		included bool
	}{
		{
			name:     "empty include patterns - include all",
			path:     "any/file.go",
			include:  []string{},
			exclude:  []string{},
			included: true,
		},
		{
			name:     "excluded overrides included",
			path:     "test.go",
			include:  []string{"*.go"},
			exclude:  []string{"test*"},
			included: false,
		},
		{
			name:     "multiple include patterns",
			path:     "file.py",
			include:  []string{"*.go", "*.py"},
			exclude:  []string{},
			included: true,
		},
		{
			name:     "multiple exclude patterns",
			path:     "test.go",
			include:  []string{"*.go"},
			exclude:  []string{"test*", "*.tmp"},
			included: false,
		},
		{
			name:     "no match in include patterns",
			path:     "file.txt",
			include:  []string{"*.go", "*.py"},
			exclude:  []string{},
			included: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			included := walker.isIncluded(tt.path, tt.include)
			excluded := walker.isExcluded(tt.path, tt.exclude)
			actualIncluded := included && !excluded

			if actualIncluded != tt.included {
				t.Errorf("Expected path %s to be included: %v, got: %v (included: %v, excluded: %v)",
					tt.path, tt.included, actualIncluded, included, excluded)
			}
		})
	}
}

func TestFileWalker_DetectLanguage_EdgeCases(t *testing.T) {
	walker := NewFileWalker()

	tests := []struct {
		path     string
		expected string
	}{
		{"file.GO", "go"},           // Case insensitive
		{"file.Py", "python"},       // Case insensitive
		{"file.JS", "javascript"},   // Case insensitive
		{"file", "unknown"},         // No extension
		{"file.unknown", "unknown"}, // Unknown extension
		{".hiddenfile", "unknown"},  // Hidden file with no extension
		{".hidden.go", "go"},        // Hidden file with extension
		{"path/to/file.rs", "rust"}, // Full path
		{"file.tar.gz", "unknown"},  // Multiple extensions
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := walker.detectLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileWalker_ProcessFile_StatError(t *testing.T) {
	walker := NewFileWalker()
	scope := FileScope{Language: "go"}

	// Test with non-existent file
	result := walker.processFile("/nonexistent/file.go", scope)

	if result.Error == nil {
		t.Error("Expected error for non-existent file")
	}

	if result.Path != "/nonexistent/file.go" {
		t.Errorf("Expected path to be preserved, got: %s", result.Path)
	}

	if result.Info != nil {
		t.Error("Expected nil file info for failed stat")
	}
}

func TestFileWalker_FastScan_WithErrors(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create some valid files
	for i := range 3 {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(testFile, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a directory we can't read to generate errors
	restrictedDir := filepath.Join(tempDir, "restricted")
	err := os.Mkdir(restrictedDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create restricted directory: %v", err)
	}

	err = os.WriteFile(filepath.Join(restrictedDir, "test.go"), []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create file in restricted directory: %v", err)
	}

	err = os.Chmod(restrictedDir, 0o000)
	if err != nil {
		t.Fatalf("Failed to make directory unreadable: %v", err)
	}
	defer os.Chmod(restrictedDir, 0o755)

	ctx := context.Background()
	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go"},
	}

	files, err := walker.FastScan(ctx, scope)
	if err != nil {
		t.Fatalf("FastScan failed: %v", err)
	}

	// Should get some files despite errors (FastScan skips errors)
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify all returned paths are valid
	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			t.Errorf("Expected .go file, got: %s", file)
		}
	}
}

func TestFileWalker_GetLanguageStats_WithErrors(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create files with different languages
	files := map[string]string{
		"main.go":   "go",
		"script.py": "python",
		"app.js":    "javascript",
		"style.css": "unknown", // Not in language map
	}

	for filename := range files {
		testFile := filepath.Join(tempDir, filename)
		err := os.WriteFile(testFile, []byte("content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	scope := FileScope{Path: tempDir}

	stats, err := walker.GetLanguageStats(ctx, scope)
	if err != nil {
		t.Fatalf("GetLanguageStats failed: %v", err)
	}

	expected := map[string]int{
		"go":         1,
		"python":     1,
		"javascript": 1,
		"unknown":    1,
	}

	for lang, expectedCount := range expected {
		if actualCount := stats[lang]; actualCount != expectedCount {
			t.Errorf("Expected %d files for language %s, got %d", expectedCount, lang, actualCount)
		}
	}
}

func TestFileWalker_Walk_EmptyDirectory(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	ctx := context.Background()
	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go"},
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	count := 0
	for result := range results {
		count++
		if result.Error != nil {
			t.Errorf("Unexpected error: %v", result.Error)
		}
	}

	if count != 0 {
		t.Errorf("Expected 0 results for empty directory, got %d", count)
	}
}

func TestFileWalker_Walk_SymlinkHandling(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create a file to link to
	targetFile := filepath.Join(tempDir, "target.go")
	err := os.WriteFile(targetFile, []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create a symlink
	symlinkFile := filepath.Join(tempDir, "link.go")
	err = os.Symlink(targetFile, symlinkFile)
	if err != nil {
		t.Skipf("Skipping symlink test: %v", err) // Skip if symlinks not supported
	}

	ctx := context.Background()
	scope := FileScope{
		Path:           tempDir,
		Include:        []string{"*.go"},
		FollowSymlinks: false, // Default behavior
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	foundFiles := make(map[string]bool)
	for result := range results {
		if result.Error == nil {
			foundFiles[result.Path] = true
		}
	}

	// Should find both the target file and symlink
	if !foundFiles[targetFile] {
		t.Error("Should find target file")
	}

	if !foundFiles[symlinkFile] {
		t.Error("Should find symlink file")
	}
}

func TestFileWalker_Worker_ChannelClose(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go"},
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Partially consume results then stop
	count := 0
	for result := range results {
		count++
		if result.Error == nil && count >= 1 {
			break // Stop early to test channel cleanup
		}
	}

	// The remaining results should be handled gracefully when channel closes
	// This tests the worker goroutine cleanup logic
}

func TestFileWalker_ScanDirectory_ContextCancellation(t *testing.T) {
	walker := NewFileWalker()
	tempDir := t.TempDir()

	// Create many files to increase chance of catching cancellation
	for i := range 20 {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(testFile, []byte("package main"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Use a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go"},
	}

	results, err := walker.Walk(ctx, scope)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Consume all results (may be partial due to cancellation)
	count := 0
	for result := range results {
		count++
		if result.Error != nil {
			t.Logf("Result error: %v", result.Error)
		}
	}

	// Due to timing, we might get 0 to all files
	t.Logf("Got %d results with quick cancellation", count)
}
