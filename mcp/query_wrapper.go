package mcp

import (
	"context"
	"encoding/json"

	"github.com/termfx/morfx/mcp/types"
)

// handleQueryTool delegates to the modular tool registry so tests and
// legacy call-sites use the same implementation as the new tool system.
func (s *StdioServer) handleQueryTool(ctx context.Context, params json.RawMessage) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := s.toolRegistry.Execute(ctx, "query", params)
	if err != nil {
		if mcpErr, ok := err.(*types.MCPError); ok {
			return nil, NewMCPError(mcpErr.Code, mcpErr.Message, mcpErr.Data)
		}
		return nil, err
	}
	return result, nil
}
