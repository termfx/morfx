package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/termfx/morfx/internal/lang/golang"
	"github.com/termfx/morfx/internal/lang/javascript"
	"github.com/termfx/morfx/internal/lang/python"
	"github.com/termfx/morfx/internal/lang/typescript"
	"github.com/termfx/morfx/internal/types"
)

// LanguageRegistry manages language providers with thread-safe operations
type LanguageRegistry struct {
	mu         sync.RWMutex
	providers  map[string]types.LanguageProvider
	aliases    map[string]string // alias -> canonical name
	extensions map[string]string // extension -> canonical name
}

// NewLanguageRegistry creates a new language registry
func NewLanguageRegistry() *LanguageRegistry {
	return &LanguageRegistry{
		providers:  make(map[string]types.LanguageProvider),
		aliases:    make(map[string]string),
		extensions: make(map[string]string),
	}
}

// Register adds a language provider to the registry
func (r *LanguageRegistry) Register(p types.LanguageProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lang := p.Lang()
	if lang == "" {
		return fmt.Errorf("provider must have a non-empty language name")
	}

	// Check for conflicts
	if _, exists := r.providers[lang]; exists {
		return fmt.Errorf("provider for language '%s' already registered", lang)
	}

	// Register the provider
	r.providers[lang] = p

	// Register aliases
	for _, alias := range p.Aliases() {
		if alias == "" {
			continue
		}
		if existing, exists := r.aliases[alias]; exists {
			return fmt.Errorf("alias '%s' conflicts with existing mapping to '%s'", alias, existing)
		}
		r.aliases[alias] = lang
	}

	// Register file extensions
	for _, ext := range p.Extensions() {
		if ext == "" {
			continue
		}
		// Normalize extension (ensure it starts with .)
		if ext[0] != '.' {
			ext = "." + ext
		}
		if existing, exists := r.extensions[ext]; exists {
			return fmt.Errorf("extension '%s' conflicts with existing mapping to '%s'", ext, existing)
		}
		r.extensions[ext] = lang
	}

	return nil
}

// GetProvider retrieves a provider by language name or alias
func (r *LanguageRegistry) GetProvider(name string) (types.LanguageProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct lookup
	if p, exists := r.providers[name]; exists {
		return p, nil
	}

	// Try alias lookup
	if canonical, exists := r.aliases[name]; exists {
		if p, exists := r.providers[canonical]; exists {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no provider found for language '%s'", name)
}

// GetProviderByExtension retrieves a provider by file extension
func (r *LanguageRegistry) GetProviderByExtension(ext string) (types.LanguageProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize extension
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}

	canonical, exists := r.extensions[ext]
	if !exists {
		return nil, fmt.Errorf("no provider found for extension '%s'", ext)
	}

	p, exists := r.providers[canonical]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found for extension '%s'", canonical, ext)
	}

	return p, nil
}

// ListProviders returns all registered language names
func (r *LanguageRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	languages := make([]string, 0, len(r.providers))
	for lang := range r.providers {
		languages = append(languages, lang)
	}
	return languages
}

// ListExtensions returns all registered file extensions
func (r *LanguageRegistry) ListExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extensions := make([]string, 0, len(r.extensions))
	for ext := range r.extensions {
		extensions = append(extensions, ext)
	}
	return extensions
}

// HasProvider checks if a provider is registered for the given language
func (r *LanguageRegistry) HasProvider(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check direct name
	if _, exists := r.providers[name]; exists {
		return true
	}

	// Check aliases
	if canonical, exists := r.aliases[name]; exists {
		_, exists := r.providers[canonical]
		return exists
	}

	return false
}

// Unregister removes a provider from the registry
func (r *LanguageRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find the provider
	p, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}

	// Remove the provider
	delete(r.providers, name)

	// Remove aliases
	for _, alias := range p.Aliases() {
		if r.aliases[alias] == name {
			delete(r.aliases, alias)
		}
	}

	// Remove extensions
	for _, ext := range p.Extensions() {
		if ext[0] != '.' {
			ext = "." + ext
		}
		if r.extensions[ext] == name {
			delete(r.extensions, ext)
		}
	}

	return nil
}

// Clear removes all providers from the registry
func (r *LanguageRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]types.LanguageProvider)
	r.aliases = make(map[string]string)
	r.extensions = make(map[string]string)
}

// GetProviderInfo returns detailed information about a provider
func (r *LanguageRegistry) GetProviderInfo(name string) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return &ProviderInfo{
		Name:       p.Lang(),
		Aliases:    p.Aliases(),
		Extensions: p.Extensions(),
	}, nil
}

// ProviderInfo contains metadata about a registered provider
type ProviderInfo struct {
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases"`
	Extensions []string `json:"extensions"`
}

// Global registry instance
var DefaultRegistry = NewLanguageRegistry()

func init() {
	// Register built-in providers
	DefaultRegistry.Register(golang.NewProvider())
	DefaultRegistry.Register(python.NewProvider())
	DefaultRegistry.Register(javascript.NewProvider())
	DefaultRegistry.Register(typescript.NewProvider())
}

// Convenience functions for the default registry

// Register adds a provider to the default registry
func Register(p types.LanguageProvider) error {
	return DefaultRegistry.Register(p)
}

// GetProvider retrieves a provider from the default registry
func GetProvider(name string) (types.LanguageProvider, error) {
	return DefaultRegistry.GetProvider(name)
}

// GetProviderByExtension retrieves a provider by extension from the default registry
func GetProviderByExtension(ext string) (types.LanguageProvider, error) {
	return DefaultRegistry.GetProviderByExtension(ext)
}

// ListProviders returns all providers from the default registry
func ListProviders() []string {
	return DefaultRegistry.ListProviders()
}

// HasProvider checks if a provider exists in the default registry
func HasProvider(name string) bool {
	return DefaultRegistry.HasProvider(name)
}

// AutoRegister loads external plugins
// Built-in providers are already registered in init()
func AutoRegister() error {
	// Try loading external plugins from default directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		pluginDir := filepath.Join(homeDir, ".morfx", "plugins")
		LoadPluginsFromDir(pluginDir)
	}

	return nil
}

// LoadPlugin dynamically loads a provider from a plugin file
func LoadPlugin(path string) error {
	plug, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for the Provider symbol
	symProvider, err := plug.Lookup("Provider")
	if err != nil {
		return fmt.Errorf("plugin %s missing Provider symbol: %w", path, err)
	}

	// Try to cast to LanguageProvider interface
	providerImpl, ok := symProvider.(types.LanguageProvider)
	if !ok {
		return fmt.Errorf("plugin %s Provider symbol is not a LanguageProvider", path)
	}

	// Register the provider
	return Register(providerImpl)
}

// LoadPluginsFromDir loads all plugin files from a directory
func LoadPluginsFromDir(dir string) error {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, not an error
	}

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory %s: %w", dir, err)
	}

	var errors []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check for .so files (plugins)
		name := entry.Name()
		if strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".dll") || strings.HasSuffix(name, ".dylib") {
			path := filepath.Join(dir, name)
			if err := LoadPlugin(path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
