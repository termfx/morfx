package types

import (
	"encoding/json"
	"testing"
)

// JSONRPCRequest for MCP testing
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
}

// JSONRPCResponse for MCP testing
type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
}

func TestJSONRPCRequest(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	if req.JSONRPC != "2.0" {
		t.Error("JSONRPC should be 2.0")
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Error("Should marshal to JSON")
	}

	var decoded JSONRPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Error("Should unmarshal from JSON")
	}
}

func TestJSONRPCResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
	}

	data, _ := json.Marshal(resp)
	if len(data) == 0 {
		t.Error("Should produce JSON output")
	}
}

func TestMCPError(t *testing.T) {
	err := MCPError{
		Code:    -1,
		Message: "test error",
	}

	if err.Error() != "test error" {
		t.Error("Error() should return message")
	}
}

func TestNewMCPError(t *testing.T) {
	err := NewMCPError(-1, "test", nil)
	if err.Code != -1 {
		t.Error("Code should be -1")
	}
}

func TestWrapError(t *testing.T) {
	origErr := &MCPError{Code: -2, Message: "original"}
	wrappedErr := WrapError(-1, "wrapped", origErr)

	if wrappedErr.Code != -1 {
		t.Error("Wrapped error should have new code")
	}
}
