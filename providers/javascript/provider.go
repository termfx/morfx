package javascript

import (
	"github.com/termfx/morfx/providers/base"
	"github.com/termfx/morfx/providers/catalog"
)

// This package provides JavaScript language support for morfx using the base provider.
// All the heavy lifting is done by the base provider with JavaScript-specific configuration.

func init() {
	catalog.Register(catalog.LanguageInfo{
		ID:         "javascript",
		Extensions: (&Config{}).Extensions(),
	})
}

// New creates a JavaScript provider using base functionality with JS-specific AST mapping
func New() *base.Provider {
	config := &Config{}
	return base.New(config)
}
