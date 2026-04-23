package mcp

import (
	"context"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/mcp/types"
	"github.com/oxhq/morfx/providers"
)

// ServerInterface is now an alias to types.ServerInterface
type ServerInterface = types.ServerInterface

// Ensure StdioServer implements ServerInterface
var _ types.ServerInterface = (*StdioServer)(nil)

// GetProviders returns the provider registry
func (s *StdioServer) GetProviders() *providers.Registry {
	return s.providers
}

// GetFileProcessor returns the file processor
func (s *StdioServer) GetFileProcessor() *core.FileProcessor {
	return s.fileProcessor
}

// GetStaging returns the staging manager
func (s *StdioServer) GetStaging() any {
	if s.staging == nil {
		return nil
	}
	return s.staging
}

// GetSafety returns the safety manager
func (s *StdioServer) GetSafety() any {
	if s.safety == nil {
		return nil
	}
	return s.safety
}

// GetSessionID returns the current persistence session identifier if available.
func (s *StdioServer) GetSessionID() string {
	if s.session != nil {
		return s.session.ID
	}
	return ""
}

// ReportProgress emits a progress notification if the context carries a token.
func (s *StdioServer) ReportProgress(ctx context.Context, progress, total float64, message string) {
	if token, ok := progressTokenFromContext(ctx); ok {
		s.sendProgressNotification(token, progress, total, message)
	}
}

// ConfirmApply requests client confirmation before applying staged changes.
// If the client does not support elicitation, auto-confirms.
func (s *StdioServer) ConfirmApply(ctx context.Context, summary string) error {
	// Skip elicitation entirely — most MCP clients (Claude Desktop, Codex)
	// don't support elicitation/create and will hang waiting for a response.
	// Auto-confirm for now; re-enable once client capability detection is in place.
	s.debugLog("Auto-confirming apply: %s", summary)
	return nil
}
