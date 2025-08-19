package main

import (
	"fmt"

	"github.com/termfx/morfx/internal/lang/golang"
	"github.com/termfx/morfx/internal/lang/javascript"
	"github.com/termfx/morfx/internal/lang/python"
	"github.com/termfx/morfx/internal/lang/typescript"
	"github.com/termfx/morfx/internal/provider"
	"github.com/termfx/morfx/internal/registry"
)

// registerBuiltinProviders registers all built-in language providers with the default registry.
// This function maintains the language-agnostic design by registering providers without
// the core knowing about specific languages. The registry handles all provider management.
func registerBuiltinProviders() error {
	// Register built-in providers
	providers := []func() provider.LanguageProvider{
		golang.NewProvider,
		python.NewProvider,
		javascript.NewProvider,
		typescript.NewProvider,
	}

	for _, providerFactory := range providers {
		p := providerFactory()
		if err := registry.RegisterProvider(p); err != nil {
			return fmt.Errorf("failed to register %s provider: %w", p.Lang(), err)
		}
	}

	return nil
}

// initializeRegistry sets up the registry with built-in providers and attempts to load
// external plugins. This function implements the auto-discovery mechanism specified
// in the architecture requirements.
func initializeRegistry() error {
	// Register built-in providers
	if err := registerBuiltinProviders(); err != nil {
		return fmt.Errorf("failed to register built-in providers: %w", err)
	}

	// Attempt to load external plugins (non-fatal if it fails)
	if err := registry.AutoRegister(); err != nil {
		// Log but don't fail - external plugins are optional
		fmt.Printf("Warning: failed to load external plugins: %v\n", err)
	}

	return nil
}
