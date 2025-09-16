package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetPromptDefinitions(t *testing.T) {
	definitions := GetPromptDefinitions()

	if len(definitions) == 0 {
		t.Error("Expected at least one prompt definition")
	}

	// Check first definition structure
	if len(definitions) > 0 {
		def := definitions[0]
		if def.Name == "" {
			t.Error("Prompt definition should have a name")
		}
		if def.Description == "" {
			t.Error("Prompt definition should have a description")
		}
	}

	// Check for expected prompts
	expectedPrompts := []string{
		"code-analysis",
		"transformation-guide",
		"confidence-explanation",
		"query-builder",
		"best-practices",
	}

	for _, expected := range expectedPrompts {
		found := false
		for _, def := range definitions {
			if def.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected prompt '%s' not found", expected)
		}
	}
}

func TestHandleListPromptsDetailed(t *testing.T) {
	server := &StdioServer{}

	req := Request{
		ID:     &[]string{"test-123"}[0],
		Method: "prompts/list",
		Params: json.RawMessage(`{}`),
	}

	response := server.handleListPrompts(req)

	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be map, got %T", response.Result)
	}

	prompts, ok := result["prompts"].([]PromptDefinition)
	if !ok {
		t.Fatalf("Expected prompts to be []PromptDefinition, got %T", result["prompts"])
	}

	if len(prompts) == 0 {
		t.Error("Expected at least one prompt")
	}
}

func TestHandleGetPrompt_CodeAnalysis(t *testing.T) {
	server := &StdioServer{
		debugLog: func(format string, args ...any) {}, // No-op for tests
	}

	params := map[string]any{
		"name": "code-analysis",
		"arguments": map[string]string{
			"language": "go",
			"code":     "func test() {}",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	req := Request{
		ID:     &[]string{"test-123"}[0],
		Method: "prompts/get",
		Params: paramsJSON,
	}

	response := server.handleGetPrompt(req)

	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}

	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be map, got %T", response.Result)
	}

	messages, ok := result["messages"].([]PromptMessage)
	if !ok {
		t.Fatalf("Expected messages to be []PromptMessage, got %T", result["messages"])
	}

	if len(messages) == 0 {
		t.Error("Expected at least one message")
	}
}

func TestHandleGetPrompt_InvalidParams(t *testing.T) {
	server := &StdioServer{
		debugLog: func(format string, args ...any) {},
	}

	invalidJSON := json.RawMessage(`{"invalid": json}`)

	req := Request{
		ID:     &[]string{"test-456"}[0],
		Method: "prompts/get",
		Params: invalidJSON,
	}

	response := server.handleGetPrompt(req)

	if response.Error == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestHandleGetPrompt_UnknownPrompt(t *testing.T) {
	server := &StdioServer{
		debugLog: func(format string, args ...any) {},
	}

	params := map[string]any{
		"name": "unknown-prompt",
	}
	paramsJSON, _ := json.Marshal(params)

	req := Request{
		ID:     &[]string{"test-789"}[0],
		Method: "prompts/get",
		Params: paramsJSON,
	}

	response := server.handleGetPrompt(req)

	if response.Error == nil {
		t.Error("Expected error for unknown prompt")
	}
}

// Test generatePromptContent function
func TestGeneratePromptContent_AllPrompts(t *testing.T) {
	server := &StdioServer{
		debugLog: func(format string, args ...any) {},
	}

	testCases := []struct {
		name      string
		args      map[string]string
		shouldErr bool
	}{
		{
			name: "code-analysis",
			args: map[string]string{
				"language": "go",
				"code":     "func test() {}",
				"focus":    "refactoring",
			},
			shouldErr: false,
		},
		{
			name: "transformation-guide",
			args: map[string]string{
				"operation": "replace",
				"target":    "function",
				"language":  "go",
			},
			shouldErr: false,
		},
		{
			name: "confidence-explanation",
			args: map[string]string{
				"score":   "0.85",
				"factors": "syntax match, semantic analysis",
			},
			shouldErr: false,
		},
		{
			name: "query-builder",
			args: map[string]string{
				"description": "find all functions",
				"language":    "go",
			},
			shouldErr: false,
		},
		{
			name: "best-practices",
			args: map[string]string{
				"language":  "go",
				"operation": "refactor",
			},
			shouldErr: false,
		},
		{
			name:      "nonexistent",
			args:      map[string]string{},
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messages, err := server.generatePromptContent(tc.name, tc.args)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error for prompt '%s'", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for prompt '%s': %v", tc.name, err)
				}

				if len(messages) == 0 {
					t.Errorf("Expected messages for prompt '%s'", tc.name)
				}

				// Check message structure
				for _, message := range messages {
					if message.Role == "" {
						t.Errorf("Message should have a role for prompt '%s'", tc.name)
					}
					if len(message.Content) == 0 {
						t.Errorf("Message should have content for prompt '%s'", tc.name)
					}
				}
			}
		})
	}
}

// Test generateCodeAnalysisPrompt specifically
func TestGenerateCodeAnalysisPrompt(t *testing.T) {
	server := &StdioServer{}

	// Test with valid arguments
	args := map[string]string{
		"language": "go",
		"code":     "func test() { return 42 }",
		"focus":    "performance",
	}

	messages, err := server.generateCodeAnalysisPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	message := messages[0]
	if message.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", message.Role)
	}

	if len(message.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(message.Content))
	}

	content := message.Content[0]
	if content.Type != "text" {
		t.Errorf("Expected content type 'text', got '%s'", content.Type)
	}

	// Check that content includes the provided arguments
	text := content.Text
	if !strings.Contains(text, "go") {
		t.Error("Content should contain language")
	}
	if !strings.Contains(text, "func test()") {
		t.Error("Content should contain code")
	}
	if !strings.Contains(text, "performance") {
		t.Error("Content should contain focus")
	}
}

func TestGenerateCodeAnalysisPrompt_MissingArgs(t *testing.T) {
	server := &StdioServer{}

	testCases := []struct {
		name string
		args map[string]string
	}{
		{"missing language", map[string]string{"code": "test"}},
		{"missing code", map[string]string{"language": "go"}},
		{"missing both", map[string]string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.generateCodeAnalysisPrompt(tc.args)

			if err == nil {
				t.Error("Expected error for missing required arguments")
			}

			mcpErr, ok := err.(*MCPError)
			if !ok {
				t.Errorf("Expected MCPError, got %T", err)
			} else if mcpErr.Code != InvalidParams {
				t.Errorf("Expected InvalidParams error, got %d", mcpErr.Code)
			}
		})
	}
}

func TestGenerateTransformationGuidePrompt(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"operation": "replace",
		"target":    "function",
		"language":  "go",
	}

	messages, err := server.generateTransformationGuidePrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	text := messages[0].Content[0].Text
	if !strings.Contains(text, "replace") {
		t.Error("Content should contain operation")
	}
	if !strings.Contains(text, "function") {
		t.Error("Content should contain target")
	}
	if !strings.Contains(text, "go") {
		t.Error("Content should contain language")
	}
}

func TestGenerateTransformationGuidePrompt_MissingArgs(t *testing.T) {
	server := &StdioServer{}

	testCases := []struct {
		name string
		args map[string]string
	}{
		{"missing operation", map[string]string{"target": "function"}},
		{"missing target", map[string]string{"operation": "replace"}},
		{"missing both", map[string]string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.generateTransformationGuidePrompt(tc.args)

			if err == nil {
				t.Error("Expected error for missing required arguments")
			}
		})
	}
}

func TestGenerateConfidenceExplanationPrompt(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"score":   "0.85",
		"factors": "syntax match, semantic analysis, code context",
	}

	messages, err := server.generateConfidenceExplanationPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	text := messages[0].Content[0].Text
	if !strings.Contains(text, "0.85") {
		t.Error("Content should contain score")
	}
	if !strings.Contains(text, "syntax match") {
		t.Error("Content should contain factors")
	}
}

func TestGenerateConfidenceExplanationPrompt_MissingScore(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"factors": "some factors",
	}

	_, err := server.generateConfidenceExplanationPrompt(args)

	if err == nil {
		t.Error("Expected error for missing score")
	}
}

func TestGenerateConfidenceExplanationPrompt_NoFactors(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"score": "0.75",
	}

	messages, err := server.generateConfidenceExplanationPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should work without factors
	text := messages[0].Content[0].Text
	if !strings.Contains(text, "0.75") {
		t.Error("Content should contain score")
	}
}

func TestGenerateQueryBuilderPrompt(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"description": "find all public functions",
		"language":    "go",
	}

	messages, err := server.generateQueryBuilderPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	text := messages[0].Content[0].Text
	if !strings.Contains(text, "find all public functions") {
		t.Error("Content should contain description")
	}
	if !strings.Contains(text, "go") {
		t.Error("Content should contain language")
	}
	if !strings.Contains(text, "JSON") {
		t.Error("Content should mention JSON format")
	}
}

func TestGenerateQueryBuilderPrompt_MissingDescription(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"language": "go",
	}

	_, err := server.generateQueryBuilderPrompt(args)

	if err == nil {
		t.Error("Expected error for missing description")
	}
}

func TestGenerateQueryBuilderPrompt_NoLanguage(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"description": "find functions",
	}

	messages, err := server.generateQueryBuilderPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should work without language
	text := messages[0].Content[0].Text
	if !strings.Contains(text, "find functions") {
		t.Error("Content should contain description")
	}
}

func TestGenerateBestPracticesPrompt(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"language":  "go",
		"operation": "refactor",
	}

	messages, err := server.generateBestPracticesPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	text := messages[0].Content[0].Text
	if !strings.Contains(text, "go") {
		t.Error("Content should contain language")
	}
	if !strings.Contains(text, "refactor") {
		t.Error("Content should contain operation")
	}
	if !strings.Contains(text, "best practices") {
		t.Error("Content should mention best practices")
	}
}

func TestGenerateBestPracticesPrompt_MissingLanguage(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"operation": "refactor",
	}

	_, err := server.generateBestPracticesPrompt(args)

	if err == nil {
		t.Error("Expected error for missing language")
	}
}

func TestGenerateBestPracticesPrompt_NoOperation(t *testing.T) {
	server := &StdioServer{}

	args := map[string]string{
		"language": "go",
	}

	messages, err := server.generateBestPracticesPrompt(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should work without operation
	text := messages[0].Content[0].Text
	if !strings.Contains(text, "go") {
		t.Error("Content should contain language")
	}
}

// Test prompt definition structure
func TestPromptDefinitionStructure(t *testing.T) {
	definitions := GetPromptDefinitions()

	for _, def := range definitions {
		if def.Name == "" {
			t.Error("Prompt definition should have a name")
		}

		if def.Description == "" {
			t.Error("Prompt definition should have a description")
		}

		// Check arguments structure
		for _, arg := range def.Arguments {
			if arg.Name == "" {
				t.Error("Argument should have a name")
			}
		}
	}
}

// Test prompt message serialization
func TestPromptMessageSerialization(t *testing.T) {
	message := PromptMessage{
		Role: "user",
		Content: []PromptContent{
			{
				Type: "text",
				Text: "Test message",
			},
		},
	}

	// Should be able to marshal to JSON
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal PromptMessage: %v", err)
	}

	// Should be able to unmarshal back
	var unmarshaled PromptMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal PromptMessage: %v", err)
	}

	if unmarshaled.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", unmarshaled.Role)
	}

	if len(unmarshaled.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(unmarshaled.Content))
	}

	if unmarshaled.Content[0].Text != "Test message" {
		t.Errorf("Expected text 'Test message', got '%s'", unmarshaled.Content[0].Text)
	}
}
