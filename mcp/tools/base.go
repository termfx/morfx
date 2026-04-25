package tools

import (
	"context"
	"encoding/json"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/mcp/types"
)

const dslSelectorDescription = "Morfx DSL selector for read tools. Use this instead of query when matching nested AST structure. Syntax: kind:name with * wildcard and $capture patterns. Operators: ! not, > contains descendant, >> direct semantic child, & and, | or, parentheses for grouping. Use attributes as key=value or shorthand type. Common attributes: arg, arg0, source, text, before, after. Common selectors: func, def, function, method, class, struct, interface, field, call, return, assignment, condition, block, loop, import. Examples: func:* > call:os.Getenv; class:* >> method:render; call:$client.$method; call:fetch arg0=\"/api/user\"; struct:* > field:Secret type=string; (func:* | method:*) > call:fetch."

const targetDSLSelectorDescription = "Morfx target_dsl selector for mutation tools. Use this instead of target when matching nested AST structure. Syntax: kind:name with * wildcard and $capture patterns. Operators: ! not, > contains descendant, >> direct semantic child, & and, | or, parentheses for grouping. Use attributes as key=value or shorthand type. Common attributes: arg, arg0, source, text, before, after. Common selectors: func, def, function, method, class, struct, interface, field, call, return, assignment, condition, block, loop, import. Examples: func:Legacy*; func:* > call:os.Getenv; class:* >> method:render; call:fetch arg0=\"/api/user\"; struct:* > field:Secret type=string."

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
	DSL         map[string]any
	Replacement map[string]any
	Target      map[string]any
	TargetDSL   map[string]any
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
		"description": "Object query to find code elements by type/name. Prefer dsl for nested AST structure, operators, and attributes.",
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
	DSL: map[string]any{
		"type":        "string",
		"description": dslSelectorDescription,
	},
	Replacement: map[string]any{
		"type":        "string",
		"description": "Replacement code",
	},
	Target: map[string]any{
		"type":        "object",
		"description": "Object target to modify by type/name. Prefer target_dsl for nested AST structure, operators, and attributes.",
		"properties": map[string]any{
			"type": map[string]any{
				"type": "string",
			},
			"name": map[string]any{
				"type": "string",
			},
		},
	},
	TargetDSL: map[string]any{
		"type":        "string",
		"description": targetDSLSelectorDescription,
	},
}

func parseRequiredQuery(raw json.RawMessage, dsl, label string) (core.AgentQuery, error) {
	query, err := core.ParseAgentQueryPayload(raw, dsl)
	if err != nil {
		return core.AgentQuery{}, types.WrapError(types.InvalidParams, "Invalid "+label+" structure", err)
	}
	return query, nil
}

func parseOptionalQuery(raw json.RawMessage, dsl, label string) (core.AgentQuery, bool, error) {
	query, ok, err := core.ParseOptionalAgentQueryPayload(raw, dsl)
	if err != nil {
		return core.AgentQuery{}, ok, types.WrapError(types.InvalidParams, "Invalid "+label+" structure", err)
	}
	return query, ok, nil
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
