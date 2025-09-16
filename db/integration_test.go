package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// TestDatabaseIntegration tests the full database workflow
func TestDatabaseIntegration(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_test.db")

	// Test Connect
	db, err := Connect(dbPath, true) // Enable debug mode
	require.NoError(t, err)
	require.NotNil(t, db)

	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Run integration workflow
	t.Run("complete workflow", func(t *testing.T) {
		testCompleteWorkflow(t, db)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		testConcurrentOperations(t, db)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		testTransactionRollback(t, db)
	})

	t.Run("bulk operations", func(t *testing.T) {
		testBulkOperations(t, db)
	})
}

func testCompleteWorkflow(t *testing.T, db *gorm.DB) {
	// Step 1: Create a session
	session := &models.Session{
		ID:           "integration-session-001",
		StagesCount:  0,
		AppliesCount: 0,
		ClientInfo:   datatypes.JSON(`{"version": "1.0.0", "platform": "test"}`),
	}

	err := db.Create(session).Error
	require.NoError(t, err)

	// Step 2: Create multiple stages
	stages := []*models.Stage{
		{
			ID:              "stage-001",
			SessionID:       session.ID,
			Language:        "go",
			Operation:       "replace",
			TargetType:      "function",
			TargetName:      "TestFunction",
			Original:        "func TestFunction() {}",
			Modified:        "func TestFunction() { // modified }",
			ConfidenceScore: 0.95,
			ConfidenceLevel: "high",
			Status:          "pending",
			ExpiresAt:       time.Now().Add(24 * time.Hour),
		},
		{
			ID:              "stage-002",
			SessionID:       session.ID,
			Language:        "javascript",
			Operation:       "insert",
			TargetType:      "class",
			TargetName:      "TestClass",
			Content:         "class TestClass {}",
			ConfidenceScore: 0.85,
			ConfidenceLevel: "medium",
			Status:          "pending",
			ExpiresAt:       time.Now().Add(24 * time.Hour),
		},
	}

	for _, stage := range stages {
		err = db.Create(stage).Error
		require.NoError(t, err)
	}

	// Step 3: Apply the first stage
	apply := &models.Apply{
		ID:          "apply-001",
		StageID:     stages[0].ID,
		BaseDigest:  "original-hash",
		AfterDigest: "modified-hash",
		AutoApplied: false,
		AppliedBy:   "test-user",
	}

	err = db.Create(apply).Error
	require.NoError(t, err)

	// Update stage status
	err = db.Model(stages[0]).Update("status", "applied").Error
	require.NoError(t, err)

	// Set AppliedAt timestamp
	now := time.Now()
	err = db.Model(stages[0]).Update("applied_at", now).Error
	require.NoError(t, err)

	// Step 4: Update session counts
	err = db.Model(session).Updates(map[string]any{
		"stages_count":  2,
		"applies_count": 1,
	}).Error
	require.NoError(t, err)

	// Step 5: Verify the complete state
	var retrievedSession models.Session
	err = db.Where("id = ?", session.ID).First(&retrievedSession).Error
	require.NoError(t, err)
	assert.Equal(t, 2, retrievedSession.StagesCount)
	assert.Equal(t, 1, retrievedSession.AppliesCount)

	// Verify stage with apply relationship
	var stageWithApply models.Stage
	err = db.Preload("Apply").Where("id = ?", stages[0].ID).First(&stageWithApply).Error
	require.NoError(t, err)
	assert.Equal(t, "applied", stageWithApply.Status)
	assert.NotNil(t, stageWithApply.Apply)
	assert.Equal(t, apply.ID, stageWithApply.Apply.ID)

	// Step 6: Test revert operation
	revertTime := time.Now()
	err = db.Model(apply).Updates(map[string]any{
		"reverted":    true,
		"reverted_by": "admin-user",
		"reverted_at": revertTime,
	}).Error
	require.NoError(t, err)

	// Verify revert
	var revertedApply models.Apply
	err = db.Where("id = ?", apply.ID).First(&revertedApply).Error
	require.NoError(t, err)
	assert.True(t, revertedApply.Reverted)
	assert.Equal(t, "admin-user", revertedApply.RevertedBy)
	assert.NotNil(t, revertedApply.RevertedAt)

	// Step 7: End the session
	endTime := time.Now()
	err = db.Model(session).Update("ended_at", endTime).Error
	require.NoError(t, err)

	var endedSession models.Session
	err = db.Where("id = ?", session.ID).First(&endedSession).Error
	require.NoError(t, err)
	assert.NotNil(t, endedSession.EndedAt)
}

func testConcurrentOperations(t *testing.T, db *gorm.DB) {
	session := &models.Session{
		ID: "concurrent-session-001",
	}
	err := db.Create(session).Error
	require.NoError(t, err)

	// Create multiple stages concurrently (simulation)
	numGoroutines := 5
	results := make(chan error, numGoroutines)

	for i := range numGoroutines {
		go func(index int) {
			stage := &models.Stage{
				ID:        fmt.Sprintf("concurrent-stage-%03d", index),
				SessionID: session.ID,
				Language:  "go",
				Operation: "test",
				Status:    "pending",
			}
			results <- db.Create(stage).Error
		}(i)
	}

	// Collect results
	for range numGoroutines {
		err := <-results
		assert.NoError(t, err, "Concurrent stage creation should succeed")
	}

	// Verify all stages were created
	var count int64
	err = db.Model(&models.Stage{}).Where("session_id = ?", session.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines), count)
}

func testTransactionRollback(t *testing.T, db *gorm.DB) {
	session := &models.Session{
		ID: "transaction-session-001",
	}
	err := db.Create(session).Error
	require.NoError(t, err)

	// Test successful transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		stage := &models.Stage{
			ID:        "transaction-stage-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		if err := tx.Create(stage).Error; err != nil {
			return err
		}

		apply := &models.Apply{
			ID:      "transaction-apply-001",
			StageID: stage.ID,
		}
		return tx.Create(apply).Error
	})
	require.NoError(t, err)

	// Verify both records exist
	var stageCount, applyCount int64
	db.Model(&models.Stage{}).Where("session_id = ?", session.ID).Count(&stageCount)
	db.Model(&models.Apply{}).Where("stage_id = ?", "transaction-stage-001").Count(&applyCount)
	assert.Equal(t, int64(1), stageCount)
	assert.Equal(t, int64(1), applyCount)

	// Test failed transaction (should rollback)
	err = db.Transaction(func(tx *gorm.DB) error {
		stage := &models.Stage{
			ID:        "transaction-stage-002",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		if err := tx.Create(stage).Error; err != nil {
			return err
		}

		// This should fail due to duplicate StageID
		apply := &models.Apply{
			ID:      "transaction-apply-002",
			StageID: "transaction-stage-001", // Existing StageID
		}
		return tx.Create(apply).Error // This will fail due to unique constraint
	})
	assert.Error(t, err, "Transaction should fail and rollback")

	// Verify rollback - stage-002 should not exist
	var rollbackStageCount int64
	db.Model(&models.Stage{}).Where("id = ?", "transaction-stage-002").Count(&rollbackStageCount)
	assert.Equal(t, int64(0), rollbackStageCount, "Failed transaction should be rolled back")
}

func testBulkOperations(t *testing.T, db *gorm.DB) {
	session := &models.Session{
		ID: "bulk-session-001",
	}
	err := db.Create(session).Error
	require.NoError(t, err)

	// Bulk create stages
	numStages := 100
	stages := make([]*models.Stage, numStages)
	for i := range numStages {
		stages[i] = &models.Stage{
			ID:        fmt.Sprintf("bulk-stage-%03d", i),
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
	}

	// Use CreateInBatches for better performance
	err = db.CreateInBatches(stages, 20).Error
	require.NoError(t, err)

	// Verify count
	var count int64
	err = db.Model(&models.Stage{}).Where("session_id = ?", session.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(numStages), count)

	// Bulk update
	err = db.Model(&models.Stage{}).Where("session_id = ?", session.ID).Update("status", "bulk_updated").Error
	require.NoError(t, err)

	// Verify bulk update
	var updatedCount int64
	err = db.Model(&models.Stage{}).
		Where("session_id = ? AND status = ?", session.ID, "bulk_updated").
		Count(&updatedCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(numStages), updatedCount)

	// Bulk delete
	err = db.Where("session_id = ? AND status = ?", session.ID, "bulk_updated").Delete(&models.Stage{}).Error
	require.NoError(t, err)

	// Verify bulk delete
	var remainingCount int64
	err = db.Model(&models.Stage{}).Where("session_id = ?", session.ID).Count(&remainingCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), remainingCount)
}

// TestDatabasePerformance tests database performance characteristics
func TestDatabasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, err := Connect(":memory:", false)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	session := &models.Session{
		ID: "perf-session-001",
	}
	err = db.Create(session).Error
	require.NoError(t, err)

	t.Run("stage creation performance", func(t *testing.T) {
		numStages := 1000
		start := time.Now()

		stages := make([]*models.Stage, numStages)
		for i := range numStages {
			stages[i] = &models.Stage{
				ID:        fmt.Sprintf("perf-stage-%04d", i),
				SessionID: session.ID,
				Language:  "go",
				Operation: "test",
				Status:    "pending",
			}
		}

		err = db.CreateInBatches(stages, 50).Error
		require.NoError(t, err)

		duration := time.Since(start)
		t.Logf(
			"Created %d stages in %v (%.2f stages/second)",
			numStages,
			duration,
			float64(numStages)/duration.Seconds(),
		)

		// Should be reasonably fast
		assert.Less(t, duration, 5*time.Second, "Stage creation should be fast")
	})

	t.Run("query performance with indexes", func(t *testing.T) {
		start := time.Now()

		// Query by SessionID (indexed)
		var stages []models.Stage
		err = db.Where("session_id = ?", session.ID).Find(&stages).Error
		require.NoError(t, err)

		duration := time.Since(start)
		t.Logf("Queried %d stages by session_id in %v", len(stages), duration)

		// Should be very fast due to index
		assert.Less(t, duration, 100*time.Millisecond, "Indexed query should be very fast")
	})

	t.Run("complex query performance", func(t *testing.T) {
		start := time.Now()

		// Complex query with joins
		var results []struct {
			StageID   string
			SessionID string
			ApplyID   *string
		}

		err = db.Table("stages").
			Select("stages.id as stage_id, stages.session_id, applies.id as apply_id").
			Joins("LEFT JOIN applies ON stages.id = applies.stage_id").
			Where("stages.session_id = ?", session.ID).
			Scan(&results).Error
		require.NoError(t, err)

		duration := time.Since(start)
		t.Logf("Complex join query returned %d results in %v", len(results), duration)

		// Should still be reasonable
		assert.Less(t, duration, 500*time.Millisecond, "Complex query should be reasonable")
	})
}

// TestDatabaseRecovery tests database recovery scenarios
func TestDatabaseRecovery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "recovery_test.db")

	// Create initial database
	db1, err := Connect(dbPath, false)
	require.NoError(t, err)

	session := &models.Session{
		ID: "recovery-session-001",
	}
	err = db1.Create(session).Error
	require.NoError(t, err)

	stage := &models.Stage{
		ID:        "recovery-stage-001",
		SessionID: session.ID,
		Language:  "go",
		Operation: "test",
		Status:    "pending",
	}
	err = db1.Create(stage).Error
	require.NoError(t, err)

	// Close first connection
	sqlDB1, _ := db1.DB()
	sqlDB1.Close()

	// Reconnect to same database
	db2, err := Connect(dbPath, false)
	require.NoError(t, err)
	defer func() {
		sqlDB2, _ := db2.DB()
		if sqlDB2 != nil {
			sqlDB2.Close()
		}
	}()

	// Verify data persisted
	var retrievedSession models.Session
	err = db2.Where("id = ?", session.ID).First(&retrievedSession).Error
	assert.NoError(t, err)

	var retrievedStage models.Stage
	err = db2.Where("id = ?", stage.ID).First(&retrievedStage).Error
	assert.NoError(t, err)
	assert.Equal(t, stage.Language, retrievedStage.Language)
}

// TestDatabaseConstraintViolations tests various constraint violations
func TestDatabaseConstraintViolations(t *testing.T) {
	db, err := Connect(":memory:", false)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	session := &models.Session{
		ID: "constraint-session-001",
	}
	err = db.Create(session).Error
	require.NoError(t, err)

	t.Run("primary key violation", func(t *testing.T) {
		stage1 := &models.Stage{
			ID:        "duplicate-stage-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(stage1).Error
		require.NoError(t, err)

		// Try to create another stage with same ID
		stage2 := &models.Stage{
			ID:        "duplicate-stage-001", // Same ID
			SessionID: session.ID,
			Language:  "javascript",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(stage2).Error
		assert.Error(t, err, "Duplicate primary key should be rejected")
	})

	t.Run("foreign key violation", func(t *testing.T) {
		// Try to create stage with non-existent session
		// Note: Stage doesn't have FK constraint to Session in the current model
		stage := &models.Stage{
			ID:        "orphan-stage-001",
			SessionID: "non-existent-session",
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(stage).Error
		// This succeeds because there's no FK constraint from Stage to Session
		assert.NoError(t, err, "Stage creation without session succeeds (no FK constraint)")

		// Test the actual FK constraint that exists: Apply -> Stage
		invalidApply := &models.Apply{
			ID:      "invalid-apply-001",
			StageID: "non-existent-stage",
		}
		err = db.Create(invalidApply).Error
		assert.Error(t, err, "Apply with non-existent Stage should be rejected due to FK constraint")
	})

	t.Run("unique constraint violation", func(t *testing.T) {
		stage := &models.Stage{
			ID:        "unique-test-stage-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(stage).Error
		require.NoError(t, err)

		apply1 := &models.Apply{
			ID:      "unique-apply-001",
			StageID: stage.ID,
		}
		err = db.Create(apply1).Error
		require.NoError(t, err)

		// Try to create another apply for same stage
		apply2 := &models.Apply{
			ID:      "unique-apply-002",
			StageID: stage.ID, // Same StageID
		}
		err = db.Create(apply2).Error
		assert.Error(t, err, "Unique constraint violation should be rejected")
	})
}

// BenchmarkDatabaseOperations benchmarks common database operations
func BenchmarkDatabaseOperations(b *testing.B) {
	db, err := Connect(":memory:", false)
	require.NoError(b, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	session := &models.Session{
		ID: "benchmark-session-001",
	}
	err = db.Create(session).Error
	require.NoError(b, err)

	b.Run("stage creation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			stage := &models.Stage{
				ID:        fmt.Sprintf("bench-stage-%d", i),
				SessionID: session.ID,
				Language:  "go",
				Operation: "test",
				Status:    "pending",
			}
			err = db.Create(stage).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Create some test data for query benchmarks
	for i := range 1000 {
		stage := &models.Stage{
			ID:        fmt.Sprintf("query-bench-stage-%d", i),
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		db.Create(stage)
	}

	b.Run("stage query by session", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			var stages []models.Stage
			err = db.Where("session_id = ?", session.ID).Find(&stages).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("stage query by id", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			var stage models.Stage
			err = db.Where("id = ?", "query-bench-stage-500").First(&stage).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
