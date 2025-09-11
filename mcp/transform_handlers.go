package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		Target      json.RawMessage `json:"target"`
		Replacement string          `json:"replacement"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid replace parameters", err)
	}

	return s.executeTransform(args.Language, args.Source, args.Target, core.TransformOp{
		Method:      "replace",
		Replacement: args.Replacement,
	})
}

// handleDeleteTool executes deletion transformation with staging
func (s *StdioServer) handleDeleteTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Target   json.RawMessage `json:"target"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid delete parameters", err)
	}

	return s.executeTransform(args.Language, args.Source, args.Target, core.TransformOp{
		Method: "delete",
	})
}

// handleInsertBeforeTool executes insert before transformation with staging
func (s *StdioServer) handleInsertBeforeTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Target   json.RawMessage `json:"target"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid insert_before parameters", err)
	}

	return s.executeTransform(args.Language, args.Source, args.Target, core.TransformOp{
		Method:  "insert_before",
		Content: args.Content,
	})
}

// handleInsertAfterTool executes insert after transformation with staging
func (s *StdioServer) handleInsertAfterTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Target   json.RawMessage `json:"target"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid insert_after parameters", err)
	}

	return s.executeTransform(args.Language, args.Source, args.Target, core.TransformOp{
		Method:  "insert_after",
		Content: args.Content,
	})
}

// executeTransform is the common transformation logic with staging
func (s *StdioServer) executeTransform(
	language, source string,
	targetJSON json.RawMessage,
	op core.TransformOp,
) (any, error) {
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
	result := provider.Transform(source, op)
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

	// Determine if should auto-apply
	shouldAutoApply := s.config.AutoApplyEnabled &&
		result.Confidence.Score >= s.config.AutoApplyThreshold

	// Create response text
	responseText := s.formatTransformResponse(op.Method, result)

	// If no staging available, return direct result
	if s.staging == nil {
		status := "completed"
		if shouldAutoApply {
			status = "applied"
			responseText += "\n✅ Changes applied (no staging available)"
		} else {
			responseText += "\n⚠️ Low confidence - review recommended"
		}

		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": responseText},
			},
			"result":   status,
			"modified": result.Modified,
		}, nil
	}

	// Create stage
	stage := &models.Stage{
		ID:        generateID("stg"),
		Language:  language,
		Operation: op.Method,

		// Target info
		TargetType:  target.Type,
		TargetName:  target.Name,
		TargetQuery: datatypes.JSON(targetJSON),

		// Content
		Original:    source,
		Modified:    result.Modified,
		Content:     op.Content,
		Diff:        result.Diff,
		BaseDigest:  calculateSHA256(source),
		AfterDigest: calculateSHA256(result.Modified),

		// Confidence
		ConfidenceScore:   result.Confidence.Score,
		ConfidenceLevel:   result.Confidence.Level,
		ConfidenceFactors: mustMarshalJSON(result.Confidence.Factors),
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
			responseText += fmt.Sprintf("\n✅ Auto-applied (ID: %s)", apply.ID)
		}
	} else {
		responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
		responseText += "\nUse 'apply' command to commit changes"
	}

	// Build response
	response := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": responseText},
		},
		"result": status,
		"id":     referenceID,
	}

	// Always include modified source when available (either applied or when it would auto-apply)
	if status == "applied" || (shouldAutoApply && result.Modified != "") {
		response["modified"] = result.Modified
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
