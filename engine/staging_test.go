package engine

import (
	"context"
	"testing"
	"time"

	"github.com/oxhq/morfx/core"
)

func TestStageStoreCreateGetListAndUpdate(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStageStore(dir)

	stage, err := store.Create(context.Background(), StageCreateRequest{
		Path:        "main.go",
		Language:    "go",
		Operation:   "replace",
		Original:    "old",
		Modified:    "new",
		Diff:        "@@",
		BaseDigest:  "abc",
		AfterDigest: "def",
		Confidence: core.ConfidenceScore{
			Score: 0.92,
			Level: "high",
		},
		Metadata: map[string]any{
			"session_id": "sess-1",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if stage.Status != StageStatusPending {
		t.Fatalf("expected pending status, got %s", stage.Status)
	}

	got, err := store.Get(context.Background(), stage.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Metadata["session_id"] != "sess-1" {
		t.Fatalf("expected metadata to round-trip")
	}

	list, err := store.List(context.Background(), StageFilter{Status: StageStatusPending})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != stage.ID {
		t.Fatalf("expected one pending stage")
	}

	stage.Status = StageStatusApplied
	now := time.Now().UTC()
	stage.AppliedAt = &now
	if err := store.Update(context.Background(), stage); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := store.Get(context.Background(), stage.ID)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if updated.Status != StageStatusApplied {
		t.Fatalf("expected applied status, got %s", updated.Status)
	}
}
