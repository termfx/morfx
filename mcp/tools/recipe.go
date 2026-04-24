package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/mcp/types"
)

// RecipeTool runs named repeatable transformations through the MCP surface.
type RecipeTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewRecipeTool creates a recipe execution tool.
func NewRecipeTool(server types.ServerInterface) *RecipeTool {
	tool := &RecipeTool{server: server}
	tool.BaseTool = &BaseTool{
		name:        "recipe",
		description: "Run named repeatable transformations with confidence gates",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Recipe name",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Optional recipe description",
				},
				"dry_run": map[string]any{
					"type":        "boolean",
					"description": "Preview changes without applying them",
				},
				"min_confidence": map[string]any{
					"type":        "number",
					"description": "Default confidence gate for all steps",
				},
				"steps": map[string]any{
					"type":        "array",
					"description": "Recipe rules to execute in order",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":           map[string]any{"type": "string"},
							"description":    map[string]any{"type": "string"},
							"method":         map[string]any{"type": "string"},
							"scope":          map[string]any{"type": "object"},
							"target":         CommonSchemas.Target,
							"replacement":    CommonSchemas.Replacement,
							"content":        map[string]any{"type": "string"},
							"min_confidence": map[string]any{"type": "number"},
							"backup":         map[string]any{"type": "boolean"},
						},
						"required": []string{"name", "method", "scope"},
					},
				},
			},
			"required": []string{"name", "steps"},
		},
		handler: tool.handle,
	}
	return tool
}

func (t *RecipeTool) handle(ctx context.Context, params json.RawMessage) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var recipe core.Recipe
	if err := json.Unmarshal(params, &recipe); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid recipe parameters", err)
	}

	notifyProgress(ctx, t.server, 5, 100, "validating recipe")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	opCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	result, err := core.ExecuteRecipe(opCtx, t.server.GetFileProcessor(), recipe)
	if err != nil {
		return nil, types.WrapError(types.TransformFailed, "Recipe failed", err)
	}

	notifyProgress(ctx, t.server, 100, 100, "recipe completed")

	return map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": formatRecipeToolResponse(result),
		}},
		"name":            result.Name,
		"dry_run":         result.DryRun,
		"steps_run":       result.StepsRun,
		"files_scanned":   result.FilesScanned,
		"files_modified":  result.FilesModified,
		"matches":         result.TotalMatches,
		"transaction_ids": result.TransactionIDs,
		"steps":           result.Steps,
	}, nil
}

func formatRecipeToolResponse(result *core.RecipeResult) string {
	if result == nil {
		return "Recipe returned no result"
	}

	mode := ""
	modifiedLabel := "Files modified"
	if result.DryRun {
		mode = " [DRY RUN]"
		modifiedLabel = "Files that would be modified"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Recipe %s completed%s\n\n", result.Name, mode)
	fmt.Fprintf(&builder, "Steps run: %d\n", result.StepsRun)
	fmt.Fprintf(&builder, "Files scanned: %d\n", result.FilesScanned)
	fmt.Fprintf(&builder, "%s: %d\n", modifiedLabel, result.FilesModified)
	fmt.Fprintf(&builder, "Total matches: %d\n", result.TotalMatches)

	if len(result.Steps) > 0 {
		builder.WriteString("\nSteps:\n")
		for _, step := range result.Steps {
			stepResult := step.Result
			if step.AppliedResult != nil {
				stepResult = step.AppliedResult
			}
			if stepResult == nil {
				fmt.Fprintf(&builder, "- %s (%s): no result\n", step.Name, step.Method)
				continue
			}
			fmt.Fprintf(
				&builder,
				"- %s (%s): %d matches, %d files, confidence %.3f >= %.3f\n",
				step.Name,
				step.Method,
				stepResult.TotalMatches,
				stepResult.FilesModified,
				stepResult.Confidence.Score,
				step.MinConfidence,
			)
		}
	}

	if result.DryRun {
		builder.WriteString("\nThis was a dry run. No files were modified.\n")
	}
	return builder.String()
}
