package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestEngineFileTransformApplyWritesAtomically(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := New(Config{AllowedRoots: []string{root}, WriteMode: WriteModeApply})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	res, err := e.FileTransform(context.Background(), FileTransformRequest{
		Language: "go",
		Path:     path,
		Op: core.TransformOp{
			Method:      "replace",
			Target:      core.AgentQuery{Type: "function", Name: "A"},
			Replacement: "func B() {}",
		},
	})
	if err != nil {
		t.Fatalf("FileTransform() error = %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected applied=true")
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(updated) != "package main\nfunc B() {}\n" {
		t.Fatalf("unexpected file content after apply: %q", string(updated))
	}
}
