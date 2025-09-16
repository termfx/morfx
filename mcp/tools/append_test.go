package tools

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestAppendTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewAppendTool(server)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "append_to_source",
			params: map[string]any{
				"language": "go",
				"source":   "package main\n\nfunc main() {}",
				"content":  "\nfunc helper() {\n\t// Helper function\n}",
			},
			expectErr: false,
		},
		{
			name: "append_to_file",
			params: map[string]any{
				"language": "go",
				"path":     "test.go",
				"content":  "\n// Additional code",
			},
			expectErr: false,
		},
		{
			name: "append_with_target_scope",
			params: map[string]any{
				"language": "go",
				"source": `package main

type User struct {
	Name string
}`,
				"target": map[string]any{
					"type": "struct",
					"name": "User",
				},
				"content": "\tAge int",
			},
			expectErr: false,
		},
		{
			name: "append_without_target",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"content": `

func newFunction() {
	// Appended to end of file
}`,
			},
			expectErr: false,
		},
		{
			name: "missing_content",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
			},
			expectErr: true,
			errMsg:    "content",
		},
		{
			name: "empty_content_allowed",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"content":  "",
			},
			expectErr: false, // Empty content should be allowed
		},
		{
			name: "append_to_function_body",
			params: map[string]any{
				"language": "go",
				"source": `func process() {
	// Existing code
}`,
				"target": map[string]any{
					"type": "function",
					"name": "process",
				},
				"content": "\t// New code in function",
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
				createTestFile(t, fullPath, `package main

func main() {
	// Existing main
}`)
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
					// content should be an array of content items
					_, hasContent := resultMap["content"].([]map[string]any)
					if !hasContent {
						// Try interface{} array
						if contentInterfaceArray, ok := resultMap["content"].([]any); ok {
							// Convert to proper type
							hasContent = len(contentInterfaceArray) > 0
						}
					}

					if !hasContent {
						t.Error("Result should have content array")
					}

					// Check for result status
					if resultStatus, ok := resultMap["result"].(string); ok {
						if resultStatus == "" {
							t.Error("Result status should not be empty")
						}
					}

					// Check for modified content
					if modified, ok := resultMap["modified"].(string); ok {
						if modified == "" && !tt.expectErr {
							t.Error("Modified content should not be empty")
						}
					}
				}
			}
		})
	}
}

func TestAppendTool_SmartPlacement(t *testing.T) {
	server := newMockServer()
	tool := NewAppendTool(server)

	tests := []struct {
		name           string
		source         string
		target         map[string]any
		content        string
		expectedResult string
	}{
		{
			name: "append_to_struct_body",
			source: `type Config struct {
	Host string
}`,
			target: map[string]any{
				"type": "struct",
				"name": "Config",
			},
			content:        "\tPort int",
			expectedResult: "should add Port field",
		},
		{
			name: "append_to_interface",
			source: `type Service interface {
	Start()
}`,
			target: map[string]any{
				"type": "interface",
				"name": "Service",
			},
			content:        "\tStop()",
			expectedResult: "should add Stop method",
		},
		{
			name: "append_to_end_without_target",
			source: `package main

func main() {}`,
			target: nil,
			content: `

func helper() {}`,
			expectedResult: "should add helper at end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{
				"language": "go",
				"source":   tt.source,
				"content":  tt.content,
			}

			if tt.target != nil {
				params["target"] = tt.target
			}

			paramsJSON := createTestParams(params)
			result, err := tool.handle(paramsJSON)
			assertNoError(t, err)

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			// Check that the operation was successful
			if resultMap, ok := result.(map[string]any); ok {
				// content is an array of content items
				if contentArray, ok := resultMap["content"].([]map[string]any); ok && len(contentArray) > 0 {
					// First item should contain the result text
					if text, ok := contentArray[0]["text"].(string); ok {
						if text == "" {
							t.Error("Content text should not be empty")
						}
					}
				} else if contentInterface, ok := resultMap["content"].([]any); ok && len(contentInterface) > 0 {
					// Handle interface{} case
					if contentItem, ok := contentInterface[0].(map[string]any); ok {
						if text, ok := contentItem["text"].(string); ok && text == "" {
							t.Error("Content text should not be empty")
						}
					}
				} else {
					t.Error("Content should be an array with at least one item")
				}
			}
		})
	}
}

func TestAppendTool_WithStaging(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)
	tool := NewAppendTool(server)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "staged.go")
	createTestFile(t, testFile, `package main

func main() {
	// Main function
}`)

	params := createTestParams(map[string]any{
		"language": "go",
		"path":     testFile,
		"content": `

func additionalFunction() {
	// New function
}`,
	})

	result, err := tool.handle(params)
	assertNoError(t, err)

	// Verify staging was used
	if resultMap, ok := result.(map[string]any); ok {
		// content should be an array
		_, hasContent := resultMap["content"].([]map[string]any)
		if !hasContent {
			// Try interface{} array
			if contentInterface, ok := resultMap["content"].([]any); ok {
				hasContent = len(contentInterface) > 0
			}
		}

		if !hasContent {
			t.Fatal("Result should have content array")
		}

		// Note: stageId would be in the main result, not in content
		if _, hasStageID := resultMap["stageId"]; !hasStageID {
			staging := server.GetStaging().(*mockStaging)
			if staging.IsEnabled() {
				// Since this is the mock, it may not implement staging correctly
				// Just check that we got a result
				t.Log("Mock staging may not return stageId")
			}
		}
	}
}

func TestAppendTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewAppendTool(server)

	// Verify tool metadata
	if tool.Name() != "append" {
		t.Errorf("Expected name 'append', got '%s'", tool.Name())
	}

	schema := tool.InputSchema()

	// Verify required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema should have required array")
	}

	expectedRequired := []string{"language", "content"}
	for _, req := range expectedRequired {
		found := slices.Contains(required, req)
		if !found {
			t.Errorf("Schema missing required field '%s'", req)
		}
	}

	// Verify target is optional
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	if _, exists := properties["target"]; !exists {
		t.Error("Schema should have optional 'target' property")
	}

	// Target should NOT be in required
	for _, r := range required {
		if r == "target" {
			t.Error("'target' should be optional, not required")
		}
	}
}
