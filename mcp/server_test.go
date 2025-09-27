package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/tools"
	"github.com/termfx/morfx/mcp/types"
	"github.com/termfx/morfx/models"
	"github.com/termfx/morfx/providers"
)

func newServerConfig() Config {
	cfg := DefaultConfig()
	cfg.LogWriter = io.Discard
	return cfg
}

// TestNewStdioServer tests server creation
func TestNewStdioServer(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip" // Skip database for unit tests
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.config.DatabaseURL != "skip" {
		t.Error("Config not set properly")
	}

	if server.providers == nil {
		t.Error("Providers registry should be initialized")
	}

	if server.toolRegistry == nil {
		t.Error("Tool registry should be initialized")
	}

	if server.fileProcessor == nil {
		t.Error("File processor should be initialized")
	}
}

// TestNewStdioServerWithDatabase tests server creation with database
func TestNewStdioServerWithDatabase(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = ":memory:" // Use in-memory SQLite for tests
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server with database: %v", err)
	}

	if server.db == nil {
		t.Error("Database should be initialized")
	}

	if server.session == nil {
		t.Error("Session should be initialized")
	}
}

// TestServerProviderAccess tests provider retrieval through registry
func TestServerProviderAccess(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test getting Go provider through the registry
	goProvider, exists := server.providers.Get("go")
	if !exists {
		t.Error("Go provider should exist")
	}

	if goProvider != nil && goProvider.Language() != "go" {
		t.Errorf("Expected language 'go', got '%s'", goProvider.Language())
	}

	// Test getting non-existent provider
	provider, exists := server.providers.Get("nonexistent")
	if exists || provider != nil {
		t.Error("Should return false and nil for non-existent provider")
	}
}

// TestServerFileProcessorQuery tests querying through file processor
func TestServerFileProcessorQuery(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.fileProcessor == nil {
		t.Fatal("File processor should be initialized")
	}

	// Test that file processor has provider registry
	// (We can't easily test actual queries without more complex setup)
	t.Log("File processor initialized successfully")
}

// TestServerLanguageDetection tests language detection using file processor
func TestServerLanguageDetection(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test provider exists for expected languages
	languages := []string{"go", "javascript", "typescript", "php", "python"}

	for _, lang := range languages {
		provider, exists := server.providers.Get(lang)
		if !exists || provider == nil {
			t.Errorf("Provider for language '%s' should be available", lang)
		}
	}
}

// TestRegisterTool tests tool registration
func TestRegisterTool(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Register a test tool
	testTool := func(ctx context.Context, params json.RawMessage) (any, error) {
		return "test result", nil
	}

	server.RegisterTool("test_tool", testTool)

	// Execute the tool via registry to ensure it was registered
	result, err := server.toolRegistry.Execute(context.Background(), "test_tool", nil)
	if err != nil {
		t.Fatalf("Registered tool should execute without error: %v", err)
	}

	if result != "test result" {
		t.Errorf("Expected tool to return 'test result', got %v", result)
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestServerClose tests server cleanup
func TestServerClose(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Call Close (should not panic)
	server.Close()

	// Multiple calls to Close should be safe
	server.Close()
}

// TestServerToolsInitialization tests that built-in tools are registered
func TestServerToolsInitialization(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check that some built-in tools are registered
	toolCount := len(server.toolRegistry.List())

	if toolCount == 0 {
		t.Error("Expected built-in tools to be registered")
	}

	t.Logf("Found %d registered tools", toolCount)
}

// TestProviderRegistryAdapter tests the provider registry adapter
func TestProviderRegistryAdapter(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test the adapter through the server's provider registry
	provider, exists := server.providers.Get("go")
	if !exists || provider == nil {
		t.Fatal("Go provider should be available")
	}

	if provider.Language() != "go" {
		t.Errorf("Expected language 'go', got '%s'", provider.Language())
	}

	// Test provider query functionality
	goCode := `package main
func test() {}`

	result := provider.Query(goCode, core.AgentQuery{
		Type: "function",
		Name: "test",
	})

	if result.Error != nil {
		t.Fatalf("Provider query failed: %v", result.Error)
	}

	if len(result.Matches) == 0 {
		t.Error("Expected to find function match")
	}
}

// TestProviderAdapter tests the provider adapter
func TestProviderAdapter(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	provider, exists := server.providers.Get("go")
	if !exists || provider == nil {
		t.Fatal("Go provider should be available")
	}

	// Test transform functionality
	goCode := `package main
func OldName() {}`

	result := provider.Transform(goCode, core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "OldName",
		},
		Replacement: "func NewName() {}",
	})

	if result.Error != nil {
		t.Fatalf("Provider transform failed: %v", result.Error)
	}

	if !contains(result.Modified, "NewName") {
		t.Error("Transform should contain new name")
	}
}

// TestServerConfigValidation tests configuration validation
func TestServerConfigValidation(t *testing.T) {
	// Test with minimal valid config
	config := Config{
		DatabaseURL: "skip",
		Debug:       false,
		// Other fields will use defaults
	}

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Minimal config should be valid: %v", err)
	}

	if server == nil {
		t.Error("Server should be created with minimal config")
	}
}

// TestServerComponentInitialization tests that all components are properly initialized
func TestServerComponentInitialization(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check all critical components are initialized
	components := map[string]any{
		"providers":     server.providers,
		"fileProcessor": server.fileProcessor,
		"toolRegistry":  server.toolRegistry,
		"staging":       server.staging,
		"safety":        server.safety,
	}

	for name, component := range components {
		if component == nil {
			t.Errorf("Component '%s' should be initialized", name)
		}
	}
}

// TestNewStdioServerDatabaseConnectionFailed tests server creation when database connection fails
func TestNewStdioServerDatabaseConnectionFailed(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "/dev/null/impossible/path/database.db"
	config.Debug = false

	// Should not fail even with invalid database URL
	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Server creation should not fail with invalid database URL: %v", err)
	}

	// Database should be nil
	if server.db != nil {
		t.Error("Database should be nil when connection fails")
	}

	// Session should be nil
	if server.session != nil {
		t.Error("Session should be nil when database connection fails")
	}

	// Staging should be nil
	if server.staging != nil {
		t.Error("Staging should be nil when database connection fails")
	}
}

// TestServerHandleRequest tests request routing
func TestServerHandleRequest(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	testCases := []struct {
		method       string
		expectedType string
		shouldError  bool
	}{
		{"initialize", "success", false},
		{"initialized", "success", false},
		{"ping", "success", false},
		{"tools/list", "success", false},
		{"prompts/list", "success", false},
		{"resources/list", "success", false},
		{"unknown/method", "error", true},
	}

	for _, tc := range testCases {
		params := json.RawMessage(`{}`)
		if tc.method == "initialize" {
			params = mustMarshal(map[string]any{
				"protocolVersion": supportedProtocolVersion,
				"capabilities":    map[string]any{},
			})
		}

		req := Request{
			JSONRPC: JSONRPCVersion,
			ID:      &[]string{"test-" + tc.method}[0],
			Method:  tc.method,
			Params:  params,
		}

		response := server.handleRequest(req)

		if tc.shouldError {
			if response.Error == nil {
				t.Errorf("Expected error for method %s", tc.method)
			}
		} else {
			if response.Error != nil {
				t.Errorf("Unexpected error for method %s: %v", tc.method, response.Error)
			}
		}
	}
}

// TestServerSendResponse tests response sending
func TestServerSendResponse(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	server.writer = bufio.NewWriter(&buf)

	response := Response{
		ID:     &[]string{"test-123"}[0],
		Result: map[string]any{"status": "ok"},
	}

	server.sendResponse(response)

	output := buf.String()
	if output == "" {
		t.Error("Expected output from sendResponse")
	}

	// Should be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
}

// TestServerLanguageAdapter tests the language method of provider adapter
func TestServerLanguageAdapter(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test the Language method through the adapter
	registryAdapter := &providerRegistryAdapter{server.providers}
	provider, exists := registryAdapter.Get("go")
	if !exists {
		t.Fatal("Go provider should exist")
	}

	language := provider.Language()
	if language != "go" {
		t.Errorf("Expected language 'go', got '%s'", language)
	}
}

// TestServerDebugLogging tests debug logging functionality
func TestServerDebugLogging(t *testing.T) {
	// Test with debug enabled
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = true

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Debug log function should be set and not panic
	server.debugLog("Test debug message: %s", "test")

	// Test with debug disabled
	config.Debug = false
	server2, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Should not panic even when debug disabled
	server2.debugLog("This should not log: %s", "test")
}

// TestServerCloseWithDatabase tests server close with database
func TestServerCloseWithDatabase(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = ":memory:"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server with database: %v", err)
	}

	if server.db == nil {
		t.Skip("Database not initialized, skipping close test")
	}

	// Close should not error
	err = server.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

// TestServerStart_EOF tests server shutdown on EOF
func TestServerStart_EOF(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Set up reader that immediately returns EOF
	emptyReader := strings.NewReader("")
	server.reader = bufio.NewReader(emptyReader)
	server.writer = bufio.NewWriter(&buf)

	// Start should return nil on EOF
	err = server.Start()
	if err != nil {
		t.Errorf("Start should return nil on EOF, got: %v", err)
	}
}

// TestServerStart_InvalidJSON tests server handling of invalid JSON
func TestServerStart_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Input with invalid JSON followed by EOF
	invalidJSON := `{"invalid": json, "missing": quote}`
	reader := strings.NewReader(invalidJSON)
	server.reader = bufio.NewReader(reader)
	server.writer = bufio.NewWriter(&buf)

	// Should handle invalid JSON gracefully and continue
	err = server.Start()
	if err != nil {
		t.Errorf("Start should handle invalid JSON gracefully: %v", err)
	}

	// Should have sent error response
	output := buf.String()
	decoder := json.NewDecoder(strings.NewReader(output))
	foundError := false
	for decoder.More() {
		var msg map[string]any
		if err := decoder.Decode(&msg); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if errObj, ok := msg["error"].(map[string]any); ok {
			if code, ok := errObj["code"].(float64); ok && int(code) == ParseError {
				foundError = true
				break
			}
		}
	}
	if !foundError {
		t.Error("Expected parse error response")
	}
}

func TestRequestSamplingRecordsHistory(t *testing.T) {
	server := createTestServer(t)
	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	params := map[string]any{"purpose": "unit-test"}
	ctx := withProgressToken(context.Background(), "token-sample")

	resultCh := make(chan struct {
		data map[string]any
		err  error
	}, 1)

	go func() {
		resp, err := server.RequestSampling(ctx, params)
		resultCh <- struct {
			data map[string]any
			err  error
		}{resp, err}
	}()

	req := waitForOutboundRequest(t, server, &buf)
	if req.Meta == nil || req.Meta["progressToken"] != "token-sample" {
		t.Fatalf("expected progress token propagated, got meta=%v", req.Meta)
	}

	responsePayload := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "sample guidance"},
				},
			},
		},
	}

	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  responsePayload,
	})

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("RequestSampling returned error: %v", res.err)
	}
	if res.data == nil {
		t.Fatalf("expected sampling result data")
	}

	history := server.sessionState.SamplingHistory()
	if len(history) != 1 {
		t.Fatalf("expected one sampling record, got %d", len(history))
	}
	if history[0].Params["purpose"] != "unit-test" {
		t.Errorf("unexpected sampling params: %#v", history[0].Params)
	}
}

func TestRequestElicitationRecordsHistory(t *testing.T) {
	server := createTestServer(t)
	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	ctx := withProgressToken(context.Background(), "token-elic")
	params := map[string]any{"title": "Confirm", "choices": []map[string]any{{"label": "OK", "value": "confirm"}}}

	resultCh := make(chan struct {
		data map[string]any
		err  error
	}, 1)

	go func() {
		resp, err := server.RequestElicitation(ctx, params)
		resultCh <- struct {
			data map[string]any
			err  error
		}{resp, err}
	}()

	req := waitForOutboundRequest(t, server, &buf)
	if req.Meta == nil || req.Meta["progressToken"] != "token-elic" {
		t.Fatalf("expected progress token propagated, got meta=%v", req.Meta)
	}

	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"choice": "confirm"},
	})

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("RequestElicitation returned error: %v", res.err)
	}

	history := server.sessionState.ElicitationHistory()
	if len(history) != 1 {
		t.Fatalf("expected one elicitation record, got %d", len(history))
	}
	if history[0].Params["title"] != "Confirm" {
		t.Errorf("unexpected elicitation params: %#v", history[0].Params)
	}
}

func TestRequestSamplingCancellationSendsNotification(t *testing.T) {
	server := createTestServer(t)
	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = withProgressToken(ctx, "token-cancel")
	params := map[string]any{"topic": "cancellation"}

	resultCh := make(chan struct {
		data map[string]any
		err  error
	}, 1)

	go func() {
		resp, err := server.RequestSampling(ctx, params)
		resultCh <- struct {
			data map[string]any
			err  error
		}{resp, err}
	}()

	req := waitForOutboundRequest(t, server, &buf)
	if req.Meta == nil || req.Meta["progressToken"] != "token-cancel" {
		t.Fatalf("expected progress token propagated, got meta=%v", req.Meta)
	}

	cancel()

	res := <-resultCh
	if !errors.Is(res.err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", res.err)
	}
	if res.data != nil {
		t.Fatalf("expected no result data on cancellation, got %#v", res.data)
	}

	if history := server.sessionState.SamplingHistory(); len(history) != 0 {
		t.Fatalf("expected no sampling history recorded, got %d entries", len(history))
	}

	note := waitForNotification(t, server, &buf, "notifications/cancelled")
	if note.Params == nil {
		t.Fatal("expected cancellation notification params")
	}
	var payload map[string]any
	if err := json.Unmarshal(note.Params, &payload); err != nil {
		t.Fatalf("failed to decode cancellation params: %v", err)
	}
	if payload["progressToken"] != "token-cancel" {
		t.Fatalf("expected progressToken 'token-cancel', got %v", payload["progressToken"])
	}
}

func TestRouterCallToolApplyTriggersSampling(t *testing.T) {
	server := createTestServer(t)
	stub := newStubStaging(true)
	stub.AddStage("stage-apply", map[string]any{"id": "stage-apply"})
	server.toolRegistry.Register("apply", tools.NewApplyTool(newApplyToolServerAdapter(server, stub)))

	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	params := mustMarshal(map[string]any{
		"name":      "apply",
		"arguments": map[string]any{"id": "stage-apply"},
	})

	req := RequestMessage{
		JSONRPC: JSONRPCVersion,
		ID:      "req-apply",
		Method:  "tools/call",
		Params:  params,
		Meta:    Meta{"progressToken": "tok-apply"},
	}

	respCh := make(chan ResponseMessage, 1)
	go func() {
		respCh <- server.router.DispatchRequest(context.Background(), req)
	}()

	elicitationReq := waitForSpecificOutboundRequest(t, server, &buf, "elicitation/create")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      elicitationReq.ID,
		Result:  map[string]any{"choice": "confirm"},
	})

	samplingReq := waitForSpecificOutboundRequest(t, server, &buf, "sampling/createMessage")
	if samplingReq.Meta == nil || samplingReq.Meta["progressToken"] != "tok-apply" {
		t.Fatalf("expected sampling request to carry progress token, got meta=%v", samplingReq.Meta)
	}

	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      samplingReq.ID,
		Result:  map[string]any{"summary": "approved"},
	})

	resp := <-respCh
	if resp.Error != nil {
		t.Fatalf("expected success response, got error: %v", resp.Error)
	}

	result, ok := resp.Result.(types.CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing or wrong type: %T", result.StructuredContent)
	}
	if structured["mode"] != "single" {
		t.Errorf("expected mode 'single', got %v", structured["mode"])
	}
	sampling, ok := structured["sampling"].(map[string]any)
	if !ok {
		t.Fatalf("expected sampling payload, got %T", structured["sampling"])
	}
	if sampling["summary"] != "approved" {
		t.Errorf("unexpected sampling payload: %v", sampling)
	}

	history := server.sessionState.SamplingHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 sampling history entry, got %d", len(history))
	}
	if history[0].Params["workflow"] != "apply" {
		t.Errorf("expected workflow 'apply', got %v", history[0].Params["workflow"])
	}
	if len(stub.Applied()) != 1 || stub.Applied()[0] != "stage-apply" {
		t.Errorf("expected stage 'stage-apply' applied, got %v", stub.Applied())
	}
}

func TestRouterCallToolApplySamplingCancellation(t *testing.T) {
	server := createTestServer(t)
	stub := newStubStaging(true)
	stub.AddStage("stage-cancel", map[string]any{"id": "stage-cancel"})
	server.toolRegistry.Register("apply", tools.NewApplyTool(newApplyToolServerAdapter(server, stub)))

	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	params := mustMarshal(map[string]any{
		"name":      "apply",
		"arguments": map[string]any{"id": "stage-cancel"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	req := RequestMessage{
		JSONRPC: JSONRPCVersion,
		ID:      "req-cancel",
		Method:  "tools/call",
		Params:  params,
		Meta:    Meta{"progressToken": "tok-cancel"},
	}

	respCh := make(chan ResponseMessage, 1)
	go func() {
		respCh <- server.router.DispatchRequest(ctx, req)
	}()

	elicitationReq := waitForSpecificOutboundRequest(t, server, &buf, "elicitation/create")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      elicitationReq.ID,
		Result:  map[string]any{"choice": "confirm"},
	})

	_ = waitForSpecificOutboundRequest(t, server, &buf, "sampling/createMessage")
	cancel()

	resp := <-respCh
	if resp.Error != nil {
		t.Fatalf("expected RPC success envelope, got error: %v", resp.Error)
	}
	result, ok := resp.Result.(types.CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	if !result.IsError {
		t.Fatal("expected error result after cancellation")
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured error content, got %T", result.StructuredContent)
	}
	if _, ok := structured["code"]; !ok {
		t.Errorf("expected structured error code, got %v", structured)
	}
	switch code := structured["code"].(type) {
	case int:
		if code != -32800 {
			t.Errorf("expected error code -32800, got %d", code)
		}
	case float64:
		if int(code) != -32800 {
			t.Errorf("expected error code -32800, got %v", code)
		}
	default:
		t.Errorf("unexpected code type %T", structured["code"])
	}
	switch code := structured["code"].(type) {
	case int:
		if code != -32800 {
			t.Errorf("expected error code -32800, got %d", code)
		}
	case float64:
		if int(code) != -32800 {
			t.Errorf("expected error code -32800, got %v", code)
		}
	default:
		t.Errorf("unexpected code type %T", structured["code"])
	}

	note := waitForNotification(t, server, &buf, "notifications/cancelled")
	if note.Params == nil {
		t.Fatal("expected cancellation notification params")
	}
	var payload map[string]any
	if err := json.Unmarshal(note.Params, &payload); err != nil {
		t.Fatalf("failed to decode cancellation payload: %v", err)
	}
	if payload["progressToken"] != "tok-cancel" {
		t.Errorf("expected progressToken 'tok-cancel', got %v", payload["progressToken"])
	}

	if history := server.sessionState.SamplingHistory(); len(history) != 0 {
		t.Fatalf("expected no sampling history recorded, got %d", len(history))
	}
}

func TestHandleInitializedRequestsRootsAndStoresClientRoots(t *testing.T) {
	server := createTestServer(t)
	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	req := Request{ID: "init-id"}
	resp := server.handleInitialized(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	rootsReq := waitForSpecificOutboundRequest(t, server, &buf, "roots/list")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      rootsReq.ID,
		Result: map[string]any{
			"roots": []any{map[string]any{"uri": "file:///workspace"}},
		},
	})

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		roots := server.sessionState.ClientRoots()
		if len(roots) == 1 && roots[0] == "file:///workspace" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected client roots to be stored")
}

func TestHandleSubscribeResourceWatchable(t *testing.T) {
	server := createTestServer(t)
	updates := make(chan types.ResourceUpdate, 1)
	resource := &watchableTestResource{
		uri:         "custom://dynamic",
		name:        "Dynamic Resource",
		description: "Test dynamic resource",
		mime:        "application/json",
		body:        "{}",
		updates:     updates,
	}

	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)
	server.RegisterResource(resource)
	// Drain the initial list_changed notification emitted during registration.
	_, _ = readFirstLine(server, &buf)

	params := mustMarshal(map[string]any{"uri": resource.URI()})
	req := Request{
		Method: "resources/subscribe",
		ID:     "sub-1",
		Params: params,
	}

	resp := server.handleSubscribeResource(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected subscribe error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	subscriptionID, _ := result["subscriptionId"].(string)
	if subscriptionID == "" {
		t.Fatal("expected subscriptionId in response")
	}

	updates <- types.ResourceUpdate{URI: resource.URI(), Type: types.ResourceUpdateTypeUpdated}
	note := waitForNotification(t, server, &buf, "notifications/resources/updated")
	if note.Params == nil {
		t.Fatal("expected notification params")
	}
	var payload map[string]any
	if err := json.Unmarshal(note.Params, &payload); err != nil {
		t.Fatalf("failed to decode notification payload: %v", err)
	}
	if payload["uri"] != resource.URI() {
		t.Fatalf("expected uri %s, got %v", resource.URI(), payload["uri"])
	}

	unsubParams := mustMarshal(map[string]any{
		"uri":            resource.URI(),
		"subscriptionId": subscriptionID,
	})
	unsubReq := Request{
		Method: "resources/unsubscribe",
		ID:     "unsub-1",
		Params: unsubParams,
	}
	resp = server.handleUnsubscribeResource(context.Background(), unsubReq)
	if resp.Error != nil {
		t.Fatalf("unexpected unsubscribe error: %v", resp.Error)
	}
	close(updates)
}

func TestRouterCallToolCancellationNotificationStopsApply(t *testing.T) {
	server := createTestServer(t)
	stub := newStubStaging(true)
	stub.AddStage("stage-notify", map[string]any{"id": "stage-notify"})
	barrier := make(chan struct{})
	stub.SetApplyBarrier(barrier)
	server.toolRegistry.Register("apply", tools.NewApplyTool(newApplyToolServerAdapter(server, stub)))

	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	params := mustMarshal(map[string]any{
		"name":      "apply",
		"arguments": map[string]any{"id": "stage-notify"},
	})

	req := RequestMessage{
		JSONRPC: JSONRPCVersion,
		ID:      "req-cancel-note",
		Method:  "tools/call",
		Params:  params,
		Meta:    Meta{"progressToken": "tok-notify"},
	}

	respCh := make(chan ResponseMessage, 1)
	go func() {
		respCh <- server.router.DispatchRequest(context.Background(), req)
	}()

	elicitationReq := waitForSpecificOutboundRequest(t, server, &buf, "elicitation/create")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      elicitationReq.ID,
		Result:  map[string]any{"choice": "confirm"},
	})

	note := NotificationMessage{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/cancelled",
		Params:  mustMarshal(map[string]any{"progressToken": "tok-notify"}),
	}
	if err := server.router.DispatchNotification(context.Background(), note); err != nil {
		t.Fatalf("dispatch notification: %v", err)
	}

	close(barrier)
	resp := <-respCh
	if resp.Error != nil {
		t.Fatalf("expected success envelope, got %v", resp.Error)
	}
	result, ok := resp.Result.(types.CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	if !result.IsError {
		t.Fatal("expected error result after cancellation notification")
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured error content, got %T", result.StructuredContent)
	}
	switch code := structured["code"].(type) {
	case int:
		if code != -32800 {
			t.Fatalf("expected cancellation code -32800, got %v", structured)
		}
	case float64:
		if code != -32800 {
			t.Fatalf("expected cancellation code -32800, got %v", structured)
		}
	default:
		t.Fatalf("expected numeric cancellation code, got %T", structured["code"])
	}
	if message, ok := structured["message"].(string); !ok || message != "Request cancelled" {
		t.Fatalf("expected cancellation message, got %v", structured)
	}
	data, ok := structured["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected structured data payload, got %v", structured["data"])
	}
	if detail, ok := data["detail"].(string); !ok || !strings.Contains(detail, "context canceled") {
		t.Fatalf("expected detail about context cancellation, got %v", detail)
	}
	if applied := stub.Applied(); len(applied) != 0 {
		t.Errorf("expected no stages applied, got %v", applied)
	}
}

func TestRouterCallToolApplyElicitationFallback(t *testing.T) {
	server := createTestServer(t)
	stub := newStubStaging(true)
	stub.AddStage("stage-elic", map[string]any{"id": "stage-elic"})
	server.toolRegistry.Register("apply", tools.NewApplyTool(newApplyToolServerAdapter(server, stub)))

	var buf bytes.Buffer
	server.writer = bufio.NewWriter(&buf)

	params := mustMarshal(map[string]any{
		"name":      "apply",
		"arguments": map[string]any{"id": "stage-elic"},
	})

	req := RequestMessage{
		JSONRPC: JSONRPCVersion,
		ID:      "req-elic",
		Method:  "tools/call",
		Params:  params,
		Meta:    Meta{"progressToken": "tok-elic"},
	}

	respCh := make(chan ResponseMessage, 1)
	go func() {
		respCh <- server.router.DispatchRequest(context.Background(), req)
	}()

	elicitationReq := waitForSpecificOutboundRequest(t, server, &buf, "elicitation/create")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      elicitationReq.ID,
		Error:   &ErrorObject{Code: MethodNotFound, Message: "unsupported"},
	})

	samplingReq := waitForSpecificOutboundRequest(t, server, &buf, "sampling/createMessage")
	server.resolvePendingResponse(ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      samplingReq.ID,
		Result:  map[string]any{"summary": "auto"},
	})

	resp := <-respCh
	if resp.Error != nil {
		t.Fatalf("expected success, got %v", resp.Error)
	}
	result, ok := resp.Result.(types.CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("missing structured content: %T", result.StructuredContent)
	}
	if structured["mode"] != "single" {
		t.Errorf("expected mode 'single', got %v", structured["mode"])
	}
	if sampling, ok := structured["sampling"].(map[string]any); !ok || sampling["summary"] != "auto" {
		t.Errorf("unexpected sampling payload: %v", structured["sampling"])
	}
	if len(stub.Applied()) != 1 || stub.Applied()[0] != "stage-elic" {
		t.Errorf("expected stage applied despite elicitation fallback, got %v", stub.Applied())
	}
}

func waitForSpecificOutboundRequest(t *testing.T, server *StdioServer, buf *bytes.Buffer, target string) RequestMessage {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		req := waitForOutboundRequest(t, server, buf)
		if req.Method == target {
			return req
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for outbound request %s", target)
		}
	}
}

type applyToolServerAdapter struct {
	inner   *StdioServer
	staging any
}

func newApplyToolServerAdapter(inner *StdioServer, staging any) *applyToolServerAdapter {
	return &applyToolServerAdapter{inner: inner, staging: staging}
}

func (a *applyToolServerAdapter) GetProviders() *providers.Registry {
	return a.inner.GetProviders()
}

func (a *applyToolServerAdapter) GetFileProcessor() *core.FileProcessor {
	return a.inner.GetFileProcessor()
}

func (a *applyToolServerAdapter) GetStaging() any {
	return a.staging
}

func (a *applyToolServerAdapter) GetSafety() any {
	return a.inner.GetSafety()
}

func (a *applyToolServerAdapter) GetSessionID() string {
	return "test-session"
}

func (a *applyToolServerAdapter) ReportProgress(ctx context.Context, progress, total float64, message string) {
	a.inner.ReportProgress(ctx, progress, total, message)
}

func (a *applyToolServerAdapter) ConfirmApply(ctx context.Context, summary string) error {
	return a.inner.ConfirmApply(ctx, summary)
}

func (a *applyToolServerAdapter) RequestSampling(ctx context.Context, params map[string]any) (map[string]any, error) {
	return a.inner.RequestSampling(ctx, params)
}

func (a *applyToolServerAdapter) RequestElicitation(ctx context.Context, params map[string]any) (map[string]any, error) {
	return a.inner.RequestElicitation(ctx, params)
}

func (a *applyToolServerAdapter) FinalizeTransform(ctx context.Context, req types.TransformRequest) (map[string]any, error) {
	return a.inner.FinalizeTransform(ctx, req)
}

func newStubStaging(enabled bool) *stubStaging {
	return &stubStaging{
		enabled: enabled,
		stages:  make(map[string]map[string]any),
	}
}

type stubStaging struct {
	mu      sync.Mutex
	enabled bool
	stages  map[string]map[string]any
	order   []string
	applied []string
	barrier chan struct{}
}

func (s *stubStaging) IsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

func (s *stubStaging) AddStage(id string, stage map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.stages[id]; !exists {
		s.order = append(s.order, id)
	}
	copy := make(map[string]any, len(stage))
	for k, v := range stage {
		copy[k] = v
	}
	copy["id"] = id
	s.stages[id] = copy
}

func (s *stubStaging) Applied() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.applied...)
}

func (s *stubStaging) SetApplyBarrier(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.barrier = ch
}

func (s *stubStaging) GetStageMap(id string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stage, ok := s.stages[id]
	if !ok {
		return nil, false
	}
	return cloneStage(stage), true
}

func (s *stubStaging) GetAllStages() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	stages := make([]any, 0, len(s.order))
	for _, id := range s.order {
		if stage, ok := s.stages[id]; ok {
			stages = append(stages, cloneStage(stage))
		}
	}
	return stages
}

func (s *stubStaging) GetLatestStage() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.order) == 0 {
		return nil
	}
	for i := len(s.order) - 1; i >= 0; i-- {
		if stage, ok := s.stages[s.order[i]]; ok {
			return cloneStage(stage)
		}
	}
	return nil
}

func (s *stubStaging) ApplyStage(ctx context.Context, stageID string, auto bool) (*models.Apply, error) {
	s.mu.Lock()
	barrier := s.barrier
	s.mu.Unlock()
	if barrier != nil {
		select {
		case <-barrier:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stages[stageID]; !exists {
		return nil, fmt.Errorf("stage not found: %s", stageID)
	}

	delete(s.stages, stageID)
	s.applied = append(s.applied, stageID)

	return &models.Apply{
		ID:          "apply-" + stageID,
		StageID:     stageID,
		AutoApplied: auto,
		AppliedBy:   "test",
	}, nil
}

// GetStage implements types.StagingStore
func (s *stubStaging) GetStage(stageID string) (*models.Stage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stage, exists := s.stages[stageID]
	if !exists {
		return nil, fmt.Errorf("stage not found: %s", stageID)
	}

	return &models.Stage{
		ID:        stageID,
		Status:    "pending",
		SessionID: "test-session",
		Modified:  fmt.Sprintf("%v", stage["content"]),
	}, nil
}

// ListPendingStages implements types.StagingStore
func (s *stubStaging) ListPendingStages(sessionID string) ([]models.Stage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stages []models.Stage
	for _, id := range s.order {
		if _, exists := s.stages[id]; exists {
			stages = append(stages, models.Stage{
				ID:        id,
				Status:    "pending",
				SessionID: sessionID,
			})
		}
	}
	return stages, nil
}

func cloneStage(stage map[string]any) map[string]any {
	copy := make(map[string]any, len(stage))
	for k, v := range stage {
		copy[k] = v
	}
	return copy
}

type watchableTestResource struct {
	uri         string
	name        string
	description string
	mime        string
	body        string
	updates     chan types.ResourceUpdate
}

func (r *watchableTestResource) Name() string {
	return r.name
}

func (r *watchableTestResource) Description() string {
	return r.description
}

func (r *watchableTestResource) URI() string {
	return r.uri
}

func (r *watchableTestResource) MimeType() string {
	return r.mime
}

func (r *watchableTestResource) Contents() (string, error) {
	return r.body, nil
}

func (r *watchableTestResource) Watch(ctx context.Context) (<-chan types.ResourceUpdate, error) {
	if r.updates == nil {
		return nil, types.ErrResourceWatchUnsupported
	}
	out := make(chan types.ResourceUpdate)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-r.updates:
				if !ok {
					return
				}
				select {
				case out <- update:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func waitForOutboundRequest(t *testing.T, server *StdioServer, buf *bytes.Buffer) RequestMessage {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if line, ok := readFirstLine(server, buf); ok {
			var msg RequestMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				t.Fatalf("failed to unmarshal outbound request: %v", err)
			}
			return msg
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for outbound request")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readFirstLine(server *StdioServer, buf *bytes.Buffer) (string, bool) {
	server.writeMu.Lock()
	defer server.writeMu.Unlock()

	data := buf.String()
	idx := strings.IndexRune(data, '\n')
	if idx == -1 {
		return "", false
	}
	line := data[:idx]
	remaining := data[idx+1:]
	buf.Reset()
	buf.WriteString(remaining)
	return line, true
}

func waitForNotification(t *testing.T, server *StdioServer, buf *bytes.Buffer, method string) NotificationMessage {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if line, ok := readFirstLine(server, buf); ok {
			var note NotificationMessage
			if err := json.Unmarshal([]byte(line), &note); err != nil {
				t.Fatalf("failed to unmarshal notification: %v", err)
			}
			if note.Method == method {
				return note
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for notification %s", method)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestServerStart_ValidRequest tests server handling of valid request
func TestServerStart_ValidRequest(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Valid JSON request
	validRequest := `{"id": "1", "method": "ping", "params": {}}`
	reader := strings.NewReader(validRequest)
	server.reader = bufio.NewReader(reader)
	server.writer = bufio.NewWriter(&buf)

	// Start should process the request and then EOF
	err = server.Start()
	if err != nil {
		t.Errorf("Start should handle valid request: %v", err)
	}

	// Should have sent response
	output := buf.String()
	if output == "" {
		t.Error("Expected response output")
	}

	// Should be valid JSON response
	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}
}

// TestServerStart_Notification tests server handling of notifications (no ID)
func TestServerStart_Notification(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = false

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Notification (no ID) - should not send response
	notification := `{"method": "initialized", "params": {}}`
	reader := strings.NewReader(notification)
	server.reader = bufio.NewReader(reader)
	server.writer = bufio.NewWriter(&buf)

	// Start should process the notification without sending response
	err = server.Start()
	if err != nil {
		t.Errorf("Start should handle notification: %v", err)
	}

	// Should not have sent response for notification
	output := buf.String()
	if output != "" {
		t.Error("Should not send response for notification")
	}
}

// TestProviderRegistryAdapter_NotFound tests adapter behavior when provider not found
func TestProviderRegistryAdapter_NotFound(t *testing.T) {
	config := newServerConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	adapter := &providerRegistryAdapter{server.providers}
	provider, exists := adapter.Get("nonexistent")

	if exists {
		t.Error("Should return false for nonexistent provider")
	}

	if provider != nil {
		t.Error("Should return nil provider for nonexistent language")
	}
}

// TestServerRequestLogging tests request logging with long requests
func TestServerRequestLogging(t *testing.T) {
	var buf bytes.Buffer
	config := newServerConfig()
	config.DatabaseURL = "skip"
	config.Debug = true // Enable debug to see log output

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a very long request that should be truncated in logs
	longParams := strings.Repeat("x", 300) // Longer than 200 char limit
	longRequest := fmt.Sprintf(`{"id": "1", "method": "ping", "params": {"data": "%s"}}`, longParams)

	reader := strings.NewReader(longRequest)
	server.reader = bufio.NewReader(reader)
	server.writer = bufio.NewWriter(&buf)

	// Should handle long request without issues
	err = server.Start()
	if err != nil {
		t.Errorf("Start should handle long request: %v", err)
	}
}
