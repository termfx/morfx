package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/termfx/morfx/mcp/types"
	"gorm.io/gorm"
)

// ApplyTool handles applying staged transformations
type ApplyTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewApplyTool creates a new apply tool
func NewApplyTool(server types.ServerInterface) *ApplyTool {
	tool := &ApplyTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "apply",
		description: "Apply staged code transformations",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Specific stage ID to apply",
				},
				"all": map[string]any{
					"type":        "boolean",
					"description": "Apply all pending stages",
				},
				"latest": map[string]any{
					"type":        "boolean",
					"description": "Apply the most recent pending stage",
				},
			},
			"required": []string{},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the apply tool
func (t *ApplyTool) handle(ctx context.Context, params json.RawMessage) (any, error) {
	var args struct {
		ID     string `json:"id,omitempty"`
		All    bool   `json:"all,omitempty"`
		Latest bool   `json:"latest,omitempty"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid apply parameters", err)
	}

	notifyProgress(ctx, t.server, 5, 100, "validating")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	stagingRaw := t.server.GetStaging()
	if stagingRaw == nil {
		return nil, types.NewMCPError(types.InvalidParams,
			"Staging not available",
			map[string]any{"reason": "Database connection required for staging"})
	}

	staging, ok := stagingRaw.(types.StagingManager)
	if !ok {
		return nil, types.NewMCPError(types.InvalidParams,
			"Staging manager does not implement required interface",
			nil)
	}

	if !staging.IsEnabled() {
		return nil, types.NewMCPError(types.InvalidParams, "staging is not enabled", nil)
	}

	sessionID := t.server.GetSessionID()

	notifyProgress(ctx, t.server, 20, 100, "staging ready")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	mode := ""
	appliedIDs := make([]string, 0)

	paramCount := 0
	if args.ID != "" {
		paramCount++
	}
	if args.All {
		paramCount++
	}
	if args.Latest {
		paramCount++
	}
	if paramCount > 1 {
		return nil, types.NewMCPError(types.InvalidParams,
			"conflicting parameters: specify only one of 'id', 'all', or 'latest'",
			nil)
	}
	if paramCount == 0 {
		args.Latest = true
	}

	notifyProgress(ctx, t.server, 35, 100, "prepared request")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	var summary map[string]any

	switch {
	case args.ID != "":
		mode = "single"
		notifyProgress(ctx, t.server, 60, 100, "applying stage")

		if sessionID == "" {
			return nil, types.NewMCPError(types.InvalidParams,
				"staging session unavailable",
				map[string]any{"reason": "server did not negotiate persistence"})
		}
		stage, err := staging.GetStage(args.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, types.NewMCPError(types.InvalidParams,
					"stage not found: "+args.ID,
					nil)
			}
			return nil, types.WrapError(types.InvalidParams, "failed to load stage", err)
		}
		if stage != nil && stage.SessionID != "" && stage.SessionID != sessionID {
			return nil, types.NewMCPError(types.InvalidParams,
				fmt.Sprintf("stage %s belongs to a different session", args.ID),
				nil)
		}
		if stage != nil && stage.Status != "pending" {
			return nil, types.NewMCPError(types.InvalidParams,
				fmt.Sprintf("stage already %s", stage.Status),
				nil)
		}

		if err := t.server.ConfirmApply(ctx, fmt.Sprintf("Apply stage %s", args.ID)); err != nil {
			return nil, err
		}

		if _, err := staging.ApplyStage(ctx, args.ID, false); err != nil {
			return nil, err
		}
		appliedIDs = append(appliedIDs, args.ID)
		summary = map[string]any{"mode": mode, "stageId": args.ID}

	case args.All:
		mode = "all"
		notifyProgress(ctx, t.server, 60, 100, "applying all stages")
		if err := t.server.ConfirmApply(ctx, "Apply all pending stages"); err != nil {
			return nil, err
		}

		stages, err := staging.ListPendingStages(sessionID)
		if err != nil || len(stages) == 0 {
			return nil, types.NewMCPError(types.InvalidParams, "no stages available", nil)
		}
		for _, stage := range stages {
			if err := isCancelled(ctx); err != nil {
				return nil, err
			}
			if _, err := staging.ApplyStage(ctx, stage.ID, false); err == nil {
				appliedIDs = append(appliedIDs, stage.ID)
			}
		}
		summary = map[string]any{"mode": mode, "appliedCount": len(appliedIDs)}

	case args.Latest:
		mode = "latest"
		notifyProgress(ctx, t.server, 60, 100, "applying latest stage")
		stages, err := staging.ListPendingStages(sessionID)
		if err != nil || len(stages) == 0 {
			return nil, types.NewMCPError(types.InvalidParams, "no stages available", nil)
		}
		stageID := stages[0].ID // Most recent stage
		if err := t.server.ConfirmApply(ctx, fmt.Sprintf("Apply latest stage %s", stageID)); err != nil {
			return nil, err
		}
		if _, err := staging.ApplyStage(ctx, stageID, false); err != nil {
			return nil, err
		}
		appliedIDs = append(appliedIDs, stageID)
		summary = map[string]any{"mode": mode, "stageId": stageID}

	default:
		return nil, types.NewMCPError(types.InvalidParams, "unsupported apply parameters", nil)
	}

	structured := map[string]any{"mode": mode}
	if len(appliedIDs) > 0 {
		structured["applied"] = append([]string{}, appliedIDs...)
	}
	if mode == "all" {
		structured["appliedCount"] = len(appliedIDs)
	}

	sampling, err := t.sampleApply(ctx, summary)
	if err != nil {
		return nil, err
	}
	if sampling != nil {
		structured["sampling"] = sampling
	}

	notifyProgress(ctx, t.server, 90, 100, "completed")

	message := "Apply operation completed"
	switch mode {
	case "single":
		message = "Applied stage: " + appliedIDs[0]
	case "latest":
		message = "Applied latest stage: " + appliedIDs[0]
	case "all":
		message = fmt.Sprintf("Applied %d stage(s)", len(appliedIDs))
	}

	return map[string]any{
		"content":           []map[string]any{{"type": "text", "text": message}},
		"applied":           appliedIDs,
		"structuredContent": structured,
	}, nil
}

func (t *ApplyTool) sampleApply(ctx context.Context, summary map[string]any) (map[string]any, error) {
	if t.server == nil {
		return nil, nil
	}
	if summary == nil {
		summary = map[string]any{}
	}

	notifyProgress(ctx, t.server, 72, 100, "requesting sampling")
	payload := map[string]any{
		"workflow": "apply",
		"summary":  summary,
	}

	resp, err := t.server.RequestSampling(ctx, payload)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return map[string]any{"error": err.Error()}, nil
	}

	notifyProgress(ctx, t.server, 78, 100, "sampling complete")
	return resp, nil
}
