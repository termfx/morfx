package registry

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/provider"
)

// MockLanguageProvider implements the LanguageProvider interface for testing
type MockLanguageProvider struct {
	provider.BaseProvider
	lang       string
	aliases    []string
	extensions []string
	language   *sitter.Language
}

func (m *MockLanguageProvider) Lang() string {
	return m.lang
}

func (m *MockLanguageProvider) Aliases() []string {
	return m.aliases
}

func (m *MockLanguageProvider) Extensions() []string {
	return m.extensions
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return m.language
}

func (m *MockLanguageProvider) TranslateKind(kind core.NodeKind) []core.NodeMapping {
	return []core.NodeMapping{}
}

func (m *MockLanguageProvider) TranslateQuery(q *core.Query) (string, error) {
	return "(function_declaration)", nil
}

func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
	return "function"
}

func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string {
	return "mockFunction"
}

func (m *MockLanguageProvider) OptimizeQuery(q *core.Query) *core.Query {
	return q
}

func (m *MockLanguageProvider) EstimateQueryCost(q *core.Query) int {
	return 1
}

func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
	return "file"
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node {
	return node
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
	return core.NodeKind(dslKind)
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Error("Expected non-nil registry")
	}

	if len(registry.providers) != 0 {
		t.Error("Expected empty registry")
	}

	// Test that aliases and extensions are cleaned up
	_, err := registry.GetProvider("golang")
	if err == nil {
		t.Error("Expected error when getting provider by alias after clear")
	}

	_, err = registry.GetProviderByExtension(".go")
	if err == nil {
		t.Error("Expected error when getting provider by extension after clear")
	}
}

func TestRegisterProvider(t *testing.T) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	err := registry.RegisterProvider(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify provider is registered
	if len(registry.providers) != 1 {
		t.Error("Expected 1 provider")
	}

	// Test that provider can be retrieved by alias
	_, err = registry.GetProvider("golang")
	if err != nil {
		t.Errorf("Expected to find provider by alias, got error: %v", err)
	}

	// Test that provider can be retrieved by extension
	_, err = registry.GetProviderForFile("test.go")
	if err != nil {
		t.Errorf("Expected to find provider by extension, got error: %v", err)
	}
}

func TestRegisterProviderDuplicate(t *testing.T) {
	registry := NewRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "go", // Same language
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register first provider
	err := registry.RegisterProvider(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Try to register duplicate
	err = registry.RegisterProvider(mockProvider2)
	if err == nil {
		t.Error("Expected error for duplicate provider")
	}
}

func TestGetProvider(t *testing.T) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register provider
	err := registry.RegisterProvider(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test getting by language name
	provider, err := registry.GetProvider("go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting by alias
	provider, err = registry.GetProvider("golang")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting non-existent provider
	_, err = registry.GetProvider("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestGetProviderForFile(t *testing.T) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go", ".mod"},
	}

	// Register provider
	err := registry.RegisterProvider(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test getting by extension
	provider, err := registry.GetProviderForFile("main.go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting by another extension
	provider, err = registry.GetProviderForFile("go.mod")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting non-existent extension
	_, err = registry.GetProviderForFile("test.unknown")
	if err == nil {
		t.Error("Expected error for non-existent extension")
	}
}

func TestListProviders(t *testing.T) {
	registry := NewRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "python",
		aliases:    []string{"py"},
		extensions: []string{".py"},
	}

	// Initially empty
	providers := registry.ListProviders()
	if len(providers) != 0 {
		t.Error("Expected empty provider list")
	}

	// Register providers
	err := registry.RegisterProvider(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = registry.RegisterProvider(mockProvider2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check list
	providers = registry.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Verify provider names
	names := make(map[string]bool)
	for _, providerName := range providers {
		names[providerName] = true
	}

	if !names["go"] || !names["python"] {
		t.Error("Expected both 'go' and 'python' providers")
	}
}

func TestUnregisterProvider(t *testing.T) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register provider
	err := registry.RegisterProvider(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify it's registered
	if len(registry.providers) != 1 {
		t.Error("Expected 1 provider")
	}

	// Unregister
	err = registry.Unregister("go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify it's removed
	if len(registry.providers) != 0 {
		t.Error("Expected 0 providers")
	}

	// Test that aliases and extensions are cleaned up
	_, err = registry.GetProvider("golang")
	if err == nil {
		t.Error("Expected error when getting provider by alias after unregister")
	}

	_, err = registry.GetProviderForFile("main.go")
	if err == nil {
		t.Error("Expected error when getting provider by extension after unregister")
	}

	// Try to unregister non-existent provider
	err = registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestGetSupportedExtensions(t *testing.T) {
	registry := NewRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go", ".mod"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "python",
		aliases:    []string{"py"},
		extensions: []string{".py", ".pyx"},
	}

	// Register providers
	err := registry.RegisterProvider(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = registry.RegisterProvider(mockProvider2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Get supported extensions
	extensions := registry.ListExtensions()
	if len(extensions) != 4 {
		t.Errorf("Expected 4 extensions, got %d", len(extensions))
	}

	// Verify all extensions are present
	expected := map[string]bool{
		".go":  true,
		".mod": true,
		".py":  true,
		".pyx": true,
	}

	for _, ext := range extensions {
		if !expected[ext] {
			t.Errorf("Unexpected extension: %s", ext)
		}
		delete(expected, ext)
	}

	if len(expected) > 0 {
		t.Errorf("Missing extensions: %v", expected)
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registration and access
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mockProvider := &MockLanguageProvider{
				lang:       fmt.Sprintf("lang%d", id),
				aliases:    []string{fmt.Sprintf("alias%d", id)},
				extensions: []string{fmt.Sprintf(".ext%d", id)},
			}
			_ = registry.RegisterProvider(mockProvider)
			_, _ = registry.GetProvider(fmt.Sprintf("lang%d", id))
		}(i)
	}
	wg.Wait()

	// Verify all providers were registered
	providers := registry.ListProviders()
	if len(providers) != 10 {
		t.Errorf("Expected 10 providers, got %d", len(providers))
	}
}

// Benchmark tests
func BenchmarkRegisterProvider(b *testing.B) {
	registry := NewRegistry()

	for i := 0; b.Loop(); i++ {
		mockProvider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}
		_ = registry.RegisterProvider(mockProvider)
		_ = registry.Unregister(fmt.Sprintf("lang%d", i)) // Clean up
	}
}

func BenchmarkGetProvider(b *testing.B) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	_ = registry.RegisterProvider(mockProvider)

	for b.Loop() {
		_, _ = registry.GetProvider("go")
	}
}

func BenchmarkGetProviderForFile(b *testing.B) {
	registry := NewRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	_ = registry.RegisterProvider(mockProvider)

	for b.Loop() {
		_, _ = registry.GetProviderForFile("main.go")
	}
}

// Test comprehensive registry functionality and edge cases
func TestRegistry_ProviderRegistrationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Registry) error
		provider *MockLanguageProvider
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "register valid provider",
			setup:    func(r *Registry) error { return nil },
			provider: &MockLanguageProvider{lang: "test", aliases: []string{"test-alias"}, extensions: []string{".test"}},
			wantErr:  false,
		},
		{
			name:     "register nil provider",
			setup:    func(r *Registry) error { return nil },
			provider: nil,
			wantErr:  true,
			errMsg:   "provider cannot be nil",
		},
		{
			name:     "register provider with empty language",
			setup:    func(r *Registry) error { return nil },
			provider: &MockLanguageProvider{lang: "", aliases: []string{}, extensions: []string{}},
			wantErr:  true,
			errMsg:   "must have a non-empty language name",
		},
		{
			name: "register duplicate provider",
			setup: func(r *Registry) error {
				return r.RegisterProvider(&MockLanguageProvider{lang: "duplicate", aliases: []string{}, extensions: []string{}})
			},
			provider: &MockLanguageProvider{lang: "duplicate", aliases: []string{}, extensions: []string{}},
			wantErr:  true,
			errMsg:   "already registered",
		},
		{
			name: "register provider with conflicting alias",
			setup: func(r *Registry) error {
				return r.RegisterProvider(&MockLanguageProvider{lang: "first", aliases: []string{"conflict"}, extensions: []string{}})
			},
			provider: &MockLanguageProvider{lang: "second", aliases: []string{"conflict"}, extensions: []string{}},
			wantErr:  true,
			errMsg:   "conflicts with existing mapping",
		},
		{
			name: "register provider with conflicting extension",
			setup: func(r *Registry) error {
				return r.RegisterProvider(&MockLanguageProvider{lang: "first", aliases: []string{}, extensions: []string{".conflict"}})
			},
			provider: &MockLanguageProvider{lang: "second", aliases: []string{}, extensions: []string{".conflict"}},
			wantErr:  true,
			errMsg:   "conflicts with existing mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			// Setup
			if tt.setup != nil {
				err := tt.setup(registry)
				if err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Test
			err := registry.RegisterProvider(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Test provider retrieval with various identifier types
func TestRegistry_ProviderRetrieval(t *testing.T) {
	registry := NewRegistry()

	// Register test providers
	providers := []*MockLanguageProvider{
		{lang: "go", aliases: []string{"golang"}, extensions: []string{".go", ".mod"}},
		{lang: "python", aliases: []string{"py"}, extensions: []string{".py", ".pyx"}},
		{lang: "javascript", aliases: []string{"js", "node"}, extensions: []string{".js", ".mjs", ".cjs"}},
	}

	for _, p := range providers {
		err := registry.RegisterProvider(p)
		if err != nil {
			t.Fatalf("Failed to register provider %s: %v", p.lang, err)
		}
	}

	tests := []struct {
		name       string
		identifier string
		wantLang   string
		wantErr    bool
	}{
		// Direct language names
		{"direct go", "go", "go", false},
		{"direct python", "python", "python", false},
		{"direct javascript", "javascript", "javascript", false},

		// Aliases
		{"golang alias", "golang", "go", false},
		{"py alias", "py", "python", false},
		{"js alias", "js", "javascript", false},
		{"node alias", "node", "javascript", false},

		// Extensions
		{"go extension", ".go", "go", false},
		{"mod extension", ".mod", "go", false},
		{"py extension", ".py", "python", false},
		{"js extension", ".js", "javascript", false},
		{"mjs extension", ".mjs", "javascript", false},

		// Extension without dot
		{"go ext no dot", "go", "go", false}, // This should match the language name, not extension

		// Non-existent
		{"non-existent lang", "nonexistent", "", true},
		{"non-existent extension", ".unknown", "", true},
		{"empty identifier", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.GetProvider(tt.identifier)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected non-nil provider")
				return
			}

			if provider.Lang() != tt.wantLang {
				t.Errorf("Expected language %s, got %s", tt.wantLang, provider.Lang())
			}
		})
	}
}

// Test file-based provider retrieval
func TestRegistry_GetProviderForFile(t *testing.T) {
	registry := NewRegistry()

	// Register providers
	err := registry.RegisterProvider(&MockLanguageProvider{
		lang: "go", aliases: []string{"golang"}, extensions: []string{".go", ".mod"},
	})
	if err != nil {
		t.Fatalf("Failed to register Go provider: %v", err)
	}

	err = registry.RegisterProvider(&MockLanguageProvider{
		lang: "python", aliases: []string{"py"}, extensions: []string{".py"},
	})
	if err != nil {
		t.Fatalf("Failed to register Python provider: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		wantLang string
		wantErr  bool
	}{
		{"go file", "main.go", "go", false},
		{"go mod file", "go.mod", "go", false},
		{"python file", "script.py", "python", false},
		{"file with path", "/path/to/main.go", "go", false},
		{"file with complex path", "/very/long/path/to/script.py", "python", false},
		{"no extension", "filename", "", true},
		{"empty filename", "", "", true},
		{"unknown extension", "file.unknown", "", true},
		{"dot file", ".gitignore", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.GetProviderForFile(tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected non-nil provider")
				return
			}

			if provider.Lang() != tt.wantLang {
				t.Errorf("Expected language %s, got %s", tt.wantLang, provider.Lang())
			}
		})
	}
}

// Test provider unregistration
func TestRegistry_UnregisterProvider(t *testing.T) {
	registry := NewRegistry()

	// Register test provider
	provider := &MockLanguageProvider{
		lang:       "test",
		aliases:    []string{"test-alias"},
		extensions: []string{".test"},
	}
	err := registry.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Verify registration
	_, err = registry.GetProvider("test")
	if err != nil {
		t.Fatalf("Provider should be registered: %v", err)
	}

	_, err = registry.GetProvider("test-alias")
	if err != nil {
		t.Fatalf("Provider alias should be registered: %v", err)
	}

	_, err = registry.GetProviderForFile("test.test")
	if err != nil {
		t.Fatalf("Provider extension should be registered: %v", err)
	}

	// Unregister
	err = registry.Unregister("test")
	if err != nil {
		t.Fatalf("Failed to unregister provider: %v", err)
	}

	// Verify removal
	_, err = registry.GetProvider("test")
	if err == nil {
		t.Error("Provider should be unregistered")
	}

	_, err = registry.GetProvider("test-alias")
	if err == nil {
		t.Error("Provider alias should be unregistered")
	}

	_, err = registry.GetProviderForFile("test.test")
	if err == nil {
		t.Error("Provider extension should be unregistered")
	}

	// Test unregistering non-existent provider
	err = registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

// Test registry introspection methods
func TestRegistry_IntrospectionMethods(t *testing.T) {
	registry := NewRegistry()

	// Initially empty
	if len(registry.ListProviders()) != 0 {
		t.Error("Expected empty provider list initially")
	}

	if len(registry.ListExtensions()) != 0 {
		t.Error("Expected empty extension list initially")
	}

	// Register providers
	providers := []*MockLanguageProvider{
		{lang: "go", aliases: []string{"golang"}, extensions: []string{".go", ".mod"}},
		{lang: "python", aliases: []string{"py"}, extensions: []string{".py"}},
		{lang: "rust", aliases: []string{"rs"}, extensions: []string{".rs"}},
	}

	for _, p := range providers {
		err := registry.RegisterProvider(p)
		if err != nil {
			t.Fatalf("Failed to register provider %s: %v", p.lang, err)
		}
	}

	// Test ListProviders
	providerList := registry.ListProviders()
	if len(providerList) != len(providers) {
		t.Errorf("Expected %d providers, got %d", len(providers), len(providerList))
	}

	expectedProviders := []string{"go", "python", "rust"}
	for _, expected := range expectedProviders {
		found := slices.Contains(providerList, expected)
		if !found {
			t.Errorf("Provider %s not found in list", expected)
		}
	}

	// Test ListExtensions
	extensionList := registry.ListExtensions()
	expectedExtensions := []string{".go", ".mod", ".py", ".rs"}
	if len(extensionList) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensionList))
	}

	for _, expected := range expectedExtensions {
		found := slices.Contains(extensionList, expected)
		if !found {
			t.Errorf("Extension %s not found in list", expected)
		}
	}

	// Test HasProvider
	tests := []struct {
		identifier string
		expected   bool
	}{
		{"go", true},
		{"golang", true},
		{".go", true},
		{"python", true},
		{"py", true},
		{".py", true},
		{"nonexistent", false},
		{".unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		has := registry.HasProvider(tt.identifier)
		if has != tt.expected {
			t.Errorf("HasProvider(%s) = %v, want %v", tt.identifier, has, tt.expected)
		}
	}

	// Test GetProviderInfo
	info, err := registry.GetProviderInfo("go")
	if err != nil {
		t.Errorf("Failed to get provider info: %v", err)
	} else {
		if info.Name != "go" {
			t.Errorf("Expected name 'go', got '%s'", info.Name)
		}
		if len(info.Aliases) == 0 {
			t.Error("Expected non-empty aliases")
		}
		if len(info.Extensions) == 0 {
			t.Error("Expected non-empty extensions")
		}
	}

	// Test GetProviderInfo for non-existent provider
	_, err = registry.GetProviderInfo("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

// Test registry clear functionality
func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	// Register providers
	err := registry.RegisterProvider(&MockLanguageProvider{
		lang: "test", aliases: []string{"test-alias"}, extensions: []string{".test"},
	})
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Verify registration
	if len(registry.ListProviders()) == 0 {
		t.Error("Expected non-empty provider list")
	}

	// Clear
	registry.Clear()

	// Verify clearing
	if len(registry.ListProviders()) != 0 {
		t.Error("Expected empty provider list after clear")
	}

	if len(registry.ListExtensions()) != 0 {
		t.Error("Expected empty extension list after clear")
	}

	if registry.HasProvider("test") {
		t.Error("Provider should be removed after clear")
	}
}

// Test extension normalization
func TestRegistry_ExtensionNormalization(t *testing.T) {
	registry := NewRegistry()

	// Test extension normalization during registration
	provider := &MockLanguageProvider{
		lang:       "test",
		aliases:    []string{},
		extensions: []string{"test", ".test2"}, // Mixed with and without dots
	}

	err := registry.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Both should work
	_, err = registry.GetProvider(".test")
	if err != nil {
		t.Errorf("Should find provider by normalized extension .test: %v", err)
	}

	_, err = registry.GetProvider(".test2")
	if err != nil {
		t.Errorf("Should find provider by extension .test2: %v", err)
	}

	// Test GetProviderByExtension (deprecated)
	_, err = registry.GetProviderByExtension("test")
	if err != nil {
		t.Errorf("Should find provider by extension without dot: %v", err)
	}

	_, err = registry.GetProviderByExtension(".test2")
	if err != nil {
		t.Errorf("Should find provider by extension with dot: %v", err)
	}
}

// Test advanced concurrent scenarios
func TestRegistry_ConcurrentComplexOperations(t *testing.T) {
	registry := NewRegistry()

	// Pre-populate with some providers
	baseProviders := []*MockLanguageProvider{
		{lang: "base1", aliases: []string{"base1-alias"}, extensions: []string{".base1"}},
		{lang: "base2", aliases: []string{"base2-alias"}, extensions: []string{".base2"}},
	}

	for _, p := range baseProviders {
		err := registry.RegisterProvider(p)
		if err != nil {
			t.Fatalf("Failed to register base provider: %v", err)
		}
	}

	// Test concurrent mixed operations
	var wg sync.WaitGroup
	numGoroutines := 20
	errors := make(chan error, numGoroutines*3) // 3 operations per goroutine

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Register new provider
			provider := &MockLanguageProvider{
				lang:       fmt.Sprintf("lang%d", id),
				aliases:    []string{fmt.Sprintf("alias%d", id)},
				extensions: []string{fmt.Sprintf(".ext%d", id)},
			}

			err := registry.RegisterProvider(provider)
			if err != nil {
				errors <- fmt.Errorf("register %d: %v", id, err)
				return
			}

			// Retrieve by different methods
			_, err = registry.GetProvider(fmt.Sprintf("lang%d", id))
			if err != nil {
				errors <- fmt.Errorf("get by name %d: %v", id, err)
				return
			}

			_, err = registry.GetProvider(fmt.Sprintf("alias%d", id))
			if err != nil {
				errors <- fmt.Errorf("get by alias %d: %v", id, err)
				return
			}

			_, err = registry.GetProviderForFile(fmt.Sprintf("test%d.ext%d", id, id))
			if err != nil {
				errors <- fmt.Errorf("get by file %d: %v", id, err)
				return
			}
		}(i)
	}

	// Concurrent list operations
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.ListProviders()
			_ = registry.ListExtensions()
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify final state
	providers := registry.ListProviders()
	expectedCount := len(baseProviders) + numGoroutines
	if len(providers) != expectedCount {
		t.Errorf("Expected %d providers, got %d", expectedCount, len(providers))
	}
}

// Test plugin loading simulation
func TestRegistry_PluginLoadingSimulation(t *testing.T) {
	registry := NewRegistry()

	t.Run("LoadPlugin with non-existent file", func(t *testing.T) {
		err := registry.LoadPlugin("nonexistent.so")
		if err == nil {
			t.Error("Expected error for non-existent plugin file")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected file not exist error, got: %v", err)
		}
	})

	t.Run("LoadPlugin with empty path", func(t *testing.T) {
		err := registry.LoadPlugin("")
		if err == nil {
			t.Error("Expected error for empty plugin path")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Expected empty path error, got: %v", err)
		}
	})

	t.Run("LoadPluginsFromDir with non-existent directory", func(t *testing.T) {
		err := registry.LoadPluginsFromDir("/nonexistent/directory")
		// Should not error for non-existent directory
		if err != nil {
			t.Errorf("Should not error for non-existent directory: %v", err)
		}
	})

	t.Run("isPluginFile detection", func(t *testing.T) {
		tests := []struct {
			filename string
			expected bool
		}{
			{"plugin.so", true},
			{"plugin.dll", true},
			{"plugin.dylib", true},
			{"plugin.txt", false},
			{"plugin", false},
			{"", false},
			{".so", true},
			{".dll", true},
			{".dylib", true},
		}

		for _, tt := range tests {
			result := registry.isPluginFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isPluginFile(%s) = %v, want %v", tt.filename, result, tt.expected)
			}
		}
	})
}

// Test default registry functions
func TestDefaultRegistryFunctions(t *testing.T) {
	// Clear default registry first
	DefaultRegistry.Clear()

	// Test convenience functions
	provider := &MockLanguageProvider{
		lang: "test", aliases: []string{"test-alias"}, extensions: []string{".test"},
	}

	err := RegisterProvider(provider)
	if err != nil {
		t.Errorf("Failed to register provider via convenience function: %v", err)
	}

	// Test other convenience functions
	_, err = GetProvider("test")
	if err != nil {
		t.Errorf("Failed to get provider via convenience function: %v", err)
	}

	providers := ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	_, err = GetProviderForFile("test.test")
	if err != nil {
		t.Errorf("Failed to get provider for file via convenience function: %v", err)
	}

	if !HasProvider("test") {
		t.Error("HasProvider should return true for registered provider")
	}

	// Clean up
	DefaultRegistry.Clear()
}

// Test deprecated functions for backward compatibility
func TestDeprecatedFunctions(t *testing.T) {
	DefaultRegistry.Clear()

	provider := &MockLanguageProvider{
		lang: "test", aliases: []string{}, extensions: []string{".test"},
	}

	// Test deprecated Register function
	err := Register(provider)
	if err != nil {
		t.Errorf("Deprecated Register function failed: %v", err)
	}

	// Test deprecated GetProviderByExtension function
	_, err = GetProviderByExtension(".test")
	if err != nil {
		t.Errorf("Deprecated GetProviderByExtension function failed: %v", err)
	}

	DefaultRegistry.Clear()
}

// Stress test with many providers
func TestRegistry_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	registry := NewRegistry()
	numProviders := 1000

	// Register many providers
	for i := range numProviders {
		provider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}

		err := registry.RegisterProvider(provider)
		if err != nil {
			t.Fatalf("Failed to register provider %d: %v", i, err)
		}
	}

	// Verify all registered
	providers := registry.ListProviders()
	if len(providers) != numProviders {
		t.Errorf("Expected %d providers, got %d", numProviders, len(providers))
	}

	// Test random access
	for i := range 100 {
		idx := i * numProviders / 100 // Spread across the range

		_, err := registry.GetProvider(fmt.Sprintf("lang%d", idx))
		if err != nil {
			t.Errorf("Failed to get provider lang%d: %v", idx, err)
		}

		_, err = registry.GetProvider(fmt.Sprintf("alias%d", idx))
		if err != nil {
			t.Errorf("Failed to get provider by alias%d: %v", idx, err)
		}

		_, err = registry.GetProviderForFile(fmt.Sprintf("test%d.ext%d", idx, idx))
		if err != nil {
			t.Errorf("Failed to get provider for file with ext%d: %v", idx, err)
		}
	}
}

// Enhanced benchmarks
func BenchmarkRegistry_RegisterProvider(b *testing.B) {
	registry := NewRegistry()

	for i := 0; b.Loop(); i++ {
		provider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}

		err := registry.RegisterProvider(provider)
		if err != nil {
			b.Errorf("Registration failed: %v", err)
		}

		// Clean up to avoid memory growth
		if i%100 == 99 {
			registry.Clear()
		}
	}
}

func BenchmarkRegistry_GetProviderColdCache(b *testing.B) {
	registry := NewRegistry()

	// Pre-register providers
	for i := range 1000 {
		provider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}
		_ = registry.RegisterProvider(provider)
	}

	for i := 0; b.Loop(); i++ {
		idx := i % 1000
		_, err := registry.GetProvider(fmt.Sprintf("lang%d", idx))
		if err != nil {
			b.Errorf("GetProvider failed: %v", err)
		}
	}
}

func BenchmarkRegistry_GetProviderForFile(b *testing.B) {
	registry := NewRegistry()

	// Register common file types
	providers := []*MockLanguageProvider{
		{lang: "go", extensions: []string{".go"}},
		{lang: "python", extensions: []string{".py"}},
		{lang: "javascript", extensions: []string{".js"}},
		{lang: "rust", extensions: []string{".rs"}},
		{lang: "cpp", extensions: []string{".cpp", ".cc", ".cxx"}},
	}

	for _, p := range providers {
		_ = registry.RegisterProvider(p)
	}

	files := []string{"main.go", "script.py", "app.js", "lib.rs", "program.cpp"}

	for i := 0; b.Loop(); i++ {
		file := files[i%len(files)]
		_, err := registry.GetProviderForFile(file)
		if err != nil {
			b.Errorf("GetProviderForFile failed: %v", err)
		}
	}
}

func BenchmarkRegistry_ConcurrentAccess(b *testing.B) {
	registry := NewRegistry()

	// Pre-register providers
	for i := range 100 {
		provider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}
		_ = registry.RegisterProvider(provider)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % 100
			_, err := registry.GetProvider(fmt.Sprintf("lang%d", idx))
			if err != nil {
				b.Errorf("Concurrent GetProvider failed: %v", err)
			}
			i++
		}
	})
}
