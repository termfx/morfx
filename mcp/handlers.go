package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/termfx/morfx/mcp/types"
)

const supportedProtocolVersion = "2025-06-18"

type listToolsResult struct {
	Tools      []types.ToolDefinition `json:"tools"`
	NextCursor *string                `json:"nextCursor,omitempty"`
}

type listPromptsResult struct {
	Prompts    []types.PromptDefinition `json:"prompts"`
	NextCursor *string                  `json:"nextCursor,omitempty"`
}

type listResourcesResult struct {
	Resources  []types.ResourceDefinition `json:"resources"`
	NextCursor *string                    `json:"nextCursor,omitempty"`
}

type listResourceTemplatesResult struct {
	ResourceTemplates []types.ResourceTemplateDefinition `json:"resourceTemplates"`
	NextCursor        *string                            `json:"nextCursor,omitempty"`
}

type readResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

type getPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// handleListTools returns available tools to the client
func (s *StdioServer) handleListTools(ctx context.Context, req Request) Response {
	params, err := decodePaginationParams(req.Params)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid pagination parameters")
	}

	definitions := s.toolRegistry.GetDefinitions()
	page, nextCursor, err := applyPagination(definitions, params.Cursor, params.Limit)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, err.Error())
	}

	result := listToolsResult{Tools: page}
	result.NextCursor = nextCursor
	return SuccessResponse(req.ID, result)
}

// handleCallTool executes a specific tool
func (s *StdioServer) handleCallTool(ctx context.Context, req Request) Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid params structure")
	}

	s.debugLog("Calling tool: %s", params.Name)

	progressStatus := "completed"
	if token, ok := req.Meta.ProgressToken(); ok {
		s.sendProgressNotification(token, 0, 100, "queued")
		defer func() {
			s.sendProgressNotification(token, 100, 100, progressStatus)
		}()
	}

	result, err := s.toolRegistry.Execute(ctx, params.Name, params.Arguments)
	if err != nil {
		if errors.Is(err, ErrToolNotFound) {
			progressStatus = "failed"
			return ErrorResponse(req.ID, MethodNotFound,
				fmt.Sprintf("Tool not found: %s", params.Name))
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			progressStatus = "cancelled"
			return SuccessResponse(req.ID, buildErrorToolResult(-32800, "Request cancelled", map[string]any{"detail": err.Error()}))
		}

		if mcpErr, ok := err.(*types.MCPError); ok {
			progressStatus = "failed"
			return SuccessResponse(req.ID, buildErrorToolResult(mcpErr.Code, mcpErr.Message, mcpErr.Data))
		}
		if legacyErr, ok := err.(*MCPError); ok {
			progressStatus = "failed"
			return SuccessResponse(req.ID, buildErrorToolResult(legacyErr.Code, legacyErr.Message, legacyErr.Data))
		}

		progressStatus = "failed"
		return ErrorResponse(req.ID, InternalError, err.Error())
	}

	return SuccessResponse(req.ID, normalizeToolResult(result))
}

// handleInitialize handles the MCP initialization handshake
func (s *StdioServer) handleInitialize(ctx context.Context, req Request) Response {
	var params struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		Capabilities    map[string]any         `json:"capabilities"`
		ClientInfo      map[string]interface{} `json:"clientInfo"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid initialize parameters")
	}

	clientName := "unknown"
	clientVersion := ""
	if params.ClientInfo != nil {
		if name, ok := params.ClientInfo["name"].(string); ok {
			clientName = name
		}
		if version, ok := params.ClientInfo["version"].(string); ok {
			clientVersion = version
		}
	}

	s.debugLog("Client initialize: %s v%s requested protocol %s", clientName, clientVersion, params.ProtocolVersion)

	negotiated := supportedProtocolVersion
	if params.ProtocolVersion != "" && params.ProtocolVersion != supportedProtocolVersion {
		s.debugLog("Client protocol %s not matched, negotiating %s", params.ProtocolVersion, negotiated)
	}

	s.sessionState.MarkInitialized(negotiated, params.Capabilities)

	result := map[string]any{
		"protocolVersion": negotiated,
		"capabilities":    s.serverCapabilities(),
		"serverInfo": map[string]any{
			"name":    "morfx",
			"version": "1.5.0",
		},
	}

	if instructions := s.defaultInstructions(); instructions != "" {
		result["instructions"] = instructions
	}

	return SuccessResponse(req.ID, result)
}

// handleInitialized confirms initialization complete
func (s *StdioServer) handleInitialized(ctx context.Context, req Request) Response {
	s.debugLog("Initialization complete")
	go func() {
		resp, err := s.RequestRoots(context.Background(), map[string]any{}, Meta{})
		if err != nil {
			s.debugLog("roots/list request failed: %v", err)
			return
		}
		if resp.Error != nil {
			s.debugLog("roots/list responded with error: %s", resp.Error.Message)
			return
		}
		var roots []string
		if result, ok := resp.Result.(map[string]any); ok {
			if items, ok := result["roots"].([]any); ok {
				for _, item := range items {
					if rootObj, ok := item.(map[string]any); ok {
						if uri, ok := rootObj["uri"].(string); ok {
							roots = append(roots, uri)
						}
					}
				}
			}
		}
		if len(roots) > 0 {
			s.sessionState.SetClientRoots(roots)
		}
	}()
	if req.ID == nil {
		return Response{}
	}
	return SuccessResponse(req.ID, map[string]any{})
}

// handlePing responds to keepalive pings
func (s *StdioServer) handlePing(ctx context.Context, req Request) Response {
	return SuccessResponse(req.ID, map[string]any{})
}

// handleListPrompts returns available prompts to the client
func (s *StdioServer) handleListPrompts(ctx context.Context, req Request) Response {
	params, err := decodePaginationParams(req.Params)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid pagination parameters")
	}

	prompts := GetPromptDefinitions()
	page, nextCursor, err := applyPagination(prompts, params.Cursor, params.Limit)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, err.Error())
	}

	result := listPromptsResult{Prompts: page}
	result.NextCursor = nextCursor
	return SuccessResponse(req.ID, result)
}

// handleGetPrompt returns the content of a specific prompt
func (s *StdioServer) handleGetPrompt(ctx context.Context, req Request) Response {
	var params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid prompt parameters")
	}

	s.debugLog("Getting prompt: %s", params.Name)

	messages, err := s.generatePromptContent(params.Name, params.Arguments)
	if err != nil {
		if mcpErr, ok := err.(*MCPError); ok {
			return ErrorResponseWithData(req.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		return ErrorResponse(req.ID, InternalError, err.Error())
	}

	return SuccessResponse(req.ID, getPromptResult{Messages: messages})
}

// handleListResources returns available resources to the client
func (s *StdioServer) handleListResources(ctx context.Context, req Request) Response {
	params, err := decodePaginationParams(req.Params)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid pagination parameters")
	}

	resources := s.ResourceDefinitions()
	page, nextCursor, err := applyPagination(resources, params.Cursor, params.Limit)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, err.Error())
	}

	result := listResourcesResult{Resources: page}
	result.NextCursor = nextCursor
	return SuccessResponse(req.ID, result)
}

// handleListResourceTemplates returns available resource templates to the client
func (s *StdioServer) handleListResourceTemplates(ctx context.Context, req Request) Response {
	params, err := decodePaginationParams(req.Params)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid pagination parameters")
	}

	templates := s.resourceTemplateRegistry.List()
	page, nextCursor, err := applyPagination(templates, params.Cursor, params.Limit)
	if err != nil {
		return ErrorResponse(req.ID, InvalidParams, err.Error())
	}

	result := listResourceTemplatesResult{ResourceTemplates: page}
	result.NextCursor = nextCursor
	return SuccessResponse(req.ID, result)
}

// handleReadResource returns the content of a specific resource
func (s *StdioServer) handleReadResource(ctx context.Context, req Request) Response {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource read parameters")
	}

	s.debugLog("Reading resource: %s", params.URI)

	content, err := s.generateResourceContent(params.URI)
	if err != nil {
		if mcpErr, ok := err.(*MCPError); ok {
			return ErrorResponseWithData(req.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		return ErrorResponse(req.ID, InternalError, err.Error())
	}

	return SuccessResponse(req.ID, readResourceResult{Contents: []ResourceContent{*content}})
}

// handleSubscribeResource subscribes to resource changes
func (s *StdioServer) handleSubscribeResource(ctx context.Context, req Request) Response {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource subscribe parameters")
	}

	s.debugLog("Subscribing to resource: %s", params.URI)
	if params.URI == "" {
		return ErrorResponse(req.ID, InvalidParams, "Resource URI is required")
	}

	if _, builtin := builtinResourceURIs[params.URI]; builtin {
		// Built-in resources are static; acknowledge the subscription without watcher wiring.
		return SuccessResponse(req.ID, map[string]any{})
	}

	resource, ok := s.lookupResource(params.URI)
	if !ok {
		return ErrorResponseWithData(req.ID, InvalidParams, "Resource not found", map[string]any{"uri": params.URI})
	}

	watchable, ok := resource.(types.WatchableResource)
	if !ok {
		return SuccessResponse(req.ID, map[string]any{})
	}

	subCtx, cancel := context.WithCancel(context.Background())
	updates, err := watchable.Watch(subCtx)
	if err != nil {
		cancel()
		if errors.Is(err, types.ErrResourceWatchUnsupported) {
			return SuccessResponse(req.ID, map[string]any{})
		}
		return ErrorResponse(req.ID, InternalError, err.Error())
	}
	if updates == nil {
		cancel()
		return SuccessResponse(req.ID, map[string]any{})
	}

	subscriptionID := s.addResourceSubscription(params.URI, cancel)
	go s.forwardResourceUpdates(params.URI, subscriptionID, updates, cancel)

	return SuccessResponse(req.ID, map[string]any{"subscriptionId": subscriptionID})
}

// handleUnsubscribeResource unsubscribes from resource changes
func (s *StdioServer) handleUnsubscribeResource(ctx context.Context, req Request) Response {
	var params struct {
		URI            string `json:"uri"`
		SubscriptionID string `json:"subscriptionId,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid resource unsubscribe parameters")
	}

	s.debugLog("Unsubscribing from resource: %s", params.URI)
	if params.SubscriptionID != "" {
		s.cancelResourceSubscription(params.URI, params.SubscriptionID)
	} else if params.URI != "" {
		s.cancelAllResourceSubscriptions(params.URI)
	}
	return SuccessResponse(req.ID, map[string]any{})
}

func (s *StdioServer) serverCapabilities() map[string]any {
	return map[string]any{
		"tools": map[string]any{
			"listChanged": true,
		},
		"resources": map[string]any{
			"subscribe":   true,
			"listChanged": true,
			"templates": map[string]any{
				"listChanged": true,
			},
		},
		"prompts": map[string]any{
			"listChanged": true,
		},
		"logging": map[string]any{},
	}
}

func (s *StdioServer) defaultInstructions() string {
	return "Use tools/list to discover available actions, then call tools/call with the requested name to query or transform code."
}

func buildErrorToolResult(code int, message string, data any) types.CallToolResult {
	structured := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		structured["data"] = data
	}
	return types.CallToolResult{
		Content: []types.ContentBlock{
			{Type: "text", Text: message},
		},
		StructuredContent: structured,
		IsError:           true,
	}
}

func normalizeToolResult(result any) types.CallToolResult {
	if result == nil {
		return types.CallToolResult{Content: []types.ContentBlock{{Type: "text", Text: ""}}}
	}
	if callResult, ok := result.(types.CallToolResult); ok {
		return callResult
	}
	if callPtr, ok := result.(*types.CallToolResult); ok && callPtr != nil {
		return *callPtr
	}

	var (
		blocks     []types.ContentBlock
		structured any = result
	)

	if asMap, ok := result.(map[string]any); ok {
		if rawContent, exists := asMap["content"]; exists {
			blocks = toContentBlocks(rawContent)
		}
		if sc, exists := asMap["structuredContent"]; exists {
			structured = sc
		} else {
			remainder := mapWithout(asMap, "content")
			if len(remainder) == 0 {
				structured = nil
			} else {
				structured = remainder
			}
		}
	}

	if len(blocks) == 0 {
		if text, ok := result.(string); ok {
			blocks = []types.ContentBlock{{Type: "text", Text: text}}
			structured = nil
		} else {
			blocks = []types.ContentBlock{{Type: "text", Text: marshalAsText(result)}}
			structured = result
		}
	}

	return types.CallToolResult{
		Content:           blocks,
		StructuredContent: structured,
	}
}

func toContentBlocks(value any) []types.ContentBlock {
	switch blocks := value.(type) {
	case []types.ContentBlock:
		return blocks
	case []map[string]any:
		return convertContentMaps(blocks)
	case []any:
		converted := make([]types.ContentBlock, 0, len(blocks))
		for _, item := range blocks {
			if m, ok := item.(map[string]any); ok {
				if block, ok := convertContentMap(m); ok {
					converted = append(converted, block)
				}
			}
		}
		return converted
	default:
		return nil
	}
}

func convertContentMaps(items []map[string]any) []types.ContentBlock {
	converted := make([]types.ContentBlock, 0, len(items))
	for _, item := range items {
		if block, ok := convertContentMap(item); ok {
			converted = append(converted, block)
		}
	}
	return converted
}

func convertContentMap(item map[string]any) (types.ContentBlock, bool) {
	typeName, _ := item["type"].(string)
	if typeName == "" {
		typeName = "text"
	}
	block := types.ContentBlock{Type: typeName}
	if text, ok := item["text"].(string); ok {
		block.Text = text
	}
	if uri, ok := item["uri"].(string); ok {
		block.URI = uri
	}
	if mime, ok := item["mimeType"].(string); ok {
		block.MimeType = mime
	}
	if annotations, ok := item["annotations"].(map[string]any); ok {
		block.Annotations = copyMap(annotations)
	}
	switch data := item["data"].(type) {
	case map[string]any:
		block.Data = copyMap(data)
	case []any:
		block.Data = map[string]any{"items": data}
	case nil:
	default:
		if block.Data == nil {
			block.Data = make(map[string]any, 1)
		}
		block.Data["value"] = data
	}

	if block.Text == "" && len(block.Data) == 1 {
		if value, ok := block.Data["value"]; ok {
			block.Text = marshalAsText(value)
		}
	}

	return block, true
}

func marshalAsText(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}

func copyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	result := make(map[string]any, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

func mapWithout(source map[string]any, keys ...string) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(source))
	for k, v := range source {
		result[k] = v
	}
	for _, key := range keys {
		delete(result, key)
	}
	delete(result, "structuredContent")
	return result
}
