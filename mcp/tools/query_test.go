package tools

import (
	"path/filepath"
	"testing"
)

func TestQueryTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewQueryTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "query_with_source",
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
			name: "query_with_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"query": map[string]any{
					"type": "function",
					"name": "main",
				},
			},
			expectErr: false, // Will create file in test
		},
		{
			name: "missing_language",
			params: map[string]any{
				"source": "package main",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
			errMsg:    "language",
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
			errMsg:    "Exactly one of 'source' or 'path' must be provided",
		},
		{
			name: "both_source_and_path",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"path":     "test.go",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: true,
			errMsg:    "Exactly one of 'source' or 'path' must be provided",
		},
		{
			name: "unsupported_language",
			params: map[string]any{
				"language": "rust",
				"source":   "fn main() {}",
				"query": map[string]any{
					"type": "function",
					"name": "main",
				},
			},
			expectErr: true,
			errMsg:    "No provider for language: rust",
		},
		{
			name: "invalid_query_type",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"query":    "invalid",
			},
			expectErr: true,
			errMsg:    "Invalid query structure",
		},
		{
			name: "empty_source",
			params: map[string]any{
				"language": "go",
				"source":   "",
				"query": map[string]any{
					"type": "function",
					"name": "test",
				},
			},
			expectErr: false, // Provider should handle empty source
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
			result, err := tool.handle(params)

			if tt.expectErr {
				assertError(t, err, tt.errMsg)
			} else {
				assertNoError(t, err)
				if result == nil {
					t.Error("Expected result but got nil")
				}

				// Verify result structure
				if resultMap, ok := result.(map[string]any); ok {
					if _, hasContent := resultMap["content"]; !hasContent {
						t.Error("Result should have 'content' field")
					}
				}
			}
		})
	}
}

func TestQueryTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewQueryTool(server)

	// Verify tool metadata
	if tool.Name() != "query" {
		t.Errorf("Expected name 'query', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Tool should have input schema")
	}

	// Verify schema structure
	if schema["type"] != "object" {
		t.Errorf("Schema type should be 'object', got %v", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	// Verify required properties exist
	requiredProps := []string{"language", "query"}
	for _, prop := range requiredProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Schema missing required property '%s'", prop)
		}
	}

	// Verify oneOf constraint
	oneOf, ok := schema["oneOf"].([]map[string]any)
	if !ok || len(oneOf) != 2 {
		t.Error("Schema should have oneOf constraint with 2 options")
	}
}

func TestQueryTool_FileHandling(t *testing.T) {
	server := newMockServer()
	tool := NewQueryTool(server)
	tmpDir := t.TempDir()

	// Test with valid file
	testFile := filepath.Join(tmpDir, "valid.go")
	createTestFile(t, testFile, `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	// Helper function
}`)

	params := createTestParams(map[string]any{
		"language": "go",
		"path":     testFile,
		"query": map[string]any{
			"type": "function",
			"name": "main",
		},
	})

	result, err := tool.handle(params)
	assertNoError(t, err)

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	// Test with non-existent file
	params = createTestParams(map[string]any{
		"language": "go",
		"path":     filepath.Join(tmpDir, "non_existent.go"),
		"query": map[string]any{
			"type": "function",
			"name": "test",
		},
	})

	_, err = tool.handle(params)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
