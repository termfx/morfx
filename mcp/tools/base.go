package tools

import (
	"context"
	"encoding/json"

	"github.com/termfx/morfx/mcp/types"
)

// BaseTool provides common tool functionality
type BaseTool struct {
	name        string
	description string
	inputSchema map[string]any
	handler     types.ToolHandler
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// InputSchema returns the tool's input schema
func (t *BaseTool) InputSchema() map[string]any {
	return t.inputSchema
}

// Handler returns the tool's handler function
func (t *BaseTool) Handler() types.ToolHandler {
	return t.handler
}

// ToolBuilder helps construct tools with fluent interface
type ToolBuilder struct {
	tool *BaseTool
}

// NewTool creates a new tool builder
func NewTool(name string) *ToolBuilder {
	return &ToolBuilder{
		tool: &BaseTool{
			name:        name,
			inputSchema: make(map[string]any),
		},
	}
}

// WithDescription sets the tool description
func (b *ToolBuilder) WithDescription(desc string) *ToolBuilder {
	b.tool.description = desc
	return b
}

// WithInputSchema sets the input schema
func (b *ToolBuilder) WithInputSchema(schema map[string]any) *ToolBuilder {
	b.tool.inputSchema = schema
	return b
}

// WithHandler sets the handler function
func (b *ToolBuilder) WithHandler(handler types.ToolHandler) *ToolBuilder {
	b.tool.handler = handler
	return b
}

// Build returns the constructed tool
func (b *ToolBuilder) Build() types.Tool {
	return b.tool
}

// CommonSchemas provides reusable schema definitions
var CommonSchemas = struct {
	Language    map[string]any
	Source      map[string]any
	Path        map[string]any
	Query       map[string]any
	Replacement map[string]any
	Target      map[string]any
}{
	Language: map[string]any{
		"type":        "string",
		"description": "Programming language",
	},
	Source: map[string]any{
		"type":        "string",
		"description": "Source code (for in-memory mode)",
	},
	Path: map[string]any{
		"type":        "string",
		"description": "File path to modify (for file writer mode)",
	},
	Query: map[string]any{
		"type":        "object",
		"description": "Query to find code elements",
		"properties": map[string]any{
			"type": map[string]any{
				"type":        "string",
				"description": "Element type (function, struct, class, etc)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Name pattern (supports wildcards)",
			},
		},
	},
	Replacement: map[string]any{
		"type":        "string",
		"description": "Replacement code",
	},
	Target: map[string]any{
		"type":        "object",
		"description": "Target to modify",
		"properties": map[string]any{
			"type": map[string]any{
				"type": "string",
			},
			"name": map[string]any{
				"type": "string",
			},
		},
	},
}

// ParseParams is a helper to unmarshal parameters with proper error handling
func ParseParams[T any](params json.RawMessage) (*T, error) {
	var result T
	if err := json.Unmarshal(params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func notifyProgress(ctx context.Context, server types.ServerInterface, progress, total float64, message string) {
	if server == nil {
		return
	}
	server.ReportProgress(ctx, progress, total, message)
}

func isCancelled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
