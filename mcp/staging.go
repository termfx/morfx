package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// StagingManager handles staging and applying transformations
type StagingManager struct {
	db     *gorm.DB
	config Config
	safety *SafetyManager
}

// IsEnabled reports whether staging support is active. The manager is enabled
// whenever it has a backing database connection.
func (sm *StagingManager) IsEnabled() bool {
	return sm != nil && sm.db != nil
}

// NewStagingManager creates a new staging manager
func NewStagingManager(db *gorm.DB, config Config, safety *SafetyManager) *StagingManager {
	return &StagingManager{
		db:     db,
		config: config,
		safety: safety,
	}
}

// CreateStage creates a new staged transformation while honoring cancellation.
func (sm *StagingManager) CreateStage(ctx context.Context, stage *models.Stage) error {
	// Validate stage is not nil
	if stage == nil {
		return fmt.Errorf("stage cannot be nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	db := sm.db.WithContext(ctx)

	// Check session limits before creating stage
	if stage.SessionID != "" && sm.config.MaxStagesPerSession > 0 {
		var pendingCount int64
		if err := db.Model(&models.Stage{}).
			Where("session_id = ? AND status = ?", stage.SessionID, "pending").
			Count(&pendingCount).Error; err != nil {
			return fmt.Errorf("failed to check stage count: %w", err)
		}

		if pendingCount >= int64(sm.config.MaxStagesPerSession) {
			return fmt.Errorf("session stage limit exceeded: %d >= %d", pendingCount, sm.config.MaxStagesPerSession)
		}
	}

	// Generate ID if not set
	if stage.ID == "" {
		stage.ID = generateID("stg")
	}

	// Set defaults only if not already set
	if stage.Status == "" {
		stage.Status = "pending"
	}

	// Set expiration only if not already set
	if stage.ExpiresAt.IsZero() {
		stage.ExpiresAt = time.Now().Add(sm.config.StagingTTL)
	}

	// Save to database
	if err := db.Create(stage).Error; err != nil {
		return err
	}

	return ctx.Err()
}

// GetStage retrieves a stage by ID
func (sm *StagingManager) GetStage(id string) (*models.Stage, error) {
	var stage models.Stage
	err := sm.db.First(&stage, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// ApplyStage applies a staged transformation while honoring cancellation.
func (sm *StagingManager) ApplyStage(ctx context.Context, stageID string, autoApplied bool) (*models.Apply, error) {
	var (
		apply      *models.Apply
		stageWrite *fileWriteGuard
	)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	err := sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Get the stage
		var stage models.Stage
		if err := tx.First(&stage, "id = ?", stageID).Error; err != nil {
			return fmt.Errorf("stage not found: %w", err)
		}

		// Check status
		if stage.Status != "pending" {
			return fmt.Errorf("stage already %s", stage.Status)
		}

		// Check expiration
		if time.Now().After(stage.ExpiresAt) {
			// Update status to expired
			stage.Status = "expired"
			tx.Save(&stage)
			return fmt.Errorf("stage expired")
		}

		// Check apply session limits before applying
		if stage.SessionID != "" && sm.config.MaxAppliesPerSession > 0 {
			var applyCount int64
			if err := tx.Model(&models.Apply{}).
				Joins("JOIN stages ON applies.stage_id = stages.id").
				Where("stages.session_id = ?", stage.SessionID).
				Count(&applyCount).Error; err != nil {
				return fmt.Errorf("failed to check apply count: %w", err)
			}

			if applyCount >= int64(sm.config.MaxAppliesPerSession) {
				return fmt.Errorf("session apply limit exceeded: %d >= %d", applyCount, sm.config.MaxAppliesPerSession)
			}
		}

		if !autoApplied {
			var err error
			stageWrite, err = sm.prepareStageWrite(&stage)
			if err != nil {
				return err
			}
		}

		// Create apply record
		apply = &models.Apply{
			ID:          generateID("apl"),
			StageID:     stageID,
			AutoApplied: autoApplied,
			AppliedBy:   "mcp",
		}

		if autoApplied {
			apply.AppliedBy = "auto"
		}

		if err := tx.Create(apply).Error; err != nil {
			return fmt.Errorf("failed to create apply record: %w", err)
		}

		// Update stage status
		now := time.Now()
		stage.Status = "applied"
		stage.AppliedAt = &now

		if err := tx.Save(&stage).Error; err != nil {
			return fmt.Errorf("failed to update stage: %w", err)
		}

		// Update session statistics if available
		if stage.SessionID != "" {
			tx.Model(&models.Session{}).
				Where("id = ?", stage.SessionID).
				Update("applies_count", gorm.Expr("applies_count + ?", 1))
		}

		return nil
	})
	if err != nil {
		if stageWrite != nil {
			_ = stageWrite.Rollback()
		}
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		if stageWrite != nil {
			_ = stageWrite.Rollback()
		}
		return nil, err
	}

	if stageWrite != nil {
		stageWrite.Commit()
	}

	return apply, nil
}

func (sm *StagingManager) prepareStageWrite(stage *models.Stage) (*fileWriteGuard, error) {
	if stage == nil {
		return nil, fmt.Errorf("stage cannot be nil")
	}

	if len(stage.ScopeAST) == 0 {
		return nil, nil
	}

	var scope map[string]any
	if err := json.Unmarshal(stage.ScopeAST, &scope); err != nil {
		return nil, fmt.Errorf("failed to decode stage scope: %w", err)
	}

	pathVal, ok := scope["file_path"].(string)
	if !ok || pathVal == "" {
		return nil, nil
	}
	path := pathVal

	if stage.Modified == "" {
		return nil, fmt.Errorf("stage %s has no modified content", stage.ID)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	if sm.safety != nil {
		op := &SafetyOperation{
			Files: []SafetyFile{{
				Path:       path,
				Size:       int64(len(stage.Modified)),
				Confidence: stage.ConfidenceScore,
			}},
			GlobalConfidence: stage.ConfidenceScore,
		}

		if err := sm.safety.ValidateOperation(op); err != nil {
			return nil, err
		}

		if stage.BaseDigest != "" {
			if err := sm.safety.ValidateFileIntegrity([]FileIntegrityCheck{{
				Path:         path,
				ExpectedHash: stage.BaseDigest,
			}}); err != nil {
				return nil, err
			}
		}

		handle, err := sm.safety.AtomicWrite(path, stage.Modified)
		if err != nil {
			return nil, err
		}

		return &fileWriteGuard{
			commitFn: func() {
				if handle != nil {
					handle.Commit()
				}
			},
			rollbackFn: func() error {
				if handle != nil {
					return handle.Rollback()
				}
				return nil
			},
		}, nil
	}

	var (
		originalBytes  []byte
		originalExists bool
		filePerm       os.FileMode = 0o644
	)

	if info, err := os.Stat(path); err == nil {
		filePerm = info.Mode().Perm()
		if data, readErr := os.ReadFile(path); readErr == nil {
			originalBytes = data
			originalExists = true
		}
	}

	if err := os.WriteFile(path, []byte(stage.Modified), filePerm); err != nil {
		return nil, err
	}

	return &fileWriteGuard{
		commitFn: func() {},
		rollbackFn: func() error {
			if originalExists {
				return os.WriteFile(path, originalBytes, filePerm)
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		},
	}, nil
}

// ListPendingStages lists all pending stages for a session
func (sm *StagingManager) ListPendingStages(sessionID string) ([]models.Stage, error) {
	var stages []models.Stage
	err := sm.db.
		Where("session_id = ? AND status = ?", sessionID, "pending").
		Order("created_at DESC").
		Find(&stages).Error
	return stages, err
}

// CleanupExpiredStages marks expired stages
func (sm *StagingManager) CleanupExpiredStages() error {
	return sm.db.Model(&models.Stage{}).
		Where("status = ? AND expires_at < ?", "pending", time.Now()).
		Update("status", "expired").Error
}

// DeleteAppliedStages removes applied stages from database
func (sm *StagingManager) DeleteAppliedStages(sessionID string) error {
	return sm.db.Where("session_id = ? AND status = ?", sessionID, "applied").
		Delete(&models.Stage{}).Error
}

// DeleteStage removes a specific stage by ID
func (sm *StagingManager) DeleteStage(stageID string) error {
	return sm.db.Where("id = ?", stageID).Delete(&models.Stage{}).Error
}
