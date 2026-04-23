package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestRegistry_Init(t *testing.T) {
	server := newMockServer()
	Init(server)

	if Registry == nil {
		t.Fatal("Registry should be initialized")
	}

	if Registry.server != server {
		t.Error("Registry should store the server reference")
	}

	// Verify all tools are registered
	expectedTools := []string{
		"query", "file_query",
		"replace", "file_replace",
		"delete", "file_delete",
		"insert_before", "insert_after",
		"append", "apply",
	}

	for _, name := range expectedTools {
		if _, exists := Registry.Get(name); !exists {
			t.Errorf("Tool '%s' should be registered", name)
		}
	}
}

func TestRegistry_Register(t *testing.T) {
	server := newMockServer()
	Init(server)

	// Create a custom tool
	customTool := &BaseTool{
		name:        "custom_tool",
		description: "Test custom tool",
		inputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		},
		handler: func(ctx context.Context, params json.RawMessage) (any, error) {
			return map[string]any{"result": "custom"}, nil
		},
	}

	Registry.Register("custom_tool", customTool)

	tool, exists := Registry.Get("custom_tool")
	if !exists {
		t.Error("Custom tool should be registered")
	}

	if tool.Name() != "custom_tool" {
		t.Errorf("Expected tool name 'custom_tool', got '%s'", tool.Name())
	}
}

func TestRegistry_Execute(t *testing.T) {
	server := newMockServer()
	Init(server)

	tests := []struct {
		name      string
		tool      string
		params    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "execute_existing_tool",
			tool: "query",
			params: map[string]any{
				"language": "go",
				"source":   "package main",
				"query": map[string]any{
					"type": "function",
					"name": "main",
				},
			},
			expectErr: false,
		},
		{
			name:      "execute_non_existing_tool",
			tool:      "non_existing",
			params:    map[string]any{},
			expectErr: true,
			errMsg:    "tool not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := createTestParams(tt.params)
			result, err := Registry.Execute(context.Background(), tt.tool, params)

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

func TestRegistry_List(t *testing.T) {
	server := newMockServer()
	Init(server)

	tools := Registry.List()

	if len(tools) < 10 {
		t.Errorf("Expected at least 10 tools, got %d", len(tools))
	}

	// Verify tools are returned in registration order
	expectedOrder := []string{
		"query", "file_query", "replace", "file_replace",
	}

	for i, expectedName := range expectedOrder {
		if i >= len(tools) {
			break
		}
		if tools[i].Name() != expectedName {
			t.Errorf("Expected tool at position %d to be '%s', got '%s'",
				i, expectedName, tools[i].Name())
		}
	}
}

func TestRegistry_ThreadSafety(t *testing.T) {
	server := newMockServer()
	Init(server)

	done := make(chan bool)

	// Concurrent reads
	for range 10 {
		go func() {
			for range 100 {
				Registry.Get("query")
				Registry.List()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := range 5 {
		go func(id int) {
			toolName := fmt.Sprintf("concurrent_%d", id)
			tool := &BaseTool{
				name:        toolName,
				description: "Concurrent test tool",
				inputSchema: map[string]any{},
				handler: func(ctx context.Context, params json.RawMessage) (any, error) {
					return nil, nil
				},
			}
			for range 50 {
				Registry.Register(toolName, tool)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 15 {
		<-done
	}

	// Verify registry is still consistent
	if _, exists := Registry.Get("query"); !exists {
		t.Error("Registry corrupted: missing 'query' tool")
	}
}
