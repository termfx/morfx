package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
)

func TestApplyStagesUsesEngineLifecycle(t *testing.T) {
	t.Run("applies specific stage", func(t *testing.T) {
		root := t.TempDir()
		lifecycle := newApplyEngine(t, root)
		stage := createPendingStage(t, lifecycle, root, "specific", "session-a", time.Now())

		applied, err := applyStages(context.Background(), lifecycle, &applyRequest{ID: stage.ID}, "single")
		if err != nil {
			t.Fatalf("applyStages() error = %v", err)
		}
		if len(applied) != 1 || applied[0] != stage.ID {
			t.Fatalf("expected stage %q to be applied, got %v", stage.ID, applied)
		}

		stored, err := lifecycle.GetStage(context.Background(), stage.ID)
		if err != nil {
			t.Fatalf("GetStage() error = %v", err)
		}
		if stored.Status != engine.StageStatusApplied {
			t.Fatalf("expected applied status, got %s", stored.Status)
		}
	})

	t.Run("applies all pending stages for session", func(t *testing.T) {
		root := t.TempDir()
		lifecycle := newApplyEngine(t, root)
		first := createPendingStage(t, lifecycle, root, "one", "session-a", time.Now().Add(-2*time.Minute))
		second := createPendingStage(t, lifecycle, root, "two", "session-a", time.Now().Add(-time.Minute))
		createPendingStage(t, lifecycle, root, "other", "session-b", time.Now())

		applied, err := applyStages(context.Background(), lifecycle, &applyRequest{
			All:       true,
			SessionID: "session-a",
		}, "all")
		if err != nil {
			t.Fatalf("applyStages() error = %v", err)
		}
		if len(applied) != 2 {
			t.Fatalf("expected 2 applied stages, got %v", applied)
		}
		if applied[0] != second.ID || applied[1] != first.ID {
			t.Fatalf("expected newest-first apply order, got %v", applied)
		}
	})

	t.Run("applies latest pending stage by default", func(t *testing.T) {
		root := t.TempDir()
		lifecycle := newApplyEngine(t, root)
		createPendingStage(t, lifecycle, root, "older", "session-a", time.Now().Add(-2*time.Minute))
		latest := createPendingStage(t, lifecycle, root, "latest", "session-a", time.Now().Add(-time.Minute))

		applied, err := applyStages(context.Background(), lifecycle, &applyRequest{}, "latest")
		if err != nil {
			t.Fatalf("applyStages() error = %v", err)
		}
		if len(applied) != 1 || applied[0] != latest.ID {
			t.Fatalf("expected latest stage %q, got %v", latest.ID, applied)
		}
	})
}

func TestListPendingStageIDsFiltersBySession(t *testing.T) {
	root := t.TempDir()
	lifecycle := newApplyEngine(t, root)
	oldest := createPendingStage(t, lifecycle, root, "oldest", "session-a", time.Now().Add(-3*time.Minute))
	newest := createPendingStage(t, lifecycle, root, "newest", "session-a", time.Now().Add(-time.Minute))
	createPendingStage(t, lifecycle, root, "other", "session-b", time.Now())

	ids, err := listPendingStageIDs(context.Background(), lifecycle, "session-a")
	if err != nil {
		t.Fatalf("listPendingStageIDs() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 stage ids, got %v", ids)
	}
	if ids[0] != newest.ID || ids[1] != oldest.ID {
		t.Fatalf("expected newest-first session ids, got %v", ids)
	}
}

func newApplyEngine(t *testing.T, root string) *engine.Engine {
	t.Helper()

	lifecycle, err := engine.New(engine.Config{
		AllowedRoots:  []string{root},
		WriteMode:     engine.WriteModeStage,
		EnableStaging: true,
		StageDir:      filepath.Join(root, ".morfx-stages"),
	})
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}
	return lifecycle
}

func createPendingStage(t *testing.T, lifecycle *engine.Engine, root string, name string, sessionID string, createdAt time.Time) engine.Stage {
	t.Helper()

	path := filepath.Join(root, name+".go")
	original := "package main\nfunc Old() {}\n"
	modified := "package main\nfunc New() {}\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	stage, err := lifecycle.CreateStage(context.Background(), engine.StageCreateRequest{
		Path:        path,
		Language:    "go",
		Operation:   "replace",
		Original:    original,
		Modified:    modified,
		BaseDigest:  sha256Of(original),
		AfterDigest: sha256Of(modified),
		Confidence:  core.ConfidenceScore{Score: 0.9, Level: "high"},
		Metadata: map[string]any{
			"session_id": sessionID,
		},
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}
	stage.CreatedAt = createdAt.UTC()
	if err := updateStageForTest(context.Background(), lifecycle, stage); err != nil {
		t.Fatalf("updateStageForTest() error = %v", err)
	}
	return stage
}

func sha256Of(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func updateStageForTest(ctx context.Context, lifecycle *engine.Engine, stage engine.Stage) error {
	store, err := engine.NewFileStageStore(filepath.Join(filepath.Dir(stage.Path), ".morfx-stages")).Get(ctx, stage.ID)
	if err == nil && store.ID == stage.ID {
		return engine.NewFileStageStore(filepath.Join(filepath.Dir(stage.Path), ".morfx-stages")).Update(ctx, stage)
	}
	stageDir := filepath.Join(filepath.Dir(stage.Path), ".morfx-stages")
	return engine.NewFileStageStore(stageDir).Update(ctx, stage)
}
