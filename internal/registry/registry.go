package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"sync"

	"github.com/termfx/morfx/internal/provider"
)

// Registry manages language providers with thread-safe operations.
// This registry is completely language-agnostic and works solely through the
// LanguageProvider interface, enabling true pluggable architecture where
// language support can be added without modifying core components.
type Registry struct {
	mu         sync.RWMutex
	providers  map[string]provider.LanguageProvider // canonical name -> provider
	aliases    map[string]string                    // alias -> canonical name
	extensions map[string]string                    // extension -> canonical name
}

// NewRegistry creates a new language registry with no built-in providers.
// Providers must be registered explicitly through RegisterProvider or LoadPlugin.
// This enforces the language-agnostic design where the core has zero knowledge
// of specific languages.
func NewRegistry() *Registry {
	return &Registry{
		providers:  make(map[string]provider.LanguageProvider),
		aliases:    make(map[string]string),
		extensions: make(map[string]string),
	}
}

// RegisterProvider adds a language provider to the registry.
// This is the primary method for registering providers, whether built-in
// or loaded from plugins. The provider must implement the LanguageProvider
// interface completely.
func (r *Registry) RegisterProvider(p provider.LanguageProvider) error {
	if p == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	// Check for nil interface with non-nil type (Go interface semantics)
	if reflect.ValueOf(p).IsNil() {
		return fmt.Errorf("provider cannot be nil")
	}

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

// GetProvider retrieves a provider by language name, alias, or file extension.
// This method supports flexible provider lookup as specified in the requirements:
// - Direct language name (e.g., "go", "python")
// - Language aliases (e.g., "golang" -> "go", "js" -> "javascript")
// - File extensions (e.g., ".go" -> "go", ".py" -> "python")
func (r *Registry) GetProvider(identifier string) (provider.LanguageProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct lookup by name
	if p, exists := r.providers[identifier]; exists {
		return p, nil
	}

	// Try alias lookup
	if canonical, exists := r.aliases[identifier]; exists {
		if p, exists := r.providers[canonical]; exists {
			return p, nil
		}
	}

	// Try extension lookup (normalize the identifier)
	ext := identifier
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	if canonical, exists := r.extensions[ext]; exists {
		if p, exists := r.providers[canonical]; exists {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no provider found for identifier '%s'", identifier)
}

// GetProviderForFile retrieves a provider based on a file's extension.
// This method enables automatic language detection from filenames, which is
// essential for CLI tools that process files without explicit language specification.
func (r *Registry) GetProviderForFile(filename string) (provider.LanguageProvider, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	ext := filepath.Ext(filename)
	if ext == "" {
		return nil, fmt.Errorf("file %s has no extension", filename)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical, exists := r.extensions[ext]
	if !exists {
		return nil, fmt.Errorf("no provider found for file extension '%s'", ext)
	}

	p, exists := r.providers[canonical]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found for extension '%s'", canonical, ext)
	}

	return p, nil
}

// GetProviderByExtension retrieves a provider by file extension for backward compatibility.
// Deprecated: Use GetProviderForFile instead.
func (r *Registry) GetProviderByExtension(ext string) (provider.LanguageProvider, error) {
	if ext == "" {
		return nil, fmt.Errorf("extension cannot be empty")
	}

	// Normalize extension
	if ext[0] != '.' {
		ext = "." + ext
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

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

// ListProviders returns all registered language names.
// This provides introspection capability for CLI tools and debugging.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	languages := make([]string, 0, len(r.providers))
	for lang := range r.providers {
		languages = append(languages, lang)
	}
	return languages
}

// ListExtensions returns all registered file extensions.
// This is useful for CLI help text and validation.
func (r *Registry) ListExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extensions := make([]string, 0, len(r.extensions))
	for ext := range r.extensions {
		extensions = append(extensions, ext)
	}
	return extensions
}

// HasProvider checks if a provider is registered for the given identifier.
// The identifier can be a language name, alias, or file extension.
func (r *Registry) HasProvider(identifier string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check direct name
	if _, exists := r.providers[identifier]; exists {
		return true
	}

	// Check aliases
	if canonical, exists := r.aliases[identifier]; exists {
		_, exists := r.providers[canonical]
		return exists
	}

	// Check extensions
	ext := identifier
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	if canonical, exists := r.extensions[ext]; exists {
		_, exists := r.providers[canonical]
		return exists
	}

	return false
}

// Unregister removes a provider from the registry.
// This is primarily used for testing and dynamic provider management.
func (r *Registry) Unregister(name string) error {
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

// Clear removes all providers from the registry.
// This is primarily used for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]provider.LanguageProvider)
	r.aliases = make(map[string]string)
	r.extensions = make(map[string]string)
}

// GetProviderInfo returns detailed information about a provider.
// This is useful for debugging and CLI introspection.
func (r *Registry) GetProviderInfo(name string) (*ProviderInfo, error) {
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

// ProviderInfo contains metadata about a registered provider.
type ProviderInfo struct {
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases"`
	Extensions []string `json:"extensions"`
}

// LoadPlugin dynamically loads a provider from a .so file.
// This method enables runtime plugin loading, allowing new language support
// to be added without recompiling the main application. The plugin must export
// a symbol that implements the LanguageProvider interface.
func (r *Registry) LoadPlugin(path string) error {
	if path == "" {
		return fmt.Errorf("plugin path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", path)
	}

	// Open the plugin
	plug, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for the Provider symbol - this is the standard interface
	// that all plugin providers must export
	symProvider, err := plug.Lookup("Provider")
	if err != nil {
		return fmt.Errorf("plugin %s missing 'Provider' symbol: %w", path, err)
	}

	// Try to cast to LanguageProvider interface
	providerImpl, ok := symProvider.(provider.LanguageProvider)
	if !ok {
		return fmt.Errorf("plugin %s 'Provider' symbol is not a LanguageProvider", path)
	}

	// Register the loaded provider
	if err := r.RegisterProvider(providerImpl); err != nil {
		return fmt.Errorf("failed to register plugin %s provider: %w", path, err)
	}

	return nil
}

// LoadPluginsFromDir loads all plugin files from a directory.
// This method scans for .so, .dll, and .dylib files and attempts to load them as plugins.
// It continues loading even if some plugins fail, collecting all errors.
func (r *Registry) LoadPluginsFromDir(dir string) error {
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

		// Check for plugin files
		name := entry.Name()
		if r.isPluginFile(name) {
			path := filepath.Join(dir, name)
			if err := r.LoadPlugin(path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// isPluginFile checks if a filename represents a valid plugin file.
func (r *Registry) isPluginFile(filename string) bool {
	return strings.HasSuffix(filename, ".so") ||
		strings.HasSuffix(filename, ".dll") ||
		strings.HasSuffix(filename, ".dylib")
}

// Global registry instance for convenience.
// While the registry itself is language-agnostic, we provide a global instance
// to maintain backward compatibility with existing code that expects a default registry.
var DefaultRegistry = NewRegistry()

// Convenience functions for the default registry.
// These functions delegate to the default registry instance, maintaining the same
// API while enabling dependency injection for testing and modular design.

// RegisterProvider adds a provider to the default registry.
func RegisterProvider(p provider.LanguageProvider) error {
	return DefaultRegistry.RegisterProvider(p)
}

// GetProvider retrieves a provider from the default registry.
func GetProvider(identifier string) (provider.LanguageProvider, error) {
	return DefaultRegistry.GetProvider(identifier)
}

// LoadPlugin loads a provider plugin into the default registry.
func LoadPlugin(path string) error {
	return DefaultRegistry.LoadPlugin(path)
}

// ListProviders returns all providers from the default registry.
func ListProviders() []string {
	return DefaultRegistry.ListProviders()
}

// GetProviderForFile retrieves a provider by file extension from the default registry.
func GetProviderForFile(filename string) (provider.LanguageProvider, error) {
	return DefaultRegistry.GetProviderForFile(filename)
}

// HasProvider checks if a provider exists in the default registry.
func HasProvider(identifier string) bool {
	return DefaultRegistry.HasProvider(identifier)
}

// LoadPluginsFromDir loads plugins from a directory into the default registry.
func LoadPluginsFromDir(dir string) error {
	return DefaultRegistry.LoadPluginsFromDir(dir)
}

// AutoRegister loads external plugins from default locations.
// This function implements the auto-discovery mechanism for plugins without
// hardcoding any specific language providers. It searches standard plugin
// directories and loads any valid plugins it finds.
func AutoRegister() error {
	// Try loading external plugins from user plugin directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		pluginDir := filepath.Join(homeDir, ".morfx", "plugins")
		if err := LoadPluginsFromDir(pluginDir); err != nil {
			// Log error but don't fail - plugin loading is optional
			return fmt.Errorf("failed to load plugins from %s: %w", pluginDir, err)
		}
	}

	// Try loading from system-wide plugin directory
	systemPluginDir := "/usr/local/lib/morfx/plugins"
	if err := LoadPluginsFromDir(systemPluginDir); err != nil {
		// Log error but don't fail - plugin loading is optional
		return fmt.Errorf("failed to load plugins from %s: %w", systemPluginDir, err)
	}

	return nil
}

// DEPRECATED functions for backward compatibility

// GetProviderByExtension retrieves a provider by extension from the default registry.
// Deprecated: Use GetProviderForFile instead.
func GetProviderByExtension(ext string) (provider.LanguageProvider, error) {
	return DefaultRegistry.GetProviderForFile("file" + ext)
}

// Register is an alias for RegisterProvider for backward compatibility.
// Deprecated: Use RegisterProvider instead.
func Register(p provider.LanguageProvider) error {
	return RegisterProvider(p)
}
