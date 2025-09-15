package mcp

import (
	"encoding/json"
	"testing"
)

// TestHandleInitialize verifies MCP initialization protocol compliance
func TestHandleInitialize(t *testing.T) {
	tests := []struct {
		name           string
		request        Request
		expectedResult map[string]any
		expectError    bool
	}{
		{
			name: "valid_initialization",
			request: Request{
				Method: "initialize",
				ID:     1,
				Params: mustMarshal(map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{},
					"clientInfo": map[string]any{
						"name":    "test-client",
						"version": "1.0.0",
					},
				}),
			},
			expectedResult: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": true,
					},
					"resources": map[string]any{
						"subscribe":   true,
						"listChanged": true,
					},
					"prompts": map[string]any{
						"listChanged": true,
					},
					"logging": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "morfx",
					"version": "1.3.0",
				},
			},
			expectError: false,
		},
		{
			name: "initialization_with_invalid_params",
			request: Request{
				Method: "initialize",
				ID:     2,
				Params: json.RawMessage(`{"invalid": "data"}`),
			},
			expectedResult: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": true,
					},
					"resources": map[string]any{
						"subscribe":   true,
						"listChanged": true,
					},
					"prompts": map[string]any{
						"listChanged": true,
					},
					"logging": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "morfx",
					"version": "1.3.0",
				},
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t)
			
			response := server.handleInitialize(tt.request)
			
			// Verify response structure
			if response.Error != nil && !tt.expectError {
				t.Fatalf("unexpected error: %v", response.Error)
			}
			
			if response.Error == nil && tt.expectError {
				t.Fatal("expected error but got none")
			}
			
			if !tt.expectError {
				// Verify specific fields
				result, ok := response.Result.(map[string]any)
				if !ok {
					t.Fatal("result is not a map")
				}
				
				if result["protocolVersion"] != tt.expectedResult["protocolVersion"] {
					t.Errorf("protocol version mismatch: got %v, want %v",
						result["protocolVersion"], tt.expectedResult["protocolVersion"])
				}
				
				// Verify server info
				serverInfo, ok := result["serverInfo"].(map[string]any)
				if !ok {
					t.Fatal("serverInfo is not a map")
				}
				
				expectedServerInfo := tt.expectedResult["serverInfo"].(map[string]any)
				if serverInfo["name"] != expectedServerInfo["name"] {
					t.Errorf("server name mismatch: got %v, want %v",
						serverInfo["name"], expectedServerInfo["name"])
				}
			}
		})
	}
}

// TestHandleListTools verifies tool listing functionality
func TestHandleListTools(t *testing.T) {
	server := createTestServer(t)
	
	request := Request{
		Method: "tools/list",
		ID:     1,
	}
	
	response := server.handleListTools(request)
	
	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}
	
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("result is not a map")
	}
	
	tools, ok := result["tools"].([]ToolDefinition)
	if !ok {
		t.Fatal("tools is not a ToolDefinition slice")
	}
	
	// Verify we have the expected tools
	expectedTools := []string{
		"query", "file_query", "replace", "file_replace",
		"delete", "file_delete", "insert_before", "insert_after",
		"apply", "append",
	}
	
	if len(tools) != len(expectedTools) {
		t.Errorf("tool count mismatch: got %d, want %d", len(tools), len(expectedTools))
	}
	
	// Verify specific tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("missing expected tool: %s", expected)
		}
	}
}

// TestHandleListResources verifies resource listing functionality
func TestHandleListResources(t *testing.T) {
	server := createTestServer(t)
	
	request := Request{
		Method: "resources/list",
		ID:     1,
	}
	
	response := server.handleListResources(request)
	
	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}
	
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("result is not a map")
	}
	
	resources, ok := result["resources"].([]ResourceDefinition)
	if !ok {
		t.Fatal("resources is not a ResourceDefinition slice")
	}
	
	// Verify we have the expected resources
	expectedResources := []string{
		"morfx://server/info",
		"morfx://server/capabilities", 
		"morfx://providers/languages",
		"morfx://session/current",
		"morfx://config/settings",
	}
	
	if len(resources) != len(expectedResources) {
		t.Errorf("resource count mismatch: got %d, want %d", len(resources), len(expectedResources))
	}
	
	// Verify specific resources exist
	resourceURIs := make(map[string]bool)
	for _, resource := range resources {
		resourceURIs[resource.URI] = true
	}
	
	for _, expected := range expectedResources {
		if !resourceURIs[expected] {
			t.Errorf("missing expected resource: %s", expected)
		}
	}
}

// TestHandleListPrompts verifies prompt listing functionality
func TestHandleListPrompts(t *testing.T) {
	server := createTestServer(t)
	
	request := Request{
		Method: "prompts/list",
		ID:     1,
	}
	
	response := server.handleListPrompts(request)
	
	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}
	
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("result is not a map")
	}
	
	prompts, ok := result["prompts"].([]PromptDefinition)
	if !ok {
		t.Fatal("prompts is not a PromptDefinition slice")
	}
	
	// Verify we have the expected prompts
	expectedPrompts := []string{
		"code-analysis",
		"transformation-guide",
		"confidence-explanation",
		"query-builder",
		"best-practices",
	}
	
	if len(prompts) != len(expectedPrompts) {
		t.Errorf("prompt count mismatch: got %d, want %d", len(prompts), len(expectedPrompts))
	}
	
	// Verify specific prompts exist
	promptNames := make(map[string]bool)
	for _, prompt := range prompts {
		promptNames[prompt.Name] = true
	}
	
	for _, expected := range expectedPrompts {
		if !promptNames[expected] {
			t.Errorf("missing expected prompt: %s", expected)
		}
	}
}

// TestMethodNotFound verifies proper error handling for unknown methods
func TestMethodNotFound(t *testing.T) {
	server := createTestServer(t)
	
	request := Request{
		Method: "unknown/method",
		ID:     1,
	}
	
	response := server.handleRequest(request)
	
	if response.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	
	if response.Error.Code != MethodNotFound {
		t.Errorf("wrong error code: got %d, want %d", response.Error.Code, MethodNotFound)
	}
}

// Helper functions for testing

func createTestServer(t *testing.T) *StdioServer {
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

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Benchmark tests for performance verification

func BenchmarkHandleInitialize(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := Request{
		Method: "initialize",
		ID:     1,
		Params: mustMarshal(map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "benchmark-client",
				"version": "1.0.0",
			},
		}),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.handleInitialize(request)
	}
}

func BenchmarkHandleListTools(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := Request{
		Method: "tools/list",
		ID:     1,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.handleListTools(request)
	}
}