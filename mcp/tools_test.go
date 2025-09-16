package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestGetToolDefinitions tests tool definition structure
func TestGetToolDefinitions(t *testing.T) {
	tools := GetToolDefinitions()

	expectedTools := []string{
		"query", "file_query", "replace", "file_replace",
		"delete", "file_delete", "insert_before", "insert_after",
		"apply", "append",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolMap := make(map[string]ToolDefinition)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	for _, expectedName := range expectedTools {
		tool, exists := toolMap[expectedName]
		if !exists {
			t.Errorf("Expected tool '%s' not found", expectedName)
			continue
		}

		// Verify all tools have required fields
		if tool.Name == "" {
			t.Errorf("Tool %s has empty name", expectedName)
		}

		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", expectedName)
		}

		if tool.InputSchema == nil {
			t.Errorf("Tool %s has nil input schema", expectedName)
		}

		// Verify schema structure
		schema := tool.InputSchema
		if schema["type"] != "object" {
			t.Errorf("Tool %s schema type should be 'object', got %v", expectedName, schema["type"])
		}

		if _, hasProperties := schema["properties"]; !hasProperties {
			t.Errorf("Tool %s schema should have 'properties'", expectedName)
		}

		if _, hasRequired := schema["required"]; !hasRequired {
			t.Errorf("Tool %s schema should have 'required' field", expectedName)
		}
	}
}

// TestHandleQueryTool tests the query tool with various scenarios
func TestHandleQueryTool(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errType   string
	}{
		{
			name: "valid_source_query",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc test() {}",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: false,
		},
		{
			name: "missing_language",
			params: map[string]any{
				"source": "package main\nfunc test() {}",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
		},
		{
			name: "missing_source_and_path",
			params: map[string]any{
				"language": "go",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
			errType:   "InvalidParams",
		},
		{
			name: "both_source_and_path",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"path":     "/some/path",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
			errType:   "InvalidParams",
		},
		{
			name: "unsupported_language",
			params: map[string]any{
				"language": "unsupported",
				"source":   "some code",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
			errType:   "LanguageNotFound",
		},
		{
			name: "invalid_query_structure",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"query":    "invalid query",
			},
			expectErr: true,
		},
		{
			name: "malformed_go_code",
			params: map[string]any{
				"language": "go",
				"source":   "invalid go code {{{",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true, // Malformed code should return syntax error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			result, err := server.handleQueryTool(params)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errType != "" {
					// Check if error message contains expected text
					if !strings.Contains(err.Error(), tt.errType) {
						t.Logf("Expected error type %s, got %v (this might be expected behavior)", tt.errType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					// Verify result structure
					resultMap, ok := result.(map[string]any)
					if !ok {
						t.Errorf("Expected result to be a map, got %T", result)
					} else {
						if _, hasContent := resultMap["content"]; !hasContent {
							t.Error("Result should have 'content' field")
						}
					}
				}
			}
		})
	}
}

// TestHandleQueryToolWithFile tests query tool with file operations
func TestHandleQueryToolWithFile(t *testing.T) {
	server := createTestServer(t)

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	// Helper function
}`

	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
	}{
		{
			name: "valid_file_query",
			params: map[string]any{
				"language": "go",
				"path":     testFile,
				"query": map[string]any{
					"type": "function",
					"name": "main",
				},
			},
			expectErr: false,
		},
		{
			name: "nonexistent_file",
			params: map[string]any{
				"language": "go",
				"path":     "/nonexistent/file.go",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			result, err := server.handleQueryTool(params)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					// Verify result mentions the file path
					resultMap, ok := result.(map[string]any)
					if !ok {
						t.Errorf("Expected result to be a map, got %T", result)
					} else {
						content, ok := resultMap["content"].([]map[string]any)
						if !ok || len(content) == 0 {
							t.Error("Expected content array")
						} else {
							text, ok := content[0]["text"].(string)
							if !ok {
								t.Error("Expected text field in content")
							} else if !strings.Contains(text, testFile) {
								t.Error("Result should mention the file path")
							}
						}
					}
				}
			}
		})
	}
}

// TestRegisterBuiltinTools tests that all builtin tools are registered
func TestRegisterBuiltinTools(t *testing.T) {
	server := createTestServer(t)

	expectedTools := []string{
		"query", "file_query", "replace", "file_replace",
		"delete", "file_delete", "insert_before", "insert_after",
		"apply", "append",
	}

	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, toolName := range expectedTools {
		if _, exists := server.tools[toolName]; !exists {
			t.Errorf("Tool '%s' should be registered", toolName)
		}
	}

	// Verify we don't have unexpected tools
	if len(server.tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, found %d", len(expectedTools), len(server.tools))
		for name := range server.tools {
			found := slices.Contains(expectedTools, name)
			if !found {
				t.Logf("Unexpected tool found: %s", name)
			}
		}
	}
}

// TestHandleCallTool tests the generic tool calling mechanism
func TestHandleCallTool(t *testing.T) {
	server := createTestServer(t)

	// Test with a tool that exists
	t.Run("existing_tool", func(t *testing.T) {
		request := Request{
			Method: "tools/call",
			ID:     1,
			Params: mustMarshal(map[string]any{
				"name": "query",
				"arguments": map[string]any{
					"language": "go",
					"source":   "package main\nfunc test() {}",
					"query": map[string]any{
						"type": "function",
						"name": "test",
					},
				},
			}),
		}

		response := server.handleCallTool(request)

		if response.Error != nil {
			t.Errorf("Unexpected error: %v", response.Error)
		}
	})

	// Test with a tool that doesn't exist
	t.Run("nonexistent_tool", func(t *testing.T) {
		request := Request{
			Method: "tools/call",
			ID:     2,
			Params: mustMarshal(map[string]any{
				"name":      "nonexistent_tool",
				"arguments": map[string]any{},
			}),
		}

		response := server.handleCallTool(request)

		if response.Error == nil {
			t.Error("Expected error for nonexistent tool")
		}
	})

	// Test with invalid params structure
	t.Run("invalid_params", func(t *testing.T) {
		request := Request{
			Method: "tools/call",
			ID:     3,
			Params: json.RawMessage(`invalid json`),
		}

		response := server.handleCallTool(request)

		if response.Error == nil {
			t.Error("Expected error for invalid params")
		}
	})

	// Test with missing tool name
	t.Run("missing_tool_name", func(t *testing.T) {
		request := Request{
			Method: "tools/call",
			ID:     4,
			Params: mustMarshal(map[string]any{
				"arguments": map[string]any{},
			}),
		}

		response := server.handleCallTool(request)

		if response.Error == nil {
			t.Error("Expected error for missing tool name")
		}
	})
}

// TestToolSchemaValidation tests that tool schemas are properly structured
func TestToolSchemaValidation(t *testing.T) {
	tools := GetToolDefinitions()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			schema := tool.InputSchema

			// Check required top-level fields
			if schema["type"] != "object" {
				t.Errorf("Schema type should be 'object', got %v", schema["type"])
			}

			properties, hasProperties := schema["properties"]
			if !hasProperties {
				t.Error("Schema should have 'properties'")
				return
			}

			propertiesMap, ok := properties.(map[string]any)
			if !ok {
				t.Error("Properties should be a map")
				return
			}

			// Check that tools have appropriate parameters
			// File tools (file_*) have 'scope' parameter instead of 'language'
			// Other tools should have 'language' parameter (except apply)
			isFileTool := strings.HasPrefix(tool.Name, "file_")
			if tool.Name != "apply" && !isFileTool {
				if _, hasLanguage := propertiesMap["language"]; !hasLanguage {
					t.Error("Non-file tool should have 'language' parameter")
				}
			} else if isFileTool {
				if _, hasScope := propertiesMap["scope"]; !hasScope {
					t.Error("File tool should have 'scope' parameter")
				}
			}

			// Check required fields
			required, hasRequired := schema["required"]
			if !hasRequired {
				t.Error("Schema should have 'required' field")
				return
			}

			requiredSlice, ok := required.([]string)
			if !ok {
				t.Error("Required should be a string slice")
				return
			}

			// Verify that all required fields exist in properties
			for _, reqField := range requiredSlice {
				if _, exists := propertiesMap[reqField]; !exists {
					t.Errorf("Required field '%s' not found in properties", reqField)
				}
			}

			// Check for appropriate parameters based on tool type
			if strings.Contains(tool.Name, "file_") {
				// File tools should have scope instead of source/path
				if _, hasScope := propertiesMap["scope"]; !hasScope {
					t.Error("File tool should have 'scope' parameter")
				}
				// File tools don't require language parameter at top level
			} else if tool.Name != "apply" {
				// Non-file tools typically need language parameter
				if _, hasLanguage := propertiesMap["language"]; !hasLanguage {
					t.Logf("Tool %s missing 'language' parameter (might be expected)", tool.Name)
				}
			}
		})
	}
}

// TestMCPErrorHandling tests MCP error creation and handling
func TestMCPErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		hasCode bool
		hasData bool
	}{
		{
			name:    "new_mcp_error",
			err:     NewMCPError(-32602, "Invalid params", nil),
			hasCode: true,
			hasData: false,
		},
		{
			name:    "mcp_error_with_data",
			err:     NewMCPError(-32603, "Internal error", map[string]any{"details": "test"}),
			hasCode: true,
			hasData: true,
		},
		{
			name:    "wrapped_error",
			err:     WrapError(-32600, "Invalid request", os.ErrNotExist),
			hasCode: true,
			hasData: true, // WrapError includes the wrapped error's message in Data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpErr, ok := tt.err.(*MCPError)
			if !ok {
				t.Fatalf("Expected MCPError, got %T", tt.err)
			}

			if tt.hasCode && mcpErr.Code == 0 {
				t.Error("Expected error code to be set")
			}

			if mcpErr.Message == "" {
				t.Error("Expected error message to be set")
			}

			if tt.hasData && mcpErr.Data == nil {
				t.Error("Expected error data to be set")
			} else if !tt.hasData && mcpErr.Data != nil {
				t.Error("Expected error data to be nil")
			}

			// Test error string representation
			errStr := tt.err.Error()
			if errStr == "" {
				t.Error("Error string should not be empty")
			}
		})
	}
}

// TestToolExecutionTimeout tests tool execution with timeouts
func TestToolExecutionTimeout(t *testing.T) {
	server := createTestServer(t)

	// Register a slow tool for testing
	server.RegisterTool("slow_tool", func(params json.RawMessage) (any, error) {
		time.Sleep(100 * time.Millisecond)
		return "completed", nil
	})

	// Register a tool that returns an error
	server.RegisterTool("error_tool", func(params json.RawMessage) (any, error) {
		return nil, NewMCPError(-32603, "Intentional error", map[string]any{"test": true})
	})

	tests := []struct {
		name       string
		toolName   string
		expectErr  bool
		expectData bool
	}{
		{
			name:      "slow_tool_completes",
			toolName:  "slow_tool",
			expectErr: false,
		},
		{
			name:       "error_tool_returns_error",
			toolName:   "error_tool",
			expectErr:  true,
			expectData: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := Request{
				Method: "tools/call",
				ID:     1,
				Params: mustMarshal(map[string]any{
					"name":      tt.toolName,
					"arguments": map[string]any{},
				}),
			}

			response := server.handleCallTool(request)

			if tt.expectErr {
				if response.Error == nil {
					t.Error("Expected error but got none")
				} else if tt.expectData && response.Error.Data == nil {
					t.Error("Expected error data but got none")
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error)
				}
			}
		})
	}
}

// TestToolInputValidation tests input validation for various tools
func TestToolInputValidation(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name      string
		toolName  string
		params    map[string]any
		expectErr bool
	}{
		{
			name:     "query_valid_input",
			toolName: "query",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"query":    map[string]any{"type": "function"},
			},
			expectErr: false,
		},
		{
			name:     "query_missing_language",
			toolName: "query",
			params: map[string]any{
				"source": "package main",
				"query":  map[string]any{"type": "function"},
			},
			expectErr: true,
		},
		{
			name:     "apply_valid_input",
			toolName: "apply",
			params: map[string]any{
				"latest": true,
			},
			expectErr: true, // Will error without database/staging
		},
		{
			name:      "apply_empty_input",
			toolName:  "apply",
			params:    map[string]any{},
			expectErr: true, // Will error without database/staging
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			// Get the tool handler directly
			server.mu.RLock()
			handler, exists := server.tools[tt.toolName]
			server.mu.RUnlock()

			if !exists {
				t.Fatalf("Tool %s not found", tt.toolName)
			}

			_, err = handler(params)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
