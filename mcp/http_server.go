package mcp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// HTTPServer handles MCP communication over HTTP
type HTTPServer struct {
	config Config
	db     *gorm.DB
	server *http.Server

	// Core MCP logic - delegate to stdio server
	core *StdioServer

	// Authentication
	apiKey         string
	noAuth         bool // Skip authentication when true
	oauthConfig    *OAuthConfig
	oauthValidator *OAuthValidator
	sessionStore   *OAuthSessionStore

	// Debug logging
	debugLog func(format string, args ...any)
}

// NewHTTPServer creates a new HTTP server that provides MCP functionality via REST API
func NewHTTPServer(
	config Config,
	host string,
	port int,
	apiKey, corsOrigin string,
	noAuth bool,
	oauthConfig *OAuthConfig,
) (*HTTPServer, error) {
	server := &HTTPServer{
		config:      config,
		apiKey:      apiKey,
		noAuth:      noAuth,
		oauthConfig: oauthConfig,
	}

	// Setup debug logging
	if config.Debug {
		server.debugLog = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "[HTTP] "+format+"\n", args...)
		}
	} else {
		server.debugLog = func(format string, args ...any) {}
	}

	// Create core MCP server to delegate logic to
	core, err := NewStdioServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create core MCP server: %w", err)
	}
	server.core = core

	// Share database connection with core server
	server.db = core.db
	if server.db == nil {
		return nil, fmt.Errorf("core server database not initialized")
	}
	server.debugLog("Database connection shared with core server")

	// Initialize OAuth validator if OAuth is enabled
	if oauthConfig != nil {
		validator, err := NewOAuthValidator(oauthConfig, config.Debug)
		if err != nil {
			return nil, fmt.Errorf("failed to create OAuth validator: %w", err)
		}
		server.oauthValidator = validator
		server.debugLog("OAuth validator initialized for provider: %s", oauthConfig.Provider)

		// Initialize OAuth session store
		server.sessionStore = NewOAuthSessionStore(5 * time.Minute) // 5min cleanup interval
		server.debugLog("OAuth session store initialized")
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", server.handleMCPEndpoint)
	mux.HandleFunc("/health", server.handleHealth)

	// OAuth endpoints (only if OAuth is configured)
	if oauthConfig != nil {
		mux.HandleFunc("/.well-known/oauth-authorization-server", server.handleAuthServerMetadata)
		mux.HandleFunc("/.well-known/oauth-protected-resource", server.handleProtectedResourceMetadata)
		mux.HandleFunc("/auth", server.handleOAuthAuth)
		mux.HandleFunc("/callback", server.handleOAuthCallback)
		mux.HandleFunc("/token", server.handleOAuthToken)
		mux.HandleFunc("/register", server.handleClientRegistration)
		mux.HandleFunc("/logout", server.handleOAuthLogout)
		server.debugLog("OAuth endpoints registered")
	}

	// Add CORS and auth middleware
	handler := server.corsMiddleware(corsOrigin, server.authMiddleware(mux))

	server.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	server.debugLog("HTTP server initialized on %s:%d", host, port)
	return server, nil
}

// Reuse existing MCP handler methods by delegating to core server
func (s *HTTPServer) handleListTools(req Request) Response {
	return s.core.handleListTools(req)
}

func (s *HTTPServer) handleCallTool(req Request) Response {
	return s.core.handleCallTool(req)
}

func (s *HTTPServer) handleInitialize(req Request) Response {
	return s.core.handleInitialize(req)
}

func (s *HTTPServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public endpoints
		publicEndpoints := []string{
			"/health",
			"/auth",
			"/callback",
			"/token",
			"/register",
			"/logout",
			"/.well-known/oauth-authorization-server",
		}
		if slices.Contains(publicEndpoints, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth if noAuth is enabled
		if s.noAuth {
			s.debugLog("Authentication disabled - proceeding without auth check")
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			// RFC9728 Section 5.1: WWW-Authenticate header MUST be included in 401 responses
			baseURL := s.getBaseURL(r)
			wwwAuth := fmt.Sprintf(
				`Bearer realm="%s", resource="%s/mcp", authorization_uri="%s/.well-known/oauth-protected-resource"`,
				baseURL,
				baseURL,
				baseURL,
			)
			w.Header().Set("WWW-Authenticate", wwwAuth)
			s.writeErrorResponse(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		// Validate Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(auth, bearerPrefix) {
			s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		token := strings.TrimPrefix(auth, bearerPrefix)

		// Try OAuth validation first if available
		if s.oauthValidator != nil {
			claims, err := s.oauthValidator.ValidateToken(token)
			if err != nil {
				s.debugLog("OAuth validation failed: %v", err)
				// RFC9728 Section 5.1: WWW-Authenticate header for OAuth failures
				baseURL := s.getBaseURL(r)
				wwwAuth := fmt.Sprintf(
					`Bearer realm="%s", resource="%s/mcp", authorization_uri="%s/.well-known/oauth-protected-resource", error="invalid_token"`,
					baseURL,
					baseURL,
					baseURL,
				)
				w.Header().Set("WWW-Authenticate", wwwAuth)
				s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid OAuth token")
				return
			}

			// Add claims to request context for later use
			ctx := context.WithValue(r.Context(), "oauth_claims", claims)
			ctx = context.WithValue(ctx, "auth_subject", s.oauthValidator.GetSubject(claims))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fallback to API key validation
		if token != s.apiKey {
			s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
func (s *HTTPServer) corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleMCPEndpoint routes MCP requests based on HTTP method and Accept header
func (s *HTTPServer) handleMCPEndpoint(w http.ResponseWriter, r *http.Request) {
	// Extract MCP protocol version if provided
	protocolVersion := r.Header.Get("MCP-Protocol-Version")
	sessionID := r.Header.Get("Mcp-Session-Id")

	if protocolVersion != "" {
		s.debugLog("MCP Protocol Version: %s", protocolVersion)
	}

	// Session management
	var mcpSession *models.MCPSession
	if sessionID != "" {
		// Lookup existing session
		err := s.db.Where("id = ? AND active = ? AND expires_at > ?", sessionID, true, time.Now()).
			First(&mcpSession).Error
		if err != nil {
			s.debugLog("Invalid or expired session: %s", sessionID)
			s.writeErrorResponse(w, http.StatusNotFound, "Invalid or expired session")
			return
		}

		// Update session activity
		s.db.Model(mcpSession).Updates(map[string]any{
			"last_activity": time.Now(),
			"request_count": gorm.Expr("request_count + 1"),
		})

		s.debugLog("Session found: %s (requests: %d)", sessionID, mcpSession.RequestCount+1)
	}

	switch r.Method {
	case "POST":
		s.handleMCPJSON(w, r, mcpSession, protocolVersion)

	case "GET":
		s.debugLog("GET /mcp endpoint check - returning capabilities")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"methods":   []string{"POST"},
			"transport": "http-jsonrpc",
		})

	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST and GET methods allowed")
	}
}

// handleMCPJSON processes MCP JSON-RPC requests via HTTP (renamed from handleMCPRequest)
func (s *HTTPServer) handleMCPJSON(
	w http.ResponseWriter,
	r *http.Request,
	mcpSession *models.MCPSession,
	protocolVersion string,
) {
	if r.Method != "POST" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method allowed")
		return
	}

	// Parse JSON-RPC request
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON-RPC request")
		return
	}

	sessionID := getSessionID(mcpSession)
	s.debugLog("HTTP JSON request: %s with ID: %v (session: %s)", req.Method, req.ID, sessionID)

	// Process request using existing MCP logic
	var response Response

	switch req.Method {
	case "tools/list":
		response = s.handleListTools(req)
	case "tools/call":
		response = s.handleCallTool(req)
	case "initialize":
		response = s.handleInitialize(req)
	default:
		response = ErrorResponse(req.ID, MethodNotFound, "Method not found: "+req.Method)
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Add session info to response headers
	if mcpSession != nil {
		w.Header().Set("Mcp-Session-Id", mcpSession.ID)
		w.Header().Set("Mcp-Request-Count", fmt.Sprintf("%d", mcpSession.RequestCount))
	}

	if protocolVersion != "" {
		w.Header().Set("MCP-Protocol-Version", protocolVersion)
	}

	// Write JSON response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.debugLog("Failed to encode response: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}

	s.debugLog("HTTP JSON response sent for ID: %v (session: %s)", req.ID, sessionID)
}

// handleHealth provides health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"database":  "connected",
	}

	// Check database connectivity via core server
	if s.core.db != nil {
		if err := s.core.db.Exec("SELECT 1").Error; err != nil {
			health["status"] = "unhealthy"
			health["database"] = "disconnected"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// writeErrorResponse writes an HTTP error response
func (s *HTTPServer) writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		s.debugLog("Starting HTTP server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for stop signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-stop:
		s.debugLog("Shutdown signal received")
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	s.debugLog("HTTP server stopped gracefully")
	return nil
}

// handleAuthServerMetadata serves OAuth 2.0 Authorization Server Metadata (RFC8414)
func (s *HTTPServer) handleAuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only GET method allowed")
		return
	}

	// Check MCP protocol version header
	protocolVersion := r.Header.Get("MCP-Protocol-Version")
	if protocolVersion != "" {
		s.debugLog("Metadata discovery for MCP protocol version: %s", protocolVersion)
	}

	baseURL := s.getBaseURL(r)

	// Build metadata response according to RFC8414
	metadata := map[string]any{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/auth",
		"token_endpoint":                        baseURL + "/token",
		"scopes_supported":                      s.oauthConfig.Scopes,
		"grant_types_supported":                 []string{"authorization_code", "client_credentials"},
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
	}

	// Add registration endpoint if we support dynamic client registration
	// TODO: implement dynamic client registration handler
	metadata["registration_endpoint"] = baseURL + "/register"

	s.debugLog("Serving authorization server metadata for base URL: %s", baseURL)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		s.debugLog("Failed to encode metadata response: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

// handleProtectedResourceMetadata serves OAuth 2.0 Protected Resource Metadata (RFC9728)
// CRITICAL for MCP - OpenAI uses this for MCP server discovery
func (s *HTTPServer) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only GET method allowed")
		return
	}

	baseURL := s.getBaseURL(r)

	// RFC9728 Protected Resource Metadata - MCP server acting as resource server
	metadata := map[string]any{
		"resource":                              baseURL + "/mcp",    // Canonical resource URI
		"authorization_servers":                 []string{baseURL},   // This server is also auth server
		"resource_documentation":                baseURL + "/docs",   // Optional documentation
		"resource_policy_uri":                   baseURL + "/policy", // Optional policy
		"scopes_supported":                      s.oauthConfig.Scopes,
		"bearer_methods_supported":              []string{"header"}, // Only Authorization: Bearer
		"resource_signing_alg_values_supported": []string{"RS256", "HS256"},
	}

	s.debugLog("Serving protected resource metadata for resource: %s", baseURL+"/mcp")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		s.debugLog("Failed to encode protected resource metadata: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

// Close closes the HTTP server and cleans up resources
func (s *HTTPServer) Close() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	// Close OAuth session store to stop cleanup goroutine
	if s.sessionStore != nil {
		s.sessionStore.Close()
		s.debugLog("OAuth session store closed")
	}

	// Close core server resources
	if s.core != nil {
		return s.core.Close()
	}

	return nil
}

// ClientRegistrationRequest represents RFC7591 client registration request
type ClientRegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	ApplicationType         string   `json:"application_type,omitempty"`
}

// ClientRegistrationResponse represents RFC7591 client registration response
type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ApplicationType         string   `json:"application_type"`
}

// handleClientRegistration implements RFC7591 Dynamic Client Registration
func (s *HTTPServer) handleClientRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method allowed")
		return
	}

	// Parse registration request
	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeClientRegistrationError(w, "invalid_request", "Invalid JSON in request body")
		return
	}

	s.debugLog("Client registration request: client_name=%s, redirect_uris=%v", req.ClientName, req.RedirectURIs)

	// Validate redirect URIs (required by RFC7591)
	if len(req.RedirectURIs) == 0 {
		s.writeClientRegistrationError(w, "invalid_redirect_uri", "At least one redirect_uri is required")
		return
	}

	// Validate redirect URIs security requirements
	for _, uri := range req.RedirectURIs {
		if err := s.validateRedirectURI(uri); err != nil {
			s.writeClientRegistrationError(w, "invalid_redirect_uri", err.Error())
			return
		}
	}

	// Set defaults for optional fields
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	tokenAuthMethod := req.TokenEndpointAuthMethod
	if tokenAuthMethod == "" {
		tokenAuthMethod = "client_secret_basic"
	}

	applicationType := req.ApplicationType
	if applicationType == "" {
		applicationType = "web"
	}

	// Generate client credentials
	clientID, err := s.generateClientID()
	if err != nil {
		s.debugLog("Failed to generate client ID: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate client credentials")
		return
	}

	clientSecret, err := s.generateClientSecret()
	if err != nil {
		s.debugLog("Failed to generate client secret: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate client credentials")
		return
	}

	// Build registration response
	response := ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientSecretExpiresAt:   0, // 0 = never expires
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		Scope:                   req.Scope,
		ClientName:              req.ClientName,
		ClientURI:               req.ClientURI,
		LogoURI:                 req.LogoURI,
		Contacts:                req.Contacts,
		TosURI:                  req.TosURI,
		PolicyURI:               req.PolicyURI,
		TokenEndpointAuthMethod: tokenAuthMethod,
		ApplicationType:         applicationType,
	}

	// Store registered client in database
	client := &models.OAuthClient{
		ID:                      generateID("cli"),
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		RedirectURIs:            mustMarshalJSON(req.RedirectURIs),
		GrantTypes:              mustMarshalJSON(grantTypes),
		ResponseTypes:           mustMarshalJSON(responseTypes),
		Scope:                   req.Scope,
		ClientName:              req.ClientName,
		ClientURI:               req.ClientURI,
		LogoURI:                 req.LogoURI,
		Contacts:                mustMarshalJSON(req.Contacts),
		TosURI:                  req.TosURI,
		PolicyURI:               req.PolicyURI,
		TokenEndpointAuthMethod: tokenAuthMethod,
		ApplicationType:         applicationType,
		SecretExpiresAt:         nil, // never expires
		Active:                  true,
		Metadata:                mustMarshalJSON(map[string]any{"user_agent": r.UserAgent()}),
	}

	if err := s.db.Create(client).Error; err != nil {
		s.debugLog("Failed to store OAuth client: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to register client")
		return
	}

	s.debugLog("Client registered successfully: ID=%s, Name=%s, DB_ID=%s", clientID, req.ClientName, client.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.debugLog("Failed to encode registration response: %v", err)
		return
	}
}

// validateRedirectURI validates redirect URI according to security requirements
func (s *HTTPServer) validateRedirectURI(uri string) error {
	// Must be HTTPS or localhost HTTP (for development)
	if !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "http://localhost") &&
		!strings.HasPrefix(uri, "http://127.0.0.1") {
		return fmt.Errorf("redirect URI must use HTTPS or be localhost for development")
	}
	return nil
}

// generateClientID creates a unique client identifier
func (s *HTTPServer) generateClientID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "mcp_" + fmt.Sprintf("%x", bytes), nil
}

// generateClientSecret creates a secure client secret
func (s *HTTPServer) generateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// writeClientRegistrationError writes RFC7591 compliant error response
func (s *HTTPServer) writeClientRegistrationError(w http.ResponseWriter, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	errorResponse := map[string]string{
		"error":             errorCode,
		"error_description": description,
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// getSessionID safely extracts session ID, returning "none" if nil
func getSessionID(session *models.MCPSession) string {
	if session == nil {
		return "none"
	}
	return session.ID
}
