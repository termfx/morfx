package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gorm.io/datatypes"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/models"
)

// handleAppendTool executes append operation with optional target
func (s *StdioServer) handleAppendTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target,omitempty"` // OPTIONAL
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid append parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	// Validate that content is provided (can be empty string, but must be present)
	// Check if the field was actually provided in the JSON
	var rawArgs map[string]json.RawMessage
	if err := json.Unmarshal(params, &rawArgs); err != nil {
		return nil, WrapError(InvalidParams, "Invalid parameters", err)
	}
	if _, hasContent := rawArgs["content"]; !hasContent {
		return nil, NewMCPError(InvalidParams, "Missing required parameter: content", nil)
	}

	// Get source code
	var actualSource string
	var originalHash string
	var isFileMode bool
	var fileLock *FileLock
	_ = fileLock // Used for lock management

	if args.Path != "" {
		// FILE WRITER MODE: Read from filesystem with safety checks

		// Acquire file lock if in file mode
		lock, err := s.safety.LockFile(args.Path)
		if err != nil {
			return nil, err
		}
		defer lock.Release()
		fileLock = lock

		content, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, WrapError(FileSystemError, "Failed to read file", err)
		}
		actualSource = string(content)

		// Calculate original hash for integrity checking
		if s.config.Safety.ValidateFileHashes {
			originalHash = calculateSHA256(actualSource)
		}

		// Validate file size
		if s.config.Safety.MaxFileSize > 0 && int64(len(actualSource)) > s.config.Safety.MaxFileSize {
			return nil, NewMCPError(FileTooLarge,
				fmt.Sprintf("File exceeds size limit: %s (%d bytes > %d bytes)",
					args.Path, len(actualSource), s.config.Safety.MaxFileSize))
		}

		isFileMode = true
	} else {
		// IN-MEMORY MODE: Use provided source
		actualSource = args.Source
		isFileMode = false
	}

	// Get provider
	provider, exists := s.providers.Get(args.Language)
	if !exists {
		return nil, NewMCPError(LanguageNotFound,
			fmt.Sprintf("No provider for language: %s", args.Language))
	}

	var result core.TransformResult
	var operationDesc string

	// Check if target specified
	if len(args.Target) > 0 {
		// Target specified - append to specific scope
		var target core.AgentQuery
		if err := json.Unmarshal(args.Target, &target); err != nil {
			return nil, WrapError(InvalidParams, "Invalid target query", err)
		}

		// Use regular transform with insert_after semantics
		op := core.TransformOp{
			Method:  "append", // Append as last element in target
			Target:  target,
			Content: args.Content,
		}

		result = provider.Transform(actualSource, op)
		operationDesc = fmt.Sprintf("Appended to %s '%s'", target.Type, target.Name)

	} else {
		// NO TARGET - use smart detection
		smartProvider, ok := provider.(interface {
			SmartAppend(source, content string) core.TransformResult
		})

		if !ok {
			// Provider doesn't support smart append, fallback to end of file
			result = s.simpleAppendToEnd(actualSource, args.Content)
			operationDesc = "Appended to end of file (no smart detection)"
		} else {
			result = smartProvider.SmartAppend(actualSource, args.Content)
			// Get strategy from metadata
			if strategy, ok := result.Metadata["strategy"].(string); ok {
				operationDesc = fmt.Sprintf("Smart placement: %s", strategy)
			} else {
				operationDesc = "Smart placement based on content type"
			}
		}
	}

	// Check for errors
	if result.Error != nil {
		errMsg := result.Error.Error()
		if args.Target != nil && strings.Contains(errMsg, "no matches") {
			return nil, NewMCPError(NoMatches, "Target not found for append",
				map[string]any{"error": errMsg})
		}
		return nil, WrapError(TransformFailed, "Append operation failed", result.Error)
	}

	// Safety validation for single file operations
	if isFileMode {
		// Validate operation safety
		safetyOp := &SafetyOperation{
			Files: []SafetyFile{
				{
					Path:       args.Path,
					Size:       int64(len(result.Modified)),
					Confidence: result.Confidence.Score,
				},
			},
			GlobalConfidence: result.Confidence.Score,
		}

		if err := s.safety.ValidateOperation(safetyOp); err != nil {
			return nil, err
		}

		// Validate file integrity (check if file was modified externally)
		if originalHash != "" {
			integrityChecks := []FileIntegrityCheck{
				{
					Path:         args.Path,
					ExpectedHash: originalHash,
				},
			}
			if err := s.safety.ValidateFileIntegrity(integrityChecks); err != nil {
				return nil, err
			}
		}
	}

	// Format response
	responseText := fmt.Sprintf("%s\n\nConfidence: %s (%.2f)\n",
		operationDesc,
		result.Confidence.Level,
		result.Confidence.Score)

	// Add confidence factors
	if len(result.Confidence.Factors) > 0 {
		responseText += "\nFactors:\n"
		for _, factor := range result.Confidence.Factors {
			sign := "+"
			if factor.Impact < 0 {
				sign = ""
			}
			responseText += fmt.Sprintf("• %s (%s%.2f): %s\n",
				factor.Name, sign, factor.Impact, factor.Reason)
		}
	}

	// Check staging
	shouldAutoApply := s.config.AutoApplyEnabled &&
		result.Confidence.Score >= s.config.AutoApplyThreshold

	// FILE WRITER MODE: Write to filesystem if auto-applying with safety
	if isFileMode && shouldAutoApply {
		if err := s.safety.AtomicWrite(args.Path, result.Modified); err != nil {
			return nil, WrapError(FileSystemError, "Failed to write file safely", err)
		}
		responseText += fmt.Sprintf("\n✅ File updated safely: %s", args.Path)

		// Add safety info
		if s.config.Safety.AtomicWrites {
			responseText += " (atomic)"
		}
		if s.config.Safety.CreateBackups {
			responseText += " (backup created)"
		}
	}

	// Add file info if in FILE WRITER MODE
	if isFileMode {
		responseText = fmt.Sprintf("File: %s\n\n%s", args.Path, responseText)
	}

	// If no staging, return direct
	if s.staging == nil {
		status := "completed"
		if shouldAutoApply {
			status = "applied"
			if !isFileMode {
				responseText += "\n✅ Changes applied (no staging available)"
			}
		} else {
			if isFileMode {
				responseText += "\n⚠️ Low confidence - file not modified. Use 'apply' to force write."
			} else {
				responseText += "\n⚠️ Low confidence - review recommended"
			}
		}

		if result.Diff != "" {
			responseText += "\nChanges:\n```diff\n" + result.Diff + "\n```"
		}

		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": responseText},
			},
			"result":   status,
			"modified": result.Modified,
			"path":     args.Path, // Include path if in file mode
		}, nil
	}

	// Create stage for persistence
	var targetQuery datatypes.JSON
	if args.Target != nil {
		targetQuery = datatypes.JSON(args.Target)
	} else {
		// Store "smart" as indicator
		targetQuery = mustMarshalJSON(map[string]string{"mode": "smart"})
	}

	stage := &models.Stage{
		ID:        generateID("stg"),
		Language:  args.Language,
		Operation: "append",

		TargetType:  "auto",
		TargetName:  operationDesc,
		TargetQuery: targetQuery,

		// Content
		Original:    actualSource,
		Modified:    result.Modified,
		Content:     args.Content,
		Diff:        result.Diff,
		BaseDigest:  calculateSHA256(actualSource),
		AfterDigest: calculateSHA256(result.Modified),

		// Confidence
		ConfidenceScore:   result.Confidence.Score,
		ConfidenceLevel:   result.Confidence.Level,
		ConfidenceFactors: mustMarshalJSON(result.Confidence.Factors),

		// Store metadata with safety info
		ScopeAST: mustMarshalJSON(map[string]any{
			"strategy":         result.Metadata,
			"file_path":        args.Path,
			"original_hash":    originalHash,
			"file_size":        len(result.Modified),
			"safety_validated": true,
		}),
	}

	// Add session
	if s.session != nil {
		stage.SessionID = s.session.ID
	}

	// Save stage
	if err := s.staging.CreateStage(stage); err != nil {
		s.debugLog("Failed to create stage: %v", err)
		return nil, WrapError(InternalError, "Failed to stage append", err)
	}

	// Auto-apply if confidence high
	status := "staged"
	referenceID := stage.ID

	if shouldAutoApply {
		apply, err := s.staging.ApplyStage(stage.ID, true)
		if err != nil {
			s.debugLog("Failed to auto-apply: %v", err)
			responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
		} else {
			status = "applied"
			referenceID = apply.ID
			if isFileMode {
				responseText += fmt.Sprintf("\n✅ Auto-applied and file updated safely (ID: %s)", apply.ID)
			} else {
				responseText += fmt.Sprintf("\n✅ Auto-applied (ID: %s)", apply.ID)
			}
		}
	} else {
		responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
		if isFileMode {
			responseText += "\nUse 'apply' command to write changes to file"
		} else {
			responseText += "\nUse 'apply' command to commit changes"
		}
	}

	// Add diff
	if result.Diff != "" {
		responseText += "\n\nChanges:\n```diff\n" + result.Diff + "\n```"
	}

	response := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": responseText},
		},
		"result": status,
		"id":     referenceID,
	}

	// Include modified source and path
	if status == "applied" || result.Modified != "" {
		response["modified"] = result.Modified
	}
	if isFileMode {
		response["path"] = args.Path
	}

	return response, nil
}

// simpleAppendToEnd creates a simple append-to-end result
func (s *StdioServer) simpleAppendToEnd(source, content string) core.TransformResult {
	// Simple append
	var modified string
	if len(source) > 0 && source[len(source)-1] != '\n' {
		modified = source + "\n\n" + content
	} else {
		modified = source + "\n" + content
	}

	if !strings.HasSuffix(modified, "\n") {
		modified += "\n"
	}

	// Simple diff
	diff := fmt.Sprintf("--- original\n+++ modified\n@@ -%d,0 +%d,%d @@\n",
		strings.Count(source, "\n"),
		strings.Count(source, "\n")+1,
		strings.Count(content, "\n")+1)

	for line := range strings.SplitSeq(content, "\n") {
		diff += "+" + line + "\n"
	}

	return core.TransformResult{
		Modified: modified,
		Diff:     diff,
		Confidence: core.ConfidenceScore{
			Score: 1.0,
			Level: "high",
			Factors: []core.ConfidenceFactor{
				{
					Name:   "end_of_file",
					Impact: 0.0,
					Reason: "Simple append to end",
				},
			},
		},
		MatchCount: 1,
		Metadata: map[string]any{
			"strategy": "End of file (fallback)",
		},
	}
}
