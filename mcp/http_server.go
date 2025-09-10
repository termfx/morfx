package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"
)

// HTTPServer handles MCP communication over HTTP
type HTTPServer struct {
	config     Config
	db         *gorm.DB
	server     *http.Server
	
	// Core MCP logic - delegate to stdio server
	core *StdioServer
	
	// Authentication
	apiKey string
	
	// Debug logging
	debugLog func(format string, args ...interface{})
}

// NewHTTPServer creates a new HTTP server that provides MCP functionality via REST API
func NewHTTPServer(config Config, host string, port int, apiKey, corsOrigin string) (*HTTPServer, error) {
	server := &HTTPServer{
		config: config,
		apiKey: apiKey,
	}
	
	// Setup debug logging
	if config.Debug {
		server.debugLog = func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, "[HTTP] "+format+"\n", args...)
		}
	} else {
		server.debugLog = func(format string, args ...interface{}) {}
	}
	
	// Create core MCP server to delegate logic to
	core, err := NewStdioServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create core MCP server: %w", err)
	}
	server.core = core
	
	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", server.handleMCPRequest)
	mux.HandleFunc("/health", server.handleHealth)
	
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
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
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

// handleMCPRequest processes MCP JSON-RPC requests via HTTP
func (s *HTTPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
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
	
	s.debugLog("HTTP request: %s with ID: %v", req.Method, req.ID)
	
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
	
	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.debugLog("Failed to encode response: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	
	s.debugLog("HTTP response sent for ID: %v", req.ID)
}

// handleHealth provides health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
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
	
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
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

// Close closes the HTTP server and cleans up resources
func (s *HTTPServer) Close() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}
	
	// Close core server resources
	if s.core != nil {
		return s.core.Close()
	}
	
	return nil
}
