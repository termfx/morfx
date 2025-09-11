package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"gorm.io/gorm"

	"github.com/termfx/morfx/db"
	"github.com/termfx/morfx/models"
	"github.com/termfx/morfx/providers"
	"github.com/termfx/morfx/providers/golang"
)

// StdioServer handles MCP communication over stdio
type StdioServer struct {
	config Config
	db     *gorm.DB

	reader *bufio.Reader
	writer *bufio.Writer

	// Tool registry
	tools map[string]ToolHandler
	mu    sync.RWMutex

	// Provider registry
	providers *providers.Registry

	// Session tracking
	session *models.Session

	// Staging manager
	staging *StagingManager

	// Debug logging
	debugLog func(format string, args ...any)
}

// ToolHandler represents a function that handles a tool call
type ToolHandler func(params json.RawMessage) (any, error)

// NewStdioServer creates a new MCP server that communicates over stdio
func NewStdioServer(config Config) (*StdioServer, error) {
	server := &StdioServer{
		config:    config,
		reader:    bufio.NewReader(os.Stdin),
		writer:    bufio.NewWriter(os.Stdout),
		tools:     make(map[string]ToolHandler),
		providers: providers.NewRegistry(),
	}

	// Set debug logger
	if config.Debug {
		server.debugLog = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
		}
	} else {
		server.debugLog = func(format string, args ...any) {}
	}

	// Initialize database if URL provided
	if config.DatabaseURL != "" && config.DatabaseURL != "skip" {
		database, err := db.Connect(config.DatabaseURL, config.Debug)
		if err != nil {
			// Only fail if database was explicitly requested
			if config.DatabaseURL != "postgres://localhost/morfx_dev" {
				return nil, fmt.Errorf("failed to connect to database: %w", err)
			}
			// Default URL failed, continue without database
			server.debugLog("Database connection failed, continuing without persistence: %v", err)
		} else {
			server.db = database

			// Create session
			session := &models.Session{
				ID: generateSessionID(),
			}
			if err := server.db.Create(session).Error; err != nil {
				server.debugLog("Failed to create session: %v", err)
			} else {
				server.session = session
				server.debugLog("Session created: %s", session.ID)
			}

			// Initialize staging manager
			server.staging = NewStagingManager(server.db, config)
		}
	}

	// Register built-in tools
	server.registerBuiltinTools()

	// Register providers
	server.providers.Register(golang.New())
	server.debugLog("Registered Go provider")

	return server, nil
}

// Start begins processing JSON-RPC requests from stdin
func (s *StdioServer) Start() error {
	sessionID := ""
	if s.session != nil {
		sessionID = s.session.ID
	}
	s.debugLog("MCP server started, session: %s", sessionID)

	// Use JSON decoder for streaming - handles multi-line JSON properly
	decoder := json.NewDecoder(s.reader)

	for {
		// Decode next JSON message (handles newlines, whitespace, etc)
		var req Request
		err := decoder.Decode(&req)

		if err == io.EOF {
			s.debugLog("EOF received, shutting down gracefully")
			return nil
		}

		if err != nil {
			// Check if it's a real error or just malformed JSON
			if err == io.ErrUnexpectedEOF {
				s.debugLog("Unexpected EOF, waiting for more data")
				continue
			}

			// More descriptive error for debugging
			errMsg := "Parse error"
			if syntaxErr, ok := err.(*json.SyntaxError); ok {
				errMsg = fmt.Sprintf("JSON syntax error at position %d: %v", syntaxErr.Offset, err)
			} else if typeErr, ok := err.(*json.UnmarshalTypeError); ok {
				errMsg = fmt.Sprintf("JSON type error: expected %s for field %s", typeErr.Type, typeErr.Field)
			} else {
				errMsg = fmt.Sprintf("JSON decode error: %v", err)
			}

			// Send parse error but continue running
			s.debugLog("%s", errMsg)
			s.sendResponse(ErrorResponse(nil, ParseError, errMsg))

			// Try to recover by creating a new decoder
			decoder = json.NewDecoder(s.reader)
			continue
		}

		// Log sanitized request (truncate long source code)
		reqLog := fmt.Sprintf("%v", req)
		if len(reqLog) > 200 {
			reqLog = reqLog[:200] + "..."
		}
		s.debugLog("Received: %s", reqLog)

		// Handle the request
		response := s.handleRequest(req)

		// Don't send response for notifications (no ID)
		if req.ID != nil {
			s.sendResponse(response)
		}
	}
}

// handleRequest routes requests to appropriate handlers
func (s *StdioServer) handleRequest(req Request) Response {
	s.debugLog("Handling method: %s", req.Method)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		return s.handleInitialized(req)
	case "ping":
		return s.handlePing(req)
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	case "prompts/list":
		// Return empty prompts list
		return SuccessResponse(req.ID, map[string]any{
			"prompts": []any{},
		})
	case "resources/list":
		// Return empty resources list
		return SuccessResponse(req.ID, map[string]any{
			"resources": []any{},
		})
	default:
		return ErrorResponse(req.ID, MethodNotFound,
			fmt.Sprintf("Method not found: %s", req.Method))
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

	fmt.Fprintf(s.writer, "%s\n", data)
	s.writer.Flush()
}

// RegisterTool adds a custom tool handler
func (s *StdioServer) RegisterTool(name string, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[name] = handler
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
