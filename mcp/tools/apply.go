package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/mcp/types"
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

	lifecycle := t.server.GetEngine()
	if lifecycle == nil {
		return nil, types.NewMCPError(types.InvalidParams, "engine lifecycle not available", nil)
	}

	notifyProgress(ctx, t.server, 20, 100, "engine ready")
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

		stage, err := lifecycle.GetStage(ctx, args.ID)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, types.NewMCPError(types.InvalidParams,
					"stage not found: "+args.ID,
					nil)
			}
			return nil, types.WrapError(types.InvalidParams, "failed to load stage", err)
		}
		if stage.Status != engine.StageStatusPending {
			return nil, types.NewMCPError(types.InvalidParams,
				fmt.Sprintf("stage already %s", stage.Status),
				nil)
		}

		if err := t.server.ConfirmApply(ctx, fmt.Sprintf("Apply stage %s", args.ID)); err != nil {
			return nil, err
		}

		result, err := lifecycle.ApplyStage(ctx, engine.StageApplyRequest{ID: args.ID})
		if err != nil {
			return nil, err
		}
		if result.Applied {
			appliedIDs = append(appliedIDs, result.StageID)
		}
		summary = map[string]any{"mode": mode, "stageId": result.StageID}

	case args.All:
		mode = "all"
		notifyProgress(ctx, t.server, 60, 100, "applying all stages")
		stages, err := lifecycle.ListStages(ctx, engine.StageFilter{Status: engine.StageStatusPending})
		if err != nil {
			return nil, types.WrapError(types.InvalidParams, "failed to list stages", err)
		}
		if len(stages) == 0 {
			return nil, types.NewMCPError(types.InvalidParams, "no stages available", nil)
		}
		if err := t.server.ConfirmApply(ctx, "Apply all pending stages"); err != nil {
			return nil, err
		}
		for _, stage := range stages {
			if err := isCancelled(ctx); err != nil {
				return nil, err
			}
			result, err := lifecycle.ApplyStage(ctx, engine.StageApplyRequest{ID: stage.ID})
			if err == nil && result.Applied {
				appliedIDs = append(appliedIDs, result.StageID)
			}
		}
		summary = map[string]any{"mode": mode, "appliedCount": len(appliedIDs)}

	case args.Latest:
		mode = "latest"
		notifyProgress(ctx, t.server, 60, 100, "applying latest stage")
		stages, err := lifecycle.ListStages(ctx, engine.StageFilter{Status: engine.StageStatusPending})
		if err != nil {
			return nil, types.WrapError(types.InvalidParams, "failed to list stages", err)
		}
		if len(stages) == 0 {
			return nil, types.NewMCPError(types.InvalidParams, "no stages available", nil)
		}
		stageID := stages[0].ID // Most recent stage
		if err := t.server.ConfirmApply(ctx, fmt.Sprintf("Apply latest stage %s", stageID)); err != nil {
			return nil, err
		}
		result, err := lifecycle.ApplyStage(ctx, engine.StageApplyRequest{ID: stageID})
		if err != nil {
			return nil, err
		}
		if result.Applied {
			appliedIDs = append(appliedIDs, result.StageID)
		}
		summary = map[string]any{"mode": mode, "stageId": result.StageID}

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
	// Skip sampling for now — most MCP clients don't support sampling/createMessage
	// and will cause the server to hang waiting for a response.
	return nil, nil
}
