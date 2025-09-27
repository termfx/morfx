package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func setupTestServerForLogging(t *testing.T) *StdioServer {
	var buf bytes.Buffer
	config := DefaultConfig()
	config.Debug = true

	server := &StdioServer{
		config:       config,
		writer:       bufio.NewWriter(&buf),
		sessionState: NewSessionState(),
	}

	return server
}

func TestHandleSetLoggingLevel_ValidLevel(t *testing.T) {
	server := setupTestServerForLogging(t)

	// Test valid logging level
	params := map[string]any{
		"level": LogLevelInfo,
	}
	paramsJSON, _ := json.Marshal(params)

	req := Request{
		ID:     toPointer("test-123"),
		Method: "logging/setLevel",
		Params: paramsJSON,
	}

	response := server.handleSetLoggingLevel(context.Background(), req)

	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}

	if response.ID == nil {
		t.Error("Expected response ID not to be nil")
	} else if responseID, ok := response.ID.(*string); !ok || *responseID != "test-123" {
		t.Errorf("Expected response ID 'test-123', got %v", response.ID)
	}

	// Check that result is empty object
	resultMap, ok := response.Result.(map[string]any)
	if !ok {
		t.Errorf("Expected result to be map, got %T", response.Result)
	}

	if len(resultMap) != 0 {
		t.Errorf("Expected empty result map, got %v", resultMap)
	}
}

func TestHandleSetLoggingLevel_InvalidParams(t *testing.T) {
	server := setupTestServerForLogging(t)

	// Test with invalid JSON
	invalidJSON := json.RawMessage(`{"invalid": json}`)

	req := Request{
		ID:     toPointer("test-456"),
		Method: "logging/setLevel",
		Params: invalidJSON,
	}

	response := server.handleSetLoggingLevel(context.Background(), req)

	if response.Error == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestSendLogNotification_DebugEnabled(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)
	server.config.Debug = true
	server.sessionState.SetLoggingLevel(LogLevelDebug)
	server.sessionState.SetLoggingLevel(LogLevelDebug)

	// Test sending debug notification
	data := LogData{"key": "value", "number": 42}
	server.sendLogNotification(LogLevelDebug, "Test debug message", data)
	server.writer.Flush()

	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Error("Expected output for debug notification when debug enabled")
	}

	// Parse the JSON output
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	// Check notification structure
	if notification["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %v", notification["jsonrpc"])
	}

	if notification["method"] != "notifications/message" {
		t.Errorf("Expected method 'notifications/message', got %v", notification["method"])
	}

	params, ok := notification["params"].(map[string]any)
	if !ok {
		t.Fatalf("Expected params to be map, got %T", notification["params"])
	}

	if params["level"] != string(LogLevelDebug) {
		t.Errorf("Expected level 'debug', got %v", params["level"])
	}

	if params["logger"] != "morfx" {
		t.Errorf("Expected logger 'morfx', got %v", params["logger"])
	}

	// Check that data was merged with message and timestamp
	dataMap, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("Expected data to be map, got %T", params["data"])
	}

	if dataMap["message"] != "Test debug message" {
		t.Errorf("Expected message in data, got %v", dataMap["message"])
	}

	if dataMap["key"] != "value" {
		t.Errorf("Expected custom data to be preserved, got %v", dataMap["key"])
	}

	if dataMap["number"].(float64) != 42 {
		t.Errorf("Expected number 42, got %v", dataMap["number"])
	}

	if _, exists := dataMap["timestamp"]; !exists {
		t.Error("Expected timestamp in data")
	}
}

func TestSendLogNotification_DebugDisabled(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)
	server.config.Debug = false // Disable debug

	// Test sending debug notification when debug disabled
	server.sendLogNotification(LogLevelDebug, "Test debug message", nil)
	server.writer.Flush()

	output := strings.TrimSpace(buf.String())
	if output != "" {
		t.Error("Expected no output for debug notification when debug disabled")
	}
}

func TestSendLogNotification_InfoLevel(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)
	server.config.Debug = false // Info should still work with debug disabled

	// Test sending info notification
	server.sendLogNotification(LogLevelInfo, "Test info message", nil)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output for info notification even when debug disabled")
	}

	// Parse and verify
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params, ok := notification["params"].(map[string]any)
	if !ok {
		t.Fatalf("Expected params to be map, got %T", notification["params"])
	}

	if params["level"] != string(LogLevelInfo) {
		t.Errorf("Expected level 'info', got %v", params["level"])
	}
}

func TestSendLogNotification_NilData(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	// Test with nil data
	server.sendLogNotification(LogLevelInfo, "Test message", nil)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output even with nil data")
	}

	// Parse and verify data was created
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	dataMap := params["data"].(map[string]any)

	if dataMap["message"] != "Test message" {
		t.Errorf("Expected message to be set, got %v", dataMap["message"])
	}

	if _, exists := dataMap["timestamp"]; !exists {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogInfo(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	// Test LogInfo with data
	data := LogData{"operation": "test", "count": 5}
	server.LogInfo("Info message", data)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from LogInfo")
	}

	// Verify it's an info level notification
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	if params["level"] != string(LogLevelInfo) {
		t.Errorf("Expected info level, got %v", params["level"])
	}

	dataMap := params["data"].(map[string]any)
	if dataMap["operation"] != "test" {
		t.Errorf("Expected custom data to be preserved, got %v", dataMap["operation"])
	}
}

func TestLogInfo_NoData(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	// Test LogInfo without data
	server.LogInfo("Info message")
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from LogInfo")
	}

	// Should still work without crash
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	if params["level"] != string(LogLevelInfo) {
		t.Errorf("Expected info level, got %v", params["level"])
	}
}

func TestLogWarning(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	data := LogData{"warning_type": "deprecated"}
	server.LogWarning("Warning message", data)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from LogWarning")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	if params["level"] != string(LogLevelWarning) {
		t.Errorf("Expected warning level, got %v", params["level"])
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	data := LogData{"error_code": "E001", "severity": "high"}
	server.LogError("Error message", data)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from LogError")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	if params["level"] != string(LogLevelError) {
		t.Errorf("Expected error level, got %v", params["level"])
	}

	dataMap := params["data"].(map[string]any)
	if dataMap["error_code"] != "E001" {
		t.Errorf("Expected error_code to be preserved, got %v", dataMap["error_code"])
	}
}

func TestLogDebug(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)
	server.config.Debug = true
	server.sessionState.SetLoggingLevel(LogLevelDebug)

	data := LogData{"debug_info": "detailed"}
	server.LogDebug("Debug message", data)
	server.writer.Flush()

	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Error("Expected output from LogDebug when debug enabled")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	if params["level"] != string(LogLevelDebug) {
		t.Errorf("Expected debug level, got %v", params["level"])
	}
}

func TestLogDebug_DebugDisabled(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)
	server.config.Debug = false

	server.LogDebug("Debug message")
	server.writer.Flush()

	output := buf.String()
	if output != "" {
		t.Error("Expected no output from LogDebug when debug disabled")
	}
}

func TestSendResourceUpdatedNotification(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	uri := "file:///test/resource.txt"
	server.sendResourceUpdatedNotification(uri)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from resource updated notification")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	if notification["method"] != "notifications/resources/updated" {
		t.Errorf("Expected method 'notifications/resources/updated', got %v", notification["method"])
	}

	params := notification["params"].(map[string]any)
	if params["uri"] != uri {
		t.Errorf("Expected uri '%s', got %v", uri, params["uri"])
	}
}

func TestSendResourceListChangedNotification(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	server.sendResourceListChangedNotification()
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from resource list changed notification")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	if notification["method"] != "notifications/resources/list_changed" {
		t.Errorf("Expected method 'notifications/resources/list_changed', got %v", notification["method"])
	}

	params := notification["params"].(map[string]any)
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}
}

func TestSendToolListChangedNotification(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	server.sendToolListChangedNotification()
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from tool list changed notification")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	if notification["method"] != "notifications/tools/list_changed" {
		t.Errorf("Expected method 'notifications/tools/list_changed', got %v", notification["method"])
	}

	params := notification["params"].(map[string]any)
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}
}

func TestSendPromptListChangedNotification(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	server.sendPromptListChangedNotification()
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from prompt list changed notification")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	if notification["method"] != "notifications/prompts/list_changed" {
		t.Errorf("Expected method 'notifications/prompts/list_changed', got %v", notification["method"])
	}

	params := notification["params"].(map[string]any)
	if len(params) != 0 {
		t.Errorf("Expected empty params, got %v", params)
	}
}

func TestSendProgressNotification(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	progressToken := "test-progress-123"
	progress := 50.0
	total := 100.0

	message := "Processing"
	server.sendProgressNotification(progressToken, progress, total, message)
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from progress notification")
	}

	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	if notification["method"] != "notifications/progress" {
		t.Errorf("Expected method 'notifications/progress', got %v", notification["method"])
	}

	params := notification["params"].(map[string]any)
	if params["progressToken"] != progressToken {
		t.Errorf("Expected progressToken '%s', got %v", progressToken, params["progressToken"])
	}

	if params["progress"] != progress {
		t.Errorf("Expected progress %f, got %v", progress, params["progress"])
	}

	if params["total"] != total {
		t.Errorf("Expected total %f, got %v", total, params["total"])
	}

	if params["message"] != message {
		t.Errorf("Expected message '%s', got %v", message, params["message"])
	}
}

func TestLogLevels_Constants(t *testing.T) {
	// Test that all log levels are defined correctly
	expectedLevels := []LogLevel{
		LogLevelDebug,
		LogLevelInfo,
		LogLevelNotice,
		LogLevelWarning,
		LogLevelError,
		LogLevelCritical,
		LogLevelAlert,
		LogLevelEmergency,
	}

	expectedValues := []string{
		"debug",
		"info",
		"notice",
		"warning",
		"error",
		"critical",
		"alert",
		"emergency",
	}

	if len(expectedLevels) != len(expectedValues) {
		t.Error("Mismatch between levels and values arrays")
	}

	for i, level := range expectedLevels {
		if string(level) != expectedValues[i] {
			t.Errorf("Expected level %d to be '%s', got '%s'", i, expectedValues[i], string(level))
		}
	}
}

func TestLogMessage_Struct(t *testing.T) {
	// Test LogMessage struct serialization
	data := LogData{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	msg := LogMessage{
		Level:  LogLevelError,
		Data:   data,
		Logger: "test-logger",
	}

	// Should be able to marshal to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal LogMessage: %v", err)
	}

	// Should be able to unmarshal back
	var unmarshaled LogMessage
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal LogMessage: %v", err)
	}

	if unmarshaled.Level != LogLevelError {
		t.Errorf("Expected level 'error', got '%s'", unmarshaled.Level)
	}

	if unmarshaled.Logger != "test-logger" {
		t.Errorf("Expected logger 'test-logger', got '%s'", unmarshaled.Logger)
	}

	if len(unmarshaled.Data) != 3 {
		t.Errorf("Expected 3 data items, got %d", len(unmarshaled.Data))
	}
}

func TestMultipleLogNotifications(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	// Send multiple notifications
	server.LogInfo("First message")
	server.LogWarning("Second message")
	server.LogError("Third message")
	server.writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected output from multiple notifications")
	}

	// Count number of JSON messages
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var notification map[string]any
		if err := json.Unmarshal([]byte(line), &notification); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestTimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	server := setupTestServerForLogging(t)
	server.writer = bufio.NewWriter(&buf)

	server.LogInfo("Test timestamp")
	server.writer.Flush()

	output := buf.String()
	var notification map[string]any
	if err := json.Unmarshal([]byte(output), &notification); err != nil {
		t.Fatalf("Failed to parse notification JSON: %v", err)
	}

	params := notification["params"].(map[string]any)
	dataMap := params["data"].(map[string]any)
	timestamp, ok := dataMap["timestamp"].(string)
	if !ok {
		t.Error("Timestamp should be a string")
	}

	// Verify timestamp is in RFC3339 format
	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Timestamp should be in RFC3339 format, got: %s", timestamp)
	}
}

// Helper function to convert string to pointer
func toPointer(s string) *string {
	return &s
}
