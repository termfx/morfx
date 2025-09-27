package mcp

import (
	"context"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
	"github.com/termfx/morfx/providers"
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
func (s *StdioServer) ConfirmApply(ctx context.Context, summary string) error {
	params := map[string]any{
		"title": "Apply staged changes",
		"prompt": []map[string]any{
			{
				"role": "assistant",
				"content": []map[string]any{
					{
						"type": "text",
						"text": summary,
					},
				},
			},
		},
		"choices": []map[string]any{
			{"label": "Apply changes", "value": "confirm"},
			{"label": "Cancel", "value": "cancel"},
		},
	}

	result, err := s.RequestElicitation(ctx, params)
	if err != nil {
		return err
	}
	if result != nil {
		if choice, ok := result["choice"].(string); ok && choice == "cancel" {
			return context.Canceled
		}
	}

	return nil
}
