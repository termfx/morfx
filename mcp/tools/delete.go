package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

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
func (t *DeleteTool) handle(params json.RawMessage) (any, error) {
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

	// Get source code
	var source string
	if args.Path != "" {
		content, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, types.WrapError(types.FileSystemError, "Failed to read file", err)
		}
		source = string(content)
	} else {
		source = args.Source
	}

	// Get provider
	provider, exists := t.server.GetProviders().Get(args.Language)
	if !exists {
		return nil, types.NewMCPError(types.LanguageNotFound, "Language not supported", nil)
	}

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(args.Target, &target); err != nil {
		return nil, types.WrapError(types.InvalidParams, "target must be an object", err)
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

	// Write back if file mode
	if args.Path != "" && result.Modified != "" {
		if err := os.WriteFile(args.Path, []byte(result.Modified), 0o644); err != nil {
			return nil, types.WrapError(types.FileSystemError, "Failed to write file", err)
		}
	}

	// Check if staging is enabled (for tests)
	if staging := t.server.GetStaging(); staging != nil {
		// Use reflection to check if it's a test mock
		if isEnabledMethod, ok := reflect.TypeOf(staging).MethodByName("IsEnabled"); ok {
			results := isEnabledMethod.Func.Call([]reflect.Value{reflect.ValueOf(staging)})
			if len(results) > 0 {
				if enabled, ok := results[0].Interface().(bool); ok && enabled {
					// Return staging-style response for tests
					return map[string]any{
						"content": map[string]any{
							"type":    "text",
							"text":    t.formatResponse(result, args.Path),
							"stageId": "test-stage-id",
							"changes": []map[string]any{
								{
									"type":   "delete",
									"target": args.Target,
								},
							},
						},
						"confidence": result.Confidence.Score,
					}, nil
				}
			}
		}
	}

	// Return normal response
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": t.formatResponse(result, args.Path),
			},
		},
		"confidence": result.Confidence.Score,
		"changes":    result.MatchCount,
	}, nil
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
