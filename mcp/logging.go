package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// LogData represents structured data for a log message
type LogData map[string]any

// LogMessage represents a log message according to MCP specification
type LogMessage struct {
	Level  LogLevel `json:"level"`
	Data   LogData  `json:"data,omitempty"`
	Logger string   `json:"logger,omitempty"`
}

// handleSetLoggingLevel handles logging level configuration
func (s *StdioServer) handleSetLoggingLevel(req Request) Response {
	var params struct {
		Level LogLevel `json:"level"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid logging level parameters")
	}

	s.debugLog("Setting logging level to: %s", params.Level)

	// Store the logging level (in a real implementation, you'd store this in server state)
	// For now, just acknowledge the setting
	return SuccessResponse(req.ID, map[string]any{})
}

// sendLogNotification sends a log message notification to the client
func (s *StdioServer) sendLogNotification(level LogLevel, message string, data LogData) {
	if !s.config.Debug && level == LogLevelDebug {
		return // Don't send debug logs unless debug mode is enabled
	}

	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/message",
		"params": map[string]any{
			"level":  level,
			"data":   data,
			"logger": "morfx",
		},
	}

	// Add the message to data
	if data == nil {
		data = make(LogData)
	}
	data["message"] = message
	data["timestamp"] = time.Now().Format(time.RFC3339)

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal log notification: %v", err)
		return
	}

	// Send notification
	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}

// LogInfo sends an info level log notification
func (s *StdioServer) LogInfo(message string, data ...LogData) {
	var logData LogData
	if len(data) > 0 {
		logData = data[0]
	}
	s.sendLogNotification(LogLevelInfo, message, logData)
}

// LogWarning sends a warning level log notification
func (s *StdioServer) LogWarning(message string, data ...LogData) {
	var logData LogData
	if len(data) > 0 {
		logData = data[0]
	}
	s.sendLogNotification(LogLevelWarning, message, logData)
}

// LogError sends an error level log notification
func (s *StdioServer) LogError(message string, data ...LogData) {
	var logData LogData
	if len(data) > 0 {
		logData = data[0]
	}
	s.sendLogNotification(LogLevelError, message, logData)
}

// LogDebug sends a debug level log notification
func (s *StdioServer) LogDebug(message string, data ...LogData) {
	var logData LogData
	if len(data) > 0 {
		logData = data[0]
	}
	s.sendLogNotification(LogLevelDebug, message, logData)
}

// sendResourceUpdatedNotification sends a notification when a resource is updated
func (s *StdioServer) sendResourceUpdatedNotification(uri string) {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/resources/updated",
		"params": map[string]any{
			"uri": uri,
		},
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal resource notification: %v", err)
		return
	}

	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}

// sendResourceListChangedNotification sends a notification when the resource list changes
func (s *StdioServer) sendResourceListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/resources/list_changed",
		"params":  map[string]any{},
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal resource list notification: %v", err)
		return
	}

	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}

// sendToolListChangedNotification sends a notification when the tool list changes
func (s *StdioServer) sendToolListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/tools/list_changed",
		"params":  map[string]any{},
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal tool list notification: %v", err)
		return
	}

	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}

// sendPromptListChangedNotification sends a notification when the prompt list changes
func (s *StdioServer) sendPromptListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/prompts/list_changed",
		"params":  map[string]any{},
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal prompt list notification: %v", err)
		return
	}

	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}

// sendProgressNotification sends a progress notification for long-running operations
func (s *StdioServer) sendProgressNotification(progressToken string, progress, total float64) {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/progress",
		"params": map[string]any{
			"progressToken": progressToken,
			"progress":      progress,
			"total":         total,
		},
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		s.debugLog("Failed to marshal progress notification: %v", err)
		return
	}

	fmt.Fprintf(s.writer, "%s\n", notificationJSON)
	s.writer.Flush()
}
