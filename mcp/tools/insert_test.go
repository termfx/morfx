package tools

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/termfx/morfx/mcp/types"
)

func TestInsertBeforeTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewInsertBeforeTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "insert_before_with_source",
			params: map[string]any{
				"language": "go",
				"source":   "package main\n\nfunc main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "// This comment goes before main\n",
			},
			expectErr: false,
		},
		{
			name: "insert_before_with_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "import \"log\"\n",
			},
			expectErr: false,
		},
		{
			name: "missing_content",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"target": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: false, // Go uses empty string as default value
		},
		{
			name: "empty_content_allowed",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc test() {}",
				"target": map[string]any{
					"type": "function",
					"name": "test",
				},
				"content": "",
			},
			expectErr: false, // Empty content should be allowed
		},
		{
			name: "multiline_content",
			params: map[string]any{
				"language": "go",
				"source":   "func main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": `// Multi-line comment
// Line 2
// Line 3
`,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if path is specified
			if path, ok := tt.params["path"].(string); ok && !tt.expectErr {
				tmpDir := t.TempDir()
				fullPath := filepath.Join(tmpDir, path)
				createTestFile(t, fullPath, "package main\n\nfunc main() {}")
				tt.params["path"] = fullPath
			}

			params := createTestParams(tt.params)
			result, err := tool.handle(context.Background(), params)

			if tt.expectErr {
				assertError(t, err, tt.errMsg)
			} else {
				assertNoError(t, err)
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestInsertAfterTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewInsertAfterTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "insert_after_with_source",
			params: map[string]any{
				"language": "go",
				"source":   "package main\n\nfunc main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "\n// This comment goes after main",
			},
			expectErr: false,
		},
		{
			name: "insert_after_with_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "\nfunc helper() {\n\t// Helper function\n}",
			},
			expectErr: false,
		},
		{
			name: "insert_after_struct",
			params: map[string]any{
				"language": "go",
				"source": `type User struct {
	Name string
}`,
				"target": map[string]any{
					"type": "struct",
					"name": "User",
				},
				"content": "\n// Methods for User go here",
			},
			expectErr: false,
		},
		{
			name: "missing_target",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"content":  "// Some content",
			},
			expectErr: true,
			errMsg:    "target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if path is specified
			if path, ok := tt.params["path"].(string); ok && !tt.expectErr {
				tmpDir := t.TempDir()
				fullPath := filepath.Join(tmpDir, path)
				createTestFile(t, fullPath, "package main\n\nfunc main() {\n\t// Main function\n}")
				tt.params["path"] = fullPath
			}

			params := createTestParams(tt.params)
			result, err := tool.handle(context.Background(), params)

			if tt.expectErr {
				assertError(t, err, tt.errMsg)
			} else {
				assertNoError(t, err)
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestInsertTools_Schema(t *testing.T) {
	server := newMockServer()

	tests := []struct {
		toolName string
		tool     types.Tool
	}{
		{"insert_before", NewInsertBeforeTool(server)},
		{"insert_after", NewInsertAfterTool(server)},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			// Verify tool metadata
			if tt.tool.Name() != tt.toolName {
				t.Errorf("Expected name '%s', got '%s'", tt.toolName, tt.tool.Name())
			}

			schema := tt.tool.InputSchema()

			// Verify required fields
			required, ok := schema["required"].([]string)
			if !ok {
				t.Fatal("Schema should have required array")
			}

			expectedRequired := []string{"language", "target", "content"}
			for _, req := range expectedRequired {
				found := slices.Contains(required, req)
				if !found {
					t.Errorf("Schema missing required field '%s'", req)
				}
			}

			// Verify content property exists
			properties, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatal("Schema should have properties")
			}

			if _, exists := properties["content"]; !exists {
				t.Error("Schema should have 'content' property")
			}
		})
	}
}

func TestInsertTools_WithStaging(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)

	tests := []struct {
		name   string
		tool   types.Tool
		params map[string]any
	}{
		{
			name: "insert_before_staging",
			tool: NewInsertBeforeTool(server),
			params: map[string]any{
				"language": "go",
				"source":   "func main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "// Before main\n",
			},
		},
		{
			name: "insert_after_staging",
			tool: NewInsertAfterTool(server),
			params: map[string]any{
				"language": "go",
				"source":   "func main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"content": "\n// After main",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := createTestParams(tt.params)
			result, err := tt.tool.Handler()(context.Background(), params)
			assertNoError(t, err)

			// Verify staging was used
			if resultMap, ok := result.(map[string]any); ok {
				// content should be an array of content blocks, not a map
				if _, hasContent := resultMap["content"].([]map[string]any); !hasContent {
					t.Fatal("Result should have content array")
				}

				if _, hasID := resultMap["id"].(string); !hasID {
					t.Error("Result should include stage identifier")
				}
				if res, ok := resultMap["result"].(string); !ok || res != "staged" {
					t.Errorf("Expected staged result, got %v", resultMap["result"])
				}
			}
		})
	}
}
