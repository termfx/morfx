package mcp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC protocol constants used by the MCP transport layer.
const JSONRPCVersion = "2.0"

// Meta represents the optional `_meta` envelope that can accompany any
// request, notification, or response in the MCP transport. The structure is
// intentionally open-ended to support spec-defined fields like
// `progressToken` while allowing experimental metadata.
type Meta map[string]any

// ProgressToken returns the `_meta.progressToken` value if present.
func (m Meta) ProgressToken() (string, bool) {
	if m == nil {
		return "", false
	}
	if token, ok := m["progressToken"].(string); ok && token != "" {
		return token, true
	}
	return "", false
}

// WithProgressToken returns a copy of the metadata map with the provided
// progress token value applied. A zero-length token removes the field.
func (m Meta) WithProgressToken(token string) Meta {
	var clone Meta
	if len(m) == 0 {
		clone = make(Meta)
	} else {
		clone = make(Meta, len(m))
		for k, v := range m {
			clone[k] = v
		}
	}
	if token == "" {
		delete(clone, "progressToken")
		return clone
	}
	clone["progressToken"] = token
	return clone
}

// RequestMessage represents a JSON-RPC 2.0 request that expects a response.
type RequestMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Meta    Meta            `json:"_meta,omitempty"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NotificationMessage represents a JSON-RPC 2.0 notification with no ID.
type NotificationMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Meta    Meta            `json:"_meta,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ResponseMessage represents a JSON-RPC 2.0 response to a request.
type ResponseMessage struct {
	JSONRPC string       `json:"jsonrpc"`
	Meta    Meta         `json:"_meta,omitempty"`
	ID      any          `json:"id"`
	Result  any          `json:"result,omitempty"`
	Error   *ErrorObject `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error payload.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// NewRequestMessage constructs a request envelope for the supplied method.
func NewRequestMessage(id any, method string, params any) (RequestMessage, error) {
	payload, err := marshalParams(params)
	if err != nil {
		return RequestMessage{}, err
	}
	return RequestMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  payload,
	}, nil
}

// NewNotificationMessage constructs a notification envelope for the method.
func NewNotificationMessage(method string, params any) (NotificationMessage, error) {
	payload, err := marshalParams(params)
	if err != nil {
		return NotificationMessage{}, err
	}
	return NotificationMessage{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  payload,
	}, nil
}

// SuccessResponse builds a success response with the provided result payload.
func SuccessResponse(id, result any) ResponseMessage {
	return ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}

// ErrorResponse builds a response containing the supplied error object.
func ErrorResponse(id any, code int, message string, data ...any) ResponseMessage {
	var extra any
	if len(data) > 0 {
		extra = data[0]
	}
	return ResponseMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
			Data:    extra,
		},
	}
}

// ensureVersion validates that a decoded message has the expected jsonrpc value.
func ensureVersion(v string) error {
	if v == JSONRPCVersion {
		return nil
	}
	if v == "" {
		return fmt.Errorf("missing jsonrpc version")
	}
	return fmt.Errorf("unsupported jsonrpc version: %s", v)
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode params: %w", err)
	}
	return raw, nil
}

// Legacy aliases retained temporarily while refactoring call sites. They will
// be removed once all handlers are migrated to the new naming.
type (
	Request      = RequestMessage
	Response     = ResponseMessage
	Error        = ErrorObject
	Notification = NotificationMessage
)
