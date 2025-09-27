package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/termfx/morfx/mcp/types"
	"github.com/termfx/morfx/models"
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

	var status string = "completed"
	var referenceID string
	autoApplied := false
	modifiedWritten := false

	// Helper to ensure we only write once we have validation in place.
	writeFile := func() (*fileWriteGuard, error) {
		if !fileMode || req.Result.Modified == "" || modifiedWritten {
			return nil, nil
		}

		var (
			originalBytes  []byte
			originalExists bool
			filePerm       os.FileMode = 0o644
		)

		if info, err := os.Stat(req.Path); err == nil {
			filePerm = info.Mode().Perm()
			if data, readErr := os.ReadFile(req.Path); readErr == nil {
				originalBytes = data
				originalExists = true
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		if sm := s.safety; sm != nil {
			op := &SafetyOperation{
				Files: []SafetyFile{
					{
						Path:       req.Path,
						Size:       int64(len(req.Result.Modified)),
						Confidence: req.Result.Confidence.Score,
					},
				},
				GlobalConfidence: req.Result.Confidence.Score,
			}
			if err := sm.ValidateOperation(op); err != nil {
				return nil, err
			}

			if originalHash != "" {
				checks := []FileIntegrityCheck{{
					Path:         req.Path,
					ExpectedHash: originalHash,
				}}
				if err := sm.ValidateFileIntegrity(checks); err != nil {
					return nil, err
				}
			}

			handle, err := sm.AtomicWrite(req.Path, req.Result.Modified)
			if err != nil {
				return nil, err
			}
			modifiedWritten = true
			return &fileWriteGuard{
				commitFn: func() {
					if handle != nil {
						handle.Commit()
					}
					modifiedWritten = true
				},
				rollbackFn: func() error {
					if handle != nil {
						if err := handle.Rollback(); err != nil {
							return err
						}
						modifiedWritten = false
						return nil
					}
					if originalExists {
						if err := os.WriteFile(req.Path, originalBytes, filePerm); err != nil {
							return err
						}
					} else if err := os.Remove(req.Path); err != nil && !os.IsNotExist(err) {
						return err
					}
					modifiedWritten = false
					return nil
				},
			}, nil
		}

		// Fallback direct write when safety manager is unavailable.
		if err := os.WriteFile(req.Path, []byte(req.Result.Modified), filePerm); err != nil {
			return nil, err
		}
		modifiedWritten = true
		return &fileWriteGuard{
			commitFn: func() {
				modifiedWritten = true
			},
			rollbackFn: func() error {
				if originalExists {
					if err := os.WriteFile(req.Path, originalBytes, filePerm); err != nil {
						return err
					}
				} else if err := os.Remove(req.Path); err != nil && !os.IsNotExist(err) {
					return err
				}
				modifiedWritten = false
				return nil
			},
		}, nil
	}

	// Attempt staging when available.
	if s.staging != nil {
		stage := s.buildStage(req, originalHash)
		applied := false

		if err := s.staging.CreateStage(ctx, stage); err != nil {
			responseText += fmt.Sprintf("\n⚠️ Failed to stage transformation: %v", err)
		} else {
			status = "staged"
			referenceID = stage.ID

			var stageWrite *fileWriteGuard

			if shouldAutoApply && fileMode {
				guard, err := writeFile()
				if err != nil {
					shouldAutoApply = false
					responseText += fmt.Sprintf("\n⚠️ Auto-apply aborted: %v", err)
				} else {
					stageWrite = guard
				}
			}

			if shouldAutoApply {
				applyResult, err := s.staging.ApplyStage(ctx, stage.ID, true)
				if err != nil {
					shouldAutoApply = false
					if stageWrite != nil {
						if rollbackErr := stageWrite.Rollback(); rollbackErr != nil {
							responseText += fmt.Sprintf("\n⚠️ Failed to rollback file write: %v", rollbackErr)
						}
					}
					responseText += fmt.Sprintf("\n⚠️ Failed to auto-apply: %v", err)
				} else {
					applied = true
					autoApplied = true
					status = "applied"
					if applyResult != nil {
						referenceID = applyResult.ID
					} else {
						referenceID = stage.ID
					}
					if stageWrite != nil {
						stageWrite.Commit()
					}
				}
			}

			if applied {
				if fileMode {
					responseText += fmt.Sprintf("\n✅ Auto-applied and saved (ID: %s)", referenceID)
				} else {
					responseText += fmt.Sprintf("\n✅ Auto-applied (ID: %s)", referenceID)
				}
			} else {
				responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
				if fileMode {
					responseText += "\nUse the apply tool to write changes to disk."
				}
			}
		}
	} else {
		// No staging available – only auto-apply can persist changes.
		if shouldAutoApply {
			guard, err := writeFile()
			if err != nil {
				shouldAutoApply = false
				responseText += fmt.Sprintf("\n⚠️ Auto-apply aborted: %v", err)
			} else {
				if guard != nil {
					guard.Commit()
				}
				autoApplied = true
				status = "applied"
				responseText += "\n✅ Auto-applied"
			}
		} else if fileMode {
			responseText += "\n⚠️ Auto-apply disabled – file not modified. Apply the diff manually."
		}
	}

	// If auto-apply requested but we could not fulfil it in staging path, and file mode,
	// ensure the file is not left modified inadvertently.
	if fileMode && shouldAutoApply && !autoApplied {
		// We reach here if staging prevented auto apply; ensure we roll back any write.
		// Since writeFile only sets modifiedWritten when successful, no action needed.
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
