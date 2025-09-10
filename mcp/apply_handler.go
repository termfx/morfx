package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

// handleApplyTool applies a staged transformation
func (s *StdioServer) handleApplyTool(params json.RawMessage) (interface{}, error) {
	var args struct {
		ID     string `json:"id,omitempty"`      // Specific stage ID
		All    bool   `json:"all,omitempty"`     // Apply all pending stages
		Latest bool   `json:"latest,omitempty"`  // Apply latest stage
	}
	
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid apply parameters", err)
	}
	
	// Check if staging is available
	if s.staging == nil {
		return nil, NewMCPError(InternalError, "Staging not available", 
			"Database connection required for staging")
	}
	
	// Handle different apply modes
	if args.ID != "" {
		// Apply specific stage
		return s.applySpecificStage(args.ID)
	}
	
	if args.All {
		// Apply all pending stages for session
		return s.applyAllStages()
	}
	
	if args.Latest {
		// Apply the most recent pending stage
		return s.applyLatestStage()
	}
	
	// Default: show pending stages
	return s.listPendingStages()
}

// applySpecificStage applies a stage by ID
func (s *StdioServer) applySpecificStage(stageID string) (interface{}, error) {
	// Get the stage first to validate
	stage, err := s.staging.GetStage(stageID)
	if err != nil {
		return nil, NewMCPError(StageNotFound, fmt.Sprintf("Stage not found: %s", stageID))
	}
	
	// Check if already applied
	if stage.Status != "pending" {
		return nil, NewMCPError(AlreadyApplied, 
			fmt.Sprintf("Stage is %s, not pending", stage.Status))
	}
	
	// Check expiration
	if time.Now().After(stage.ExpiresAt) {
		return nil, NewMCPError(StageExpired, "Stage has expired",
			fmt.Sprintf("Expired at %s", stage.ExpiresAt.Format(time.RFC3339)))
	}
	
	// Apply the stage
	apply, err := s.staging.ApplyStage(stageID, false)
	if err != nil {
		return nil, WrapError(TransformFailed, "Failed to apply stage", err)
	}
	
	// Format response
	responseText := fmt.Sprintf("✅ Applied stage %s\n", stageID)
	responseText += fmt.Sprintf("Apply ID: %s\n", apply.ID)
	responseText += fmt.Sprintf("\nOperation: %s %s '%s'\n", 
		stage.Operation, stage.TargetType, stage.TargetName)
	
	if stage.Diff != "" {
		responseText += "\nChanges applied:\n```diff\n" + stage.Diff + "\n```"
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": responseText},
		},
		"result":   "applied",
		"id":       apply.ID,
		"modified": stage.Modified,
	}, nil
}

// applyAllStages applies all pending stages for the session
func (s *StdioServer) applyAllStages() (interface{}, error) {
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	
	// Get all pending stages
	stages, err := s.staging.ListPendingStages(sessionID)
	if err != nil {
		return nil, WrapError(InternalError, "Failed to list stages", err)
	}
	
	if len(stages) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "No pending stages to apply"},
			},
			"result": "completed",
		}, nil
	}
	
	// Apply each stage
	applied := 0
	failed := 0
	var details []string
	
	for _, stage := range stages {
		if time.Now().After(stage.ExpiresAt) {
			failed++
			details = append(details, fmt.Sprintf("• %s: expired", stage.ID))
			continue
		}
		
		_, err := s.staging.ApplyStage(stage.ID, false)
		if err != nil {
			failed++
			details = append(details, fmt.Sprintf("• %s: %v", stage.ID, err))
		} else {
			applied++
			details = append(details, fmt.Sprintf("• %s: applied", stage.ID))
		}
	}
	
	// Format response
	responseText := fmt.Sprintf("Applied %d stages, %d failed\n\n", applied, failed)
	for _, detail := range details {
		responseText += detail + "\n"
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": responseText},
		},
		"result": "completed",
		"applied": applied,
		"failed": failed,
	}, nil
}

// applyLatestStage applies the most recent pending stage
func (s *StdioServer) applyLatestStage() (interface{}, error) {
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	
	// Get pending stages
	stages, err := s.staging.ListPendingStages(sessionID)
	if err != nil {
		return nil, WrapError(InternalError, "Failed to list stages", err)
	}
	
	if len(stages) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "No pending stages to apply"},
			},
			"result": "completed",
		}, nil
	}
	
	// Apply the first (most recent) stage
	return s.applySpecificStage(stages[0].ID)
}

// listPendingStages shows all pending stages
func (s *StdioServer) listPendingStages() (interface{}, error) {
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	
	// Get pending stages
	stages, err := s.staging.ListPendingStages(sessionID)
	if err != nil {
		return nil, WrapError(InternalError, "Failed to list stages", err)
	}
	
	if len(stages) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "No pending stages"},
			},
			"result": "completed",
		}, nil
	}
	
	// Format stage list
	responseText := fmt.Sprintf("📋 %d pending stage(s):\n\n", len(stages))
	
	for i, stage := range stages {
		responseText += fmt.Sprintf("%d. %s (ID: %s)\n", i+1, stage.ID, stage.ID[:8])
		responseText += fmt.Sprintf("   Operation: %s %s '%s'\n", 
			stage.Operation, stage.TargetType, stage.TargetName)
		responseText += fmt.Sprintf("   Confidence: %s (%.2f)\n", 
			stage.ConfidenceLevel, stage.ConfidenceScore)
		
		// Check expiration
		remaining := time.Until(stage.ExpiresAt)
		if remaining < 0 {
			responseText += "   Status: ❌ EXPIRED\n"
		} else if remaining < 5*time.Minute {
			responseText += fmt.Sprintf("   Expires: ⚠️ %v\n", remaining.Round(time.Second))
		} else {
			responseText += fmt.Sprintf("   Expires: %v\n", remaining.Round(time.Minute))
		}
		responseText += "\n"
	}
	
	responseText += "To apply:\n"
	responseText += "• apply {id} - Apply specific stage\n"
	responseText += "• apply --latest - Apply most recent stage\n"
	responseText += "• apply --all - Apply all pending stages"
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": responseText},
		},
		"result": "completed",
		"count":  len(stages),
	}, nil
}