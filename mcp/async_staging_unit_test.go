package mcp

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/termfx/morfx/models"
)

// newAsyncStagingForTest creates an async staging manager backed by an in-memory DB.
func newAsyncStagingForTest(t *testing.T) *AsyncStagingManager {
	t.Helper()
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false
	config.StagingTTL = time.Minute
	config.LogWriter = io.Discard
	asm := NewAsyncStagingManager(db, config, nil)
	t.Cleanup(func() {
		asm.Close()
	})
	return asm
}

func TestAsyncStagingManager_New(t *testing.T) {
	t.Parallel()
	asm := newAsyncStagingForTest(t)
	if asm == nil {
		t.Fatal("expected async staging manager to be created")
	}
}

func TestAsyncStagingManager_CreateStageAsync_QuickSuccess(t *testing.T) {
	t.Parallel()
	asm := newAsyncStagingForTest(t)
	stage := &models.Stage{Language: "go", Operation: "replace"}

	errCh := asm.CreateStageAsync(stage)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected enqueue error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("stage enqueue timed out")
	}

	if stage.ID == "" {
		t.Fatal("stage ID should be set after enqueue")
	}

	var stored models.Stage
	if err := asm.db.First(&stored, "id = ?", stage.ID).Error; err != nil {
		t.Fatalf("expected stage persisted: %v", err)
	}
}

func TestAsyncStagingManager_CreateStageAsyncWithContext_Cancelled(t *testing.T) {
	t.Parallel()
	asm := newAsyncStagingForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh := asm.CreateStageAsyncWithContext(ctx, &models.Stage{Language: "go"})
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected cancellation result")
	}
}
