package mcp

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// TestNewStagingManager verifies staging manager creation
func TestNewStagingManager(t *testing.T) {
	db := setupTestDB(t)

	config := Config{
		StagingTTL: 24 * time.Hour,
	}

	sm := NewStagingManager(db, config)

	if sm == nil {
		t.Fatal("StagingManager should not be nil")
	}

	if sm.db != db {
		t.Error("Database should be set correctly")
	}

	if sm.config.StagingTTL != config.StagingTTL {
		t.Error("Config should be set correctly")
	}
}

// TestCreateStage verifies stage creation functionality
func TestCreateStage(t *testing.T) {
	db := setupTestDB(t)
	config := Config{
		StagingTTL: 24 * time.Hour,
	}
	sm := NewStagingManager(db, config)

	tests := []struct {
		name        string
		stage       *models.Stage
		expectError bool
		description string
	}{
		{
			name: "valid_stage_creation",
			stage: &models.Stage{
				Language:  "go",
				Original:  "package main\nfunc main() {}",
				Modified:  "package main\nfunc newMain() {}",
				Operation: "replace",
			},
			expectError: false,
			description: "Valid stage creation should succeed",
		},
		{
			name: "stage_with_predefined_id",
			stage: &models.Stage{
				ID:        "custom-stage-id",
				Language:  "javascript",
				Original:  "function test() {}",
				Modified:  "function newTest() {}",
				Operation: "replace",
			},
			expectError: false,
			description: "Stage with predefined ID should be preserved",
		},
		{
			name: "minimal_stage",
			stage: &models.Stage{
				Language:  "python",
				Operation: "delete",
			},
			expectError: false,
			description: "Minimal stage should be created successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.stage.ID
			err := sm.CreateStage(tt.stage)

			if tt.expectError && err == nil {
				t.Fatalf("Expected error for %s, but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.description, err)
			}

			if !tt.expectError {
				// Verify stage was created properly
				if tt.stage.ID == "" {
					t.Error("Stage ID should be generated if empty")
				}

				if originalID != "" && tt.stage.ID != originalID {
					t.Error("Predefined ID should be preserved")
				}

				if tt.stage.Status != "pending" {
					t.Errorf("Status should be 'pending', got %s", tt.stage.Status)
				}

				if tt.stage.ExpiresAt.IsZero() {
					t.Error("ExpiresAt should be set")
				}

				// Verify it exists in database
				var retrieved models.Stage
				err := db.First(&retrieved, "id = ?", tt.stage.ID).Error
				if err != nil {
					t.Errorf("Failed to retrieve created stage: %v", err)
				}
			}
		})
	}
} // TestGetStage verifies stage retrieval functionality
func TestGetStage(t *testing.T) {
	db := setupTestDB(t)
	config := Config{StagingTTL: 24 * time.Hour}
	sm := NewStagingManager(db, config)

	// Create a test stage
	stage := &models.Stage{
		ID:        "test-stage-get",
		Language:  "go",
		Original:  "original code",
		Modified:  "modified code",
		Operation: "replace",
	}

	err := sm.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	tests := []struct {
		name        string
		stageID     string
		expectError bool
		description string
	}{
		{
			name:        "existing_stage",
			stageID:     "test-stage-get",
			expectError: false,
			description: "Retrieving existing stage should succeed",
		},
		{
			name:        "nonexistent_stage",
			stageID:     "nonexistent-stage-id",
			expectError: true,
			description: "Retrieving nonexistent stage should fail",
		},
		{
			name:        "empty_id",
			stageID:     "",
			expectError: true,
			description: "Empty stage ID should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := sm.GetStage(tt.stageID)

			if tt.expectError && err == nil {
				t.Fatalf("Expected error for %s, but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.description, err)
			}

			if !tt.expectError && retrieved != nil {
				if retrieved.ID != tt.stageID {
					t.Errorf("Retrieved stage ID mismatch: got %s, want %s",
						retrieved.ID, tt.stageID)
				}

				if retrieved.Language != "go" {
					t.Errorf("Retrieved stage language mismatch: got %s, want go",
						retrieved.Language)
				}
			}
		})
	}
}

// TestListPendingStages verifies listing pending stages
func TestListPendingStages(t *testing.T) {
	db := setupTestDB(t)
	config := Config{StagingTTL: 24 * time.Hour}
	sm := NewStagingManager(db, config)

	// Create multiple test stages with different statuses
	stages := []*models.Stage{
		{
			ID:        "pending-1",
			SessionID: "test-session",
			Language:  "go",
			Status:    "pending",
			Operation: "replace",
		},
		{
			ID:        "pending-2",
			SessionID: "test-session",
			Language:  "javascript",
			Status:    "pending",
			Operation: "delete",
		},
		{
			ID:        "applied-1",
			SessionID: "test-session",
			Language:  "python",
			Status:    "applied",
			Operation: "replace",
		},
	}

	for _, stage := range stages {
		err := sm.CreateStage(stage)
		if err != nil {
			t.Fatalf("Failed to create test stage %s: %v", stage.ID, err)
		}
	}

	// Manually set one to applied status to test filtering
	err := db.Model(&models.Stage{}).Where("id = ?", "applied-1").Update("status", "applied").Error
	if err != nil {
		t.Fatalf("Failed to update stage status: %v", err)
	}

	pendingStages, err := sm.ListPendingStages("test-session")
	if err != nil {
		t.Fatalf("Failed to list pending stages: %v", err)
	}

	t.Logf("Found %d pending stages", len(pendingStages))
	for i, stage := range pendingStages {
		t.Logf("Stage %d: ID=%s, SessionID=%s, Status=%s",
			i, stage.ID, stage.SessionID, stage.Status)
	}

	// Should have 2 pending stages for the session (applied-1 was changed to applied status)
	if len(pendingStages) != 2 {
		t.Errorf("Expected 2 pending stages for test-session, got %d", len(pendingStages))
	}

	// Verify all returned stages are pending and belong to the correct session
	for _, stage := range pendingStages {
		if stage.Status != "pending" {
			t.Errorf("Expected pending status, got %s for stage %s",
				stage.Status, stage.ID)
		}
		if stage.SessionID != "test-session" {
			t.Errorf("Expected session 'test-session', got %s for stage %s",
				stage.SessionID, stage.ID)
		}
	}
} // TestCleanupExpiredStages verifies expired stage cleanup
func TestCleanupExpiredStages(t *testing.T) {
	db := setupTestDB(t)
	config := Config{StagingTTL: 1 * time.Second} // Very short TTL for testing
	sm := NewStagingManager(db, config)

	// Create a stage that will expire quickly
	stage := &models.Stage{
		ID:        "expire-test",
		Language:  "go",
		Operation: "replace",
	}

	err := sm.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Run cleanup
	err = sm.CleanupExpiredStages()
	if err != nil {
		t.Fatalf("Failed to cleanup expired stages: %v", err)
	}

	// Verify stage was marked as expired, not deleted
	retrieved, err := sm.GetStage("expire-test")
	if err != nil {
		t.Fatalf("Failed to retrieve stage after cleanup: %v", err)
	}

	if retrieved.Status != "expired" {
		t.Errorf("Expected status 'expired', got %s", retrieved.Status)
	}
}

// Helper function to setup test database
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate both Stage and Apply models
	err = db.AutoMigrate(&models.Stage{}, &models.Apply{}, &models.Session{})
	if err != nil {
		t.Fatalf("Failed to migrate models: %v", err)
	}

	return db
}

// TestApplyStage verifies stage application functionality
func TestApplyStage(t *testing.T) {
	db := setupTestDB(t)
	config := Config{StagingTTL: 24 * time.Hour}
	sm := NewStagingManager(db, config)

	// Create a test stage
	stage := &models.Stage{
		ID:        "apply-test",
		Language:  "go",
		Original:  "original code",
		Modified:  "modified code",
		Operation: "replace",
		Status:    "pending",
	}

	err := sm.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	tests := []struct {
		name        string
		stageID     string
		expectError bool
		description string
	}{
		{
			name:        "apply_existing_stage",
			stageID:     "apply-test",
			expectError: false,
			description: "Applying existing stage should succeed",
		},
		{
			name:        "apply_nonexistent_stage",
			stageID:     "nonexistent-apply-test",
			expectError: true,
			description: "Applying nonexistent stage should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sm.ApplyStage(tt.stageID, false) // autoApplied = false

			if tt.expectError && err == nil {
				t.Fatalf("Expected error for %s, but got none", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.description, err)
			}

			// For successful application, verify stage status changed
			if !tt.expectError {
				updated, getErr := sm.GetStage(tt.stageID)
				if getErr != nil {
					t.Fatalf("Failed to get updated stage: %v", getErr)
				}

				if updated.Status != "applied" {
					t.Errorf("Expected status 'applied', got %s", updated.Status)
				}
			}
		})
	}
}

// Benchmark tests for performance verification
func BenchmarkGetStage(b *testing.B) {
	db := setupTestDB(&testing.T{})
	config := Config{StagingTTL: 24 * time.Hour}
	sm := NewStagingManager(db, config)

	// Create a test stage
	stage := &models.Stage{
		ID:        "bench-get-test",
		Language:  "go",
		Original:  "original code",
		Modified:  "modified code",
		Operation: "replace",
	}
	sm.CreateStage(stage)

	for b.Loop() {
		sm.GetStage("bench-get-test")
	}
}
