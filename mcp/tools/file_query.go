package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// FileQueryTool handles code queries across multiple files
type FileQueryTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewFileQueryTool creates a new file query tool
func NewFileQueryTool(server types.ServerInterface) *FileQueryTool {
	tool := &FileQueryTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "file_query",
		description: "Find code elements across multiple files using natural language queries",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"scope": map[string]any{
					"type":        "object",
					"description": "File scope to search",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Root directory path to scan",
						},
						"include": map[string]any{
							"type":        "array",
							"description": "File patterns to include (*.go, **/*.ts)",
							"items":       map[string]any{"type": "string"},
						},
						"exclude": map[string]any{
							"type":        "array",
							"description": "File patterns to exclude",
							"items":       map[string]any{"type": "string"},
						},
						"language": map[string]any{
							"type":        "string",
							"description": "Programming language filter",
						},
						"max_files": map[string]any{
							"type":        "integer",
							"description": "Maximum files to process (0 = unlimited)",
						},
					},
					"required": []string{"path"},
				},
				"query": CommonSchemas.Query,
			},
			"required": []string{"scope", "query"},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the file query tool
func (t *FileQueryTool) handle(params json.RawMessage) (any, error) {
	var args struct {
		Scope *core.FileScope `json:"scope"`
		Query json.RawMessage `json:"query"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid file query parameters", err)
	}

	// Validate scope is provided
	if args.Scope == nil {
		return nil, types.NewMCPError(types.InvalidParams, "scope is required", nil)
	}

	// Validate scope path
	if args.Scope.Path == "" {
		return nil, types.NewMCPError(types.InvalidParams, "scope.path is required", nil)
	}

	// Check if path exists
	if _, err := os.Stat(args.Scope.Path); os.IsNotExist(err) {
		return nil, types.NewMCPError(types.InvalidParams, "path does not exist: "+args.Scope.Path, nil)
	}

	// Parse the query
	var query core.AgentQuery
	if err := json.Unmarshal(args.Query, &query); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid query structure", err)
	}

	// Execute file query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileProcessor := t.server.GetFileProcessor()
	matches, err := fileProcessor.QueryFiles(ctx, *args.Scope, query)
	if err != nil {
		return nil, types.WrapError(types.TransformFailed, "File query failed", err)
	}

	// Format response
	responseText := t.formatFileQueryResponse(matches, *args.Scope)

	// Format response for tests compatibility
	// Always use the map format for content to be consistent
	fileList := make([]any, 0)
	if len(matches) > 0 {
		seen := make(map[string]bool)
		for _, m := range matches {
			if !seen[m.FilePath] {
				seen[m.FilePath] = true
				fileList = append(fileList, map[string]any{
					"path":    m.FilePath,
					"matches": 1,
				})
			}
		}
	}

	return map[string]any{
		"content": map[string]any{
			"type":  "text",
			"text":  responseText,
			"files": fileList,
		},
		"matches": len(matches),
		"files":   t.countUniqueFiles(matches),
	}, nil
}

// formatFileQueryResponse formats file query matches as human-readable text
func (t *FileQueryTool) formatFileQueryResponse(matches []core.FileMatch, scope core.FileScope) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found in %s", scope.Path)
	}

	// Group matches by file
	fileGroups := make(map[string][]core.FileMatch)
	for _, fm := range matches {
		fileGroups[fm.FilePath] = append(fileGroups[fm.FilePath], fm)
	}

	var response string
	response = fmt.Sprintf("Found %d matches across %d files in %s:\n\n",
		len(matches), len(fileGroups), scope.Path)

	for filePath, fileMatches := range fileGroups {
		response += fmt.Sprintf("📄 %s (%d matches):\n", filePath, len(fileMatches))
		for _, match := range fileMatches {
			response += fmt.Sprintf("  • %s '%s' at line %d, column %d\n",
				match.Type, match.Name,
				match.Location.Line, match.Location.Column)
			if match.Content != "" {
				// Show first line of content
				firstLine := match.Content
				if idx := len(firstLine); idx > 80 {
					firstLine = firstLine[:77] + "..."
				}
				response += fmt.Sprintf("    %s\n", firstLine)
			}
		}
		response += "\n"
	}

	return response
}

// countUniqueFiles counts unique files in matches
func (t *FileQueryTool) countUniqueFiles(matches []core.FileMatch) int {
	unique := make(map[string]bool)
	for _, m := range matches {
		unique[m.FilePath] = true
	}
	return len(unique)
}
