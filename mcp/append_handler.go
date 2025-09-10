package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	
	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/models"
	"gorm.io/datatypes"
)

// handleAppendTool executes append operation with optional target
func (s *StdioServer) handleAppendTool(params json.RawMessage) (interface{}, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Target   json.RawMessage `json:"target,omitempty"` // OPTIONAL
		Content  string          `json:"content"`
	}
	
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid append parameters", err)
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
			Method:  "append",  // Append as last element in target
			Target:  target,
			Content: args.Content,
		}
		
		result = provider.Transform(args.Source, op)
		operationDesc = fmt.Sprintf("Appended to %s '%s'", target.Type, target.Name)
		
	} else {
		// NO TARGET - use smart detection
		smartProvider, ok := provider.(interface {
			SmartAppend(source, content string) core.TransformResult
		})
		
		if !ok {
			// Provider doesn't support smart append, fallback to end of file
			result = s.simpleAppendToEnd(args.Source, args.Content)
			operationDesc = "Appended to end of file (no smart detection)"
		} else {
			result = smartProvider.SmartAppend(args.Source, args.Content)
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
			return nil, NewMCPError(NoMatches, "Target not found for append", errMsg)
		}
		return nil, WrapError(TransformFailed, "Append operation failed", result.Error)
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
	
	// If no staging, return direct
	if s.staging == nil {
		status := "completed"
		if shouldAutoApply {
			status = "applied"
		}
		
		if result.Diff != "" {
			responseText += "\nChanges:\n```diff\n" + result.Diff + "\n```"
		}
		
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": responseText},
			},
			"result":   status,
			"modified": result.Modified,
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
		Original:    args.Source,
		Modified:    result.Modified,
		Content:     args.Content,
		Diff:        result.Diff,
		BaseDigest:  calculateSHA256(args.Source),
		AfterDigest: calculateSHA256(result.Modified),
		
		// Confidence
		ConfidenceScore:   result.Confidence.Score,
		ConfidenceLevel:   result.Confidence.Level,
		ConfidenceFactors: mustMarshalJSON(result.Confidence.Factors),
		
		// Store metadata
		ScopeAST: mustMarshalJSON(result.Metadata),
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
			responseText += fmt.Sprintf("\n✅ Auto-applied (ID: %s)", apply.ID)
		}
	} else {
		responseText += fmt.Sprintf("\n📋 Staged for review (ID: %s)", stage.ID)
		responseText += "\nUse 'apply' command to commit changes"
	}
	
	// Add diff
	if result.Diff != "" {
		responseText += "\n\nChanges:\n```diff\n" + result.Diff + "\n```"
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": responseText},
		},
		"result": status,
		"id":     referenceID,
		"modified": func() interface{} {
			if status == "applied" {
				return result.Modified
			}
			return nil
		}(),
	}, nil
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
	
	for _, line := range strings.Split(content, "\n") {
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
		Metadata: map[string]interface{}{
			"strategy": "End of file (fallback)",
		},
	}
}