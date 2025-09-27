// Package types provides shared types and interfaces for MCP components
// This avoids circular dependencies between packages
package types

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/models"
	"github.com/termfx/morfx/providers"
)

// ServerInterface defines what tools need from the server
type ServerInterface interface {
	GetProviders() *providers.Registry
	GetFileProcessor() *core.FileProcessor
	GetStaging() any
	GetSafety() any
	GetSessionID() string
	ReportProgress(ctx context.Context, progress, total float64, message string)
	ConfirmApply(ctx context.Context, summary string) error
	RequestSampling(ctx context.Context, params map[string]any) (map[string]any, error)
	RequestElicitation(ctx context.Context, params map[string]any) (map[string]any, error)
	FinalizeTransform(ctx context.Context, req TransformRequest) (map[string]any, error)
}

// StagingStore captures the operations ApplyTool expects from a staging manager implementation.
type StagingStore interface {
	ListPendingStages(sessionID string) ([]models.Stage, error)
	GetStage(stageID string) (*models.Stage, error)
	ApplyStage(ctx context.Context, stageID string, autoApplied bool) (*models.Apply, error)
}

// StagingToggle allows staged operations to advertise whether they are active.
type StagingToggle interface {
	IsEnabled() bool
}

// StagingManager combines the core staging operations needed by tools
type StagingManager interface {
	StagingStore
	StagingToggle
}

// ToolHandler represents a function that handles a tool call
type ToolHandler func(ctx context.Context, params json.RawMessage) (any, error)

// Component represents a registrable MCP component (tool, prompt, resource)
type Component interface {
	Name() string
	Description() string
}

// Tool represents an executable tool with handler
type Tool interface {
	Component
	Handler() ToolHandler
	InputSchema() map[string]any
}

// TransformRequest captures the information needed to finalize a transformation.
type TransformRequest struct {
	Language       string               `json:"language"`
	Operation      string               `json:"operation"`
	Target         core.AgentQuery      `json:"target"`
	TargetJSON     json.RawMessage      `json:"target_json"`
	Path           string               `json:"path,omitempty"`
	OriginalSource string               `json:"original_source"`
	Result         core.TransformResult `json:"result"`
	ResponseText   string               `json:"response_text"`
	Content        string               `json:"content,omitempty"`
}

// PromptArgument represents an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Prompt represents a system prompt
type Prompt interface {
	Component
	Content() string
	Arguments() []PromptArgument
}

// Resource represents a readable resource
type Resource interface {
	Component
	URI() string
	MimeType() string
	Contents() (string, error)
}

// WatchableResource is implemented by resources that can push update notifications.
type WatchableResource interface {
	Resource
	Watch(ctx context.Context) (<-chan ResourceUpdate, error)
}

// ErrResourceWatchUnsupported is returned when a resource does not support subscriptions.
var ErrResourceWatchUnsupported = errors.New("resource does not support watch")

// ResourceUpdateType identifies the kind of update emitted by a watchable resource.
type ResourceUpdateType string

const (
	ResourceUpdateTypeUpdated     ResourceUpdateType = "updated"
	ResourceUpdateTypeRemoved     ResourceUpdateType = "removed"
	ResourceUpdateTypeListChanged ResourceUpdateType = "list_changed"
)

// ResourceUpdate describes a change emitted by a watchable resource.
type ResourceUpdate struct {
	URI  string             `json:"uri,omitempty"`
	Type ResourceUpdateType `json:"type,omitempty"`
	Data map[string]any     `json:"data,omitempty"`
}

// DefaultJSONSchemaURI represents the canonical JSON Schema reference for responses.
const DefaultJSONSchemaURI = "https://json-schema.org/draft/2020-12/schema"

// NormalizeSchema clones the provided schema and injects required defaults.
func NormalizeSchema(schema map[string]any) map[string]any {
	cloned := cloneSchemaMap(schema)
	if cloned == nil {
		cloned = map[string]any{}
	}
	if _, ok := cloned["type"]; !ok {
		cloned["type"] = "object"
	}
	if _, ok := cloned["$schema"]; !ok {
		cloned["$schema"] = DefaultJSONSchemaURI
	}
	return cloned
}

func cloneSchemaMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneSchemaValue(value)
	}
	return result
}

func cloneSchemaSlice(source []any) []any {
	if source == nil {
		return nil
	}
	result := make([]any, len(source))
	for i, value := range source {
		result[i] = cloneSchemaValue(value)
	}
	return result
}

func cloneSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneSchemaMap(typed)
	case []any:
		return cloneSchemaSlice(typed)
	default:
		return typed
	}
}

// ToolDefinition mirrors the spec-defined Tool metadata exposed to clients.
type ToolDefinition struct {
	Name          string         `json:"name"`
	Title         string         `json:"title,omitempty"`
	Description   string         `json:"description,omitempty"`
	InputSchema   map[string]any `json:"inputSchema,omitempty"`
	OutputSchema  map[string]any `json:"outputSchema,omitempty"`
	Annotations   map[string]any `json:"annotations,omitempty"`
	StructuredKey string         `json:"structuredResultKey,omitempty"`
}

// PromptDefinition describes a prompt for the MCP client.
type PromptDefinition struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
	Annotations map[string]any   `json:"annotations,omitempty"`
}

// ResourceDefinition describes a resource for the MCP client.
type ResourceDefinition struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
	Size        *int64         `json:"size,omitempty"`
}

// SamplingRecord captures a server-initiated sampling exchange with the client.
type SamplingRecord struct {
	Timestamp time.Time      `json:"timestamp"`
	Params    map[string]any `json:"params"`
	Result    map[string]any `json:"result,omitempty"`
}

// ElicitationRecord captures an elicitation interaction with the client.
type ElicitationRecord struct {
	Timestamp time.Time      `json:"timestamp"`
	Params    map[string]any `json:"params"`
	Result    map[string]any `json:"result,omitempty"`
}

// ResourceTemplateDefinition describes a templated resource entry point exposed by the server.
type ResourceTemplateDefinition struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	URITemplate string         `json:"uriTemplate"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// Error codes for MCP
const (
	InvalidParams    = -32602
	FileSystemError  = -32001
	LanguageNotFound = -32002
	SyntaxError      = -32003
	TransformFailed  = -32004
	CustomErrorStart = -32999
)

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ContentBlock represents a unit of textual content returned by prompts or tools.
type ContentBlock struct {
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`
	URI         string         `json:"uri,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// CallToolResult models the standard MCP response payload for tool invocations.
type CallToolResult struct {
	Content           []ContentBlock `json:"content"`
	StructuredContent any            `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
}

// Error implements the error interface
func (e *MCPError) Error() string {
	return e.Message
}

// NewMCPError creates a new MCP error
func NewMCPError(code int, message string, data any) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// WrapError wraps an error with MCP error code
func WrapError(code int, message string, err error) *MCPError {
	data := map[string]any{
		"error": err.Error(),
	}
	return NewMCPError(code, message, data)
}
