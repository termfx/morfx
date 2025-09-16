package mcp

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestGetResourceDefinitions(t *testing.T) {
	resources := GetResourceDefinitions()

	if len(resources) != 5 {
		t.Errorf("Expected 5 resources, got %d", len(resources))
	}

	expectedURIs := []string{
		"morfx://server/info",
		"morfx://server/capabilities",
		"morfx://providers/languages",
		"morfx://session/current",
		"morfx://config/settings",
	}

	for i, resource := range resources {
		if resource.URI != expectedURIs[i] {
			t.Errorf("Expected URI %s, got %s", expectedURIs[i], resource.URI)
		}

		if resource.Name == "" {
			t.Errorf("Resource %s has empty name", resource.URI)
		}

		if resource.MimeType != "application/json" {
			t.Errorf("Expected JSON mime type for %s, got %s", resource.URI, resource.MimeType)
		}

		if resource.Annotations == nil {
			t.Errorf("Resource %s has nil annotations", resource.URI)
		}

		if resource.Annotations["readonly"] != true {
			t.Errorf("Resource %s should be readonly", resource.URI)
		}
	}
}

func TestHandleReadResource_ServerInfo(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://server/info"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	contents, ok := result["contents"].([]ResourceContent)
	if !ok {
		t.Fatal("Contents is not a ResourceContent slice")
	}

	if len(contents) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(contents))
	}

	content := contents[0]
	if content.URI != "morfx://server/info" {
		t.Errorf("Expected URI morfx://server/info, got %s", content.URI)
	}

	if content.MimeType != "application/json" {
		t.Errorf("Expected JSON mime type, got %s", content.MimeType)
	}

	// Verify JSON content is valid
	var info map[string]any
	if err := json.Unmarshal([]byte(content.Text), &info); err != nil {
		t.Fatalf("Invalid JSON content: %v", err)
	}

	if info["name"] != "Morfx MCP Server" {
		t.Errorf("Expected server name 'Morfx MCP Server', got %v", info["name"])
	}

	if info["version"] != "1.3.0" {
		t.Errorf("Expected version '1.3.0', got %v", info["version"])
	}
}

func TestHandleReadResource_ServerCapabilities(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://server/capabilities"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	contents, ok := result["contents"].([]ResourceContent)
	if !ok {
		t.Fatal("Contents is not a ResourceContent slice")
	}

	content := contents[0]

	// Verify JSON content structure
	var capabilities map[string]any
	if err := json.Unmarshal([]byte(content.Text), &capabilities); err != nil {
		t.Fatalf("Invalid JSON content: %v", err)
	}

	if capabilities["protocol_version"] != "2024-11-05" {
		t.Errorf("Expected protocol version '2024-11-05', got %v", capabilities["protocol_version"])
	}

	tools, ok := capabilities["tools"].(map[string]any)
	if !ok {
		t.Fatal("Tools section is missing or invalid")
	}

	if tools["count"] == nil {
		t.Error("Tools count is missing")
	}
}

func TestHandleReadResource_SupportedLanguages(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://providers/languages"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	contents, ok := result["contents"].([]ResourceContent)
	if !ok {
		t.Fatal("Contents is not a ResourceContent slice")
	}

	content := contents[0]

	// Verify JSON content structure
	var languages map[string]any
	if err := json.Unmarshal([]byte(content.Text), &languages); err != nil {
		t.Fatalf("Invalid JSON content: %v", err)
	}

	supported, ok := languages["supported"].([]any)
	if !ok {
		t.Fatal("Supported languages section is missing or invalid")
	}

	if len(supported) == 0 {
		t.Error("Expected at least one supported language")
	}

	// Check Go language support
	goLang := supported[0].(map[string]any)
	if goLang["name"] != "go" {
		t.Errorf("Expected Go language support, got %v", goLang["name"])
	}
}

func TestHandleReadResource_CurrentSession(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://session/current"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	contents, ok := result["contents"].([]ResourceContent)
	if !ok {
		t.Fatal("Contents is not a ResourceContent slice")
	}

	content := contents[0]

	// Verify JSON content structure
	var session map[string]any
	if err := json.Unmarshal([]byte(content.Text), &session); err != nil {
		t.Fatalf("Invalid JSON content: %v", err)
	}

	if session["status"] != "active" {
		t.Errorf("Expected session status 'active', got %v", session["status"])
	}

	if session["mode"] != "stateless" {
		t.Errorf("Expected mode 'stateless', got %v", session["mode"])
	}
}

func TestHandleReadResource_ConfigSettings(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://config/settings"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	contents, ok := result["contents"].([]ResourceContent)
	if !ok {
		t.Fatal("Contents is not a ResourceContent slice")
	}

	content := contents[0]

	// Verify JSON content structure
	var config map[string]any
	if err := json.Unmarshal([]byte(content.Text), &config); err != nil {
		t.Fatalf("Invalid JSON content: %v", config)
	}

	if config["debug"] == nil {
		t.Error("Debug setting is missing")
	}

	if config["database_url"] == nil {
		t.Error("Database URL setting is missing")
	}
}

func TestHandleReadResource_InvalidURI(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://invalid/resource"})
	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: params,
	}

	response := server.handleReadResource(request)

	if response.Error == nil {
		t.Fatal("Expected error for invalid URI")
	}

	if response.Error.Code != MethodNotFound {
		t.Errorf("Expected MethodNotFound error code, got %d", response.Error.Code)
	}
}

func TestHandleReadResource_InvalidParams(t *testing.T) {
	server := createTestServer(t)

	request := Request{
		Method: "resources/read",
		ID:     1,
		Params: json.RawMessage(`{"invalid": "params"}`),
	}

	response := server.handleReadResource(request)

	if response.Error == nil {
		t.Fatal("Expected error for invalid params")
	}

	// The error code should be -32601 (MethodNotFound) or -32602 (InvalidParams)
	// Accept either as both are valid for this scenario
	if response.Error.Code != InvalidParams && response.Error.Code != MethodNotFound {
		t.Errorf("Expected InvalidParams (-32602) or MethodNotFound (-32601) error code, got %d", response.Error.Code)
	}
}

func TestHandleSubscribeResource(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://server/info"})
	request := Request{
		Method: "resources/subscribe",
		ID:     1,
		Params: params,
	}

	response := server.handleSubscribeResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Should return empty result for acknowledgment
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestHandleUnsubscribeResource(t *testing.T) {
	server := createTestServer(t)

	params, _ := json.Marshal(map[string]string{"uri": "morfx://server/info"})
	request := Request{
		Method: "resources/unsubscribe",
		ID:     1,
		Params: params,
	}

	response := server.handleUnsubscribeResource(request)

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Should return empty result for acknowledgment
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestGenerateResourceContent_AllResources(t *testing.T) {
	server := createTestServer(t)

	resourceURIs := []string{
		"morfx://server/info",
		"morfx://server/capabilities",
		"morfx://providers/languages",
		"morfx://session/current",
		"morfx://config/settings",
	}

	for _, uri := range resourceURIs {
		t.Run(uri, func(t *testing.T) {
			content, err := server.generateResourceContent(uri)
			if err != nil {
				t.Fatalf("Failed to generate content for %s: %v", uri, err)
			}

			if content == nil {
				t.Fatalf("Generated content is nil for %s", uri)
			}

			if content.URI != uri {
				t.Errorf("Expected URI %s, got %s", uri, content.URI)
			}

			if content.MimeType != "application/json" {
				t.Errorf("Expected JSON mime type for %s, got %s", uri, content.MimeType)
			}

			if content.Text == "" {
				t.Errorf("Generated content text is empty for %s", uri)
			}

			// Verify content is valid JSON
			var data map[string]any
			if err := json.Unmarshal([]byte(content.Text), &data); err != nil {
				t.Errorf("Generated content is not valid JSON for %s: %v", uri, err)
			}
		})
	}
}

func TestResourceContent_JSONStructure(t *testing.T) {
	server := createTestServer(t)

	// Test server info JSON structure
	content, err := server.generateServerInfo()
	if err != nil {
		t.Fatalf("Failed to generate server info: %v", err)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(content.Text), &info); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"name", "version", "runtime", "uptime", "database", "features"}
	for _, field := range requiredFields {
		if info[field] == nil {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Check runtime info
	runtime, ok := info["runtime"].(map[string]any)
	if !ok {
		t.Fatal("Runtime info is not a map")
	}

	if runtime["go_version"] == nil {
		t.Error("Missing Go version in runtime info")
	}

	if runtime["platform"] == nil {
		t.Error("Missing platform in runtime info")
	}

	// Check features
	features, ok := info["features"].(map[string]any)
	if !ok {
		t.Fatal("Features info is not a map")
	}

	if features["file_ops"] != true {
		t.Error("File operations feature should be enabled")
	}

	if features["in_memory"] != true {
		t.Error("In-memory feature should be enabled")
	}
}

func TestResourceDefinition_Annotations(t *testing.T) {
	resources := GetResourceDefinitions()

	for _, resource := range resources {
		if resource.Annotations == nil {
			t.Errorf("Resource %s has nil annotations", resource.URI)
		}

		category, ok := resource.Annotations["category"].(string)
		if !ok {
			t.Errorf("Resource %s missing category annotation", resource.URI)
		}

		readonly, ok := resource.Annotations["readonly"].(bool)
		if !ok {
			t.Errorf("Resource %s missing readonly annotation", resource.URI)
		}

		if !readonly {
			t.Errorf("Resource %s should be readonly", resource.URI)
		}

		// Verify category makes sense
		validCategories := []string{"system", "providers", "session", "config"}
		found := slices.Contains(validCategories, category)

		if !found {
			t.Errorf("Resource %s has invalid category: %s", resource.URI, category)
		}

		// Verify URI format
		if !strings.HasPrefix(resource.URI, "morfx://") {
			t.Errorf("Resource %s has invalid URI format", resource.URI)
		}
	}
}
