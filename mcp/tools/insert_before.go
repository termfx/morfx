package tools

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// InsertBeforeTool handles inserting code before elements
type InsertBeforeTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewInsertBeforeTool creates a new insert before tool
func NewInsertBeforeTool(server types.ServerInterface) *InsertBeforeTool {
	tool := &InsertBeforeTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "insert_before",
		description: "Insert code before elements matching a query",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language": CommonSchemas.Language,
				"source":   CommonSchemas.Source,
				"path":     CommonSchemas.Path,
				"target":   CommonSchemas.Target,
				"content": map[string]any{
					"type":        "string",
					"description": "Code to insert",
				},
			},
			"required": []string{"language", "target", "content"},
			"oneOf": []map[string]any{
				{"required": []string{"source"}},
				{"required": []string{"path"}},
			},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the insert before tool
func (t *InsertBeforeTool) handle(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Target   json.RawMessage `json:"target"`
		Content  string          `json:"content"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid insert_before parameters", err)
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
		return nil, types.WrapError(types.InvalidParams, "Invalid target structure", err)
	}

	// Execute transformation
	op := core.TransformOp{
		Method:      "insert_before",
		Target:      target,
		Replacement: args.Content,
	}

	result := provider.Transform(source, op)
	if result.Error != nil {
		return nil, types.WrapError(types.TransformFailed, "Insert before operation failed", result.Error)
	}

	// Write back if file mode
	if args.Path != "" && result.Modified != "" {
		if err := os.WriteFile(args.Path, []byte(result.Modified), 0o644); err != nil {
			return nil, types.WrapError(types.FileSystemError, "Failed to write file", err)
		}
	}

	// Return response
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

// formatResponse formats the insertion result
func (t *InsertBeforeTool) formatResponse(result core.TransformResult, path string) string {
	if result.Error != nil {
		return "Insert before operation failed"
	}

	response := "✅ Insert before operation completed successfully\n\n"

	if path != "" {
		response += "📄 File: " + path + "\n\n"
	}

	response += "Insertions made:\n"
	if result.MatchCount > 0 {
		response += fmt.Sprintf("  %d locations modified\n", result.MatchCount)
	}

	response += fmt.Sprintf("\nConfidence: %.1f%%", result.Confidence.Score*100)

	return response
}
