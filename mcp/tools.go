package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	
	"github.com/termfx/morfx/core"
)

// ToolDefinition describes a tool for the client
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns all available tool definitions
func GetToolDefinitions() []ToolDefinition {
	tools := []ToolDefinition{
		{
			Name:        "query",
			Description: "Find code elements using natural language queries",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language (go, python, javascript, etc)",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code to analyze",
					},
					"query": map[string]interface{}{
						"type":        "object",
						"description": "Query to find code elements",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Element type (function, struct, class, etc)",
							},
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Name pattern (supports wildcards)",
							},
						},
					},
				},
				"required": []string{"language", "source", "query"},
			},
		},
		{
			Name:        "replace",
			Description: "Replace code elements matching a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code",
					},
					"target": map[string]interface{}{
						"type":        "object",
						"description": "Target to replace",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
							},
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"replacement": map[string]interface{}{
						"type":        "string",
						"description": "Replacement code",
					},
				},
				"required": []string{"language", "source", "target", "replacement"},
			},
		},
		{
			Name:        "delete",
			Description: "Delete code elements matching a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code",
					},
					"target": map[string]interface{}{
						"type":        "object",
						"description": "Target to delete",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
							},
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
				"required": []string{"language", "source", "target"},
			},
		},
		{
			Name:        "insert_before",
			Description: "Insert code before elements matching a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code",
					},
					"target": map[string]interface{}{
						"type":        "object",
						"description": "Target location",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
							},
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Code to insert",
					},
				},
				"required": []string{"language", "source", "target", "content"},
			},
		},
		{
			Name:        "insert_after",
			Description: "Insert code after elements matching a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code",
					},
					"target": map[string]interface{}{
						"type":        "object",
						"description": "Target location",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
							},
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Code to insert",
					},
				},
				"required": []string{"language", "source", "target", "content"},
			},
		},
		{
			Name:        "apply",
			Description: "Apply staged code transformations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Specific stage ID to apply",
					},
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Apply all pending stages",
					},
					"latest": map[string]interface{}{
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
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source code",
					},
					"target": map[string]interface{}{
						"type":        "object",
						"description": "Optional target scope (struct, function, etc)",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
							},
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Code to append",
					},
				},
				"required": []string{"language", "source", "content"},
			},
		},
	}
	
	return tools
}

// registerBuiltinTools registers all built-in tool handlers
func (s *StdioServer) registerBuiltinTools() {
	// Query tool
	s.RegisterTool("query", s.handleQueryTool)
	
	// Transformation tools
	s.RegisterTool("replace", s.handleReplaceTool)
	s.RegisterTool("delete", s.handleDeleteTool)
	s.RegisterTool("insert_before", s.handleInsertBeforeTool)
	s.RegisterTool("insert_after", s.handleInsertAfterTool)
	s.RegisterTool("append", s.handleAppendTool)
	
	// Staging tools
	s.RegisterTool("apply", s.handleApplyTool)
	
	// TODO Phase 6:
	// s.RegisterTool("validate", s.handleValidateTool)
}

// handleQueryTool executes code queries using language providers
func (s *StdioServer) handleQueryTool(params json.RawMessage) (interface{}, error) {
	var args struct {
		Language string          `json:"language"`
		Source   string          `json:"source"`
		Query    json.RawMessage `json:"query"`
	}
	
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid query parameters", err)
	}
	
	// Get provider for language
	provider, exists := s.providers.Get(args.Language)
	if !exists {
		return nil, NewMCPError(LanguageNotFound, 
			fmt.Sprintf("No provider for language: %s", args.Language),
			"Supported languages: go, python, javascript, laravel")
	}
	
	// Parse the query
	var query core.AgentQuery
	if err := json.Unmarshal(args.Query, &query); err != nil {
		return nil, WrapError(InvalidParams, "Invalid query structure", err)
	}
	
	// Execute query
	result := provider.Query(args.Source, query)
	if result.Error != nil {
		// Check if it's a syntax error or other
		errMsg := result.Error.Error()
		if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "syntax") {
			return nil, NewMCPError(SyntaxError, "Failed to parse source code", errMsg)
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
	
	// Return as MCP content blocks
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": responseText,
			},
		},
	}, nil
}
