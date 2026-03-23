package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/oxhq/morfx/mcp/types"
	"github.com/oxhq/morfx/models"
	"gorm.io/datatypes"
)

// FinalizeTransform implements types.ServerInterface. It centralises staging, auto-apply,
// and response formatting so tool handlers stay lean and consistent.
type fileWriteGuard struct {
	commitFn   func()
	rollbackFn func() error
}

func (g *fileWriteGuard) Commit() {
	if g == nil || g.commitFn == nil {
		return
	}
	g.commitFn()
}

func (g *fileWriteGuard) Rollback() error {
	if g == nil || g.rollbackFn == nil {
		return nil
	}
	return g.rollbackFn()
}

func (s *StdioServer) FinalizeTransform(ctx context.Context, req types.TransformRequest) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	fileMode := req.Path != ""
	responseText := req.ResponseText
	shouldAutoApply := s.config.AutoApplyEnabled && req.Result.Confidence.Score >= s.config.AutoApplyThreshold

	originalHash := ""
	if req.OriginalSource != "" {
		originalHash = calculateSHA256(req.OriginalSource)
	}

	status := "completed"
	var referenceID string
	autoApplied := false

	// writeFileDirect writes the modified content to disk, bypassing staging.
	writeFileDirect := func() error {
		if !fileMode || req.Result.Modified == "" {
			return nil
		}

		var filePerm os.FileMode = 0o644
		if info, err := os.Stat(req.Path); err == nil {
			filePerm = info.Mode().Perm()
		}

		// Use safety manager if available
		if sm := s.safety; sm != nil {
			op := &SafetyOperation{
				Files: []SafetyFile{{
					Path:       req.Path,
					Size:       int64(len(req.Result.Modified)),
					Confidence: req.Result.Confidence.Score,
				}},
				GlobalConfidence: req.Result.Confidence.Score,
			}
			if err := sm.ValidateOperation(op); err != nil {
				return err
			}
			if originalHash != "" {
				checks := []FileIntegrityCheck{{Path: req.Path, ExpectedHash: originalHash}}
				if err := sm.ValidateFileIntegrity(checks); err != nil {
					return err
				}
			}
			handle, err := sm.AtomicWrite(req.Path, req.Result.Modified)
			if err != nil {
				return err
			}
			if handle != nil {
				handle.Commit()
			}
			return nil
		}

		// Fallback: direct write
		return os.WriteFile(req.Path, []byte(req.Result.Modified), filePerm)
	}

	// Try staging path first
	staged := false
	if s.staging != nil {
		stage := s.buildStage(req, originalHash)
		if err := s.staging.CreateStage(ctx, stage); err != nil {
			s.debugLog("Staging failed, will fallback to direct write: %v", err)
			// Don't block — fall through to direct write below
		} else {
			staged = true
			status = "staged"
			referenceID = stage.ID

			if shouldAutoApply {
				// Write file first, then mark stage as applied
				if fileMode {
					if err := writeFileDirect(); err != nil {
						responseText += fmt.Sprintf("\n⚠️ Auto-apply file write failed: %v", err)
						responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
						responseText += "\nUse the apply tool to write changes to disk."
					} else {
						// File written, now mark stage as applied in DB
						applyResult, err := s.staging.ApplyStage(ctx, stage.ID, true)
						if err != nil {
							// File is already written, just log the DB error
							s.debugLog("Stage apply record failed (file already written): %v", err)
						}
						autoApplied = true
						status = "applied"
						if applyResult != nil {
							referenceID = applyResult.ID
						}
						responseText += fmt.Sprintf("\n✅ Auto-applied and saved (ID: %s)", referenceID)
					}
				} else {
					// Source mode: mark as applied without file write
					applyResult, err := s.staging.ApplyStage(ctx, stage.ID, true)
					if err != nil {
						responseText += fmt.Sprintf("\n⚠️ Failed to auto-apply: %v", err)
					} else {
						autoApplied = true
						status = "applied"
						if applyResult != nil {
							referenceID = applyResult.ID
						}
						responseText += fmt.Sprintf("\n✅ Auto-applied (ID: %s)", referenceID)
					}
				}
			}

			// If not auto-applied, inform about pending stage
			if !autoApplied {
				responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
				if fileMode {
					responseText += "\nUse the apply tool to write changes to disk."
				}
			}
		}
	}

	// Fallback: no staging available or staging failed — write directly if auto-apply enabled
	if !staged && fileMode && shouldAutoApply {
		if err := writeFileDirect(); err != nil {
			responseText += fmt.Sprintf("\n⚠️ Failed to write file: %v", err)
		} else {
			autoApplied = true
			status = "applied"
			responseText += "\n✅ Applied directly (no staging available)"
		}
	}

	// If file mode but nothing was written, warn clearly
	if fileMode && !autoApplied && !staged {
		if !shouldAutoApply {
			responseText += "\n⚠️ Confidence below auto-apply threshold – file not modified."
		}
	}

	resp := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"confidence": req.Result.Confidence.Score,
		"matches":    req.Result.MatchCount,
	}

	if fileMode {
		resp["path"] = req.Path
	}
	if status != "" {
		resp["result"] = status
	}
	if referenceID != "" {
		resp["id"] = referenceID
	}
	if req.Result.Modified != "" {
		resp["modified"] = req.Result.Modified
	}

	return resp, nil
}

func (s *StdioServer) buildStage(req types.TransformRequest, originalHash string) *models.Stage {
	targetJSON := req.TargetJSON
	if len(targetJSON) == 0 {
		if encoded, err := json.Marshal(req.Target); err == nil {
			targetJSON = encoded
		}
	}

	stageID := generateID("stg")
	stage := &models.Stage{
		ID:        stageID,
		Language:  req.Language,
		Operation: req.Operation,

		TargetType:  req.Target.Type,
		TargetName:  req.Target.Name,
		TargetQuery: datatypes.JSON(targetJSON),

		Original: req.OriginalSource,
		Modified: req.Result.Modified,
		Content:  req.Content,
		Diff:     req.Result.Diff,

		BaseDigest:  originalHash,
		AfterDigest: calculateSHA256(req.Result.Modified),

		ConfidenceScore:   req.Result.Confidence.Score,
		ConfidenceLevel:   req.Result.Confidence.Level,
		ConfidenceFactors: mustMarshalJSON(req.Result.Confidence.Factors),
	}

	if s.session != nil {
		stage.SessionID = s.session.ID
	}

	if req.Path != "" {
		scope := map[string]any{
			"file_path":        req.Path,
			"safety_validated": s.safety != nil,
			"file_size":        len(req.Result.Modified),
		}
		if originalHash != "" {
			scope["original_hash"] = originalHash
		}
		stage.ScopeAST = mustMarshalJSON(scope)
	}

	return stage
}

func calculateSHA256(content string) string {
	if content == "" {
		return ""
	}
	sum := sha256Sum([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func mustMarshalJSON(value any) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte("null"))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(data)
}

func sha256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}
