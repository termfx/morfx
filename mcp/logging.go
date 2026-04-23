package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
func (s *StdioServer) handleSetLoggingLevel(ctx context.Context, req Request) Response {
	var params struct {
		Level LogLevel `json:"level"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid logging level parameters")
	}

	s.sessionState.SetLoggingLevel(params.Level)

	if s.config.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Logging level set to: %s\n", params.Level)
	}

	return SuccessResponse(req.ID, map[string]any{})
}

// sendLogNotification sends a log message notification to the client
func (s *StdioServer) sendLogNotification(level LogLevel, message string, data LogData) {
	if !shouldEmitLog(s.sessionState.LoggingLevel(), level) {
		return
	}

	// Create or use existing data map
	if data == nil {
		data = make(LogData)
	}
	data["message"] = message
	data["timestamp"] = time.Now().Format(time.RFC3339)

	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/message",
		"params": map[string]any{
			"level":  level,
			"data":   data,
			"logger": "morfx",
		},
	}

	s.emitNotification(notification)
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

	s.emitNotification(notification)
}

// sendResourceListChangedNotification sends a notification when the resource list changes
func (s *StdioServer) sendResourceListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/resources/list_changed",
		"params":  map[string]any{},
	}

	s.emitNotification(notification)
}

// sendToolListChangedNotification sends a notification when the tool list changes
func (s *StdioServer) sendToolListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/tools/list_changed",
		"params":  map[string]any{},
	}

	s.emitNotification(notification)
}

// sendPromptListChangedNotification sends a notification when the prompt list changes
func (s *StdioServer) sendPromptListChangedNotification() {
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/prompts/list_changed",
		"params":  map[string]any{},
	}

	s.emitNotification(notification)
}

// sendCancelledNotification informs the client that a server-initiated request was aborted.
func (s *StdioServer) sendCancelledNotification(requestID string, progressToken string) {
	params := map[string]any{}
	if requestID != "" {
		params["requestId"] = requestID
	}
	if progressToken != "" {
		params["progressToken"] = progressToken
	}

	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/cancelled",
		"params":  params,
	}

	s.emitNotification(notification)
}

// sendProgressNotification sends a progress notification for long-running operations
func (s *StdioServer) sendProgressNotification(progressToken string, progress, total float64, message string) {
	params := map[string]any{
		"progressToken": progressToken,
		"progress":      progress,
		"total":         total,
	}
	if message != "" {
		params["message"] = message
	}

	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/progress",
		"params":  params,
	}

	s.emitNotification(notification)
}

func shouldEmitLog(min LogLevel, level LogLevel) bool {
	order := map[LogLevel]int{
		LogLevelDebug:     0,
		LogLevelInfo:      1,
		LogLevelNotice:    2,
		LogLevelWarning:   3,
		LogLevelError:     4,
		LogLevelCritical:  5,
		LogLevelAlert:     6,
		LogLevelEmergency: 7,
	}
	// Default to info if unknown
	minRank, ok := order[min]
	if !ok {
		minRank = order[LogLevelInfo]
	}
	levelRank, ok := order[level]
	if !ok {
		levelRank = order[LogLevelInfo]
	}
	return levelRank >= minRank
}

func (s *StdioServer) emitNotification(payload map[string]any) {
	payload["jsonrpc"] = JSONRPCVersion
	notificationJSON, err := json.Marshal(payload)
	if err != nil {
		s.debugLog("Failed to marshal notification %v: %v", payload["method"], err)
		return
	}
	s.writeFrame(notificationJSON)
}
