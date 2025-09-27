package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/termfx/morfx/mcp/types"
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
					"protocolVersion": supportedProtocolVersion,
					"capabilities":    map[string]any{},
					"clientInfo": map[string]any{
						"name":    "test-client",
						"version": "1.0.0",
					},
				}),
			},
			expectedResult: map[string]any{
				"protocolVersion": supportedProtocolVersion,
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
					"version": "1.5.0",
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
				"protocolVersion": supportedProtocolVersion,
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
					"version": "1.5.0",
				},
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t)

			response := server.handleInitialize(context.Background(), tt.request)

			assertJSONRPC(t, response)

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

	response := server.handleListTools(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}

	res, ok := response.Result.(listToolsResult)
	if !ok {
		t.Fatalf("result is not listToolsResult: %T", response.Result)
	}

	tools := res.Tools

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

	first := tools[0]
	if first.StructuredKey != "structuredContent" {
		t.Errorf("expected structured key 'structuredContent', got %q", first.StructuredKey)
	}
	if first.OutputSchema == nil {
		t.Error("expected output schema to be populated")
	}
	if stability, ok := first.Annotations["stability"].(string); !ok || stability == "" {
		t.Errorf("expected stability annotation, got %v", first.Annotations["stability"])
	}
	if progress, ok := first.Annotations["progress"].(bool); !ok || !progress {
		t.Errorf("expected progress annotation true, got %v", first.Annotations["progress"])
	}
}

// TestHandleListResources verifies resource listing functionality
func TestHandleListResources(t *testing.T) {
	server := createTestServer(t)

	request := Request{
		Method: "resources/list",
		ID:     1,
	}

	response := server.handleListResources(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}

	res, ok := response.Result.(listResourcesResult)
	if !ok {
		t.Fatalf("result is not listResourcesResult: %T", response.Result)
	}

	resources := res.Resources

	// Verify we have the expected resources
	expectedResources := []string{
		"morfx://server/info",
		"morfx://server/capabilities",
		"morfx://providers/languages",
		"morfx://session/current",
		"morfx://config/settings",
	}

	if len(resources) < len(expectedResources) {
		t.Errorf("resource count mismatch: got %d, want at least %d", len(resources), len(expectedResources))
	}

	resourceURIs := make(map[string]bool, len(resources))
	for _, resource := range resources {
		resourceURIs[resource.URI] = true
	}

	var serverInfo *types.ResourceDefinition
	for i := range resources {
		if resources[i].URI == "morfx://server/info" {
			serverInfo = &resources[i]
			break
		}
	}
	if serverInfo == nil {
		t.Fatal("expected morfx://server/info definition")
	}
	if serverInfo.Size == nil || *serverInfo.Size == 0 {
		t.Errorf("expected server info resource to report size, got %v", serverInfo.Size)
	}
	if cache, ok := serverInfo.Annotations["cacheControl"]; !ok || cache != "static" {
		t.Errorf("expected cacheControl=static annotation, got %v", serverInfo.Annotations["cacheControl"])
	}
	if audience, ok := serverInfo.Annotations["audience"]; !ok || audience != "developer" {
		t.Errorf("expected audience=developer annotation, got %v", serverInfo.Annotations["audience"])
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

	response := server.handleListPrompts(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Fatalf("unexpected error: %v", response.Error)
	}

	res, ok := response.Result.(listPromptsResult)
	if !ok {
		t.Fatalf("result is not listPromptsResult: %T", response.Result)
	}

	prompts := res.Prompts

	// Verify we have the expected prompts
	expectedPrompts := []string{
		"code-analysis",
		"transformation-guide",
		"confidence-explanation",
		"query-builder",
		"best-practices",
	}

	if len(prompts) < len(expectedPrompts) {
		t.Errorf("prompt count mismatch: got %d, want at least %d", len(prompts), len(expectedPrompts))
	}

	promptNames := make(map[string]bool, len(prompts))
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
		JSONRPC: JSONRPCVersion,
		Method:  "unknown/method",
		ID:      1,
	}

	response := server.handleRequest(request)

	assertJSONRPC(t, response)

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
	config.LogWriter = io.Discard

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

func assertJSONRPC(t *testing.T, resp Response) {
	t.Helper()
	if resp.JSONRPC != JSONRPCVersion {
		t.Fatalf("expected jsonrpc %s, got %s", JSONRPCVersion, resp.JSONRPC)
	}
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

	ctx := context.Background()
	for b.Loop() {
		server.handleInitialize(ctx, request)
	}
}

func BenchmarkHandleListTools(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := Request{
		Method: "tools/list",
		ID:     1,
	}

	for b.Loop() {
		server.handleListTools(context.Background(), request)
	}
}

// TestHandleCallToolWithMCPError tests call tool with MCP errors
func TestHandleCallToolWithMCPError(t *testing.T) {
	server := createTestServer(t)

	// Register a tool that returns an MCP error
	server.RegisterTool("error_tool", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, NewMCPError(-32603, "Intentional test error", map[string]any{
			"details": "This is a test error",
			"code":    "TEST_ERROR",
		})
	})

	request := Request{
		Method: "tools/call",
		ID:     1,
		Params: mustMarshal(map[string]any{
			"name":      "error_tool",
			"arguments": map[string]any{},
		}),
	}

	response := server.handleCallTool(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Fatalf("Unexpected transport error: %v", response.Error)
	}

	callResult, ok := response.Result.(types.CallToolResult)
	if !ok {
		t.Fatalf("Expected CallToolResult, got %T", response.Result)
	}

	if !callResult.IsError {
		t.Fatal("Expected CallToolResult to be marked as error")
	}

	structured, ok := callResult.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("Expected structured content map, got %T", callResult.StructuredContent)
	}

	switch code := structured["code"].(type) {
	case int:
		if code != -32603 {
			t.Errorf("Expected structured error code -32603, got %d", code)
		}
	case float64:
		if int(code) != -32603 {
			t.Errorf("Expected structured error code -32603, got %v", code)
		}
	default:
		t.Errorf("Unexpected code type %T", structured["code"])
	}

	if structured["message"] != "Intentional test error" {
		t.Errorf("Expected structured error message, got %v", structured["message"])
	}
}

// TestHandleCallToolWithGenericError tests call tool with generic errors
func TestHandleCallToolWithGenericError(t *testing.T) {
	server := createTestServer(t)

	// Register a tool that returns a generic error
	server.RegisterTool("generic_error_tool", func(ctx context.Context, params json.RawMessage) (any, error) {
		return nil, errors.New("generic error message")
	})

	request := Request{
		Method: "tools/call",
		ID:     1,
		Params: mustMarshal(map[string]any{
			"name":      "generic_error_tool",
			"arguments": map[string]any{},
		}),
	}

	response := server.handleCallTool(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error == nil {
		t.Fatal("Expected error response")
	}

	// Generic errors should be converted to InternalError
	if response.Error.Code != InternalError {
		t.Errorf("Expected error code %d, got %d", InternalError, response.Error.Code)
	}

	if !strings.Contains(response.Error.Message, "generic error message") {
		t.Errorf("Expected error message to contain 'generic error message', got %s", response.Error.Message)
	}
}

// TestHandleInitializedNotification tests the initialized notification handler
func TestHandleInitializedNotification(t *testing.T) {
	server := createTestServer(t)

	// Test notification (no ID)
	request := Request{
		Method: "initialized",
		ID:     nil, // Notification has no ID
	}

	response := server.handleInitialized(context.Background(), request)

	// Should return empty response for notification
	if response.JSONRPC != "" {
		t.Error("Notification response should be empty")
	}

	// Test with ID (shouldn't happen but handle gracefully)
	request.ID = 1
	response = server.handleInitialized(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}
}

// TestHandlePing tests the ping handler
func TestHandlePing(t *testing.T) {
	server := createTestServer(t)

	request := Request{
		Method: "ping",
		ID:     "ping-123",
	}

	response := server.handlePing(context.Background(), request)

	assertJSONRPC(t, response)

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	if response.ID != "ping-123" {
		t.Errorf("Expected ID 'ping-123', got %v", response.ID)
	}

	// Ping should return empty result
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Errorf("Expected result to be a map, got %T", response.Result)
	} else if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

// TestHandleRequestRouting tests the request routing functionality
func TestHandleRequestRouting(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name            string
		method          string
		expectError     bool
		expectedErrCode int
	}{
		{"initialize", "initialize", false, 0},
		{"initialized", "initialized", false, 0},
		{"ping", "ping", false, 0},
		{"tools_list", "tools/list", false, 0},
		{"tools_call", "tools/call", true, InvalidParams}, // Missing params
		{"resources_list", "resources/list", false, 0},
		{"resources_read", "resources/read", true, InvalidParams}, // Missing params
		{"prompts_list", "prompts/list", false, 0},
		{"prompts_get", "prompts/get", true, InvalidParams}, // Missing params
		{"unknown_method", "unknown/method", true, MethodNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := Request{
				JSONRPC: JSONRPCVersion,
				Method:  tt.method,
				ID:      1,
			}

			switch tt.method {
			case "initialize":
				request.Params = mustMarshal(map[string]any{
					"protocolVersion": supportedProtocolVersion,
					"capabilities":    map[string]any{},
				})
			case "tools/call", "resources/read", "prompts/get":
				// leave params nil to trigger invalid params as expected
			}

			response := server.handleRequest(request)

			assertJSONRPC(t, response)

			if tt.expectError {
				if response.Error == nil {
					t.Error("Expected error but got none")
				} else if tt.expectedErrCode != 0 && response.Error.Code != tt.expectedErrCode {
					t.Errorf("Expected error code %d, got %d", tt.expectedErrCode, response.Error.Code)
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error)
				}
			}
		})
	}
}

// TestHandleInitializeWithVaryingClientInfo tests initialize with different client info
func TestHandleInitializeWithVaryingClientInfo(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name       string
		clientInfo map[string]any
	}{
		{
			name: "complete_client_info",
			clientInfo: map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
		{
			name: "minimal_client_info",
			clientInfo: map[string]any{
				"name": "minimal-client",
			},
		},
		{
			name:       "empty_client_info",
			clientInfo: map[string]any{},
		},
		{
			name:       "nil_client_info",
			clientInfo: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{
				"protocolVersion": supportedProtocolVersion,
				"capabilities":    map[string]any{},
			}

			if tt.clientInfo != nil {
				params["clientInfo"] = tt.clientInfo
			}

			request := Request{
				Method: "initialize",
				ID:     1,
				Params: mustMarshal(params),
			}

			response := server.handleInitialize(context.Background(), request)

			assertJSONRPC(t, response)

			if response.Error != nil {
				t.Errorf("Unexpected error: %v", response.Error)
			}

			// All should succeed and return proper server info
			result, ok := response.Result.(map[string]any)
			if !ok {
				t.Fatal("Result should be a map")
			}

			serverInfo := result["serverInfo"].(map[string]any)
			if serverInfo["name"] != "morfx" {
				t.Errorf("Expected server name 'morfx', got %v", serverInfo["name"])
			}
		})
	}
}
