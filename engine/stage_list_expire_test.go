package engine

import (
	"context"
	"testing"
	"time"
)

func TestExpireStagesMarksOnlyPendingExpiredStages(t *testing.T) {
	dir := t.TempDir()
	e, err := New(Config{
		WriteMode:     WriteModeStage,
		EnableStaging: true,
		StageDir:      dir,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	expired, err := e.CreateStage(context.Background(), StageCreateRequest{
		Path:      "one.go",
		Language:  "go",
		Operation: "replace",
		Original:  "a",
		Modified:  "b",
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateStage() expired fixture error = %v", err)
	}

	_, err = e.CreateStage(context.Background(), StageCreateRequest{
		Path:      "two.go",
		Language:  "go",
		Operation: "replace",
		Original:  "a",
		Modified:  "b",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateStage() live fixture error = %v", err)
	}

	count, err := e.ExpireStages(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ExpireStages() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one expired stage, got %d", count)
	}

	stage, err := e.GetStage(context.Background(), expired.ID)
	if err != nil {
		t.Fatalf("GetStage() error = %v", err)
	}
	if stage.Status != StageStatusExpired {
		t.Fatalf("expected expired status, got %s", stage.Status)
	}
}
