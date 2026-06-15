package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestEngineFileTransformStageDoesNotWriteButReturnsStageID(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := New(Config{
		AllowedRoots:  []string{root},
		WriteMode:     WriteModeStage,
		EnableStaging: true,
		StageDir:      filepath.Join(root, ".morfx-stages"),
	})
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
	if res.Applied {
		t.Fatalf("expected applied=false in stage mode")
	}
	if res.StageID == "" {
		t.Fatalf("expected stage id")
	}

	after, _ := os.ReadFile(path)
	if string(after) != "package main\nfunc A() {}\n" {
		t.Fatalf("expected file unchanged, got %q", string(after))
	}
}

func TestEngineFileTransformStageReturnsLifecycleState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := New(Config{
		AllowedRoots:  []string{root},
		WriteMode:     WriteModeStage,
		EnableStaging: true,
		StageDir:      filepath.Join(root, ".morfx-stages"),
	})
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
	if res.StageID == "" {
		t.Fatal("expected stage id")
	}
	if res.StageStatus != StageStatusPending {
		t.Fatalf("expected pending status, got %s", res.StageStatus)
	}
}
