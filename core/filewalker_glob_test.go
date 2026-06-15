package core

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFileWalkerWalkMatchesDotSlashDoubleStarIncludePatterns(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "helper.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(helper.go) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "helper.txt"), []byte("helper"), 0o644); err != nil {
		t.Fatalf("WriteFile(helper.txt) error = %v", err)
	}

	fw := NewFileWalker()
	results, err := fw.Walk(context.Background(), FileScope{
		Path:    root,
		Include: []string{"./**/*.go"},
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	var found []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected walk result error: %v", result.Error)
		}
		found = append(found, filepath.Base(result.Path))
	}

	slices.Sort(found)
	expected := []string{"helper.go", "main.go"}
	if !slices.Equal(found, expected) {
		t.Fatalf("Walk() matched %v, expected %v", found, expected)
	}
}

func TestFileWalkerWalkHonorsDotSlashExcludePatterns(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "vendor"), 0o755); err != nil {
		t.Fatalf("MkdirAll(vendor) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "vendor", "dep.go"), []byte("package vendor\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(dep.go) error = %v", err)
	}

	fw := NewFileWalker()
	results, err := fw.Walk(context.Background(), FileScope{
		Path:    root,
		Include: []string{"**/*.go"},
		Exclude: []string{"./vendor/**"},
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	var found []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected walk result error: %v", result.Error)
		}
		found = append(found, filepath.Base(result.Path))
	}

	slices.Sort(found)
	expected := []string{"main.go"}
	if !slices.Equal(found, expected) {
		t.Fatalf("Walk() matched %v, expected %v", found, expected)
	}
}
