package python

import "github.com/termfx/morfx/providers/base"

// This package provides Python language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with Python-specific configuration.

// New creates a Python provider using base functionality with Python-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
