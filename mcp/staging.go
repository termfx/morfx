package mcp

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// StagingManager handles staging and applying transformations
type StagingManager struct {
	db     *gorm.DB
	config Config
}

// NewStagingManager creates a new staging manager
func NewStagingManager(db *gorm.DB, config Config) *StagingManager {
	return &StagingManager{
		db:     db,
		config: config,
	}
}

// CreateStage creates a new staged transformation
func (sm *StagingManager) CreateStage(stage *models.Stage) error {
	// Generate ID if not set
	if stage.ID == "" {
		stage.ID = generateID("stg")
	}

	// Set defaults
	stage.Status = "pending"
	stage.ExpiresAt = time.Now().Add(sm.config.StagingTTL)

	// Save to database
	return sm.db.Create(stage).Error
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

// ApplyStage applies a staged transformation
func (sm *StagingManager) ApplyStage(stageID string, autoApplied bool) (*models.Apply, error) {
	// Start transaction
	return sm.applyInTransaction(stageID, autoApplied)
}

func (sm *StagingManager) applyInTransaction(stageID string, autoApplied bool) (*models.Apply, error) {
	var apply *models.Apply

	err := sm.db.Transaction(func(tx *gorm.DB) error {
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
		return nil, err
	}

	return apply, nil
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
