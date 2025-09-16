package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// ReplaceTool handles code element replacement
type ReplaceTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewReplaceTool creates a new replace tool
func NewReplaceTool(server types.ServerInterface) *ReplaceTool {
	tool := &ReplaceTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "replace",
		description: "Replace code elements matching a query",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language":    CommonSchemas.Language,
				"source":      CommonSchemas.Source,
				"path":        CommonSchemas.Path,
				"target":      CommonSchemas.Target,
				"replacement": CommonSchemas.Replacement,
			},
			"required": []string{"language", "target", "replacement"},
			"oneOf": []map[string]any{
				{"required": []string{"source"}},
				{"required": []string{"path"}},
			},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the replace tool
func (t *ReplaceTool) handle(params json.RawMessage) (any, error) {
	var args struct {
		Language    string          `json:"language"`
		Source      string          `json:"source"`
		Path        string          `json:"path"`
		Target      json.RawMessage `json:"target"`
		Replacement string          `json:"replacement"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid replace parameters", err)
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
		Method:      "replace",
		Target:      target,
		Replacement: args.Replacement,
	}

	result := provider.Transform(source, op)
	if result.Error != nil {
		return nil, types.WrapError(types.TransformFailed, "Replace operation failed", result.Error)
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
		"matches":    result.MatchCount,
	}, nil
}

// formatResponse formats the transformation result
func (t *ReplaceTool) formatResponse(result core.TransformResult, path string) string {
	if result.Error != nil {
		return "Replace operation failed: " + result.Error.Error()
	}

	response := "✅ Replace operation completed successfully\n\n"

	if path != "" {
		response += "📄 File: " + path + "\n\n"
	}

	if result.MatchCount > 0 {
		response += fmt.Sprintf("Replacements made: %d\n", result.MatchCount)
	}

	if result.Diff != "" {
		response += "\nChanges:\n" + result.Diff + "\n"
	}

	response += "\nConfidence: " + formatConfidence(result.Confidence.Score)

	return response
}

// formatConfidence formats confidence score as visual indicator
func formatConfidence(confidence float64) string {
	bars := int(confidence * 10)
	filled := strings.Repeat("█", bars)
	empty := strings.Repeat("░", 10-bars)
	return filled + empty + " " + fmt.Sprintf("%.1f%%", confidence*100)
}
