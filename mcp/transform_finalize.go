package mcp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/mcp/types"
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

	stageMetadata := map[string]any{
		"target_type": req.Target.Type,
		"target_name": req.Target.Name,
	}
	if s.session != nil {
		stageMetadata["session_id"] = s.session.ID
	}
	if req.Path != "" {
		stageMetadata["file_path"] = req.Path
	}

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
	if s.engine != nil {
		stage, err := s.engine.CreateStage(ctx, engine.StageCreateRequest{
			Path:        req.Path,
			Language:    req.Language,
			Operation:   req.Operation,
			Original:    req.OriginalSource,
			Modified:    req.Result.Modified,
			Diff:        req.Result.Diff,
			BaseDigest:  originalHash,
			AfterDigest: calculateSHA256(req.Result.Modified),
			Confidence:  req.Result.Confidence,
			Metadata:    stageMetadata,
		})
		if err != nil {
			s.debugLog("Engine staging failed, will fallback to direct write: %v", err)
		} else {
			staged = true
			status = "staged"
			referenceID = stage.ID

			if fileMode && shouldAutoApply {
				applyResult, err := s.engine.ApplyStage(ctx, engine.StageApplyRequest{
					ID:          stage.ID,
					AutoApplied: true,
				})
				if err != nil {
					s.debugLog("Engine auto-apply failed, leaving stage pending: %v", err)
					responseText += fmt.Sprintf("\n⚠️ Auto-apply failed: %v", err)
				} else if applyResult.Applied {
					autoApplied = true
					status = "applied"
					referenceID = applyResult.StageID
					responseText += fmt.Sprintf("\n✅ Auto-applied and saved (ID: %s)", referenceID)
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
	if req.Result.Diff != "" {
		resp["diff"] = req.Result.Diff
	}

	return resp, nil
}

func calculateSHA256(content string) string {
	if content == "" {
		return ""
	}
	sum := sha256Sum([]byte(content))
	return fmt.Sprintf("%x", sum)
}

func sha256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}
