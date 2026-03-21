package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStageTableName(t *testing.T) {
	stage := Stage{}
	assert.Equal(t, "stages", stage.TableName())
}

func TestApplyTableName(t *testing.T) {
	apply := Apply{}
	assert.Equal(t, "applies", apply.TableName())
}

func TestSessionTableName(t *testing.T) {
	session := Session{}
	assert.Equal(t, "sessions", session.TableName())
}

func TestStageModel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tests := []struct {
		name          string
		stage         Stage
		expectedError bool
		errorContains string
	}{
		{
			name: "valid stage with minimal fields",
			stage: Stage{
				ID:        "stage-001",
				SessionID: "session-001",
				Language:  "go",
				Operation: "replace",
				Status:    "pending",
			},
			expectedError: false,
		},
		{
			name: "valid stage with all fields",
			stage: Stage{
				ID:                "stage-002",
				SessionID:         "session-001",
				Language:          "javascript",
				Operation:         "insert",
				TargetType:        "function",
				TargetName:        "calculateSum",
				TargetQuery:       datatypes.JSON(`{"type": "function", "name": "calculateSum"}`),
				Original:          "function original() {}",
				Modified:          "function modified() {}",
				Content:           "new content",
				Diff:              "@@ -1,1 +1,1 @@\n-original\n+modified",
				BaseDigest:        "abc123",
				AfterDigest:       "def456",
				ConfidenceScore:   0.95,
				ConfidenceLevel:   "high",
				ConfidenceFactors: datatypes.JSON(`{"syntax": 0.9, "context": 1.0}`),
				ScopeAST:          datatypes.JSON(`{"type": "Program", "body": []}`),
				Status:            "pending",
				ExpiresAt:         time.Now().Add(24 * time.Hour),
			},
			expectedError: false,
		},
		{
			name: "stage with empty required fields",
			stage: Stage{
				ID: "stage-003",
				// Missing SessionID, Language, Operation
			},
			expectedError: false, // SQLite doesn't enforce NOT NULL for varchar fields by default
		},
		{
			name: "stage with invalid JSON in TargetQuery",
			stage: Stage{
				ID:          "stage-004",
				SessionID:   "session-001",
				Language:    "python",
				Operation:   "delete",
				TargetQuery: datatypes.JSON(`{invalid json`),
				Status:      "pending",
			},
			expectedError: false, // GORM doesn't validate JSON syntax at insertion
		},
		{
			name: "stage with very long strings",
			stage: Stage{
				ID:        "stage-005",
				SessionID: "session-001",
				Language:  "java",
				Operation: "modify",
				Original:  string(make([]byte, 10000)), // Very long content
				Modified:  string(make([]byte, 10000)),
				Status:    "pending",
			},
			expectedError: false,
		},
	}

	// Create a session first for foreign key constraint
	session := Session{ID: "session-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.stage).Error

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify the stage was created
				var retrieved Stage
				err = db.Where("id = ?", tt.stage.ID).First(&retrieved).Error
				assert.NoError(t, err)
				assert.Equal(t, tt.stage.Language, retrieved.Language)
				assert.Equal(t, tt.stage.Operation, retrieved.Operation)

				// Verify timestamps are set
				assert.False(t, retrieved.CreatedAt.IsZero())

				// Test JSON fields if they were set
				if tt.stage.TargetQuery != nil {
					assert.Equal(t, tt.stage.TargetQuery, retrieved.TargetQuery)
				}
			}
		})
	}
}

func TestApplyModel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create session and multiple stages for different test cases
	session := Session{ID: "apply-session-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	// Create different stages for each test case to avoid conflicts
	stages := []Stage{
		{ID: "apply-stage-001", SessionID: session.ID, Language: "go", Operation: "replace", Status: "pending"},
		{ID: "apply-stage-002", SessionID: session.ID, Language: "js", Operation: "insert", Status: "pending"},
		{ID: "apply-stage-003", SessionID: session.ID, Language: "py", Operation: "delete", Status: "pending"},
	}

	for _, stage := range stages {
		err = db.Create(&stage).Error
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		apply         Apply
		expectedError bool
		errorContains string
	}{
		{
			name: "valid apply with minimal fields",
			apply: Apply{
				ID:      "apply-001",
				StageID: stages[0].ID, // Use first stage
			},
			expectedError: false,
		},
		{
			name: "valid apply with all fields",
			apply: Apply{
				ID:          "apply-002",
				StageID:     stages[1].ID, // Use second stage
				BaseDigest:  "abc123",
				AfterDigest: "def456",
				AutoApplied: true,
				AppliedBy:   "user@example.com",
				Reverted:    false,
				RevertedBy:  "",
			},
			expectedError: false,
		},
		{
			name: "apply with reverted fields",
			apply: Apply{
				ID:         "apply-003",
				StageID:    stages[2].ID, // Use third stage
				Reverted:   true,
				RevertedBy: "admin@example.com",
			},
			expectedError: false,
		},
		{
			name: "apply with non-existent stage ID",
			apply: Apply{
				ID:      "apply-004",
				StageID: "non-existent-stage",
			},
			expectedError: true, // Foreign key constraint
		},
		{
			name: "apply with duplicate stage ID",
			apply: Apply{
				ID:      "apply-005",
				StageID: stages[0].ID, // This will conflict with apply-001
			},
			expectedError: true, // Unique constraint on StageID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set RevertedAt if Reverted is true
			if tt.apply.Reverted && tt.apply.RevertedBy != "" {
				now := time.Now()
				tt.apply.RevertedAt = &now
			}

			err := db.Create(&tt.apply).Error

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify the apply was created
				var retrieved Apply
				err = db.Where("id = ?", tt.apply.ID).First(&retrieved).Error
				assert.NoError(t, err)
				assert.Equal(t, tt.apply.StageID, retrieved.StageID)
				assert.Equal(t, tt.apply.AutoApplied, retrieved.AutoApplied)
				assert.Equal(t, tt.apply.Reverted, retrieved.Reverted)

				// Verify timestamps are set
				assert.False(t, retrieved.AppliedAt.IsZero())

				if tt.apply.Reverted && tt.apply.RevertedAt != nil {
					assert.NotNil(t, retrieved.RevertedAt)
				}
			}
		})
	}
}

func TestSessionModel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tests := []struct {
		name          string
		session       Session
		expectedError bool
		errorContains string
	}{
		{
			name: "valid session with minimal fields",
			session: Session{
				ID: "session-001",
			},
			expectedError: false,
		},
		{
			name: "valid session with all fields",
			session: Session{
				ID:           "session-002",
				StagesCount:  5,
				AppliesCount: 3,
				ClientInfo:   datatypes.JSON(`{"version": "1.0.0", "platform": "linux"}`),
			},
			expectedError: false,
		},
		{
			name: "session with ended timestamp",
			session: Session{
				ID:           "session-003",
				StagesCount:  10,
				AppliesCount: 8,
			},
			expectedError: false,
		},
		{
			name: "session with invalid JSON",
			session: Session{
				ID:         "session-004",
				ClientInfo: datatypes.JSON(`{invalid json`),
			},
			expectedError: false, // GORM doesn't validate JSON syntax
		},
		{
			name: "session with empty ID",
			session: Session{
				ID: "",
			},
			expectedError: false, // SQLite allows empty string as primary key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set EndedAt for the ended session test
			if tt.name == "session with ended timestamp" {
				now := time.Now()
				tt.session.EndedAt = &now
			}

			err := db.Create(&tt.session).Error

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify the session was created
				var retrieved Session
				err = db.Where("id = ?", tt.session.ID).First(&retrieved).Error
				assert.NoError(t, err)
				assert.Equal(t, tt.session.StagesCount, retrieved.StagesCount)
				assert.Equal(t, tt.session.AppliesCount, retrieved.AppliesCount)

				// Verify timestamps are set
				assert.False(t, retrieved.StartedAt.IsZero())

				if tt.session.EndedAt != nil {
					assert.NotNil(t, retrieved.EndedAt)
				}

				// Test JSON field if set
				if tt.session.ClientInfo != nil {
					assert.Equal(t, tt.session.ClientInfo, retrieved.ClientInfo)
				}
			}
		})
	}
}

func TestModelRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create test data
	session := Session{
		ID:           "session-rel-001",
		StagesCount:  2,
		AppliesCount: 1,
	}
	err := db.Create(&session).Error
	require.NoError(t, err)

	stage1 := Stage{
		ID:        "stage-rel-001",
		SessionID: session.ID,
		Language:  "go",
		Operation: "replace",
		Status:    "applied",
	}
	err = db.Create(&stage1).Error
	require.NoError(t, err)

	stage2 := Stage{
		ID:        "stage-rel-002",
		SessionID: session.ID,
		Language:  "javascript",
		Operation: "insert",
		Status:    "pending",
	}
	err = db.Create(&stage2).Error
	require.NoError(t, err)

	apply := Apply{
		ID:        "apply-rel-001",
		StageID:   stage1.ID,
		AppliedBy: "test-user",
	}
	err = db.Create(&apply).Error
	require.NoError(t, err)

	// Test Stage -> Apply relationship
	t.Run("stage with apply relationship", func(t *testing.T) {
		var stageWithApply Stage
		err = db.Preload("Apply").Where("id = ?", stage1.ID).First(&stageWithApply).Error
		assert.NoError(t, err)
		assert.NotNil(t, stageWithApply.Apply)
		assert.Equal(t, apply.ID, stageWithApply.Apply.ID)
		assert.Equal(t, apply.AppliedBy, stageWithApply.Apply.AppliedBy)
	})

	// Test Stage without Apply relationship
	t.Run("stage without apply relationship", func(t *testing.T) {
		var stageWithoutApply Stage
		err = db.Preload("Apply").Where("id = ?", stage2.ID).First(&stageWithoutApply).Error
		assert.NoError(t, err)
		assert.Nil(t, stageWithoutApply.Apply)
	})

	// Test Apply -> Stage relationship
	t.Run("apply with stage relationship", func(t *testing.T) {
		var applyWithStage Apply
		err = db.Preload("Stage").Where("id = ?", apply.ID).First(&applyWithStage).Error
		assert.NoError(t, err)
		assert.Equal(t, stage1.ID, applyWithStage.Stage.ID)
		assert.Equal(t, stage1.Language, applyWithStage.Stage.Language)
	})

	// Test cascading operations
	t.Run("foreign key constraint on apply deletion", func(t *testing.T) {
		// Try to delete stage that has an apply record
		err = db.Delete(&stage1).Error
		assert.Error(t, err) // Should fail due to foreign key constraint

		// Delete apply first, then stage should succeed
		err = db.Delete(&apply).Error
		assert.NoError(t, err)

		err = db.Delete(&stage1).Error
		assert.NoError(t, err)
	})
}

func TestJSONFieldOperations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	session := Session{ID: "session-json-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	t.Run("stage with complex JSON fields", func(t *testing.T) {
		targetQuery := map[string]any{
			"type":   "function",
			"name":   "calculateTotal",
			"params": []string{"price", "tax"},
			"nested": map[string]any{
				"level1": map[string]any{
					"level2": "deep value",
				},
			},
		}

		confidenceFactors := map[string]any{
			"syntax":    0.95,
			"context":   0.85,
			"semantics": 0.90,
			"tests":     []string{"unit", "integration"},
		}

		scopeAST := map[string]any{
			"type": "Program",
			"body": []map[string]any{
				{
					"type": "FunctionDeclaration",
					"name": "calculateTotal",
				},
			},
		}

		targetQueryJSON, err := json.Marshal(targetQuery)
		require.NoError(t, err)

		confidenceFactorsJSON, err := json.Marshal(confidenceFactors)
		require.NoError(t, err)

		scopeASTJSON, err := json.Marshal(scopeAST)
		require.NoError(t, err)

		stage := Stage{
			ID:                "stage-json-001",
			SessionID:         session.ID,
			Language:          "javascript",
			Operation:         "replace",
			TargetQuery:       datatypes.JSON(targetQueryJSON),
			ConfidenceFactors: datatypes.JSON(confidenceFactorsJSON),
			ScopeAST:          datatypes.JSON(scopeASTJSON),
			Status:            "pending",
		}

		err = db.Create(&stage).Error
		assert.NoError(t, err)

		// Retrieve and verify JSON fields
		var retrieved Stage
		err = db.Where("id = ?", stage.ID).First(&retrieved).Error
		assert.NoError(t, err)

		// Verify JSON fields can be unmarshaled
		var retrievedTargetQuery map[string]any
		err = json.Unmarshal(retrieved.TargetQuery, &retrievedTargetQuery)
		assert.NoError(t, err)
		assert.Equal(t, targetQuery["type"], retrievedTargetQuery["type"])
		assert.Equal(t, targetQuery["name"], retrievedTargetQuery["name"])

		var retrievedConfidenceFactors map[string]any
		err = json.Unmarshal(retrieved.ConfidenceFactors, &retrievedConfidenceFactors)
		assert.NoError(t, err)
		assert.Equal(t, 0.95, retrievedConfidenceFactors["syntax"])

		var retrievedScopeAST map[string]any
		err = json.Unmarshal(retrieved.ScopeAST, &retrievedScopeAST)
		assert.NoError(t, err)
		assert.Equal(t, "Program", retrievedScopeAST["type"])
	})

	t.Run("session with client info JSON", func(t *testing.T) {
		clientInfo := map[string]any{
			"version":  "1.2.3",
			"platform": "darwin",
			"arch":     "arm64",
			"features": []string{"auto-apply", "rollback"},
			"config": map[string]any{
				"timeout": 30000,
				"retries": 3,
			},
		}

		clientInfoJSON, err := json.Marshal(clientInfo)
		require.NoError(t, err)

		session := Session{
			ID:         "session-json-002",
			ClientInfo: datatypes.JSON(clientInfoJSON),
		}

		err = db.Create(&session).Error
		assert.NoError(t, err)

		// Retrieve and verify
		var retrieved Session
		err = db.Where("id = ?", session.ID).First(&retrieved).Error
		assert.NoError(t, err)

		var retrievedClientInfo map[string]any
		err = json.Unmarshal(retrieved.ClientInfo, &retrievedClientInfo)
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3", retrievedClientInfo["version"])
		assert.Equal(t, "darwin", retrievedClientInfo["platform"])
	})
}

func TestModelValidation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	t.Run("confidence score validation", func(t *testing.T) {
		session := Session{ID: "session-validation-001"}
		err := db.Create(&session).Error
		require.NoError(t, err)

		// Test valid confidence scores
		validScores := []float64{0.0, 0.5, 0.99, 1.0}
		for i, score := range validScores {
			stage := Stage{
				ID:              fmt.Sprintf("stage-conf-%d", i),
				SessionID:       session.ID,
				Language:        "go",
				Operation:       "test",
				ConfidenceScore: score,
				Status:          "pending",
			}
			err = db.Create(&stage).Error
			assert.NoError(t, err, "Valid confidence score %f should be accepted", score)
		}

		// Test edge case confidence scores (these might be accepted by GORM but are logically invalid)
		edgeCases := []float64{-0.1, 1.1, 2.0}
		for i, score := range edgeCases {
			stage := Stage{
				ID:              fmt.Sprintf("stage-edge-%d", i),
				SessionID:       session.ID,
				Language:        "go",
				Operation:       "test",
				ConfidenceScore: score,
				Status:          "pending",
			}
			err = db.Create(&stage).Error
			// Note: GORM doesn't enforce decimal constraints by default in SQLite
			// This test documents the current behavior
			if err != nil {
				t.Logf("Edge case confidence score %f was rejected: %v", score, err)
			}
		}
	})

	t.Run("timestamp validation", func(t *testing.T) {
		session := Session{ID: "session-time-001"}
		err := db.Create(&session).Error
		require.NoError(t, err)

		// Test with future expiration
		futureTime := time.Now().Add(24 * time.Hour)
		stage := Stage{
			ID:        "stage-time-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			ExpiresAt: futureTime,
			Status:    "pending",
		}
		err = db.Create(&stage).Error
		assert.NoError(t, err)

		// Verify timestamps
		var retrieved Stage
		err = db.Where("id = ?", stage.ID).First(&retrieved).Error
		assert.NoError(t, err)
		assert.True(t, retrieved.CreatedAt.Before(time.Now().Add(time.Second)))
		assert.True(t, retrieved.ExpiresAt.After(time.Now()))
	})
}

func TestDefaultValues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	session := Session{ID: "session-defaults-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	t.Run("stage default status", func(t *testing.T) {
		stage := Stage{
			ID:        "stage-defaults-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			// Status not set, should default to 'pending'
		}
		err = db.Create(&stage).Error
		assert.NoError(t, err)

		var retrieved Stage
		err = db.Where("id = ?", stage.ID).First(&retrieved).Error
		assert.NoError(t, err)
		assert.Equal(t, "pending", retrieved.Status)
	})

	t.Run("apply default values", func(t *testing.T) {
		stage := Stage{
			ID:        "stage-for-apply-001",
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(&stage).Error
		require.NoError(t, err)

		apply := Apply{
			ID:      "apply-defaults-001",
			StageID: stage.ID,
			// AutoApplied and Reverted not set, should default to false
		}
		err = db.Create(&apply).Error
		assert.NoError(t, err)

		var retrieved Apply
		err = db.Where("id = ?", apply.ID).First(&retrieved).Error
		assert.NoError(t, err)
		assert.False(t, retrieved.AutoApplied)
		assert.False(t, retrieved.Reverted)
	})

	t.Run("session default counts", func(t *testing.T) {
		session := Session{
			ID: "session-defaults-002",
			// StagesCount and AppliesCount not set, should default to 0
		}
		err = db.Create(&session).Error
		assert.NoError(t, err)

		var retrieved Session
		err = db.Where("id = ?", session.ID).First(&retrieved).Error
		assert.NoError(t, err)
		assert.Equal(t, 0, retrieved.StagesCount)
		assert.Equal(t, 0, retrieved.AppliesCount)
	})
}

func TestStringFieldConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	session := Session{ID: "session-strings-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	t.Run("varchar length constraints", func(t *testing.T) {
		// Test ID length constraint (varchar(20))
		longID := string(make([]byte, 25)) // Longer than 20 chars
		for i := range longID {
			longID = longID[:i] + "a" + longID[i+1:]
		}

		stage := Stage{
			ID:        longID,
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}

		err = db.Create(&stage).Error
		// SQLite doesn't enforce varchar lengths by default, but this documents the intention
		if err != nil {
			t.Logf("Long ID was rejected: %v", err)
		}

		// Test other varchar fields
		stage2 := Stage{
			ID:        "stage-strings-001",
			SessionID: session.ID,
			Language:  string(make([]byte, 100)), // Longer than varchar(50)
			Operation: "test",
			Status:    "pending",
		}

		err = db.Create(&stage2).Error
		if err != nil {
			t.Logf("Long language field was rejected: %v", err)
		}
	})
}

func TestIndexConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Test SessionID index on Stage
	session := Session{ID: "session-index-001"}
	err := db.Create(&session).Error
	require.NoError(t, err)

	// Create multiple stages with same SessionID (should be allowed)
	for i := range 3 {
		stage := Stage{
			ID:        fmt.Sprintf("stage-index-%03d", i),
			SessionID: session.ID,
			Language:  "go",
			Operation: "test",
			Status:    "pending",
		}
		err = db.Create(&stage).Error
		assert.NoError(t, err, "Multiple stages with same SessionID should be allowed")
	}

	// Test unique index on Apply.StageID
	stage := Stage{
		ID:        "stage-unique-001",
		SessionID: session.ID,
		Language:  "go",
		Operation: "test",
		Status:    "pending",
	}
	err = db.Create(&stage).Error
	require.NoError(t, err)

	apply1 := Apply{
		ID:      "apply-unique-001",
		StageID: stage.ID,
	}
	err = db.Create(&apply1).Error
	assert.NoError(t, err)

	// Try to create another apply with same StageID
	apply2 := Apply{
		ID:      "apply-unique-002",
		StageID: stage.ID, // Same StageID
	}
	err = db.Create(&apply2).Error
	assert.Error(t, err, "Duplicate StageID in Apply should be rejected")
}

// Helper functions

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Enable foreign keys
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&Stage{}, &Apply{}, &Session{})
	require.NoError(t, err)

	return db
}

func cleanupTestDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}
