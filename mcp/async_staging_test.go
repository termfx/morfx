package mcp

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

func setupAsyncStagingDB(t *testing.T) *gorm.DB {
	// Use a temporary file instead of :memory: for concurrent access
	tempDB := t.TempDir() + "/test.db"
	db, err := gorm.Open(sqlite.Open(tempDB+"?cache=shared&mode=rwc"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.Session{}, &models.Stage{}, &models.Apply{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestNewAsyncStagingManager(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()

	asm := NewAsyncStagingManager(db, config)

	if asm == nil {
		t.Fatal("AsyncStagingManager should not be nil")
	}

	if asm.StagingManager == nil {
		t.Error("StagingManager should be initialized")
	}

	if asm.workers != 10 {
		t.Errorf("Expected 10 workers, got %d", asm.workers)
	}

	if asm.stageChan == nil {
		t.Error("Stage channel should be initialized")
	}

	if asm.resultChan == nil {
		t.Error("Result channel should be initialized")
	}

	if asm.ctx == nil {
		t.Error("Context should be initialized")
	}

	if asm.cancel == nil {
		t.Error("Cancel function should be initialized")
	}

	// Cleanup
	asm.Close()
}

func TestAsyncStagingManager_CreateStageAsync_Success(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false // Reduce noise in tests

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	stage := &models.Stage{
		ID:              "async-test-stage",
		SessionID:       "test-session",
		Operation:       "replace",
		TargetType:      "function",
		TargetName:      "testFunc",
		ConfidenceLevel: "high",
		ConfidenceScore: 0.95,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
	}

	// Create stage asynchronously
	callback := asm.CreateStageAsync(stage)

	// Wait for result
	select {
	case err := <-callback:
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for async stage creation")
	}

	// Verify stage was created
	retrieved, err := asm.GetStage("async-test-stage")
	if err != nil {
		t.Fatalf("Failed to retrieve created stage: %v", err)
	}

	if retrieved.ID != stage.ID {
		t.Errorf("Expected stage ID %s, got %s", stage.ID, retrieved.ID)
	}
}

func TestAsyncStagingManager_CreateStageAsync_QueueFull(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Fill the queue by sending many requests quickly
	var callbacks []<-chan error
	numRequests := 150 // More than the channel buffer size (100)

	for range numRequests {
		stage := &models.Stage{
			ID:        generateID("stage"),
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		callback := asm.CreateStageAsync(stage)
		callbacks = append(callbacks, callback)
	}

	// Wait for all to complete (should fallback to sync for some)
	successCount := 0
	for _, callback := range callbacks {
		select {
		case err := <-callback:
			if err == nil {
				successCount++
			}
		case <-time.After(10 * time.Second):
			t.Error("Timeout waiting for async stage creation")
		}
	}

	// Should have created at least some stages successfully
	if successCount == 0 {
		t.Error("Expected at least some stages to be created successfully")
	}

	t.Logf("Successfully created %d/%d stages", successCount, numRequests)
}

func TestAsyncStagingManager_BatchCreateStages(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Create multiple stages for batch processing
	stages := []*models.Stage{
		{
			ID:        "batch-stage-1",
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "batch-stage-2",
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "batch-stage-3",
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}

	// Process batch
	results := asm.BatchCreateStages(stages)

	if len(results) != len(stages) {
		t.Errorf("Expected %d results, got %d", len(stages), len(results))
	}

	// Check all were successful
	for i, err := range results {
		if err != nil {
			t.Errorf("Stage %d failed: %v", i, err)
		}
	}

	// Verify all stages were created
	for _, stage := range stages {
		retrieved, err := asm.GetStage(stage.ID)
		if err != nil {
			t.Errorf("Failed to retrieve stage %s: %v", stage.ID, err)
		} else if retrieved.ID != stage.ID {
			t.Errorf("Stage ID mismatch: expected %s, got %s", stage.ID, retrieved.ID)
		}
	}
}

func TestAsyncStagingManager_BatchCreateStages_WithErrors(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Create a unique ID base for this test to avoid conflicts
	baseID := fmt.Sprintf("duplicate-stage-%d", time.Now().UnixNano())

	// Create the first stage successfully
	firstStage := &models.Stage{
		ID:        baseID,
		SessionID: "test-session",
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Create first stage directly to ensure it exists
	err := asm.CreateStage(firstStage)
	if err != nil {
		t.Fatalf("Failed to create first stage: %v", err)
	}

	// Now try to create stages with duplicate IDs
	stages := []*models.Stage{
		{
			ID:        baseID + "-new", // Different ID, should succeed
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        baseID, // Same ID as pre-existing stage, should fail
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}

	// Process batch
	results := asm.BatchCreateStages(stages)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// First should succeed (new ID), second should fail (duplicate)
	if results[0] != nil {
		t.Errorf("First stage should succeed, got error: %v", results[0])
	}

	if results[1] == nil {
		t.Error("Second stage should fail due to duplicate ID")
	}
}

func TestAsyncStagingManager_WorkerPool(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Test concurrent stage creation
	numConcurrent := 20
	var wg sync.WaitGroup
	errors := make([]error, numConcurrent)

	for i := range numConcurrent {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			stage := &models.Stage{
				ID:        generateID("stage"),
				SessionID: "test-session",
				Status:    "pending",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}

			callback := asm.CreateStageAsync(stage)
			select {
			case err := <-callback:
				errors[index] = err
			case <-time.After(5 * time.Second):
				errors[index] = &MCPError{Code: InternalError, Message: "timeout"}
			}
		}(i)
	}

	wg.Wait()

	// Check results
	successCount := 0
	for i, err := range errors {
		if err == nil {
			successCount++
		} else {
			t.Logf("Worker %d failed: %v", i, err)
		}
	}

	// Should have high success rate with worker pool
	if successCount < numConcurrent/2 {
		t.Errorf("Low success rate: %d/%d", successCount, numConcurrent)
	}

	t.Logf("Worker pool test: %d/%d successful", successCount, numConcurrent)
}

func TestAsyncStagingManager_Close(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)

	// Create a stage to verify workers are running
	stage := &models.Stage{
		ID:        "close-test-stage",
		SessionID: "test-session",
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	callback := asm.CreateStageAsync(stage)

	// Wait for stage to be processed
	select {
	case err := <-callback:
		if err != nil {
			t.Fatalf("Stage creation failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for stage creation")
	}

	// Close should not panic and should clean up resources
	asm.Close()

	// Verify context is cancelled
	select {
	case <-asm.ctx.Done():
		// Expected - context should be cancelled
	default:
		t.Error("Context should be cancelled after Close()")
	}

	// Creating stages after close should still work (fallback to sync)
	afterCloseStage := &models.Stage{
		ID:        "after-close-stage",
		SessionID: "test-session",
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	afterCallback := asm.CreateStageAsync(afterCloseStage)

	select {
	case err := <-afterCallback:
		// Should fallback to sync processing
		if err != nil {
			t.Logf("After close creation failed (expected): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Should receive callback even after close")
	}
}

func TestAsyncStagingManager_StageWorker_ContextCancellation(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)

	// Cancel context immediately
	asm.cancel()

	// Give workers time to exit
	time.Sleep(100 * time.Millisecond)

	// Try to create a stage - should use fallback
	stage := &models.Stage{
		ID:        "cancelled-context-stage",
		SessionID: "test-session",
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	callback := asm.CreateStageAsync(stage)

	// Should get callback (via fallback mechanism)
	select {
	case <-callback:
		// Expected - fallback should still work
	case <-time.After(2 * time.Second):
		t.Error("Should receive callback from fallback mechanism")
	}

	asm.Close()
}

func TestAsyncStagingManager_ResultCollector(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = true // Enable debug to trigger metrics logging

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Create multiple stages to trigger result collection
	stages := []*models.Stage{
		{
			ID:        "metrics-stage-1",
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "metrics-stage-2",
			SessionID: "test-session",
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}

	var callbacks []<-chan error
	for _, stage := range stages {
		callback := asm.CreateStageAsync(stage)
		callbacks = append(callbacks, callback)
	}

	// Wait for all stages to complete
	for _, callback := range callbacks {
		select {
		case err := <-callback:
			if err != nil {
				t.Errorf("Stage creation failed: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for stage creation")
		}
	}

	// Result collector should be processing metrics
	// We can't easily test the metrics output without more complex setup,
	// but we verify the mechanism doesn't crash
	time.Sleep(100 * time.Millisecond)
}

func TestAsyncStagingManager_DebugLog(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = true

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Test debugLog function (should not panic)
	asm.debugLog("Test debug message: %s", "test")

	// Test with disabled debug
	config.Debug = false
	asm.config = config
	asm.debugLog("This should not log: %s", "test")
}

func TestAsyncStagingManager_Integration(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Test full integration: create, verify, apply
	stage := &models.Stage{
		ID:              "integration-stage",
		SessionID:       "test-session",
		Operation:       "replace",
		TargetType:      "function",
		TargetName:      "integrationTest",
		ConfidenceLevel: "high",
		ConfidenceScore: 0.90,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
	}

	// 1. Create stage asynchronously
	callback := asm.CreateStageAsync(stage)

	select {
	case err := <-callback:
		if err != nil {
			t.Fatalf("Stage creation failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for stage creation")
	}

	// 2. Verify stage exists
	retrieved, err := asm.GetStage("integration-stage")
	if err != nil {
		t.Fatalf("Failed to retrieve stage: %v", err)
	}

	if retrieved.Operation != "replace" {
		t.Errorf("Expected operation 'replace', got '%s'", retrieved.Operation)
	}

	// 3. List pending stages
	pending, err := asm.ListPendingStages("test-session")
	if err != nil {
		t.Fatalf("Failed to list pending stages: %v", err)
	}

	found := false
	for _, s := range pending {
		if s.ID == "integration-stage" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Stage should appear in pending list")
	}

	// 4. Apply stage
	apply, err := asm.ApplyStage("integration-stage", false)
	if err != nil {
		t.Fatalf("Failed to apply stage: %v", err)
	}

	if apply.StageID != "integration-stage" {
		t.Errorf("Expected apply stage ID 'integration-stage', got '%s'", apply.StageID)
	}
}

func TestAsyncStagingManager_ErrorHandling(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()
	config.Debug = false

	asm := NewAsyncStagingManager(db, config)
	defer asm.Close()

	// Test with nil stage
	var nilStage *models.Stage
	callback := asm.CreateStageAsync(nilStage)

	select {
	case err := <-callback:
		if err == nil {
			t.Error("Expected error for nil stage")
		}
	case <-time.After(2 * time.Second):
		t.Error("Should receive error callback")
	}
}

func TestAsyncStagingManager_MultipleClose(t *testing.T) {
	db := setupAsyncStagingDB(t)
	config := DefaultConfig()

	asm := NewAsyncStagingManager(db, config)

	// Multiple close calls should be safe
	asm.Close()
	asm.Close()
	asm.Close()

	// Should not panic
}
