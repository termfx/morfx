package mcp

import (
	"context"
	"fmt"
	"sync"
)

// RequestHandler processes a JSON-RPC request message and returns a response.
type RequestHandler func(ctx context.Context, msg RequestMessage) ResponseMessage

// NotificationHandler processes a JSON-RPC notification.
type NotificationHandler func(ctx context.Context, msg NotificationMessage) error

// Router maintains a registry of MCP request and notification handlers and
// provides centralized dispatch with JSON-RPC compliance checks.
type Router struct {
	mu                   sync.RWMutex
	requestHandlers      map[string]RequestHandler
	notificationHandlers map[string]NotificationHandler
}

// NewRouter creates an empty router instance.
func NewRouter() *Router {
	return &Router{
		requestHandlers:      make(map[string]RequestHandler),
		notificationHandlers: make(map[string]NotificationHandler),
	}
}

// RegisterRequest associates a handler with a JSON-RPC method name. Existing
// registrations are replaced.
func (r *Router) RegisterRequest(method string, handler RequestHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestHandlers[method] = handler
}

// RegisterNotification associates a notification handler with a method name.
func (r *Router) RegisterNotification(method string, handler NotificationHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notificationHandlers[method] = handler
}

// DispatchRequest routes a request message to the appropriate handler. It
// returns a JSON-RPC error response if validation fails or the method is
// unknown.
func (r *Router) DispatchRequest(ctx context.Context, msg RequestMessage) ResponseMessage {
	if err := ensureVersion(msg.JSONRPC); err != nil {
		return ErrorResponse(msg.ID, InvalidRequest, err.Error())
	}

	r.mu.RLock()
	handler, ok := r.requestHandlers[msg.Method]
	r.mu.RUnlock()
	if !ok {
		return ErrorResponse(msg.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", msg.Method))
	}

	resp := handler(ctx, msg)
	if resp.JSONRPC == "" {
		resp.JSONRPC = JSONRPCVersion
	}
	return resp
}

// DispatchNotification routes a notification message. If validation fails or
// the method is unknown the handler returns an error for logging.
func (r *Router) DispatchNotification(ctx context.Context, msg NotificationMessage) error {
	if err := ensureVersion(msg.JSONRPC); err != nil {
		return err
	}

	r.mu.RLock()
	handler, ok := r.notificationHandlers[msg.Method]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("notification handler not registered: %s", msg.Method)
	}

	return handler(ctx, msg)
}
