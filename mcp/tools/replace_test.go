package tools

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func TestReplaceTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewReplaceTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "replace_with_source",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc old() {}",
				"target": map[string]any{
					"type": "function",
					"name": "old",
				},
				"replacement": "func new() {\n\t// New implementation\n}",
			},
			expectErr: false,
		},
		{
			name: "replace_with_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"target": map[string]any{
					"type": "function",
					"name": "main",
				},
				"replacement": "func main() {\n\t// Modified\n}",
			},
			expectErr: false,
		},
		{
			name: "missing_replacement",
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
			name: "missing_target",
			params: map[string]any{
				"language":    "go",
				"source":      "package main",
				"replacement": "func new() {}",
			},
			expectErr: true,
			errMsg:    "target",
		},
		{
			name: "invalid_target_type",
			params: map[string]any{
				"language":    "go",
				"source":      "package main",
				"target":      "invalid",
				"replacement": "func new() {}",
			},
			expectErr: true,
			errMsg:    "Invalid target structure",
		},
		{
			name: "empty_replacement",
			params: map[string]any{
				"language": "go",
				"source":   "package main\nfunc test() {}",
				"target": map[string]any{
					"type": "function",
					"name": "test",
				},
				"replacement": "",
			},
			expectErr: false, // Empty replacement should be allowed (similar to delete)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if path is specified
			if path, ok := tt.params["path"].(string); ok && !tt.expectErr {
				tmpDir := t.TempDir()
				fullPath := filepath.Join(tmpDir, path)
				createTestFile(t, fullPath, "package main\nfunc main() {}")
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

				// Verify result contains transformation
				if resultMap, ok := result.(map[string]any); ok {
					if _, hasContent := resultMap["content"]; !hasContent {
						t.Error("Result should have 'content' field")
					}
				}
			}
		})
	}
}

func TestReplaceTool_WithStaging(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)
	tool := NewReplaceTool(server)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "staged.go")
	createTestFile(t, testFile, `package main

func oldFunction() {
	// Original implementation
}`)

	params := createTestParams(map[string]any{
		"language": "go",
		"path":     testFile,
		"target": map[string]any{
			"type": "function",
			"name": "oldFunction",
		},
		"replacement": `func newFunction() {
	// New implementation
}`,
	})

	result, err := tool.handle(context.Background(), params)
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
}

func TestReplaceTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewReplaceTool(server)

	// Verify tool metadata
	if tool.Name() != "replace" {
		t.Errorf("Expected name 'replace', got '%s'", tool.Name())
	}

	schema := tool.InputSchema()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	// Verify required properties
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema should have required array")
	}

	expectedRequired := []string{"language", "target", "replacement"}
	for _, req := range expectedRequired {
		found := slices.Contains(required, req)
		if !found {
			t.Errorf("Schema missing required field '%s'", req)
		}
	}

	// Verify replacement property exists
	if _, exists := properties["replacement"]; !exists {
		t.Error("Schema should have 'replacement' property")
	}
}
