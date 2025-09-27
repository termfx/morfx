package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/termfx/morfx/mcp/types"
)

// ErrToolNotFound indicates that a requested tool is not registered.
var ErrToolNotFound = errors.New("tool not found")

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
func (tr *ToolRegistry) Execute(ctx context.Context, name string, params json.RawMessage) (any, error) {
	tool, exists := tr.Get(name)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	// Tools need server context, so we wrap the handler
	handler := tool.Handler()
	return handler(ctx, params)
}

// GetDefinitions returns tool definitions for MCP protocol
func (tr *ToolRegistry) GetDefinitions() []types.ToolDefinition {
	tools := tr.List()
	definitions := make([]types.ToolDefinition, 0, len(tools))

	for _, tool := range tools {
		name := tool.Name()
		title := strings.Title(strings.ReplaceAll(name, "_", " "))
		category := classifyToolCategory(name)
		annotations := map[string]any{
			"name":             name,
			"kind":             toolKind(category),
			"category":         category,
			"scope":            toolScope(name),
			"scoped":           strings.HasPrefix(name, "file_"),
			"entrypoint":       fmt.Sprintf("tools/call:%s", name),
			"stability":        toolStability(name),
			"audience":         "developer",
			"progress":         toolSupportsProgress(name),
			"output":           "call-tool-result",
			"structuredResult": "structuredContent",
		}
		definitions = append(definitions, types.ToolDefinition{
			Name:          name,
			Title:         title,
			Description:   tool.Description(),
			InputSchema:   types.NormalizeSchema(tool.InputSchema()),
			OutputSchema:  buildCallToolResultSchema(),
			StructuredKey: "structuredContent",
			Annotations:   annotations,
		})
	}

	return definitions
}

func classifyToolCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "file_"):
		return "file-transform"
	case name == "query" || name == "file_query":
		return "analysis"
	case name == "apply":
		return "staging"
	default:
		return "code-transform"
	}
}

func toolKind(category string) string {
	switch category {
	case "analysis":
		return "analysis"
	case "staging":
		return "workflow"
	default:
		return "transformation"
	}
}

func toolScope(name string) string {
	if strings.HasPrefix(name, "file_") {
		return "file"
	}
	if name == "apply" {
		return "workspace"
	}
	return "workspace"
}

func toolStability(name string) string {
	switch name {
	case "apply":
		return "beta"
	default:
		return "stable"
	}
}

var builtinProgressTools = map[string]struct{}{
	"append":        {},
	"apply":         {},
	"delete":        {},
	"insert_after":  {},
	"insert_before": {},
	"query":         {},
	"replace":       {},
}

func toolSupportsProgress(name string) bool {
	if strings.HasPrefix(name, "file_") {
		return true
	}
	_, ok := builtinProgressTools[name]
	return ok
}

func buildCallToolResultSchema() map[string]any {
	return types.NormalizeSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":        map[string]any{"type": "string"},
						"text":        map[string]any{"type": "string"},
						"uri":         map[string]any{"type": "string", "format": "uri"},
						"mimeType":    map[string]any{"type": "string"},
						"data":        map[string]any{"type": "object"},
						"annotations": map[string]any{"type": "object"},
					},
					"required":             []string{"type"},
					"additionalProperties": true,
				},
			},
			"structuredContent": map[string]any{"type": "object"},
			"isError":           map[string]any{"type": "boolean"},
		},
		"required":             []string{"content"},
		"additionalProperties": true,
	})
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

// ResourceTemplateRegistry manages resource template registration
type ResourceTemplateRegistry struct {
	*BaseRegistry[types.ResourceTemplateDefinition]
}

// NewResourceTemplateRegistry creates a new resource template registry
func NewResourceTemplateRegistry() *ResourceTemplateRegistry {
	return &ResourceTemplateRegistry{
		BaseRegistry: NewBaseRegistry[types.ResourceTemplateDefinition](),
	}
}
