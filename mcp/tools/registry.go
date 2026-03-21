package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/termfx/morfx/mcp/types"
)

// toolRegistry manages tools internally
type toolRegistry struct {
	mu      sync.RWMutex
	tools   map[string]types.Tool
	ordered []string
	server  types.ServerInterface
}

// Registry holds all registered tools
var Registry *toolRegistry

// Init initializes the tool registry with the server
func Init(server types.ServerInterface) {
	Registry = &toolRegistry{
		tools:   make(map[string]types.Tool),
		ordered: make([]string, 0),
		server:  server,
	}
	RegisterAll(server)
}

// Register adds a tool to the registry
func (r *toolRegistry) Register(name string, tool types.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		r.ordered = append(r.ordered, name)
	}
	r.tools[name] = tool
}

// Get retrieves a tool by name
func (r *toolRegistry) Get(name string) (types.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all tools in registration order
func (r *toolRegistry) List() []types.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.Tool, 0, len(r.ordered))
	for _, name := range r.ordered {
		result = append(result, r.tools[name])
	}
	return result
}

// Execute runs a tool by name with the given parameters
func (r *toolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (any, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	handler := tool.Handler()
	return handler(ctx, params)
}

// RegisterAll registers all built-in tools
func RegisterAll(server types.ServerInterface) {
	// Query tools
	Registry.Register("query", NewQueryTool(server))
	Registry.Register("file_query", NewFileQueryTool(server))

	// Transformation tools
	Registry.Register("replace", NewReplaceTool(server))
	Registry.Register("file_replace", NewFileReplaceTool(server))
	Registry.Register("delete", NewDeleteTool(server))
	Registry.Register("file_delete", NewFileDeleteTool(server))
	Registry.Register("insert_before", NewInsertBeforeTool(server))
	Registry.Register("insert_after", NewInsertAfterTool(server))
	Registry.Register("append", NewAppendTool(server))

	// Staging tools
	Registry.Register("apply", NewApplyTool(server))
}

// Get retrieves a tool by name
func Get(name string) (types.Tool, bool) {
	return Registry.Get(name)
}

// Execute runs a tool by name
func Execute(ctx context.Context, name string, params []byte) (any, error) {
	return Registry.Execute(ctx, name, params)
}

// GetDefinitions returns all tool definitions
func GetDefinitions() []types.ToolDefinition {
	tools := Registry.List()
	definitions := make([]types.ToolDefinition, 0, len(tools))

	for _, tool := range tools {
		definitions = append(definitions, types.ToolDefinition{
			Name:        tool.Name(),
			Title:       tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: map[string]any{"title": tool.Name()},
		})
	}

	return definitions
}
