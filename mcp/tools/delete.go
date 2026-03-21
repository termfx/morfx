package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// DeleteTool handles code element deletion
type DeleteTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewDeleteTool creates a new delete tool
func NewDeleteTool(server types.ServerInterface) *DeleteTool {
	tool := &DeleteTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "delete",
		description: "Delete code elements matching a query",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language": CommonSchemas.Language,
				"source":   CommonSchemas.Source,
				"path":     CommonSchemas.Path,
				"target":   CommonSchemas.Target,
			},
			"required": []string{"language", "target"},
			"oneOf": []map[string]any{
				{"required": []string{"source"}},
				{"required": []string{"path"}},
			},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the delete tool
func (t *DeleteTool) handle(ctx context.Context, params json.RawMessage) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid delete parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, types.NewMCPError(types.InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	notifyProgress(ctx, t.server, 5, 100, "validating")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	// Get source code
	var source string
	if args.Path != "" {
		content, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, types.WrapError(types.FileSystemError, "Failed to read file", err)
		}
		source = string(content)
		notifyProgress(ctx, t.server, 15, 100, "loaded file")
	} else {
		source = args.Source
	}
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	// Get provider
	provider, exists := t.server.GetProviders().Get(args.Language)
	if !exists {
		return nil, types.NewMCPError(types.LanguageNotFound, "Language not supported", nil)
	}
	notifyProgress(ctx, t.server, 25, 100, "resolved provider")

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(args.Target, &target); err != nil {
		return nil, types.WrapError(types.InvalidParams, "target must be an object", err)
	}
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	// Execute transformation
	op := core.TransformOp{
		Method: "delete",
		Target: target,
	}

	result := provider.Transform(source, op)
	if result.Error != nil {
		return nil, types.WrapError(types.TransformFailed, "Delete operation failed", result.Error)
	}
	notifyProgress(ctx, t.server, 70, 100, "transformed source")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	notifyProgress(ctx, t.server, 90, 100, "finalizing")

	return t.server.FinalizeTransform(ctx, types.TransformRequest{
		Language:       args.Language,
		Operation:      "delete",
		Target:         target,
		TargetJSON:     args.Target,
		Path:           args.Path,
		OriginalSource: source,
		Result:         result,
		ResponseText:   t.formatResponse(result, args.Path),
	})
}

// formatResponse formats the deletion result
func (t *DeleteTool) formatResponse(result core.TransformResult, path string) string {
	if result.Error != nil {
		return "Delete operation failed: " + result.Error.Error()
	}

	response := "✅ Delete operation completed successfully\n\n"

	if path != "" {
		response += "📄 File: " + path + "\n\n"
	}

	if result.MatchCount > 0 {
		response += fmt.Sprintf("Deletions made: %d\n", result.MatchCount)
	}

	response += fmt.Sprintf("\nConfidence: %.1f%%", result.Confidence.Score*100)

	return response
}
