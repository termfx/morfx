package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// AppendTool handles appending code to elements or files
type AppendTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewAppendTool creates a new append tool
func NewAppendTool(server types.ServerInterface) *AppendTool {
	tool := &AppendTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "append",
		description: "Append code to source - uses target if specified, otherwise intelligently places content",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language": CommonSchemas.Language,
				"source":   CommonSchemas.Source,
				"path":     CommonSchemas.Path,
				"content": map[string]any{
					"type":        "string",
					"description": "Code to append",
				},
				"target": map[string]any{
					"type":        "object",
					"description": "Optional target scope (struct, function, etc)",
					"properties": map[string]any{
						"type": map[string]any{"type": "string"},
						"name": map[string]any{"type": "string"},
					},
				},
			},
			"required": []string{"language", "content"},
			"oneOf": []map[string]any{
				{"required": []string{"source"}},
				{"required": []string{"path"}},
			},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the append tool
func (t *AppendTool) handle(ctx context.Context, params json.RawMessage) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target,omitempty"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid append parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, types.NewMCPError(types.InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}
	notifyProgress(ctx, t.server, 5, 100, "validating")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	// Validate that content is provided (can be empty string, but must be present)
	// Check if the field was actually provided in the JSON
	var rawArgs map[string]json.RawMessage
	if err := json.Unmarshal(params, &rawArgs); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid parameters", err)
	}
	if _, hasContent := rawArgs["content"]; !hasContent {
		return nil, types.NewMCPError(types.InvalidParams, "Missing required parameter: content", nil)
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

	// Build transform operation
	op := core.TransformOp{
		Method:  "append",
		Content: args.Content,
	}

	// Parse optional target
	if len(args.Target) > 0 && string(args.Target) != "null" {
		var target core.AgentQuery
		if err := json.Unmarshal(args.Target, &target); err != nil {
			return nil, types.WrapError(types.InvalidParams, "Invalid target structure", err)
		}
		op.Target = target
	}
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	// Execute transformation
	result := provider.Transform(source, op)
	if result.Error != nil {
		return nil, types.WrapError(types.TransformFailed, "Append operation failed", result.Error)
	}
	notifyProgress(ctx, t.server, 70, 100, "transformed source")
	if err := isCancelled(ctx); err != nil {
		return nil, err
	}

	notifyProgress(ctx, t.server, 90, 100, "finalizing")

	return t.server.FinalizeTransform(ctx, types.TransformRequest{
		Language:       args.Language,
		Operation:      "append",
		Target:         op.Target,
		TargetJSON:     args.Target,
		Path:           args.Path,
		OriginalSource: source,
		Result:         result,
		ResponseText:   t.formatResponse(result, args.Path),
		Content:        args.Content,
	})
}

// formatResponse formats the append result
func (t *AppendTool) formatResponse(result core.TransformResult, path string) string {
	if result.Error != nil {
		return "Append operation failed"
	}

	response := "✅ Append operation completed successfully\n\n"

	if path != "" {
		response += "📄 File: " + path + "\n\n"
	}

	response += "Content appended:\n"
	if result.MatchCount > 0 {
		response += fmt.Sprintf("  %d locations modified\n", result.MatchCount)
	}

	response += fmt.Sprintf("\nConfidence: %.1f%%", result.Confidence.Score*100)

	return response
}
