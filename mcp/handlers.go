package mcp

import (
	"encoding/json"
	"fmt"
)

// handleListTools returns available tools to the client
func (s *StdioServer) handleListTools(req Request) Response {
	tools := GetToolDefinitions()

	return SuccessResponse(req.ID, map[string]any{
		"tools": tools,
	})
}

// handleCallTool executes a specific tool
func (s *StdioServer) handleCallTool(req Request) Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid params structure")
	}

	s.debugLog("Calling tool: %s", params.Name)

	// Look up tool handler
	s.mu.RLock()
	handler, exists := s.tools[params.Name]
	s.mu.RUnlock()

	if !exists {
		return ErrorResponse(req.ID, MethodNotFound,
			fmt.Sprintf("Tool not found: %s", params.Name))
	}

	// Execute tool
	result, err := handler(params.Arguments)
	if err != nil {
		// Check if it's an MCP error with code
		if mcpErr, ok := err.(*MCPError); ok {
			return ErrorResponseWithData(req.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		// Generic error
		return ErrorResponse(req.ID, InternalError, err.Error())
	}

	return SuccessResponse(req.ID, result)
}

// handleInitialize handles the MCP initialization handshake
func (s *StdioServer) handleInitialize(req Request) Response {
	// Parse initialize params
	var params struct {
		ProtocolVersion string   `json:"protocolVersion"`
		Capabilities    struct{} `json:"capabilities"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}

	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
		s.debugLog("Client: %s v%s, Protocol: %s",
			params.ClientInfo.Name,
			params.ClientInfo.Version,
			params.ProtocolVersion)
	}

	// Return server capabilities
	return SuccessResponse(req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": true, // Server can notify when tools list changes
			},
			"resources": map[string]any{
				"subscribe":   true, // Server supports resource subscriptions
				"listChanged": true, // Server can notify when resources list changes
			},
			"prompts": map[string]any{
				"listChanged": true, // Server can notify when prompts list changes
			},
			"logging": map[string]any{}, // Server supports logging messages
		},
		"serverInfo": map[string]any{
			"name":    "morfx",
			"version": "1.3.0",
		},
	})
}

// handleInitialized confirms initialization complete
func (s *StdioServer) handleInitialized(req Request) Response {
	s.debugLog("Initialization complete")
	// Notifications have no ID and expect no response
	if req.ID == nil {
		// Return empty response that won't be sent
		return Response{}
	}
	// This shouldn't happen, but handle it
	return SuccessResponse(req.ID, map[string]any{})
}

// handlePing responds to keepalive pings
func (s *StdioServer) handlePing(req Request) Response {
	return SuccessResponse(req.ID, map[string]any{})
}
