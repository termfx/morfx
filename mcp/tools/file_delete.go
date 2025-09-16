package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

// FileDeleteTool handles deletion across multiple files
type FileDeleteTool struct {
	*BaseTool
	server types.ServerInterface
}

// NewFileDeleteTool creates a new file delete tool
func NewFileDeleteTool(server types.ServerInterface) *FileDeleteTool {
	tool := &FileDeleteTool{
		server: server,
	}

	tool.BaseTool = &BaseTool{
		name:        "file_delete",
		description: "Delete code elements across multiple files",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"scope": map[string]any{
					"type":        "object",
					"description": "File scope to process",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Root directory path",
						},
						"include": map[string]any{
							"type":        "array",
							"description": "File patterns to include",
							"items":       map[string]any{"type": "string"},
						},
					},
					"required": []string{"path"},
				},
				"target": CommonSchemas.Target,
				"dry_run": map[string]any{
					"type":        "boolean",
					"description": "Preview changes without applying",
				},
				"backup": map[string]any{
					"type":        "boolean",
					"description": "Create backup files",
				},
			},
			"required": []string{"scope", "target"},
		},
		handler: tool.handle,
	}

	return tool
}

// handle executes the file delete tool
func (t *FileDeleteTool) handle(params json.RawMessage) (any, error) {
	var args struct {
		Scope  core.FileScope  `json:"scope"`
		Target json.RawMessage `json:"target"`
		DryRun bool            `json:"dry_run"`
		Backup bool            `json:"backup"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid file delete parameters", err)
	}

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(args.Target, &target); err != nil {
		return nil, types.WrapError(types.InvalidParams, "Invalid target structure", err)
	}

	// Create transform operation
	fileOp := core.FileTransformOp{
		TransformOp: core.TransformOp{
			Method: "delete",
			Target: target,
		},
		Scope:    args.Scope,
		DryRun:   args.DryRun,
		Backup:   args.Backup,
		Parallel: true,
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fileProcessor := t.server.GetFileProcessor()
	result, err := fileProcessor.TransformFiles(ctx, fileOp)
	if err != nil {
		return nil, types.WrapError(types.TransformFailed, "File delete failed", err)
	}

	// Format response
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": t.formatResponse(result, args.DryRun),
			},
		},
		"files_processed": result.FilesScanned,
		"files_modified":  result.FilesModified,
		"dry_run":         args.DryRun,
	}, nil
}

// formatResponse formats the file delete results
func (t *FileDeleteTool) formatResponse(result *core.FileTransformResult, dryRun bool) string {
	mode := ""
	if dryRun {
		mode = " [DRY RUN]"
	}

	response := fmt.Sprintf("✅ File delete operation completed%s\n\n", mode)
	response += fmt.Sprintf("Files scanned: %d\n", result.FilesScanned)
	response += fmt.Sprintf("Files modified: %d\n", result.FilesModified)
	response += fmt.Sprintf("Total deletions: %d\n", result.TotalMatches)

	if len(result.Files) > 0 {
		response += "\nModified files:\n"
		for _, file := range result.Files {
			if file.MatchCount > 0 {
				response += fmt.Sprintf("📄 %s: %d deletions\n", file.FilePath, file.MatchCount)
			}
		}
	}

	if dryRun {
		response += "\n⚠️  This was a dry run. No files were actually modified."
	}

	return response
}
