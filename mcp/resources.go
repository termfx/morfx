package mcp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/termfx/morfx/mcp/resources"
	"github.com/termfx/morfx/mcp/types"
)

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// ResourceDefinitions returns all available resource definitions

var builtinResourceURIs = map[string]struct{}{
	"morfx://server/info":         {},
	"morfx://server/capabilities": {},
	"morfx://providers/languages": {},
	"morfx://session/current":     {},
	"morfx://config/settings":     {},
}

func resourceContentLength(content *ResourceContent) int64 {
	if content == nil {
		return 0
	}
	if content.Text != "" {
		return int64(len(content.Text))
	}
	if len(content.Blob) > 0 {
		return int64(len(content.Blob))
	}
	return 0
}

func int64Ptr(v int64) *int64 {
	return &v
}

func (s *StdioServer) lookupResource(uri string) (types.Resource, bool) {
	if s != nil && s.resourceRegistry != nil {
		if res, ok := s.resourceRegistry.Get(uri); ok {
			return res, true
		}
	}
	if res, ok := resources.Registry.Get(uri); ok {
		return res, true
	}
	return nil, false
}

func (s *StdioServer) ResourceDefinitions() []types.ResourceDefinition {
	defs := []types.ResourceDefinition{
		{
			URI:         "morfx://server/info",
			Name:        "server-info",
			Title:       "Server Information",
			Description: "Current server status, configuration, and runtime information",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "system",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://server/capabilities",
			Name:        "server-capabilities",
			Title:       "Server Capabilities",
			Description: "Detailed information about server capabilities and supported features",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "system",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://providers/languages",
			Name:        "providers-languages",
			Title:       "Supported Languages",
			Description: "List of programming languages supported for code transformations",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "providers",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://session/current",
			Name:        "session-current",
			Title:       "Current Session",
			Description: "Information about the current transformation session",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "session",
				"readonly": true,
			},
		},
		{
			URI:         "morfx://config/settings",
			Name:        "config-settings",
			Title:       "Configuration Settings",
			Description: "Current server configuration and settings",
			MimeType:    "application/json",
			Annotations: map[string]any{
				"category": "config",
				"readonly": true,
			},
		},
	}

	for i := range defs {
		enrichResourceDefinition(&defs[i], nil)
		if s != nil {
			s.enrichRuntimeResourceMetadata(&defs[i])
		}
	}

	appendDynamic := func(res types.Resource) {
		uri := res.URI()
		definition := types.ResourceDefinition{
			URI:         uri,
			Name:        uri,
			Title:       res.Name(),
			Description: res.Description(),
			MimeType:    res.MimeType(),
			Annotations: map[string]any{
				"title": res.Name(),
			},
		}
		enrichResourceDefinition(&definition, res)
		if s != nil {
			s.enrichRuntimeResourceMetadata(&definition)
		}
		defs = append(defs, definition)
	}

	seen := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		seen[def.URI] = struct{}{}
	}

	for _, res := range resources.Registry.List() {
		if _, exists := seen[res.URI()]; exists {
			continue
		}
		appendDynamic(res)
		seen[res.URI()] = struct{}{}
	}

	if s != nil && s.resourceRegistry != nil {
		for _, res := range s.resourceRegistry.List() {
			if _, exists := seen[res.URI()]; exists {
				continue
			}
			appendDynamic(res)
			seen[res.URI()] = struct{}{}
		}
	}

	return defs
}

func (s *StdioServer) enrichRuntimeResourceMetadata(def *types.ResourceDefinition) {
	if def == nil {
		return
	}
	if def.Annotations == nil {
		def.Annotations = make(map[string]any)
	}
	if _, ok := def.Annotations["audience"]; !ok {
		def.Annotations["audience"] = "developer"
	}

	if _, builtin := builtinResourceURIs[def.URI]; builtin {
		if _, ok := def.Annotations["cacheControl"]; !ok {
			def.Annotations["cacheControl"] = "static"
		}
		if _, ok := def.Annotations["stability"]; !ok {
			def.Annotations["stability"] = "stable"
		}
		if def.Size == nil {
			if def.URI == "morfx://server/capabilities" {
				return
			}
			var content *ResourceContent
			switch def.URI {
			case "morfx://server/info":
				content, _ = s.generateServerInfo()
			case "morfx://server/capabilities":
				content, _ = s.generateServerCapabilities()
			case "morfx://providers/languages":
				content, _ = s.generateSupportedLanguages()
			case "morfx://session/current":
				content, _ = s.generateCurrentSession()
			case "morfx://config/settings":
				content, _ = s.generateConfigSettings()
			}
			if size := resourceContentLength(content); size > 0 {
				def.Size = int64Ptr(size)
			}
		}
		return
	}

	if _, ok := def.Annotations["cacheControl"]; !ok {
		def.Annotations["cacheControl"] = "dynamic"
	}
	if _, ok := def.Annotations["stability"]; !ok {
		def.Annotations["stability"] = "experimental"
	}
	if def.Size != nil {
		return
	}
	if res, ok := resources.Registry.Get(def.URI); ok {
		if text, err := res.Contents(); err == nil {
			def.Size = int64Ptr(int64(len(text)))
		}
	}
}

func enrichResourceDefinition(def *types.ResourceDefinition, res types.Resource) {
	if def.Title == "" {
		def.Title = def.Name
	}
	if def.Annotations == nil {
		def.Annotations = make(map[string]any)
	}
	if _, ok := def.Annotations["category"]; !ok {
		def.Annotations["category"] = resourceCategory(def.URI)
	}
	if _, ok := def.Annotations["readonly"]; !ok {
		def.Annotations["readonly"] = true
	}
	if res != nil {
		if text, err := res.Contents(); err == nil {
			size := int64(len(text))
			def.Size = &size
		}
	}
}

func resourceCategory(uri string) string {
	if idx := strings.Index(uri, "://"); idx != -1 {
		return uri[:idx]
	}
	return "custom"
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
		if res, ok := s.lookupResource(uri); ok {
			text, err := res.Contents()
			if err != nil {
				return nil, types.WrapError(InternalError, "Failed to read resource", err)
			}
			return &ResourceContent{
				URI:      uri,
				MimeType: res.MimeType(),
				Text:     text,
			}, nil
		}
		return nil, NewMCPError(MethodNotFound, "Resource not found", map[string]any{
			"uri": uri,
		})
	}
}

// generateServerInfo creates server information resource content
func (s *StdioServer) generateServerInfo() (*ResourceContent, error) {
	dbInfo := map[string]any{
		"enabled": s.db != nil,
	}
	if s.db != nil {
		dbInfo["type"] = "sqlite"
	} else {
		dbInfo["type"] = "none"
	}
	if s.config.DatabaseURL != "" {
		dbInfo["has_url"] = true
	}

	info := map[string]any{
		"name":    "Morfx MCP Server",
		"version": "1.5.0",
		"runtime": map[string]any{
			"go_version": runtime.Version(),
			"platform":   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"cpus":       runtime.NumCPU(),
		},
		"uptime": map[string]any{
			"started_at": time.Now().Format(time.RFC3339),
		},
		"database": dbInfo,
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
		"protocol_version": supportedProtocolVersion,
		"tools": map[string]any{
			"count":    len(s.toolRegistry.List()),
			"features": []string{"query", "transform", "stage", "apply"},
		},
		"resources": map[string]any{
			"count":    len(s.ResourceDefinitions()),
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
		"auto_apply_enabled":      s.config.AutoApplyEnabled,
		"auto_apply_threshold":    s.config.AutoApplyThreshold,
		"staging_ttl_minutes":     s.config.StagingTTL.Minutes(),
		"max_stages_per_session":  s.config.MaxStagesPerSession,
		"max_applies_per_session": s.config.MaxAppliesPerSession,
		"debug":                   s.config.Debug,
	}

	if s.config.DatabaseURL != "" {
		settings["database_url_present"] = true
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
