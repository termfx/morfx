package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWalker_DetectLanguage(t *testing.T) {
	tempDir := t.TempDir()
	walker := NewFileWalker()

	tests := []struct {
		filename string
		expected string
	}{
		{"test.go", "go"},
		{"test.js", "javascript"},
		{"test.ts", "typescript"},
		{"test.py", "python"},
		{"test.pyw", "python"},
		{"test.pyi", "python"},
		{"test.java", "java"},
		{"test.cpp", "cpp"},
		{"test.c", "c"},
		{"test.rb", "ruby"},
		{"test.rs", "rust"},
		{"test.php", "php"},
		{"test.phtml", "php"},
		{"test.php5", "php"},
		{"test.jsx", "javascript"},
		{"test.tsx", "typescript"},
		{"test.mjs", "javascript"},
		{"test.cjs", "javascript"},
		{"test.h", "c"},
		{"test.hpp", "cpp"},
		{"test.cs", "csharp"},
		{"test.kt", "kotlin"},
		{"test.swift", "swift"},
		{"test.dart", "dart"},
		{"test.scala", "scala"},
		{"test.clj", "clojure"},
		{"test.ml", "ocaml"},
		{"test.hs", "haskell"},
		{"test.elm", "elm"},
		{"test.ex", "elixir"},
		{"test.erl", "erlang"},
		{"test.html", "unknown"}, // Not supported
		{"test.css", "unknown"},  // Not supported
		{"test.json", "unknown"}, // Not supported
		{"test.unknown", "unknown"},
		{"no_extension", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Create the file
			err := os.WriteFile(filePath, []byte("test content"), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			language := walker.detectLanguage(filePath)
			if language != tt.expected {
				t.Errorf("detectLanguage(%s) = %s, expected %s", tt.filename, language, tt.expected)
			}
		})
	}
}

func TestFileWalker_FastScan(t *testing.T) {
	tempDir := t.TempDir()
	walker := NewFileWalker()

	// Create test files
	files := []string{
		"test1.go",
		"test2.js",
		"test3.py",
		"subdir/test4.go",
		"subdir/test5.txt",
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		dir := filepath.Dir(filePath)

		// Create directory if needed
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create file
		err = os.WriteFile(filePath, []byte("test content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go", "*.js"},
		MaxFiles: 10,
	}

	ctx := context.Background()
	results, err := walker.FastScan(ctx, scope)
	if err != nil {
		t.Fatalf("FastScan failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("FastScan should return some results")
	}

	// Verify results contain expected files
	found := make(map[string]bool)
	for _, filePath := range results {
		found[filepath.Base(filePath)] = true
	}

	expectedFiles := []string{"test1.go", "test2.js", "test4.go"}
	for _, expected := range expectedFiles {
		if !found[expected] {
			t.Errorf("Expected file %s not found in FastScan results", expected)
		}
	}
}

func TestFileWalker_FastScan_WithError(t *testing.T) {
	walker := NewFileWalker()

	// Test with non-existent directory
	scope := FileScope{
		Path: "/nonexistent/directory",
	}

	ctx := context.Background()
	_, err := walker.FastScan(ctx, scope)

	// Should handle error gracefully
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestFileWalker_GetLanguageStats(t *testing.T) {
	tempDir := t.TempDir()
	walker := NewFileWalker()

	// Create test files with different supported languages
	files := map[string]string{
		"main.go":          "package main\nfunc main() {}",
		"script.js":        "console.log('hello');",
		"app.py":           "print('hello')",
		"Helper.java":      "public class Helper {}",
		"utils.cpp":        "#include <iostream>",
		"test.rs":          "fn main() {}",
		"subdir/helper.go": "package helper",
		"subdir/utils.js":  "function utils() {}",
	}

	for file, content := range files {
		filePath := filepath.Join(tempDir, file)
		dir := filepath.Dir(filePath)

		// Create directory if needed
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create file
		err = os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	scope := FileScope{
		Path: tempDir,
	}

	ctx := context.Background()
	stats, err := walker.GetLanguageStats(ctx, scope)
	if err != nil {
		t.Fatalf("GetLanguageStats failed: %v", err)
	}

	if len(stats) == 0 {
		t.Error("GetLanguageStats should return some statistics")
	}

	// Verify expected languages are present
	expectedLanguages := []string{"go", "javascript", "python", "java", "cpp", "rust"}
	found := make(map[string]bool)

	for lang := range stats {
		found[lang] = true
	}

	for _, expected := range expectedLanguages {
		if !found[expected] {
			t.Errorf("Expected language '%s' not found in stats", expected)
		}
	}

	// Verify Go files count (should be 2: main.go and helper.go)
	if goCount, exists := stats["go"]; exists {
		if goCount != 2 {
			t.Errorf("Expected 2 Go files, got %d", goCount)
		}
	} else {
		t.Error("Go language stats not found")
	}
}

func TestFileWalker_GetLanguageStats_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	walker := NewFileWalker()

	scope := FileScope{
		Path: tempDir,
	}

	ctx := context.Background()
	stats, err := walker.GetLanguageStats(ctx, scope)
	if err != nil {
		t.Fatalf("GetLanguageStats failed on empty directory: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("Expected empty stats for empty directory, got %d entries", len(stats))
	}
}

func TestFileWalker_GetLanguageStats_WithFilters(t *testing.T) {
	tempDir := t.TempDir()
	walker := NewFileWalker()

	// Create test files
	files := map[string]string{
		"main.go":    "package main",
		"test.js":    "console.log('test');",
		"styles.css": "body {}",
		"ignore.go":  "package ignore", // This should be excluded
	}

	for file, content := range files {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	scope := FileScope{
		Path:    tempDir,
		Include: []string{"*.go", "*.js"},
		Exclude: []string{"ignore.*"},
	}

	ctx := context.Background()
	stats, err := walker.GetLanguageStats(ctx, scope)
	if err != nil {
		t.Fatalf("GetLanguageStats failed: %v", err)
	}

	// Should have Go and JavaScript, but not CSS
	if _, exists := stats["css"]; exists {
		t.Error("CSS should not be included due to include filter")
	}

	// Should have Go files but not the ignored one
	if goCount, exists := stats["go"]; exists {
		if goCount != 1 {
			t.Errorf("Expected 1 Go file (main.go), got %d", goCount)
		}
	} else {
		t.Error("Go language stats not found")
	}

	// Should have JavaScript
	if _, exists := stats["javascript"]; !exists {
		t.Error("JavaScript language stats not found")
	}
}
