package mcp

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/termfx/morfx/mcp/types"
)

// Registry is a generic registry for MCP components
type Registry[T any] interface {
	Register(name string, component T)
	Get(name string) (T, bool)
	List() []T
	Names() []string
}

// BaseRegistry provides a thread-safe generic registry implementation
type BaseRegistry[T any] struct {
	mu         sync.RWMutex
	components map[string]T
	ordered    []string // Maintain registration order
}

// NewBaseRegistry creates a new generic registry
func NewBaseRegistry[T any]() *BaseRegistry[T] {
	return &BaseRegistry[T]{
		components: make(map[string]T),
		ordered:    make([]string, 0),
	}
}

// Register adds a component to the registry
func (r *BaseRegistry[T]) Register(name string, component T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[name]; !exists {
		r.ordered = append(r.ordered, name)
	}
	r.components[name] = component
}

// Get retrieves a component by name
func (r *BaseRegistry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	component, exists := r.components[name]
	return component, exists
}

// List returns all components in registration order
func (r *BaseRegistry[T]) List() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]T, 0, len(r.ordered))
	for _, name := range r.ordered {
		result = append(result, r.components[name])
	}
	return result
}

// Names returns all component names in registration order
func (r *BaseRegistry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.ordered))
	copy(result, r.ordered)
	return result
}

// ToolRegistry manages tool registration and execution
type ToolRegistry struct {
	*BaseRegistry[types.Tool]
	server ServerInterface // Back-reference for handler context
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(server ServerInterface) *ToolRegistry {
	return &ToolRegistry{
		BaseRegistry: NewBaseRegistry[types.Tool](),
		server:       server,
	}
}

// Execute runs a tool by name with the given parameters
func (tr *ToolRegistry) Execute(name string, params json.RawMessage) (any, error) {
	tool, exists := tr.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Tools need server context, so we wrap the handler
	handler := tool.Handler()
	return handler(params)
}

// GetDefinitions returns tool definitions for MCP protocol
func (tr *ToolRegistry) GetDefinitions() []types.ToolDefinition {
	tools := tr.List()
	definitions := make([]types.ToolDefinition, 0, len(tools))

	for _, tool := range tools {
		definitions = append(definitions, types.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}

	return definitions
}

// PromptRegistry manages prompt registration
type PromptRegistry struct {
	*BaseRegistry[types.Prompt]
}

// NewPromptRegistry creates a new prompt registry
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		BaseRegistry: NewBaseRegistry[types.Prompt](),
	}
}

// ResourceRegistry manages resource registration
type ResourceRegistry struct {
	*BaseRegistry[types.Resource]
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		BaseRegistry: NewBaseRegistry[types.Resource](),
	}
}
