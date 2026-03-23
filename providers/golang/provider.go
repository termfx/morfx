package golang

import (
	"github.com/oxhq/morfx/providers/base"
	"github.com/oxhq/morfx/providers/catalog"
)

// This package provides Go language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with Go-specific configuration.

func init() {
	catalog.Register(catalog.LanguageInfo{
		ID:         "go",
		Extensions: (&Config{}).Extensions(),
	})
}

// New creates a Go provider using base functionality with Go-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
