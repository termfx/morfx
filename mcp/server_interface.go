package mcp

import (
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
	return s.staging
}

// GetSafety returns the safety manager
func (s *StdioServer) GetSafety() any {
	return s.safety
}
