package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/termfx/morfx/core"
)

// TestNewStdioServer tests server creation
func TestNewStdioServer(t *testing.T) {
	config := DefaultConfig()
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

	if server.tools == nil {
		t.Error("Tools registry should be initialized")
	}

	if server.fileProcessor == nil {
		t.Error("File processor should be initialized")
	}
}

// TestNewStdioServerWithDatabase tests server creation with database
func TestNewStdioServerWithDatabase(t *testing.T) {
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Register a test tool
	testTool := func(params json.RawMessage) (any, error) {
		return "test result", nil
	}

	server.RegisterTool("test_tool", testTool)

	// Check if tool was registered
	server.mu.RLock()
	_, exists := server.tools["test_tool"]
	server.mu.RUnlock()

	if !exists {
		t.Error("Tool should be registered")
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
	config := DefaultConfig()
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
	config := DefaultConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check that some built-in tools are registered
	server.mu.RLock()
	toolCount := len(server.tools)
	server.mu.RUnlock()

	if toolCount == 0 {
		t.Error("Expected built-in tools to be registered")
	}

	t.Logf("Found %d registered tools", toolCount)
}

// TestProviderRegistryAdapter tests the provider registry adapter
func TestProviderRegistryAdapter(t *testing.T) {
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
	config.DatabaseURL = "skip"

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check all critical components are initialized
	components := map[string]any{
		"providers":     server.providers,
		"fileProcessor": server.fileProcessor,
		"tools":         server.tools,
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
		req := Request{
			ID:     &[]string{"test-" + tc.method}[0],
			Method: tc.method,
			Params: json.RawMessage(`{}`),
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	if !strings.Contains(output, "Parse error") && !strings.Contains(output, "JSON") {
		t.Error("Expected parse error in output")
	}
}

// TestServerStart_ValidRequest tests server handling of valid request
func TestServerStart_ValidRequest(t *testing.T) {
	var buf bytes.Buffer
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
	config := DefaultConfig()
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
