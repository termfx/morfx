package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gorm.io/datatypes"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/models"
)

// handleReplaceTool executes replacement transformation with staging
func (s *StdioServer) handleReplaceTool(params json.RawMessage) (any, error) {
	var args struct {
		Language    string          `json:"language"`
		Source      string          `json:"source"`
		Path        string          `json:"path"`
		Target      json.RawMessage `json:"target"`
		Replacement string          `json:"replacement"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid replace parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	return s.executeTransform(args.Language, args.Source, args.Path, args.Target, core.TransformOp{
		Method:      "replace",
		Replacement: args.Replacement,
	})
}

// handleDeleteTool executes deletion transformation with staging
func (s *StdioServer) handleDeleteTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid delete parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	return s.executeTransform(args.Language, args.Source, args.Path, args.Target, core.TransformOp{
		Method: "delete",
	})
}

// handleInsertBeforeTool executes insert before transformation with staging
func (s *StdioServer) handleInsertBeforeTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid insert_before parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	return s.executeTransform(args.Language, args.Source, args.Path, args.Target, core.TransformOp{
		Method:  "insert_before",
		Content: args.Content,
	})
}

// handleInsertAfterTool executes insert after transformation with staging
func (s *StdioServer) handleInsertAfterTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid insert_after parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	return s.executeTransform(args.Language, args.Source, args.Path, args.Target, core.TransformOp{
		Method:  "insert_after",
		Content: args.Content,
	})
}

// executeTransform is the common transformation logic with staging and file writing
func (s *StdioServer) executeTransform(
	language, source, path string,
	targetJSON json.RawMessage,
	op core.TransformOp,
) (any, error) {
	// Get source code
	var actualSource string
	var originalHash string
	var isFileMode bool
	var fileLock *FileLock
	_ = fileLock // Used for lock management

	if path != "" {
		// FILE WRITER MODE: Read from filesystem with safety checks

		// Acquire file lock if in file mode
		lock, err := s.safety.LockFile(path)
		if err != nil {
			return nil, err
		}
		defer lock.Release()
		fileLock = lock

		content, err := os.ReadFile(path)
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
					path, len(actualSource), s.config.Safety.MaxFileSize))
		}

		isFileMode = true
	} else {
		// IN-MEMORY MODE: Use provided source
		actualSource = source
		isFileMode = false
	}

	// Get provider
	provider, exists := s.providers.Get(language)
	if !exists {
		return nil, NewMCPError(LanguageNotFound,
			fmt.Sprintf("No provider for language: %s", language),
			map[string]any{
				"requested": language,
				"supported": []string{"go"},
			})
	}

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(targetJSON, &target); err != nil {
		return nil, WrapError(InvalidParams, "Invalid target query", err)
	}
	op.Target = target

	// Execute transformation
	result := provider.Transform(actualSource, op)
	if result.Error != nil {
		errMsg := result.Error.Error()
		if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "syntax") {
			return nil, NewMCPError(SyntaxError, "Failed to parse source",
				map[string]any{"details": errMsg})
		}
		if strings.Contains(errMsg, "no matches") || strings.Contains(errMsg, "no targets") {
			return nil, NewMCPError(NoMatches, fmt.Sprintf("No targets found for %s", op.Method),
				map[string]any{"details": errMsg})
		}
		return nil, WrapError(TransformFailed, fmt.Sprintf("%s operation failed", op.Method), result.Error)
	}

	// Safety validation for single file operations
	if isFileMode {
		// Validate operation safety
		safetyOp := &SafetyOperation{
			Files: []SafetyFile{
				{
					Path:       path,
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
					Path:         path,
					ExpectedHash: originalHash,
				},
			}
			if err := s.safety.ValidateFileIntegrity(integrityChecks); err != nil {
				return nil, err
			}
		}
	}

	// Determine if should auto-apply
	shouldAutoApply := s.config.AutoApplyEnabled &&
		result.Confidence.Score >= s.config.AutoApplyThreshold

	// Create response text
	responseText := s.formatTransformResponse(op.Method, result)

	// FILE WRITER MODE: Write to filesystem if auto-applying with safety
	if isFileMode && shouldAutoApply {
		if err := s.safety.AtomicWrite(path, result.Modified); err != nil {
			return nil, WrapError(FileSystemError, "Failed to write file safely", err)
		}
		responseText += fmt.Sprintf("\n✅ File updated safely: %s", path)

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
		responseText = fmt.Sprintf("File: %s\n\n%s", path, responseText)
	}

	// If no staging available, return direct result
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

		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": responseText},
			},
			"result":   status,
			"modified": result.Modified,
			"path":     path, // Include path if in file mode
		}, nil
	}

	// Create stage with additional safety metadata
	stage := &models.Stage{
		ID:        generateID("stg"),
		Language:  language,
		Operation: op.Method,

		// Target info
		TargetType:  target.Type,
		TargetName:  target.Name,
		TargetQuery: datatypes.JSON(targetJSON),

		// Content
		Original:    actualSource,
		Modified:    result.Modified,
		Content:     op.Content,
		Diff:        result.Diff,
		BaseDigest:  calculateSHA256(actualSource),
		AfterDigest: calculateSHA256(result.Modified),

		// Confidence
		ConfidenceScore:   result.Confidence.Score,
		ConfidenceLevel:   result.Confidence.Level,
		ConfidenceFactors: mustMarshalJSON(result.Confidence.Factors),

		// Safety metadata
		ScopeAST: mustMarshalJSON(map[string]any{
			"file_path":        path,
			"original_hash":    originalHash,
			"file_size":        len(result.Modified),
			"safety_validated": true,
		}),
	}

	// Add session if available
	if s.session != nil {
		stage.SessionID = s.session.ID
	}

	// Save stage
	if err := s.staging.CreateStage(stage); err != nil {
		s.debugLog("Failed to create stage: %v", err)
		return nil, WrapError(InternalError, "Failed to stage transformation", err)
	}

	// Auto-apply if confidence is high
	status := "staged"
	referenceID := stage.ID

	if shouldAutoApply {
		apply, err := s.staging.ApplyStage(stage.ID, true)
		if err != nil {
			s.debugLog("Failed to auto-apply: %v", err)
			responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
			responseText += "\n⚠️ Auto-apply failed, manual review required"
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

	// Build response
	response := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": responseText},
		},
		"result": status,
		"id":     referenceID,
	}

	// Always include modified source when available
	if status == "applied" || result.Modified != "" {
		response["modified"] = result.Modified
	}

	// Include path if in file mode
	if isFileMode {
		response["path"] = path
	}

	return response, nil
}

// formatTransformResponse formats the transformation result text
func (s *StdioServer) formatTransformResponse(method string, result core.TransformResult) string {
	// Operation summary
	var action string
	switch method {
	case "replace":
		action = "Replaced"
	case "delete":
		action = "Deleted"
	case "insert_before":
		action = "Inserted before"
	case "insert_after":
		action = "Inserted after"
	default:
		action = "Transformed"
	}

	text := fmt.Sprintf("%s %d target(s)\n\nConfidence: %s (%.2f)\n",
		action, result.MatchCount, result.Confidence.Level, result.Confidence.Score)

	// Confidence factors
	if len(result.Confidence.Factors) > 0 {
		text += "\nFactors:\n"
		for _, factor := range result.Confidence.Factors {
			sign := "+"
			if factor.Impact < 0 {
				sign = ""
			}
			text += fmt.Sprintf("• %s (%s%.2f): %s\n",
				factor.Name, sign, factor.Impact, factor.Reason)
		}
	}

	// Diff preview
	if result.Diff != "" {
		text += "\nChanges:\n```diff\n" + result.Diff + "\n```"
	}

	return text
}

// calculateSHA256 generates SHA256 hash
func calculateSHA256(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// mustMarshalJSON safely marshals to JSON
func mustMarshalJSON(v any) datatypes.JSON {
	data, err := json.Marshal(v)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(data)
}
