package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// QueryTool handles code element queries
type QueryTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewQueryTool creates a new query tool
func NewQueryTool(server types.ServerInterface) *QueryTool {
	tool := &QueryTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "query",
		description: "Find code elements using natural language queries",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language": CommonSchemas.Language,
				"source":   CommonSchemas.Source,
				"path":     CommonSchemas.Path,
				"query":    CommonSchemas.Query,
			},
			"required": []string{"language", "query"},
			"oneOf": []map[string]any{
				{"required": []string{"source"}},
				{"required": []string{"path"}},
			},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the query tool
func (t *QueryTool) handle(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   *string         `json:"source,omitempty"`
		Path     *string         `json:"path,omitempty"`
		Query    json.RawMessage `json:"query"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid query parameters", err)
	}

	// Validate that exactly one of source or path is provided
	sourceProvided := args.Source != nil
	pathProvided := args.Path != nil

	if (!sourceProvided && !pathProvided) || (sourceProvided && pathProvided) {
		return nil, types.NewMCPError(types.InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	// Get source code
	var source string
	if pathProvided {
		// FILE WRITER MODE: Read from filesystem
		content, err := os.ReadFile(*args.Path)
		if err != nil {
			return nil, types.WrapError(types.FileSystemError, "Failed to read file", err)
		}
		source = string(content)
	} else {
		// IN-MEMORY MODE: Use provided source (can be empty string)
		source = *args.Source
	}

	// Get provider for language
	provider, exists := t.server.GetProviders().Get(args.Language)
	if !exists {
		return nil, types.NewMCPError(types.LanguageNotFound,
			fmt.Sprintf("No provider for language: %s", args.Language),
			map[string]any{
				"requested": args.Language,
				"supported": []string{"go", "python", "javascript", "typescript", "php"},
			})
	}

	// Parse the query
	var query core.AgentQuery
	if err := json.Unmarshal(args.Query, &query); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid query structure", err)
	}

	// Execute query
	result := provider.Query(source, query)
	if result.Error != nil {
		// Check if it's a syntax error or other
		errMsg := result.Error.Error()
		if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "syntax") {
			return nil, types.NewMCPError(types.SyntaxError, "Failed to parse source code",
				map[string]any{"details": errMsg})
		}
		return nil, types.WrapError(types.TransformFailed, "Query execution failed", result.Error)
	}

	// Format matches as human-readable text
	var responseText string
	if len(result.Matches) == 0 {
		responseText = "No matches found"
	} else {
		responseText = fmt.Sprintf("Found %d match", len(result.Matches))
		if len(result.Matches) != 1 {
			responseText += "es"
		}
		responseText += ":\n\n"

		for _, match := range result.Matches {
			responseText += fmt.Sprintf("• %s '%s' at line %d, column %d",
				match.Type, match.Name,
				match.Location.Line, match.Location.Column)
			if match.Content != "" {
				responseText += fmt.Sprintf("\n  Content: %s", match.Content)
			}
			responseText += "\n"
		}
	}

	// Add file info if in FILE WRITER MODE
	if pathProvided {
		responseText = fmt.Sprintf("File: %s\n\n%s", *args.Path, responseText)
	}

	// Return as MCP content blocks
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
	}, nil
}
