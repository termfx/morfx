package typescript

import "github.com/termfx/morfx/providers/base"

// This package provides TypeScript language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with TypeScript-specific configuration.

// New creates a TypeScript provider using base functionality with TS-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
