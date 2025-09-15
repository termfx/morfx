package mcp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ResourceDefinition describes a resource available to the client
type ResourceDefinition struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// GetResourceDefinitions returns all available resource definitions
func GetResourceDefinitions() []ResourceDefinition {
	return []ResourceDefinition{
		{
			URI:         "morfx://server/info",
			Name:        "Server Information",
			Description: "Current server status, configuration, and runtime information",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "system",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://server/capabilities",
			Name:        "Server Capabilities",
			Description: "Detailed information about server capabilities and supported features",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "system",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://providers/languages",
			Name:        "Supported Languages",
			Description: "List of programming languages supported for code transformations",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "providers",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://session/current",
			Name:        "Current Session",
			Description: "Information about the current transformation session",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "session",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://config/settings",
			Name:        "Configuration Settings",
			Description: "Current server configuration and settings",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "config",
				"readonly": true,
			},
		},
	}
}

// handleListResources returns available resources to the client
func (s *StdioServer) handleListResources(req Request) Response {
	resources := GetResourceDefinitions()

	return SuccessResponse(req.ID, map[string]any{
		"resources": resources,
	})
}

// handleReadResource returns the content of a specific resource
func (s *StdioServer) handleReadResource(req Request) Response {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource read parameters")
	}

	s.debugLog("Reading resource: %s", params.URI)

	// Generate resource content based on URI
	content, err := s.generateResourceContent(params.URI)
	if err != nil {
		if mcpErr, ok := err.(*MCPError); ok {
			return ErrorResponseWithData(req.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		return ErrorResponse(req.ID, InternalError, err.Error())
	}

	return SuccessResponse(req.ID, map[string]any{
		"contents": []ResourceContent{*content},
	})
}

// handleSubscribeResource subscribes to resource changes
func (s *StdioServer) handleSubscribeResource(req Request) Response {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource subscribe parameters")
	}

	s.debugLog("Subscribing to resource: %s", params.URI)

	// For now, just acknowledge the subscription
	// In a full implementation, you'd track subscriptions and send notifications
	return SuccessResponse(req.ID, map[string]any{})
}

// handleUnsubscribeResource unsubscribes from resource changes
func (s *StdioServer) handleUnsubscribeResource(req Request) Response {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource unsubscribe parameters")
	}

	s.debugLog("Unsubscribing from resource: %s", params.URI)

	// For now, just acknowledge the unsubscription
	return SuccessResponse(req.ID, map[string]any{})
}

// generateResourceContent creates the actual content for a resource URI
func (s *StdioServer) generateResourceContent(uri string) (*ResourceContent, error) {
	switch uri {
	case "morfx://server/info":
		return s.generateServerInfo()
	case "morfx://server/capabilities":
		return s.generateServerCapabilities()
	case "morfx://providers/languages":
		return s.generateSupportedLanguages()
	case "morfx://session/current":
		return s.generateCurrentSession()
	case "morfx://config/settings":
		return s.generateConfigSettings()
	default:
		return nil, NewMCPError(MethodNotFound, "Resource not found", map[string]any{
			"uri": uri,
		})
	}
}

// generateServerInfo creates server information resource content
func (s *StdioServer) generateServerInfo() (*ResourceContent, error) {
	info := map[string]any{
		"name":    "Morfx MCP Server",
		"version": "1.3.0",
		"runtime": map[string]any{
			"go_version": runtime.Version(),
			"platform":   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"cpus":       runtime.NumCPU(),
		},
		"uptime": map[string]any{
			"started_at": time.Now().Format(time.RFC3339),
		},
		"database": map[string]any{
			"enabled": s.db != nil,
			"type":    "sqlite",
			"url":     s.config.DatabaseURL,
		},
		"features": map[string]any{
			"staging":   s.staging != nil,
			"sessions":  s.session != nil,
			"file_ops":  true,
			"in_memory": true,
		},
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, WrapError(InternalError, "Failed to marshal server info", err)
	}

	return &ResourceContent{
		URI:      "morfx://server/info",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// generateServerCapabilities creates server capabilities resource content
func (s *StdioServer) generateServerCapabilities() (*ResourceContent, error) {
	capabilities := map[string]any{
		"protocol_version": "2024-11-05",
		"tools": map[string]any{
			"count":    len(s.tools),
			"features": []string{"query", "transform", "stage", "apply"},
		},
		"resources": map[string]any{
			"count":    len(GetResourceDefinitions()),
			"features": []string{"read", "subscribe", "notifications"},
		},
		"prompts": map[string]any{
			"count":    5, // Number of available prompts
			"features": []string{"get", "list"},
		},
		"languages": map[string]any{
			"supported": []string{"go"},
			"planned":   []string{"python", "javascript", "typescript"},
		},
		"transformations": []string{
			"query", "replace", "delete", "insert_before", "insert_after", "append",
		},
		"file_operations": map[string]any{
			"supported": true,
			"features":  []string{"file_query", "file_replace", "file_delete", "backup"},
		},
	}

	data, err := json.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		return nil, WrapError(InternalError, "Failed to marshal capabilities", err)
	}

	return &ResourceContent{
		URI:      "morfx://server/capabilities",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// generateSupportedLanguages creates supported languages resource content
func (s *StdioServer) generateSupportedLanguages() (*ResourceContent, error) {
	languages := map[string]any{
		"supported": []map[string]any{
			{
				"name":         "go",
				"display_name": "Go",
				"extensions":   []string{".go"},
				"features": map[string]any{
					"ast_parsing":        true,
					"transformations":    true,
					"confidence_scoring": true,
				},
			},
		},
		"planned": []map[string]any{
			{
				"name":         "python",
				"display_name": "Python",
				"extensions":   []string{".py"},
				"status":       "in_development",
			},
			{
				"name":         "javascript",
				"display_name": "JavaScript",
				"extensions":   []string{".js", ".jsx"},
				"status":       "planned",
			},
			{
				"name":         "typescript",
				"display_name": "TypeScript",
				"extensions":   []string{".ts", ".tsx"},
				"status":       "planned",
			},
		},
	}

	data, err := json.MarshalIndent(languages, "", "  ")
	if err != nil {
		return nil, WrapError(InternalError, "Failed to marshal languages", err)
	}

	return &ResourceContent{
		URI:      "morfx://providers/languages",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// generateCurrentSession creates current session resource content
func (s *StdioServer) generateCurrentSession() (*ResourceContent, error) {
	sessionInfo := map[string]any{
		"session_id": nil,
		"status":     "active",
		"database":   s.db != nil,
		"staging":    s.staging != nil,
		"mode":       "stateless", // Default to stateless when no database
	}

	if s.session != nil {
		sessionInfo["session_id"] = s.session.ID
		sessionInfo["started_at"] = s.session.StartedAt.Format(time.RFC3339)
		sessionInfo["stages_count"] = s.session.StagesCount
		sessionInfo["applies_count"] = s.session.AppliesCount
		sessionInfo["mode"] = "persistent"
	} else {
		sessionInfo["message"] = "Running in stateless mode - no persistence available"
		sessionInfo["limitations"] = []string{
			"No staging of transformations",
			"No session history",
			"Transformations applied immediately or returned as text",
		}
	}

	data, err := json.MarshalIndent(sessionInfo, "", "  ")
	if err != nil {
		return nil, WrapError(InternalError, "Failed to marshal session info", err)
	}

	return &ResourceContent{
		URI:      "morfx://session/current",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// generateConfigSettings creates configuration settings resource content
func (s *StdioServer) generateConfigSettings() (*ResourceContent, error) {
	settings := map[string]any{
		"database_url":            s.config.DatabaseURL,
		"auto_apply_enabled":      s.config.AutoApplyEnabled,
		"auto_apply_threshold":    s.config.AutoApplyThreshold,
		"staging_ttl_minutes":     s.config.StagingTTL.Minutes(),
		"max_stages_per_session":  s.config.MaxStagesPerSession,
		"max_applies_per_session": s.config.MaxAppliesPerSession,
		"debug":                   s.config.Debug,
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, WrapError(InternalError, "Failed to marshal config settings", err)
	}

	return &ResourceContent{
		URI:      "morfx://config/settings",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}
