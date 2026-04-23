package mcp

import "fmt"

// Error codes following JSON-RPC 2.0 standard and custom domain errors
const (
	// JSON-RPC 2.0 standard error codes
	ParseError     = -32700 // Invalid JSON was received
	InvalidRequest = -32600 // The JSON sent is not a valid Request object
	MethodNotFound = -32601 // The method does not exist
	InvalidParams  = -32602 // Invalid method parameters
	InternalError  = -32603 // Internal JSON-RPC error

	// Custom domain error codes (10xxx range)
	LanguageNotFound = 10001 // No provider for the specified language
	SyntaxError      = 10002 // Source code parsing failed
	NoMatches        = 10003 // Query returned no results
	TransformFailed  = 10004 // Transformation operation failed
	StageNotFound    = 10005 // Staging ID doesn't exist
	StageExpired     = 10006 // Staging has expired
	AlreadyApplied   = 10007 // Stage was already applied
	DatabaseError    = 10008 // Database operation failed
	ConfidenceTooLow = 10009 // Confidence below threshold
	ValidationFailed = 10010 // Code validation failed
	FileSystemError  = 10011 // File system operation failed

	// Safety error codes (11xxx range)
	SafetyViolation      = 11001 // General safety violation
	FileTooLarge         = 11002 // File exceeds size limit
	TooManyFiles         = 11003 // Too many files in operation
	TotalSizeTooLarge    = 11004 // Total operation size too large
	FileModified         = 11005 // File was modified externally
	FileLocked           = 11006 // File is locked by another process
	LockTimeout          = 11007 // Could not acquire file lock
	AtomicWriteFailed    = 11008 // Atomic write operation failed
	BackupFailed         = 11009 // Backup creation failed
	RollbackFailed       = 11010 // Rollback operation failed
	PerFileConfidenceLow = 11011 // Individual file confidence too low
)

// MCPError represents a structured error for the MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("%s (%d): %v", e.Message, e.Code, e.Data)
	}
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

// NewMCPError creates a new MCP error with optional data
func NewMCPError(code int, message string, data ...any) *MCPError {
	err := &MCPError{
		Code:    code,
		Message: message,
	}
	if len(data) > 0 {
		err.Data = data[0]
	}
	return err
}

// WrapError wraps a regular error into an MCP error
func WrapError(code int, message string, err error) *MCPError {
	if err == nil {
		return NewMCPError(code, message)
	}
	return NewMCPError(code, message, err.Error())
}

// ErrorResponseWithData creates a JSON-RPC error response with additional data
func ErrorResponseWithData(id any, code int, message string, data any) Response {
	resp := ErrorResponse(id, code, message)
	if resp.Error != nil {
		resp.Error.Data = data
	}
	return resp
}
