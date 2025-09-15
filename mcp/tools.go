package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/termfx/morfx/core"
)

// ToolDefinition describes a tool for the client
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// GetToolDefinitions returns all available tool definitions
func GetToolDefinitions() []ToolDefinition {
	tools := []ToolDefinition{
		{
			Name:        "query",
			Description: "Find code elements using natural language queries",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language (go, python, javascript, etc)",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code to analyze (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to analyze (for file writer mode)",
					},
					"query": map[string]any{
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
				},
				"required": []string{"language", "query"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
		{
			Name:        "file_query",
			Description: "Find code elements across multiple files using natural language queries",
			InputSchema: map[string]any{
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
							"max_files": map[string]any{
								"type":        "integer",
								"description": "Maximum files to process (0 = unlimited)",
							},
							"language": map[string]any{
								"type":        "string",
								"description": "Programming language filter",
							},
						},
						"required": []string{"path"},
					},
					"query": map[string]any{
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
				},
				"required": []string{"scope", "query"},
			},
		},
		{
			Name:        "replace",
			Description: "Replace code elements matching a query",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to modify (for file writer mode)",
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Target to replace",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
							},
							"name": map[string]any{
								"type": "string",
							},
						},
					},
					"replacement": map[string]any{
						"type":        "string",
						"description": "Replacement code",
					},
				},
				"required": []string{"language", "target", "replacement"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
		{
			Name:        "file_replace",
			Description: "Replace code elements across multiple files",
			InputSchema: map[string]any{
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
							"exclude": map[string]any{
								"type":        "array",
								"description": "File patterns to exclude",
								"items":       map[string]any{"type": "string"},
							},
						},
						"required": []string{"path"},
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Target to replace",
						"properties": map[string]any{
							"type": map[string]any{"type": "string"},
							"name": map[string]any{"type": "string"},
						},
					},
					"replacement": map[string]any{
						"type":        "string",
						"description": "Replacement code",
					},
					"dry_run": map[string]any{
						"type":        "boolean",
						"description": "Preview changes without applying",
					},
					"backup": map[string]any{
						"type":        "boolean",
						"description": "Create backup files",
					},
				},
				"required": []string{"scope", "target", "replacement"},
			},
		},
		{
			Name:        "delete",
			Description: "Delete code elements matching a query",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to modify (for file writer mode)",
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Target to delete",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
							},
							"name": map[string]any{
								"type": "string",
							},
						},
					},
				},
				"required": []string{"language", "target"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
		{
			Name:        "file_delete",
			Description: "Delete code elements across multiple files",
			InputSchema: map[string]any{
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
					"target": map[string]any{
						"type":        "object",
						"description": "Target to delete",
						"properties": map[string]any{
							"type": map[string]any{"type": "string"},
							"name": map[string]any{"type": "string"},
						},
					},
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
		},
		{
			Name:        "insert_before",
			Description: "Insert code before elements matching a query",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to modify (for file writer mode)",
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Target location",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
							},
							"name": map[string]any{
								"type": "string",
							},
						},
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Code to insert",
					},
				},
				"required": []string{"language", "target", "content"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
		{
			Name:        "insert_after",
			Description: "Insert code after elements matching a query",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to modify (for file writer mode)",
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Target location",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
							},
							"name": map[string]any{
								"type": "string",
							},
						},
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Code to insert",
					},
				},
				"required": []string{"language", "target", "content"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
		{
			Name:        "apply",
			Description: "Apply staged code transformations",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Specific stage ID to apply",
					},
					"all": map[string]any{
						"type":        "boolean",
						"description": "Apply all pending stages",
					},
					"latest": map[string]any{
						"type":        "boolean",
						"description": "Apply the most recent pending stage",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "append",
			Description: "Append code to source - uses target if specified, otherwise intelligently places content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Source code (for in-memory mode)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File path to modify (for file writer mode)",
					},
					"target": map[string]any{
						"type":        "object",
						"description": "Optional target scope (struct, function, etc)",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
							},
							"name": map[string]any{
								"type": "string",
							},
						},
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Code to append",
					},
				},
				"required": []string{"language", "content"},
				"oneOf": []map[string]any{
					{"required": []string{"source"}},
					{"required": []string{"path"}},
				},
			},
		},
	}

	return tools
}

// registerBuiltinTools registers all built-in tool handlers
func (s *StdioServer) registerBuiltinTools() {
	// Query tools
	s.RegisterTool("query", s.handleQueryTool)
	s.RegisterTool("file_query", s.handleFileQueryTool)

	// Transformation tools
	s.RegisterTool("replace", s.handleReplaceTool)
	s.RegisterTool("file_replace", s.handleFileReplaceTool)
	s.RegisterTool("delete", s.handleDeleteTool)
	s.RegisterTool("file_delete", s.handleFileDeleteTool)
	s.RegisterTool("insert_before", s.handleInsertBeforeTool)
	s.RegisterTool("insert_after", s.handleInsertAfterTool)
	s.RegisterTool("append", s.handleAppendTool)

	// Staging tools
	s.RegisterTool("apply", s.handleApplyTool)

	// TODO Phase 6:
	// s.RegisterTool("validate", s.handleValidateTool)
}

// handleQueryTool executes code queries using language providers
func (s *StdioServer) handleQueryTool(params json.RawMessage) (any, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Path     string          `json:"path"`
		Query    json.RawMessage `json:"query"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid query parameters", err)
	}

	// Validate that exactly one of source or path is provided
	if (args.Source == "" && args.Path == "") || (args.Source != "" && args.Path != "") {
		return nil, NewMCPError(InvalidParams, "Exactly one of 'source' or 'path' must be provided", nil)
	}

	// Get source code
	var source string
	if args.Path != "" {
		// FILE WRITER MODE: Read from filesystem
		content, err := os.ReadFile(args.Path)
		if err != nil {
			return nil, WrapError(FileSystemError, "Failed to read file", err)
		}
		source = string(content)
	} else {
		// IN-MEMORY MODE: Use provided source
		source = args.Source
	}

	// Get provider for language
	provider, exists := s.providers.Get(args.Language)
	if !exists {
		return nil, NewMCPError(LanguageNotFound,
			fmt.Sprintf("No provider for language: %s", args.Language),
			map[string]any{
				"requested": args.Language,
				"supported": []string{"go"},
			})
	}

	// Parse the query
	var query core.AgentQuery
	if err := json.Unmarshal(args.Query, &query); err != nil {
		return nil, WrapError(InvalidParams, "Invalid query structure", err)
	}

	// Execute query
	result := provider.Query(source, query)
	if result.Error != nil {
		// Check if it's a syntax error or other
		errMsg := result.Error.Error()
		if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "syntax") {
			return nil, NewMCPError(SyntaxError, "Failed to parse source code",
				map[string]any{"details": errMsg})
		}
		return nil, WrapError(TransformFailed, "Query execution failed", result.Error)
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
	if args.Path != "" {
		responseText = fmt.Sprintf("File: %s\n\n%s", args.Path, responseText)
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
