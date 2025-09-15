package php

import "github.com/termfx/morfx/providers/base"

// This package provides PHP language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with PHP-specific configuration.

// New creates a PHP provider using base functionality with PHP-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
