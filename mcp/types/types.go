// Package types provides shared types and interfaces for MCP components
// This avoids circular dependencies between packages
package types

import (
	"encoding/json"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/providers"
)

// ServerInterface defines what tools need from the server
type ServerInterface interface {
	GetProviders() *providers.Registry
	GetFileProcessor() *core.FileProcessor
	GetStaging() any
	GetSafety() any
}

// ToolHandler represents a function that handles a tool call
type ToolHandler func(params json.RawMessage) (any, error)

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

// ToolDefinition describes a tool for the MCP client
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// PromptDefinition describes a prompt for the MCP client
type PromptDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// ResourceDefinition describes a resource for the MCP client
type ResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Error codes for MCP
const (
	InvalidParams    = -32602
	FileSystemError  = -32001
	LanguageNotFound = -32002
	SyntaxError      = -32003
	TransformFailed  = -32004
)

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
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
