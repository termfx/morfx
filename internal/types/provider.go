package types

import (
	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// Type aliases for core contracts to maintain compatibility
type (
	NodeMapping          = core.NodeMapping
	ProviderCapabilities = core.ProviderCapabilities
	QueryOptions         = core.QueryOptions
	QuickCheckDiagnostic = core.QuickCheckDiagnostic
)

// LanguageProvider is now an alias to the new provider.LanguageProvider interface.
// This maintains backward compatibility while the new minimal interface in
// internal/provider/contract.go becomes the primary definition.
//
// DEPRECATED: Use provider.LanguageProvider directly. This alias exists for
// backward compatibility and will be removed in a future version.
type LanguageProvider = provider.LanguageProvider
