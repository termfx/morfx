package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/db"
	"github.com/termfx/morfx/mcp/prompts"
	"github.com/termfx/morfx/mcp/resources"
	"github.com/termfx/morfx/mcp/tools"
	"github.com/termfx/morfx/mcp/types"
	"github.com/termfx/morfx/models"
	"github.com/termfx/morfx/providers"
	"github.com/termfx/morfx/providers/golang"
	"github.com/termfx/morfx/providers/javascript"
	"github.com/termfx/morfx/providers/php"
	"github.com/termfx/morfx/providers/python"
	"github.com/termfx/morfx/providers/typescript"
)

// StdioServer handles MCP communication over stdio
type StdioServer struct {
	config Config
	db     *gorm.DB

	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex

	// Modular registries
	toolRegistry             *ToolRegistry
	promptRegistry           *PromptRegistry
	resourceRegistry         *ResourceRegistry
	resourceTemplateRegistry *ResourceTemplateRegistry
	router                   *Router
	sessionState             *SessionState

	// Provider registry
	providers *providers.Registry

	// File processor for filesystem operations
	fileProcessor *core.FileProcessor

	// Session tracking
	session *models.Session

	// Staging manager
	staging *StagingManager

	// Safety manager
	safety *SafetyManager

	// Debug logging
	debugLog func(format string, args ...any)

	// Pending request tracking for server-initiated workflows
	pendingMu sync.Mutex
	pending   map[string]chan ResponseMessage
	idCounter atomic.Uint64

	// In-flight request cancellation tracking
	inflightMu      sync.Mutex
	inflightCancels map[string]context.CancelFunc

	// Resource subscription management
	resourceSubsMu    sync.Mutex
	resourceSubs      map[string]map[string]*resourceSubscription
	resourceSubIDSeed atomic.Uint64

	inboundCount  atomic.Int64
	outboundCount atomic.Int64
}

type resourceSubscription struct {
	cancel context.CancelFunc
}

// MCPMetrics captures lightweight counters for observability.
type MCPMetrics struct {
	InboundMessages  int64 `json:"inbound_messages"`
	OutboundMessages int64 `json:"outbound_messages"`
	PendingRequests  int   `json:"pending_requests"`
}

// NewStdioServer creates a new MCP server that communicates over stdio
func NewStdioServer(config Config) (*StdioServer, error) {
	server := &StdioServer{
		config:          config,
		reader:          bufio.NewReader(os.Stdin),
		writer:          bufio.NewWriter(os.Stdout),
		providers:       providers.NewRegistry(),
		router:          NewRouter(),
		sessionState:    NewSessionState(),
		pending:         make(map[string]chan ResponseMessage),
		inflightCancels: make(map[string]context.CancelFunc),
		resourceSubs:    make(map[string]map[string]*resourceSubscription),
	}

	// Initialize modular registries
	server.toolRegistry = NewToolRegistry(server)
	server.promptRegistry = NewPromptRegistry()
	server.resourceRegistry = NewResourceRegistry()
	server.resourceTemplateRegistry = NewResourceTemplateRegistry()

	// Set debug logger
	logWriter := config.LogWriter
	if logWriter == nil {
		logWriter = os.Stderr
	}

	if config.Debug {
		server.debugLog = func(format string, args ...any) {
			fmt.Fprintf(logWriter, "[DEBUG] "+format+"\n", args...)
		}
	} else {
		server.debugLog = func(format string, args ...any) {}
	}

	// Initialize database if URL provided
	if config.DatabaseURL != "" && config.DatabaseURL != "skip" {
		database, err := db.Connect(config.DatabaseURL, config.Debug)
		if err != nil {
			// Log the error but continue without database for better compatibility
			server.debugLog("Database connection failed, continuing without persistence: %v", err)
			// Don't fail the server initialization - just continue without database features
			// Explicitly set these to nil to ensure clean state
			server.db = nil
			server.session = nil
			server.staging = nil
		} else {
			server.db = database

			// Create session
			session := &models.Session{
				ID: generateSessionID(),
			}
			if err := server.db.Create(session).Error; err != nil {
				server.debugLog("Failed to create session: %v", err)
				server.session = nil
			} else {
				server.session = session
				server.debugLog("Session created: %s", session.ID)
			}

			// Staging manager will be initialized after safety setup
		}
	}

	// NEW: Initialize modular registries
	tools.Init(server)
	prompts.Init()
	resources.Init()
	registerDefaultResourceTemplates(server.resourceTemplateRegistry)
	server.debugLog("Initialized modular components")

	// Mirror global tool registrations into the server registry for legacy handlers
	for _, tool := range tools.Registry.List() {
		server.toolRegistry.Register(tool.Name(), tool)
	}

	// Register providers
	server.providers.Register(golang.New())
	server.debugLog("Registered Go provider")

	server.providers.Register(javascript.New())
	server.debugLog("Registered JavaScript provider")

	server.providers.Register(typescript.New())
	server.debugLog("Registered TypeScript provider")

	server.providers.Register(php.New())
	server.debugLog("Registered PHP provider")

	server.providers.Register(python.New())
	server.debugLog("Registered Python provider")

	// Register dynamic spec resource via standard resources endpoint
	resources.Registry.Register("config://query-spec", resources.NewDynamicResource(
		"Query Spec",
		"Supported languages and query types",
		"config://query-spec",
		"application/json",
		func() (string, error) {
			// Build JSON spec dynamically
			type langSpec struct {
				Language   string   `json:"language"`
				Extensions []string `json:"extensions"`
				Types      []string `json:"types"`
			}
			providersList := server.providers.Languages()
			// sort is not critical; keep order short
			var specs []langSpec
			for _, lang := range providersList {
				if p, ok := server.providers.Get(lang); ok {
					specs = append(specs, langSpec{
						Language:   p.Language(),
						Extensions: p.Extensions(),
						Types:      p.SupportedQueryTypes(),
					})
				}
			}
			b, _ := json.MarshalIndent(specs, "", "  ")
			return string(b), nil
		},
	))

	// Initialize file processor with providers
	server.fileProcessor = core.NewFileProcessor(&providerRegistryAdapter{server.providers})
	server.debugLog("Initialized file processor")

	// Initialize safety manager
	server.safety = NewSafetyManager(config.Safety)
	server.debugLog("Initialized safety manager")
	server.fileProcessor.SetSafety(newFileSafetyAdapter(server.safety))

	// Initialize staging manager once safety is available
	if server.db != nil {
		server.staging = NewStagingManager(server.db, config, server.safety)
		server.debugLog("Initialized staging manager")
	}

	// Register request/notification handlers with the router
	server.registerHandlers()

	return server, nil
}

// providerRegistryAdapter adapts providers.Registry to core.ProviderRegistry
type providerRegistryAdapter struct {
	*providers.Registry
}

func (pra *providerRegistryAdapter) Get(language string) (core.Provider, bool) {
	provider, exists := pra.Registry.Get(language)
	if !exists {
		return nil, false
	}
	return &providerAdapter{provider}, true
}

// providerAdapter adapts providers.Provider to core.Provider
type providerAdapter struct {
	providers.Provider
}

func (pa *providerAdapter) Language() string {
	return pa.Provider.Language()
}

func (pa *providerAdapter) Query(source string, query core.AgentQuery) core.QueryResult {
	return pa.Provider.Query(source, query)
}

func (pa *providerAdapter) Transform(source string, op core.TransformOp) core.TransformResult {
	return pa.Provider.Transform(source, op)
}

func (s *StdioServer) registerHandlers() {
	s.router.RegisterRequest("initialize", s.wrapRequestHandler(s.handleInitialize))
	s.router.RegisterRequest("initialized", s.wrapRequestHandler(s.handleInitialized))
	s.router.RegisterNotification("notifications/initialized", s.wrapNotificationHandler(s.handleInitialized))
	s.router.RegisterRequest("ping", s.wrapRequestHandler(s.handlePing))
	s.router.RegisterRequest("tools/list", s.wrapRequestHandler(s.handleListTools))
	s.router.RegisterRequest("tools/call", s.wrapRequestHandler(s.handleCallTool))
	s.router.RegisterRequest("prompts/list", s.wrapRequestHandler(s.handleListPrompts))
	s.router.RegisterRequest("prompts/get", s.wrapRequestHandler(s.handleGetPrompt))
	s.router.RegisterRequest("resources/list", s.wrapRequestHandler(s.handleListResources))
	s.router.RegisterRequest("resources/read", s.wrapRequestHandler(s.handleReadResource))
	s.router.RegisterRequest("resources/templates/list", s.wrapRequestHandler(s.handleListResourceTemplates))
	s.router.RegisterRequest("resources/subscribe", s.wrapRequestHandler(s.handleSubscribeResource))
	s.router.RegisterRequest("resources/unsubscribe", s.wrapRequestHandler(s.handleUnsubscribeResource))
	s.router.RegisterRequest("logging/setLevel", s.wrapRequestHandler(s.handleSetLoggingLevel))
	s.router.RegisterNotification("notifications/cancelled", s.handleCancelledNotification)
}

func (s *StdioServer) wrapRequestHandler(fn func(context.Context, Request) Response) RequestHandler {
	return func(ctx context.Context, msg RequestMessage) ResponseMessage {
		reqID := stringifyID(msg.ID)
		progressToken, _ := msg.Meta.ProgressToken()
		keys := []string{reqID}
		if progressToken != "" {
			keys = append(keys, progressToken)
		}

		reqCtx, cancel := context.WithCancel(ctx)
		if progressToken != "" {
			reqCtx = withProgressToken(reqCtx, progressToken)
		}
		if len(keys) > 0 {
			s.registerCancellation(cancel, keys...)
			defer func() {
				s.clearCancellation(keys...)
				cancel()
			}()
		} else {
			defer cancel()
		}

		return fn(reqCtx, msg)
	}
}

func (s *StdioServer) wrapNotificationHandler(fn func(context.Context, Request) Response) NotificationHandler {
	return func(ctx context.Context, msg NotificationMessage) error {
		req := RequestMessage{
			JSONRPC: msg.JSONRPC,
			Meta:    msg.Meta,
			Method:  msg.Method,
			Params:  msg.Params,
		}
		fn(ctx, req)
		return nil
	}
}

// handleRequest is retained for tests that exercise legacy routing behaviour.
// It delegates to the router using a background context.
func (s *StdioServer) handleRequest(req Request) Response {
	return s.router.DispatchRequest(context.Background(), req)
}

// Start begins processing JSON-RPC requests from stdin
func (s *StdioServer) Start() error {
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	s.debugLog("MCP server started, session: %s", sessionID)

	decoder := json.NewDecoder(s.reader)

	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err == io.EOF {
			s.debugLog("EOF received, shutting down gracefully")
			return nil
		}
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				s.debugLog("Unexpected EOF, waiting for more data")
				continue
			}
			s.debugLog("JSON decode error: %v", err)
			s.sendResponse(ErrorResponse(nil, ParseError, err.Error()))
			decoder = json.NewDecoder(s.reader)
			continue
		}

		rawStr := string(raw)
		if len(rawStr) > 512 {
			rawStr = rawStr[:512] + "..."
		}
		s.debugLog("Received: %s", rawStr)
		s.inboundCount.Add(1)

		var envelope struct {
			JSONRPC string           `json:"jsonrpc"`
			ID      *json.RawMessage `json:"id"`
			Method  string           `json:"method"`
		}

		if err := json.Unmarshal(raw, &envelope); err != nil {
			s.debugLog("Failed to parse message envelope: %v", err)
			s.sendResponse(ErrorResponse(nil, ParseError, "Invalid JSON-RPC message"))
			continue
		}

		ctx := context.Background()

		if envelope.ID != nil {
			if envelope.Method == "" {
				var resp ResponseMessage
				if err := json.Unmarshal(raw, &resp); err != nil {
					s.debugLog("Failed to parse response: %v", err)
					continue
				}
				if !s.resolvePendingResponse(resp) {
					s.debugLog("No pending request for response id: %v", resp.ID)
				}
				continue
			}

			var req RequestMessage
			if err := json.Unmarshal(raw, &req); err != nil {
				s.debugLog("Failed to parse request: %v", err)
				s.sendResponse(ErrorResponse(nil, ParseError, "Invalid request"))
				continue
			}
			ctx := context.WithValue(ctx, requestContextKey, req)
			resp := s.router.DispatchRequest(ctx, req)
			s.sendResponse(resp)
			continue
		}

		var note NotificationMessage
		if err := json.Unmarshal(raw, &note); err != nil {
			s.debugLog("Failed to parse notification: %v", err)
			continue
		}
		if err := s.router.DispatchNotification(ctx, note); err != nil {
			s.debugLog("Notification dispatch error: %v", err)
		}
	}
}

// sendResponse writes a response to stdout
func (s *StdioServer) sendResponse(resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		s.debugLog("Failed to marshal response: %v", err)
		return
	}

	s.debugLog("Sending: %s", string(data))
	s.writeFrame(data)
}

func (s *StdioServer) sendRequestMessage(req RequestMessage) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	if len(data) > 512 {
		s.debugLog("Sending request %v...", string(data[:512]))
	} else {
		s.debugLog("Sending request: %s", string(data))
	}

	s.writeFrame(data)
	return nil
}

func (s *StdioServer) writeFrame(data []byte) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	fmt.Fprintf(s.writer, "%s\n", data)
	s.outboundCount.Add(1)
	_ = s.writer.Flush()
}

func (s *StdioServer) resolvePendingResponse(resp ResponseMessage) bool {
	id := stringifyID(resp.ID)
	if id == "" {
		return false
	}

	s.pendingMu.Lock()
	ch, ok := s.pending[id]
	if ok {
		delete(s.pending, id)
	}
	s.pendingMu.Unlock()

	if ok {
		ch <- resp
		close(ch)
	}
	return ok
}

// Metrics exposes lightweight counters useful for debugging and observability.
func (s *StdioServer) Metrics() MCPMetrics {
	s.pendingMu.Lock()
	pending := len(s.pending)
	s.pendingMu.Unlock()

	return MCPMetrics{
		InboundMessages:  s.inboundCount.Load(),
		OutboundMessages: s.outboundCount.Load(),
		PendingRequests:  pending,
	}
}

func stringifyID(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (s *StdioServer) registerCancellation(cancel context.CancelFunc, keys ...string) {
	if cancel == nil {
		return
	}
	s.inflightMu.Lock()
	for _, key := range keys {
		if key == "" {
			continue
		}
		s.inflightCancels[key] = cancel
	}
	s.inflightMu.Unlock()
}

func (s *StdioServer) clearCancellation(keys ...string) {
	s.inflightMu.Lock()
	for _, key := range keys {
		if key == "" {
			continue
		}
		delete(s.inflightCancels, key)
	}
	s.inflightMu.Unlock()
}

func (s *StdioServer) cancelByKey(key string) bool {
	if key == "" {
		return false
	}

	s.inflightMu.Lock()
	cancel, ok := s.inflightCancels[key]
	if ok {
		delete(s.inflightCancels, key)
	}
	s.inflightMu.Unlock()

	if ok && cancel != nil {
		cancel()
	}
	return ok
}

// callClient sends a JSON-RPC request to the client and waits for a response.
func (s *StdioServer) callClient(ctx context.Context, method string, params any, meta Meta) (ResponseMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	id := fmt.Sprintf("srv-%d", s.idCounter.Add(1))
	resultCh := make(chan ResponseMessage, 1)

	s.pendingMu.Lock()
	s.pending[id] = resultCh
	s.pendingMu.Unlock()

	req, err := NewRequestMessage(id, method, params)
	if err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return ResponseMessage{}, err
	}
	req.Meta = meta
	progressToken, _ := meta.ProgressToken()

	if err := s.sendRequestMessage(req); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return ResponseMessage{}, err
	}

	select {
	case resp := <-resultCh:
		return resp, nil
	case <-ctx.Done():
		s.pendingMu.Lock()
		if ch, ok := s.pending[id]; ok {
			delete(s.pending, id)
			close(ch)
		}
		s.pendingMu.Unlock()
		s.sendCancelledNotification(id, progressToken)
		return ResponseMessage{}, ctx.Err()
	}
}

// RequestSamplingMessage asks the client to generate a sampling message per MCP spec.
func (s *StdioServer) RequestSamplingMessage(ctx context.Context, params any, meta Meta) (ResponseMessage, error) {
	return s.callClient(ctx, "sampling/createMessage", params, meta)
}

// RequestSampling requests a sampling message from the client and records it.
func (s *StdioServer) RequestSampling(ctx context.Context, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = make(map[string]any)
	}
	meta := Meta{}
	if token, ok := progressTokenFromContext(ctx); ok {
		meta = meta.WithProgressToken(token)
	}

	resp, err := s.RequestSamplingMessage(ctx, params, meta)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		if resp.Error.Code == MethodNotFound {
			s.debugLog("Client does not support sampling/createMessage; skipping")
			return nil, nil
		}
		return nil, fmt.Errorf("sampling error: %s", resp.Error.Message)
	}

	result := normalizeResponseMap(resp.Result)
	s.sessionState.AppendSamplingRecord(params, result)
	return result, nil
}

// sendElicitationRequest triggers an elicitation/create workflow on the client.
func (s *StdioServer) sendElicitationRequest(ctx context.Context, params any, meta Meta) (ResponseMessage, error) {
	return s.callClient(ctx, "elicitation/create", params, meta)
}

// RequestElicitation requests an elicitation flow and records the interaction.
func (s *StdioServer) RequestElicitation(ctx context.Context, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = make(map[string]any)
	}
	meta := Meta{}
	if token, ok := progressTokenFromContext(ctx); ok {
		meta = meta.WithProgressToken(token)
	}

	resp, err := s.sendElicitationRequest(ctx, params, meta)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		if resp.Error.Code == MethodNotFound {
			s.debugLog("Client does not support elicitation/create; continuing without confirmation")
			return nil, nil
		}
		return nil, fmt.Errorf("elicitation error: %s", resp.Error.Message)
	}

	result := normalizeResponseMap(resp.Result)
	s.sessionState.AppendElicitationRecord(params, result)
	return result, nil
}

// RequestRoots negotiates shared roots/list with the client.
func (s *StdioServer) RequestRoots(ctx context.Context, params any, meta Meta) (ResponseMessage, error) {
	return s.callClient(ctx, "roots/list", params, meta)
}

func (s *StdioServer) handleCancelledNotification(ctx context.Context, msg NotificationMessage) error {
	var params struct {
		RequestID     string `json:"requestId,omitempty"`
		ProgressToken string `json:"progressToken,omitempty"`
	}

	if len(msg.Params) > 0 {
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			s.debugLog("Invalid cancellation payload: %v", err)
			return nil
		}
	}

	handled := false
	if params.ProgressToken != "" {
		handled = s.cancelByKey(params.ProgressToken) || handled
	}
	if params.RequestID != "" {
		handled = s.cancelByKey(params.RequestID) || handled
	}

	if !handled {
		s.debugLog("Cancellation received for unknown key: token=%s id=%s", params.ProgressToken, params.RequestID)
	}

	return nil
}

// RegisterTool adds or overrides a tool handler in the modular registry.
// Primarily used in tests to stub tool behavior.
func (s *StdioServer) RegisterTool(name string, handler types.ToolHandler) {
	customTool := tools.NewTool(name).
		WithDescription("Custom tool registered at runtime").
		WithInputSchema(map[string]any{"type": "object"}).
		WithHandler(handler).
		Build()

	s.toolRegistry.Register(name, customTool)
}

// RegisterResource adds or overrides a resource in the server registry.
func (s *StdioServer) RegisterResource(resource types.Resource) {
	if resource == nil {
		return
	}
	s.resourceRegistry.Register(resource.URI(), resource)
	s.sendResourceListChangedNotification()
}

// Close cleans up resources
func (s *StdioServer) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func normalizeResponseMap(result any) map[string]any {
	if result == nil {
		return nil
	}
	if existing, ok := result.(map[string]any); ok {
		return existing
	}
	if raw, err := json.Marshal(result); err == nil {
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err == nil {
			return decoded
		}
	}
	return map[string]any{"value": result}
}

func (s *StdioServer) addResourceSubscription(uri string, cancel context.CancelFunc) string {
	if cancel == nil {
		return ""
	}
	id := fmt.Sprintf("res-sub-%d", s.resourceSubIDSeed.Add(1))
	s.resourceSubsMu.Lock()
	if _, ok := s.resourceSubs[uri]; !ok {
		s.resourceSubs[uri] = make(map[string]*resourceSubscription)
	}
	s.resourceSubs[uri][id] = &resourceSubscription{cancel: cancel}
	s.resourceSubsMu.Unlock()
	return id
}

func (s *StdioServer) removeResourceSubscription(uri, id string) {
	if id == "" {
		return
	}
	s.resourceSubsMu.Lock()
	defer s.resourceSubsMu.Unlock()
	if subs, ok := s.resourceSubs[uri]; ok {
		delete(subs, id)
		if len(subs) == 0 {
			delete(s.resourceSubs, uri)
		}
	}
}

func (s *StdioServer) cancelResourceSubscription(uri, id string) {
	s.resourceSubsMu.Lock()
	sub, ok := s.resourceSubs[uri][id]
	if ok {
		delete(s.resourceSubs[uri], id)
		if len(s.resourceSubs[uri]) == 0 {
			delete(s.resourceSubs, uri)
		}
	}
	s.resourceSubsMu.Unlock()
	if ok && sub != nil && sub.cancel != nil {
		sub.cancel()
	}
}

func (s *StdioServer) cancelAllResourceSubscriptions(uri string) {
	s.resourceSubsMu.Lock()
	subs := s.resourceSubs[uri]
	delete(s.resourceSubs, uri)
	s.resourceSubsMu.Unlock()
	for _, sub := range subs {
		if sub != nil && sub.cancel != nil {
			sub.cancel()
		}
	}
}

func (s *StdioServer) forwardResourceUpdates(uri, id string, updates <-chan types.ResourceUpdate, cancel context.CancelFunc) {
	defer func() {
		if cancel != nil {
			cancel()
		}
		s.removeResourceSubscription(uri, id)
	}()

	if updates == nil {
		s.debugLog("resource %s watcher returned nil channel", uri)
		return
	}

	for update := range updates {
		target := update.URI
		if target == "" {
			target = uri
		}
		switch update.Type {
		case types.ResourceUpdateTypeListChanged:
			s.sendResourceListChangedNotification()
		case types.ResourceUpdateTypeRemoved:
			if target != "" {
				s.sendResourceUpdatedNotification(target)
			}
			s.sendResourceListChangedNotification()
		default:
			if target != "" {
				s.sendResourceUpdatedNotification(target)
			}
		}
	}
}

type contextKey string

const requestContextKey contextKey = "mcp_request_message"
