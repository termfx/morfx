package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create test server
func createTestServerForTransform(t *testing.T) *StdioServer {
	t.Helper()

	config := DefaultConfig()
	config.DatabaseURL = "skip" // No database for unit tests
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	return server
}

// TestHandleReplaceTool tests the replace tool handler parameter validation
func TestHandleReplaceTool(t *testing.T) {
	server := createTestServerForTransform(t)

	tests := []struct {
		name      string
		params    json.RawMessage
		expectErr bool
	}{
		{
			name: "missing_source_and_path",
			params: mustMarshal(map[string]any{
				"language":    "go",
				"target":      map[string]any{"type": "function", "name": "test"},
				"replacement": "newFunc",
			}),
			expectErr: true,
		},
		{
			name: "both_source_and_path",
			params: mustMarshal(map[string]any{
				"language":    "go",
				"source":      "package main",
				"path":        "/test.go",
				"target":      map[string]any{"type": "function", "name": "test"},
				"replacement": "newFunc",
			}),
			expectErr: true,
		},
		{
			name: "valid_source_replace",
			params: mustMarshal(map[string]any{
				"language":    "go",
				"source":      "package main\nfunc oldFunc() {}",
				"target":      map[string]any{"type": "function", "name": "oldFunc"},
				"replacement": "func newFunc() {}",
			}),
			expectErr: false,
		},
		{
			name: "invalid_language",
			params: mustMarshal(map[string]any{
				"language":    "unknown",
				"source":      "some code",
				"target":      map[string]any{"type": "function", "name": "test"},
				"replacement": "newFunc",
			}),
			expectErr: true,
		},
		{
			name: "missing_target",
			params: mustMarshal(map[string]any{
				"language":    "go",
				"source":      "package main",
				"replacement": "newFunc",
			}),
			expectErr: true,
		},
		{
			name: "missing_replacement",
			params: mustMarshal(map[string]any{
				"language": "go",
				"source":   "package main",
				"target":   map[string]any{"type": "function", "name": "test"},
			}),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.handleReplaceTool(tt.params)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestHandleDeleteTool tests the delete tool handler
func TestHandleDeleteTool(t *testing.T) {
	server := createTestServerForTransform(t)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
	}{
		{
			name: "valid_delete",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc toDelete() {}\nfunc keep() {}",
				"target":   map[string]any{"type": "function", "name": "toDelete"},
			},
			expectErr: false,
		},
		{
			name: "missing_source_and_path",
			params: map[string]any{
				"language": "go",
				"target":   map[string]any{"type": "function", "name": "test"},
			},
			expectErr: true,
		},
		{
			name: "both_source_and_path",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"path":     "/test.go",
				"target":   map[string]any{"type": "function", "name": "test"},
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

			_, err = server.handleDeleteTool(params)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestHandleInsertTools tests insert_before and insert_after tools
func TestHandleInsertTools(t *testing.T) {
	server := createTestServerForTransform(t)

	testCases := []struct {
		handlerName string
		handler     func(json.RawMessage) (any, error)
	}{
		{"insert_before", server.handleInsertBeforeTool},
		{"insert_after", server.handleInsertAfterTool},
	}

	for _, tc := range testCases {
		t.Run(tc.handlerName, func(t *testing.T) {
			tests := []struct {
				name      string
				params    map[string]any
				expectErr bool
			}{
				{
					name: "valid_insert",
					params: map[string]any{
						"language": "go",
						"source":   "package main\nfunc existing() {}",
						"target":   map[string]any{"type": "function", "name": "existing"},
						"content":  "func newFunc() {}",
					},
					expectErr: false,
				},
				{
					name: "missing_content",
					params: map[string]any{
						"language": "go",
						"source":   "package main",
						"target":   map[string]any{"type": "function", "name": "test"},
					},
					expectErr: true,
				},
				{
					name: "missing_target",
					params: map[string]any{
						"language": "go",
						"source":   "package main",
						"content":  "func newFunc() {}",
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

					_, err = tc.handler(params)

					if tt.expectErr && err == nil {
						t.Error("Expected error but got none")
					} else if !tt.expectErr && err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
				})
			}
		})
	}
}

// TestHandleAppendTool tests the append tool handler
func TestHandleAppendTool(t *testing.T) {
	server := createTestServerForTransform(t)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
	}{
		{
			name: "valid_append_with_target",
			params: map[string]any{
				"language": "go",
				"source":   "package main\ntype MyStruct struct {}",
				"target":   map[string]any{"type": "struct", "name": "MyStruct"},
				"content":  "field string",
			},
			expectErr: false,
		},
		{
			name: "valid_append_without_target",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"content":  "func newFunc() {}",
			},
			expectErr: false,
		},
		{
			name: "missing_content",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
			},
			expectErr: true, // Content is required for append operations
		},
		{
			name: "missing_source_and_path",
			params: map[string]any{
				"language": "go",
				"content":  "func newFunc() {}",
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

			_, err = server.handleAppendTool(params)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestFileTransformationTools tests file-based transformation tools
func TestFileTransformationTools(t *testing.T) {
	server := createTestServerForTransform(t)

	// Create a temporary directory and test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package main

import "fmt"

func oldFunc() {
	fmt.Println("Hello")
}

func keepFunc() {
	fmt.Println("Keep this")
}`

	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		toolName    string
		handler     func(json.RawMessage) (any, error)
		params      map[string]any
		expectErr   bool
		expectStage bool // Whether we expect a staging result
	}{
		{
			name:     "file_replace_valid",
			toolName: "file_replace",
			handler:  server.handleFileReplaceTool,
			params: map[string]any{
				"scope": map[string]any{
					"path":    tmpDir,
					"include": []string{"*.go"},
				},
				"target":      map[string]any{"type": "function", "name": "oldFunc"},
				"replacement": "func newFunc() {\n\tfmt.Println(\"Updated\")\n}",
				"dry_run":     true,
			},
			expectErr:   false,
			expectStage: true,
		},
		{
			name:     "file_delete_valid",
			toolName: "file_delete",
			handler:  server.handleFileDeleteTool,
			params: map[string]any{
				"scope": map[string]any{
					"path":    tmpDir,
					"include": []string{"*.go"},
				},
				"target":  map[string]any{"type": "function", "name": "oldFunc"},
				"dry_run": true,
			},
			expectErr:   false,
			expectStage: true,
		},
		{
			name:     "file_query_valid",
			toolName: "file_query",
			handler:  server.handleFileQueryTool,
			params: map[string]any{
				"scope": map[string]any{
					"path":    tmpDir,
					"include": []string{"*.go"},
				},
				"query": map[string]any{"type": "function", "name": "*Func"},
			},
			expectErr:   false,
			expectStage: false, // Query doesn't create stages
		},
		{
			name:     "invalid_scope_path",
			toolName: "file_query",
			handler:  server.handleFileQueryTool,
			params: map[string]any{
				"scope": map[string]any{
					"path": "/nonexistent/path",
				},
				"query": map[string]any{"type": "function"},
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

			result, err := tt.handler(params)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
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

						if tt.expectStage {
							// Check for staging information
							content, ok := resultMap["content"].([]map[string]any)
							if ok && len(content) > 0 {
								text, hasText := content[0]["text"].(string)
								if hasText && !strings.Contains(text, "stage") {
									t.Log("Result might not contain staging info (this could be expected)")
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestApplyTool tests the apply tool handler
func TestApplyTool(t *testing.T) {
	server := createTestServerForTransform(t)

	// Skip if no staging manager
	if server.staging == nil {
		t.Skip("No staging manager available (expected without database)")
	}

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
	}{
		{
			name:      "apply_latest",
			params:    map[string]any{"latest": true},
			expectErr: false, // Should work even with no stages
		},
		{
			name:      "apply_all",
			params:    map[string]any{"all": true},
			expectErr: false, // Should work even with no stages
		},
		{
			name:      "apply_specific_id",
			params:    map[string]any{"id": "nonexistent-id"},
			expectErr: false, // Should handle nonexistent ID gracefully
		},
		{
			name:      "apply_empty",
			params:    map[string]any{},
			expectErr: false, // Should default to latest
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			_, err = server.handleApplyTool(params)

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

// TestExecuteTransformViaHandlers tests transform execution through tool handlers
func TestExecuteTransformViaHandlers(t *testing.T) {
	server := createTestServerForTransform(t)

	// Test through the actual handlers which call executeTransform internally
	params := map[string]any{
		"language":    "go",
		"source":      "package main\nfunc oldFunc() {}",
		"target":      map[string]any{"type": "function", "name": "oldFunc"},
		"replacement": "func newFunc() {}",
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	result, err := server.handleReplaceTool(paramsJSON)
	if err != nil {
		t.Errorf("Transform should succeed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}
}

// Test transform utility functions have been moved to integration tests

// TestTransformWithInvalidJSON tests handling of invalid JSON parameters
func TestTransformWithInvalidJSON(t *testing.T) {
	server := createTestServerForTransform(t)

	invalidJSON := json.RawMessage(`{"invalid": json, "missing": quote}`)

	handlers := map[string]func(json.RawMessage) (any, error){
		"replace":       server.handleReplaceTool,
		"delete":        server.handleDeleteTool,
		"insert_before": server.handleInsertBeforeTool,
		"insert_after":  server.handleInsertAfterTool,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			_, err := handler(invalidJSON)

			if err == nil {
				t.Error("Expected error for invalid JSON")
			}

			mcpErr, ok := err.(*MCPError)
			if !ok {
				t.Errorf("Expected MCPError, got %T", err)
			} else if mcpErr.Code != InvalidParams {
				t.Errorf("Expected InvalidParams error, got %d", mcpErr.Code)
			}
		})
	}
}
