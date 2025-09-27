package python

import (
	"github.com/termfx/morfx/providers/base"
	"github.com/termfx/morfx/providers/catalog"
)

// This package provides Python language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with Python-specific configuration.

func init() {
	catalog.Register(catalog.LanguageInfo{
		ID:         "python",
		Extensions: (&Config{}).Extensions(),
	})
}

// New creates a Python provider using base functionality with Python-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
