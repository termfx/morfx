package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oxhq/morfx/core"
)

func TestApplyStageWritesFileAndMarksStageApplied(t *testing.T) {
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

	stage, err := e.CreateStage(context.Background(), StageCreateRequest{
		Path:        path,
		Language:    "go",
		Operation:   "replace",
		Original:    "package main\nfunc A() {}\n",
		Modified:    "package main\nfunc B() {}\n",
		BaseDigest:  calculateSHA256("package main\nfunc A() {}\n"),
		AfterDigest: calculateSHA256("package main\nfunc B() {}\n"),
		Confidence:  core.ConfidenceScore{Score: 0.9, Level: "high"},
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}

	result, err := e.ApplyStage(context.Background(), StageApplyRequest{ID: stage.ID})
	if err != nil {
		t.Fatalf("ApplyStage() error = %v", err)
	}
	if !result.Applied || result.Status != StageStatusApplied {
		t.Fatalf("expected applied result, got %+v", result)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(contents) != "package main\nfunc B() {}\n" {
		t.Fatalf("unexpected file contents: %q", string(contents))
	}
}

func TestApplyStageRejectsExpiredAndDigestMismatchStages(t *testing.T) {
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

	expired, err := e.CreateStage(context.Background(), StageCreateRequest{
		Path:      path,
		Language:  "go",
		Operation: "replace",
		Original:  "package main\nfunc A() {}\n",
		Modified:  "package main\nfunc B() {}\n",
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateStage() expired error = %v", err)
	}
	if _, err := e.ApplyStage(context.Background(), StageApplyRequest{ID: expired.ID}); err == nil {
		t.Fatal("expected expired stage apply to fail")
	}

	mismatch, err := e.CreateStage(context.Background(), StageCreateRequest{
		Path:       path,
		Language:   "go",
		Operation:  "replace",
		Original:   "package main\nfunc A() {}\n",
		Modified:   "package main\nfunc B() {}\n",
		BaseDigest: calculateSHA256("different"),
	})
	if err != nil {
		t.Fatalf("CreateStage() mismatch error = %v", err)
	}
	if _, err := e.ApplyStage(context.Background(), StageApplyRequest{ID: mismatch.ID}); err == nil {
		t.Fatal("expected digest mismatch apply to fail")
	}
}
