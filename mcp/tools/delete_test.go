package tools

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func TestDeleteTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewDeleteTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "delete_with_source",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc unused() {}\nfunc main() {}",
				"target": map[string]any{
					"type": "function",
					"name": "unused",
				},
			},
			expectErr: false,
		},
		{
			name: "delete_with_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"target": map[string]any{
					"type": "function",
					"name": "helper",
				},
			},
			expectErr: false,
		},
		{
			name: "missing_target",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
			},
			expectErr: true,
			errMsg:    "target",
		},
		{
			name: "invalid_target_type",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"target":   []string{"invalid"},
			},
			expectErr: true,
			errMsg:    "target must be an object",
		},
		{
			name: "delete_non_existent",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"target": map[string]any{
					"type": "function",
					"name": "nonExistent",
				},
			},
			expectErr: false, // Provider should handle gracefully
		},
		{
			name: "delete_multiple_matches",
			params: map[string]any{
				"language": "go",
				"source": `package main
func test() {}
func test() {} // Duplicate`,
				"target": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: false, // Provider should handle multiple matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if path is specified
			if path, ok := tt.params["path"].(string); ok && !tt.expectErr {
				tmpDir := t.TempDir()
				fullPath := filepath.Join(tmpDir, path)
				createTestFile(t, fullPath, `package main
func main() {}
func helper() {
	// Helper function
}`)
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

func TestDeleteTool_WithStaging(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)
	tool := NewDeleteTool(server)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "staged.go")
	createTestFile(t, testFile, `package main

func toDelete() {
	// Will be deleted
}

func main() {
	// Keep this
}`)

	params := createTestParams(map[string]any{
		"language": "go",
		"path":     testFile,
		"target": map[string]any{
			"type": "function",
			"name": "toDelete",
		},
	})

	result, err := tool.handle(context.Background(), params)
	assertNoError(t, err)

	// Verify staging was used
	if resultMap, ok := result.(map[string]any); ok {
		blocks, hasBlocks := resultMap["content"].([]map[string]any)
		if !hasBlocks || len(blocks) == 0 {
			t.Fatal("Result should include content blocks")
		}

		if res, ok := resultMap["result"].(string); !ok || res != "staged" {
			t.Errorf("Expected result to be 'staged', got %v", resultMap["result"])
		}

		if _, hasID := resultMap["id"].(string); !hasID {
			t.Error("Result should include stage identifier")
		}
	}
}

func TestDeleteTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewDeleteTool(server)

	// Verify tool metadata
	if tool.Name() != "delete" {
		t.Errorf("Expected name 'delete', got '%s'", tool.Name())
	}

	schema := tool.InputSchema()

	// Verify required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema should have required array")
	}

	expectedRequired := []string{"language", "target"}
	for _, req := range expectedRequired {
		found := slices.Contains(required, req)
		if !found {
			t.Errorf("Schema missing required field '%s'", req)
		}
	}

	// Verify no replacement field (unlike replace tool)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	if _, exists := properties["replacement"]; exists {
		t.Error("Delete tool should not have 'replacement' property")
	}
}
